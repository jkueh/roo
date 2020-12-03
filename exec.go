package main

import (
	"os"
	"os/exec"
)

func executeCommand(commands ...string) error {
	cmd := exec.Command(commands[0], commands[1:]...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}
