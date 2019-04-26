package main

import (
	"bytes"
	"io"
	"os/exec"
)

func UnixPipe() io.Writer {
	var buf bytes.Buffer
	cmd := exec.Command("netstat", "-anp")
	cmd.Stdout = &buf
	cmd.Start()
	cmd.Wait()
	return &buf
}
