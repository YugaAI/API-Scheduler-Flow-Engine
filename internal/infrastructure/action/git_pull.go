package action

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type GitPullAction struct{}

type GitPullConfig struct {
	RepoDir string `json:"repo_dir"`
	Branch  string `json:"branch,omitempty"`
}

func (a *GitPullAction) Name() string {
	return "git_pull"
}

func (a *GitPullAction) Execute(ctx context.Context, config json.RawMessage) (string, error) {
	var cfg GitPullConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", fmt.Errorf("invalid config for git_pull: %w", err)
	}

	if cfg.RepoDir == "" {
		return "", fmt.Errorf("repo_dir is required for git_pull")
	}

	args := []string{"pull"}
	if cfg.Branch != "" {
		args = append(args, "origin", cfg.Branch)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cfg.RepoDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n-- STDERR --\n" + stderr.String()
	}

	if err != nil {
		return output, fmt.Errorf("git pull failed: %w", err)
	}

	return output, nil
}
