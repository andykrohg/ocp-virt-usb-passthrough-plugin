//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

// isElevatedWindows checks if the process is running with Administrator privileges
func isElevatedWindows() bool {
	var sid *windows.SID

	// Get the well-known SID for the Administrators group
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	// Check if the current process token is a member of the Administrators group
	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}

	return member
}

// relaunchElevatedWindows re-launches the current process with elevated privileges on Windows
func relaunchElevatedWindows() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Convert args to UTF16 for Windows
	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	exePtr, _ := syscall.UTF16PtrFromString(executable)

	// Build arguments string (skip first arg which is the executable path)
	var argsStr string
	if len(os.Args) > 1 {
		for i, arg := range os.Args[1:] {
			if i > 0 {
				argsStr += " "
			}
			// Quote arguments that contain spaces
			if containsSpace(arg) {
				argsStr += fmt.Sprintf("\"%s\"", arg)
			} else {
				argsStr += arg
			}
		}
	}

	var argsPtr *uint16
	if argsStr != "" {
		argsPtr, _ = syscall.UTF16PtrFromString(argsStr)
	}

	// ShellExecute with "runas" verb triggers UAC prompt
	err = windows.ShellExecute(0, verbPtr, exePtr, argsPtr, nil, windows.SW_NORMAL)
	if err != nil {
		return fmt.Errorf("failed to elevate: %w", err)
	}

	// Exit the non-elevated process
	os.Exit(0)
	return nil
}

func containsSpace(s string) bool {
	for _, c := range s {
		if c == ' ' {
			return true
		}
	}
	return false
}

// init function is called before main() on Windows builds
func init() {
	// Override the platform-independent functions with Windows-specific ones
	isElevatedFunc = isElevatedWindows
	relaunchElevatedFunc = relaunchElevatedWindows
}
