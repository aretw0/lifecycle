package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/aretw0/lifecycle/pkg/proc"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "child" {
		fmt.Println("Child running...")
		time.Sleep(1 * time.Hour)
		return
	}

	cmd := exec.Command(os.Args[0], "child")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Parent starting child via proc.Start...")
	if err := proc.Start(cmd); err != nil {
		fmt.Printf("Failed to start: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Child PID: %d. Parent exiting now.\n", cmd.Process.Pid)
	// Parent exits immediately.
	// We expect OS to kill child.
}
