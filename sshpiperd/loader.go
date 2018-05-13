package main

import (
	_ "github.com/tg123/sshpiper/sshpiperd/upstream/mysql"
	_ "github.com/tg123/sshpiper/sshpiperd/upstream/workingdir"

	_ "github.com/tg123/sshpiper/sshpiperd/challenger/pam"
	_ "github.com/tg123/sshpiper/sshpiperd/challenger/welcometext"

	_ "github.com/tg123/sshpiper/sshpiperd/auditor/typescriptlogger"
	_ "github.com/tg123/sshpiper/sshpiperd/auditor/externalfilterlogger"
)
