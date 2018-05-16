package externalfilterlogger

import (
	"log"
	"os"
	"path"

	"golang.org/x/crypto/ssh"

	"github.com/tg123/sshpiper/sshpiperd/auditor"
)

type plugin struct {
	Config struct {
		OutputDir string `long:"auditor-externalfilterlogger-outputdir" default:"/var/sshpiper" description:"Place where logged typescript files were saved"  env:"SSHPIPERD_AUDITOR_EXTERNALFILTERLOGGER_OUTPUTDIR"  ini-name:"auditor-typescriptlogger-outputdir"`
		Filter string `long:"auditor-externalfilterlogger-bin" default:"cat" description:"Filter program to pass the log"  env:"SSHPIPERD_AUDITOR_EXTERNALFILTERLOGGER_BIN"  ini-name:"auditor-externalfilterlogger-bin"`
	}
	logger *log.Logger
}

func (p *plugin) GetName() string {
	return "external-filter-logger"
}

func (p *plugin) GetOpts() interface{} {
	return &p.Config
}

func (p *plugin) Create(conn ssh.ConnMetadata) (auditor.Auditor, error) {
	dir := path.Join(p.Config.OutputDir, conn.User())
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return nil, err
	}
	filtercmd := p.Config.Filter

	return newFilePtyLogger(p.logger, dir, filtercmd)
}

func (p *plugin) Init(logger *log.Logger) error {
	p.logger = logger
	return nil
}

func (l *filePtyLogger) GetUpstreamHook() auditor.Hook {
	return l.loggingTty
}

func (l *filePtyLogger) GetDownstreamHook() auditor.Hook {
	return l.loggingDownstream
}

func init() {
	auditor.Register("external-filter-logger", new(plugin))
}
