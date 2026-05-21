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

// BuildAction menjalankan build command via shell.
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
		return "", &ScriptError{
			Err:       fmt.Errorf("invalid config for build: %w", err),
			Retryable: false,
			Reason:    "invalid_config",
		}
	}

	if cfg.Command == "" {
		return "", &ScriptError{
			Err:       errors.New("command is required for build"),
			Retryable: false,
			Reason:    "missing_command",
		}
	}

	// resolveShell dari shared.go — tidak hardcode "sh"
	shell, err := resolveShell()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, shell, "-c", cfg.Command)
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

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
				Err:       fmt.Errorf("build cancelled: %w", ctxErr),
				Retryable: false,
				Reason:    "context_cancelled",
			}
		}

		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return output.String(), &ScriptError{
				Err:       fmt.Errorf("build exited with code %d: %w", exitErr.ExitCode(), runErr),
				Retryable: true,
				Reason:    fmt.Sprintf("exit_code_%d", exitErr.ExitCode()),
			}
		}

		return output.String(), &ScriptError{
			Err:       fmt.Errorf("build failed: %w", runErr),
			Retryable: false,
			Reason:    "execution_error",
		}
	}

	return output.String(), nil
}
