package action

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// DockerPushAction menjalankan docker push command.
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
		return "", &ScriptError{
			Err:       fmt.Errorf("invalid config for docker_push: %w", err),
			Retryable: false,
			Reason:    "invalid_config",
		}
	}

	if cfg.ImageName == "" {
		return "", &ScriptError{
			Err:       errors.New("image_name is required for docker_push"),
			Retryable: false,
			Reason:    "missing_image_name",
		}
	}

	// resolveBinary dari shared.go — tidak hardcode path "docker"
	dockerBin, err := resolveBinary("docker")
	if err != nil {
		return "", err
	}

	tag := cfg.Tag
	if tag == "" {
		tag = "latest"
	}

	imageRef := fmt.Sprintf("%s:%s", cfg.ImageName, tag)
	if cfg.Registry != "" {
		imageRef = fmt.Sprintf("%s/%s", cfg.Registry, imageRef)
	}

	cmd := exec.CommandContext(ctx, dockerBin, "push", imageRef)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	var output strings.Builder
	output.WriteString(stdout.String())
	if stderr.Len() > 0 {
		output.WriteString("\n-- STDERR --\n")
		output.WriteString(stderr.String())
	}

	if runErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return output.String(), &ScriptError{
				Err:       fmt.Errorf("docker push cancelled: %w", ctxErr),
				Retryable: false,
				Reason:    "context_cancelled",
			}
		}

		// docker push dengan exit code non-zero bisa transient (network, registry down)
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return output.String(), &ScriptError{
				Err:       fmt.Errorf("docker push exited with code %d: %w", exitErr.ExitCode(), runErr),
				Retryable: true,
				Reason:    fmt.Sprintf("exit_code_%d", exitErr.ExitCode()),
			}
		}

		return output.String(), &ScriptError{
			Err:       fmt.Errorf("docker push failed: %w", runErr),
			Retryable: false,
			Reason:    "execution_error",
		}
	}

	return output.String(), nil
}
