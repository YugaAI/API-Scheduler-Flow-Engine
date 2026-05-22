package action

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
		return "", &ScriptError{
			Err:       fmt.Errorf("invalid config for run_script: %w", err),
			Retryable: false,
			Reason:    "invalid_config",
		}
	}

	if cfg.Command == "" {
		return "", &ScriptError{
			Err:       errors.New("command is required for run_script"),
			Retryable: false,
			Reason:    "missing_command",
		}
	}

	// Sanitasi minimal — cegah null byte injection yang bisa bypass shell parsing
	// Tidak memvalidasi isi command karena seluruh string diserahkan ke shell interpreter.
	// Security enforcement ada di layer auth/role (siapa yang boleh create/trigger flow).
	if strings.ContainsRune(cfg.Command, 0) {
		return "", &ScriptError{
			Err:       errors.New("command contains null byte"),
			Retryable: false,
			Reason:    "invalid_command",
		}
	}

	// Validasi WorkDir sebelum exec — fail fast dengan pesan jelas
	if cfg.WorkDir != "" {
		if _, err := os.Stat(cfg.WorkDir); err != nil {
			if os.IsNotExist(err) {
				return "", &ScriptError{
					Err:       fmt.Errorf("workdir does not exist: %s", cfg.WorkDir),
					Retryable: false,
					Reason:    "workdir_not_found",
				}
			}
			return "", &ScriptError{
				Err:       fmt.Errorf("workdir inaccessible: %w", err),
				Retryable: false,
				Reason:    "workdir_inaccessible",
			}
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

	// Build output dengan strings.Builder — satu allocation, efisien
	var output strings.Builder
	output.WriteString(stdout.String())
	if stderr.Len() > 0 {
		output.WriteString("\n-- STDERR --\n")
		output.WriteString(stderr.String())
	}

	if runErr != nil {
		// Cek context cancellation/timeout — prioritas sebelum classify error lain
		if ctxErr := ctx.Err(); ctxErr != nil {
			return output.String(), &ScriptError{
				Err:       fmt.Errorf("command cancelled: %w", ctxErr),
				Retryable: false,
				Reason:    "context_cancelled",
			}
		}

		// ExitError = command jalan tapi exit code != 0 → retryable (transient failure)
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return output.String(), &ScriptError{
				Err:       fmt.Errorf("command exited with code %d: %w", exitErr.ExitCode(), runErr),
				Retryable: true,
				Reason:    fmt.Sprintf("exit_code_%d", exitErr.ExitCode()),
			}
		}

		// Error lain (structural) — binary tidak ada, permission, dll → non-retryable
		return output.String(), &ScriptError{
			Err:       fmt.Errorf("command execution failed: %w", runErr),
			Retryable: false,
			Reason:    "execution_error",
		}
	}

	return output.String(), nil
}
