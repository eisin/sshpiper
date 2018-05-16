package externalfilterlogger

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	msgChannelOpenConfirm = 91
	msgChannelData        = 94
	msgChannelClose       = 97
	msgChannelRequest     = 98
)

type filePtyLogger struct {
	logger      *log.Logger
	initialized bool
	outputdir   string
	filterbin   string

	typescript    *os.File
	timing        *os.File
	log           *os.File
	filtercmd     *exec.Cmd
	cmdStdinPipe  io.WriteCloser
	cmdStdoutPipe io.ReadCloser
	reader        io.Reader

	oldtime            time.Time
	sshClientSessionId map[uint32]uint32
	sshSessionPty      map[uint32]bool
}

func newFilePtyLogger(logger *log.Logger, outputdir string, filterbin string) (*filePtyLogger, error) {
	return &filePtyLogger{
		logger:             logger,
		initialized:        false,
		outputdir:          outputdir,
		filterbin:          filterbin,
		typescript:         nil,
		timing:             nil,
		log:                nil,
		filtercmd:          nil,
		cmdStdinPipe:       nil,
		cmdStdoutPipe:      nil,
		reader:             nil,
		oldtime:            time.Now(),
		sshClientSessionId: make(map[uint32]uint32),
		sshSessionPty:      make(map[uint32]bool),
	}, nil
}

func (l *filePtyLogger) initialize() (*filePtyLogger, error) {
	if l.initialized {
		return nil, nil
	}

	now := time.Now()

	filename := fmt.Sprintf("%d", now.Unix())

	typescript, err := os.OpenFile(path.Join(l.outputdir, fmt.Sprintf("%v.typescript", filename)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		return nil, err
	}

	timing, err := os.OpenFile(path.Join(l.outputdir, fmt.Sprintf("%v.timing", filename)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		return nil, err
	}

	log, err := os.OpenFile(path.Join(l.outputdir, fmt.Sprintf("%v.log", filename)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		return nil, err
	}

	cmd := exec.Command(l.filterbin)
	cmd_stdout, _ := cmd.StdoutPipe()
	cmd_stdin, _ := cmd.StdinPipe()
	if err := cmd.Start(); err != nil {
		//return nil, err
		l.logger.Println(err.Error())
	}

	reader := bufio.NewReader(cmd_stdout)

	go func() {
		defer log.Close()
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil && err != io.EOF {
				if !strings.Contains(err.Error(), "file already closed") {
					fmt.Println(err.Error())
				}
				break
			}
			if err == io.EOF && len(line) == 0 {
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

	l.typescript = typescript
	l.timing = timing
	l.log = log
	l.filtercmd = cmd
	l.cmdStdinPipe = cmd_stdin
	l.cmdStdoutPipe = cmd_stdout
	l.reader = reader
	l.oldtime = time.Now()
	l.initialized = true
	return nil, nil
}

func (l *filePtyLogger) loggingDownstream(conn ssh.ConnMetadata, msg []byte) ([]byte, error) {
	if msg[0] == msgChannelRequest {
		reqtype_length := binary.BigEndian.Uint32(msg[5:])
		reqtype := string(msg[9 : 9+reqtype_length])
		if reqtype == "pty-req" {
			sessionServerId := binary.BigEndian.Uint32(msg[1:])
			sessionClientId := l.sshClientSessionId[sessionServerId]
			l.sshSessionPty[sessionClientId] = true

			if _, err := l.initialize(); err != nil {
				l.logger.Println(err.Error())
			}
		}
	}
	if msg[0] == msgChannelClose {
		sessionServerId := binary.BigEndian.Uint32(msg[1:])
		if sessionClientId, ok := l.sshClientSessionId[sessionServerId]; ok {
			delete(l.sshSessionPty, sessionClientId)
			delete(l.sshClientSessionId, sessionServerId)
		}
	}
	return msg, nil
}

func (l *filePtyLogger) loggingTty(conn ssh.ConnMetadata, msg []byte) ([]byte, error) {

	if msg[0] == msgChannelOpenConfirm {
		sessionServerId := binary.BigEndian.Uint32(msg[1:])
		sessionClientId := binary.BigEndian.Uint32(msg[5:])
		l.sshClientSessionId[sessionServerId] = sessionClientId
	}

	if msg[0] == msgChannelData {
		if !l.initialized {
			return msg, nil
		}
		sessionServerId := binary.BigEndian.Uint32(msg[1:])
		if v, ok := l.sshSessionPty[sessionServerId]; !v || !ok {
			return msg, nil
		}

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

		if err != nil && strings.Contains(err.Error(), "file already closed") {
			// do nothing
		} else if err != nil {
			l.logger.Println(err)
			//return msg, err
		}

	}

	return msg, nil
}

func (l *filePtyLogger) Close() (err error) {
	if l.initialized {
		_, err = l.typescript.Write([]byte(fmt.Sprintf("Script done on %v\n", time.Now().Format(time.ANSIC))))
		_, err = l.cmdStdinPipe.Write([]byte(fmt.Sprintf("Script done on %v\n", time.Now().Format(time.ANSIC))))

		l.typescript.Close()
		l.timing.Close()
		l.cmdStdinPipe.Close()
		_ = l.filtercmd.Wait()
	}

	return nil // TODO
}
