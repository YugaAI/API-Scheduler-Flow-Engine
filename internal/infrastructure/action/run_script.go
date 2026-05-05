package action

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// RunScriptAction implements the Action interface to run shell commands.
type RunScriptAction struct{}

// RunScriptConfig defines the expected JSON configuration for the run_script action.
type RunScriptConfig struct {
	Command string `json:"command"`
	WorkDir string `json:"workdir,omitempty"`
}

func (a *RunScriptAction) Name() string {
	return "run_script"
}

func (a *RunScriptAction) Execute(ctx context.Context, config json.RawMessage) (string, error) {
	var cfg RunScriptConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", fmt.Errorf("invalid config for run_script: %w", err)
	}

	if cfg.Command == "" {
		return "", fmt.Errorf("command is required for run_script")
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", cfg.Command)
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n-- STDERR --\n" + stderr.String()
	}

	if err != nil {
		return output, fmt.Errorf("command execution failed: %w", err)
	}

	return output, nil
}
