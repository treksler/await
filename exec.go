package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

func Exec(cmd string, args []string) {
        path, err := exec.LookPath(cmd)
        if err != nil {
                log.Fatal(err)
        }
        if err := syscall.Exec(path, args, os.Environ()); err != nil {
                log.Fatal(err)
        }
}
