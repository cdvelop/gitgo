package devflow

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// RunCommand executes a shell command
// It returns the output (trimmed) and an error if the command fails
func RunCommand(name string, args ...string) (string, error) {
	// Execute
	cmd := exec.Command(name, args...)
	outputBytes, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(outputBytes))

	if err != nil {
		cmdStr := name + " " + strings.Join(args, " ")
		return output, fmt.Errorf("command failed: %s\nError: %w\nOutput: %s", cmdStr, err, output)
	}

	return output, nil
}

// RunCommandSilent executes a command (alias for RunCommand now, as RunCommand is also silent on success)
// kept for backward compatibility if needed, or we can remove it.
// The previous implementation was identical except for logging.
func RunCommandSilent(name string, args ...string) (string, error) {
	return RunCommand(name, args...)
}

// RunShellCommand executes a shell command in a cross-platform way
// On Windows: uses cmd.exe /C
// On Unix (Linux/macOS): uses sh -c
func RunShellCommand(command string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return RunCommand("cmd.exe", "/C", command)
	default: // linux, darwin, etc.
		return RunCommand("sh", "-c", command)
	}
}
