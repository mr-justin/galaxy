package utils

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func SSHCmd(host string, command string, background bool, debug bool) {

	// Assuming the deployed hosts will have a galaxy user created at some
	// point
	username := "galaxy"
	if strings.Contains(host, "127.0.0.1:2222") {
		username = "vagrant"
	}

	port := "22"
	hostPort := strings.SplitN(host, ":", 2)
	if len(hostPort) > 1 {
		host, port = hostPort[0], hostPort[1]
	}

	cmd := exec.Command("/usr/bin/ssh",
		//"-i", config.PrivateKey,
		"-o", "RequestTTY=yes",
		username+"@"+host,
		"-p", port,
		"-C", "/bin/bash", "-i", "-l", "-c", "'source .bashrc && "+command+"'")

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Connecting to %s...\n", host)
	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0

			// This works on both Unix and Windows. Although package
			// syscall is generally platform dependent, WaitStatus is
			// defined for both Unix and Windows and in both cases has
			// an ExitStatus() method with the same signature.
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				fmt.Printf("Command finished with error: %v\n", err)
				os.Exit(status.ExitStatus())
			}
		} else {
			fmt.Printf("Command finished with error: %v\n", err)
			os.Exit(1)
		}
	}

}