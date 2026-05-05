package action

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type DockerPushAction struct{}

type DockerPushConfig struct {
	ImageName string `json:"image_name"`
	Tag       string `json:"tag,omitempty"`
	Registry  string `json:"registry,omitempty"`
}

func (a *DockerPushAction) Name() string {
	return "docker_push"
}

func (a *DockerPushAction) Execute(ctx context.Context, config json.RawMessage) (string, error) {
	var cfg DockerPushConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", fmt.Errorf("invalid config for docker_push: %w", err)
	}

	if cfg.ImageName == "" {
		return "", fmt.Errorf("image_name is required for docker_push")
	}

	tag := cfg.Tag
	if tag == "" {
		tag = "latest"
	}
	
	imageRef := fmt.Sprintf("%s:%s", cfg.ImageName, tag)
	if cfg.Registry != "" {
		imageRef = fmt.Sprintf("%s/%s", cfg.Registry, imageRef)
	}

	args := []string{"push", imageRef}

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
		return output, fmt.Errorf("docker push failed: %w", err)
	}

	return output, nil
}
