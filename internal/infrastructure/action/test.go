package action

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type TestAction struct{}

type TestConfig struct {
	Command string `json:"command"`
	WorkDir string `json:"workdir,omitempty"`
}

func (a *TestAction) Name() string {
	return "test"
}

func (a *TestAction) Execute(ctx context.Context, config json.RawMessage) (string, error) {
	var cfg TestConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", fmt.Errorf("invalid config for test: %w", err)
	}

	if cfg.Command == "" {
		return "", fmt.Errorf("command is required for test")
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
		return output, fmt.Errorf("test failed: %w", err)
	}

	return output, nil
}
