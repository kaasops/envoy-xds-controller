package utils

import (
	"bytes"
	"os"
)

var prefix = []byte("running: ")

type CmdWriter struct {
	File *os.File
}

func (cw *CmdWriter) Write(p []byte) (n int, err error) {
	if bytes.HasPrefix(p, prefix) {
		return cw.File.Write(bytes.TrimPrefix(p, prefix))
	}
	return 0, nil
}
