package action

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type DockerBuildAction struct{}

type DockerBuildConfig struct {
	ImageName      string `json:"image_name"`
	Tag            string `json:"tag,omitempty"`
	DockerfilePath string `json:"dockerfile_path,omitempty"`
	ContextDir     string `json:"context_dir,omitempty"`
}

func (a *DockerBuildAction) Name() string {
	return "docker_build"
}

func (a *DockerBuildAction) Execute(ctx context.Context, config json.RawMessage) (string, error) {
	var cfg DockerBuildConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", fmt.Errorf("invalid config for docker_build: %w", err)
	}

	if cfg.ImageName == "" {
		return "", fmt.Errorf("image_name is required for docker_build")
	}

	tag := cfg.Tag
	if tag == "" {
		tag = "latest"
	}
	imageRef := fmt.Sprintf("%s:%s", cfg.ImageName, tag)

	args := []string{"build", "-t", imageRef}

	if cfg.DockerfilePath != "" {
		args = append(args, "-f", cfg.DockerfilePath)
	}

	ctxDir := cfg.ContextDir
	if ctxDir == "" {
		ctxDir = "."
	}
	args = append(args, ctxDir)

	cmd := exec.CommandContext(ctx, "docker", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n-- STDERR --\n" + stderr.String()
	}

	if err != nil {
		return output, fmt.Errorf("docker build failed: %w", err)
	}

	return output, nil
}
