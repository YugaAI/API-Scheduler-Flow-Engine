package action

import (
	"fmt"
	"strings"
)

var allowedCommands = []string{
	"git",
	"docker",
	"docker-compose",
	"npm",
	"go",
	"make",
}

func validateCommand(command string, allowed []string) error {
	command = strings.TrimSpace(command)

	if command == "" {
		return fmt.Errorf("empty command")
	}

	// Block multiline command
	if strings.Contains(command, "\n") {
		return fmt.Errorf("multiline command is not allowed")
	}

	// Block dangerous shell operators
	forbidden := []string{
		";",
		"&&",
		"||",
		"|",
		">",
		">>",
		"<",
		"`",
		"$(",
	}

	for _, pattern := range forbidden {
		if strings.Contains(command, pattern) {
			return fmt.Errorf("forbidden shell operator detected: %s", pattern)
		}
	}

	fields := strings.Fields(command)
	if len(fields) == 0 {
		return fmt.Errorf("invalid command")
	}

	baseCommand := fields[0]

	for _, allowedCmd := range allowed {
		if baseCommand == allowedCmd {
			return nil
		}
	}

	return fmt.Errorf("command %q is not allowed", baseCommand)
}
