package main

import (
	"os"
	"os/exec"
)

func executeCommand(commands ...string) error {
	cmd := exec.Command("/usr/bin/env", commands...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}
