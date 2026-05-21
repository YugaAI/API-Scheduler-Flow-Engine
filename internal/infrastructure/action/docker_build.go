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

// DockerBuildAction menjalankan docker build command.
type DockerBuildAction struct{}

type DockerBuildConfig struct {
	ImageName      string `json:"image_name"`
	Tag            string `json:"tag,omitempty"`
	DockerfilePath string `json:"dockerfile_path,omitempty"`

	// ContextDir harus diisi secara eksplisit — tidak ada default "." yang berbahaya di container.
	// Relative path berbahaya karena WORKDIR container adalah "/", bukan project root.
	ContextDir string `json:"context_dir"`
}

func (a *DockerBuildAction) Name() string {
	return "docker_build"
}

func (a *DockerBuildAction) Execute(ctx context.Context, config json.RawMessage) (string, error) {
	var cfg DockerBuildConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", &ScriptError{
			Err:       fmt.Errorf("invalid config for docker_build: %w", err),
			Retryable: false,
			Reason:    "invalid_config",
		}
	}

	if cfg.ImageName == "" {
		return "", &ScriptError{
			Err:       errors.New("image_name is required for docker_build"),
			Retryable: false,
			Reason:    "missing_image_name",
		}
	}

	// context_dir wajib diisi secara eksplisit — cegah default "." yang berbahaya di container
	if cfg.ContextDir == "" {
		return "", &ScriptError{
			Err:       errors.New("context_dir is required for docker_build — do not rely on default '.' inside container"),
			Retryable: false,
			Reason:    "missing_context_dir",
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

	args := []string{"build", "-t", imageRef}
	if cfg.DockerfilePath != "" {
		args = append(args, "-f", cfg.DockerfilePath)
	}
	args = append(args, cfg.ContextDir)

	cmd := exec.CommandContext(ctx, dockerBin, args...)

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
				Err:       fmt.Errorf("docker build cancelled: %w", ctxErr),
				Retryable: false,
				Reason:    "context_cancelled",
			}
		}

		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return output.String(), &ScriptError{
				Err:       fmt.Errorf("docker build exited with code %d: %w", exitErr.ExitCode(), runErr),
				Retryable: true,
				Reason:    fmt.Sprintf("exit_code_%d", exitErr.ExitCode()),
			}
		}

		return output.String(), &ScriptError{
			Err:       fmt.Errorf("docker build failed: %w", runErr),
			Retryable: false,
			Reason:    "execution_error",
		}
	}

	return output.String(), nil
}
