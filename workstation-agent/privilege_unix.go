//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

// isElevatedUnix checks if the process is running with root privileges (UID 0)
func isElevatedUnix() bool {
	return os.Geteuid() == 0
}

// relaunchElevatedUnix re-launches the current process with sudo
func relaunchElevatedUnix() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build command with sudo
	args := append([]string{executable}, os.Args[1:]...)
	cmd := exec.Command("sudo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// init function is called before main() on non-Windows builds
func init() {
	// Override the platform-independent functions with Unix-specific ones
	isElevatedFunc = isElevatedUnix
	relaunchElevatedFunc = relaunchElevatedUnix
}
