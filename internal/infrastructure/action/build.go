package action

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type BuildAction struct{}

type BuildConfig struct {
	Command string `json:"command"`
	WorkDir string `json:"workdir,omitempty"`
}

func (a *BuildAction) Name() string {
	return "build"
}

func (a *BuildAction) Execute(ctx context.Context, config json.RawMessage) (string, error) {
	var cfg BuildConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", fmt.Errorf("invalid config for build: %w", err)
	}

	if cfg.Command == "" {
		return "", fmt.Errorf("command is required for build")
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
		return output, fmt.Errorf("build failed: %w", err)
	}

	return output, nil
}
