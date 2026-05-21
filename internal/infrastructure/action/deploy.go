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

// DeployAction menjalankan deploy command via shell.
type DeployAction struct{}

type DeployConfig struct {
	Command string `json:"command"`
	WorkDir string `json:"workdir,omitempty"`
}

func (a *DeployAction) Name() string {
	return "deploy"
}

func (a *DeployAction) Execute(ctx context.Context, config json.RawMessage) (string, error) {
	var cfg DeployConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", &ScriptError{
			Err:       fmt.Errorf("invalid config for deploy: %w", err),
			Retryable: false,
			Reason:    "invalid_config",
		}
	}

	if cfg.Command == "" {
		return "", &ScriptError{
			Err:       errors.New("command is required for deploy"),
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
				Err:       fmt.Errorf("deploy cancelled: %w", ctxErr),
				Retryable: false,
				Reason:    "context_cancelled",
			}
		}

		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return output.String(), &ScriptError{
				Err:       fmt.Errorf("deploy exited with code %d: %w", exitErr.ExitCode(), runErr),
				Retryable: true,
				Reason:    fmt.Sprintf("exit_code_%d", exitErr.ExitCode()),
			}
		}

		return output.String(), &ScriptError{
			Err:       fmt.Errorf("deploy failed: %w", runErr),
			Retryable: false,
			Reason:    "execution_error",
		}
	}

	return output.String(), nil
}
