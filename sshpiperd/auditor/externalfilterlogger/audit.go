package externalfilterlogger

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	msgChannelData = 94
)

type filePtyLogger struct {
	typescript *os.File
	timing     *os.File
	log           *os.File
	filtercmd     *exec.Cmd
	cmdStdinPipe  io.WriteCloser
	cmdStdoutPipe io.ReadCloser
	reader        io.Reader

	oldtime time.Time
}

func newFilePtyLogger(outputdir string, filterbin string) (*filePtyLogger, error) {

	now := time.Now()

	cmd := exec.Command(filterbin)
	cmd_stdout, _ := cmd.StdoutPipe()
	cmd_stdin, _ := cmd.StdinPipe()
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(cmd_stdout)

	filename := fmt.Sprintf("%d", now.Unix())

	typescript, err := os.OpenFile(path.Join(outputdir, fmt.Sprintf("%v.typescript", filename)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		return nil, err
	}

	timing, err := os.OpenFile(path.Join(outputdir, fmt.Sprintf("%v.timing", filename)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		return nil, err
	}

	log, err := os.OpenFile(path.Join(outputdir, fmt.Sprintf("%v.log", filename)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		return nil, err
	}

	go func() {
		defer log.Close()
		for {
                        line, err := reader.ReadBytes('\n')
                        if err != nil && err != io.EOF {
                                fmt.Println(err.Error())
				break
                        }
                        if err == io.EOF && len(line) == 0{
                                break
                        }
			log.Write(line)
		}
	}()

	_, err = typescript.Write([]byte(fmt.Sprintf("Script started on %v\n", now.Format(time.ANSIC))))

	if err != nil {
		return nil, err
	}

	_, err = log.Write([]byte(fmt.Sprintf("Script started on %v\n", now.Format(time.ANSIC))))

	if err != nil {
		return nil, err
	}

	return &filePtyLogger{
		typescript: typescript,
		timing:     timing,
		log:           log,
		filtercmd:     cmd,
		cmdStdinPipe:  cmd_stdin,
		cmdStdoutPipe: cmd_stdout,
		reader:        reader,
		oldtime:    time.Now(),
	}, nil
}

func (l *filePtyLogger) loggingTty(conn ssh.ConnMetadata, msg []byte) ([]byte, error) {

	if msg[0] == msgChannelData {

		buf := msg[9:]

		now := time.Now()

		delta := now.Sub(l.oldtime)

		// see term-utils/script.c
		fmt.Fprintf(l.timing, "%v.%06v %v\n", int64(delta/time.Second), int64(delta/time.Microsecond), len(buf))

		l.oldtime = now

		_, err := l.typescript.Write(buf)

		if err != nil {
			return msg, err
		}

		_, err = l.cmdStdinPipe.Write(buf)

		if err != nil {
			return msg, err
		}

	}

	return msg, nil
}

func (l *filePtyLogger) Close() (err error) {
	_, err = l.typescript.Write([]byte(fmt.Sprintf("Script done on %v\n", time.Now().Format(time.ANSIC))))
	_, err = l.cmdStdinPipe.Write([]byte(fmt.Sprintf("Script done on %v\n", time.Now().Format(time.ANSIC))))

	l.typescript.Close()
	l.timing.Close()
        _ = l.filtercmd.Wait()

	return nil // TODO
}
