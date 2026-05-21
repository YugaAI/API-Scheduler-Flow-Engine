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

// GitPullAction menjalankan git pull di repository directory.
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
		return "", &ScriptError{
			Err:       fmt.Errorf("invalid config for git_pull: %w", err),
			Retryable: false,
			Reason:    "invalid_config",
		}
	}

	if cfg.RepoDir == "" {
		return "", &ScriptError{
			Err:       errors.New("repo_dir is required for git_pull"),
			Retryable: false,
			Reason:    "missing_repo_dir",
		}
	}

	// Validasi repo_dir exist sebelum exec — fail fast
	if _, err := os.Stat(cfg.RepoDir); err != nil {
		if os.IsNotExist(err) {
			return "", &ScriptError{
				Err:       fmt.Errorf("repo_dir does not exist: %s", cfg.RepoDir),
				Retryable: false,
				Reason:    "repo_dir_not_found",
			}
		}
		return "", &ScriptError{
			Err:       fmt.Errorf("repo_dir inaccessible: %w", err),
			Retryable: false,
			Reason:    "repo_dir_inaccessible",
		}
	}

	// resolveBinary dari shared.go — tidak hardcode path "git"
	gitBin, err := resolveBinary("git")
	if err != nil {
		return "", err
	}

	args := []string{"pull"}
	if cfg.Branch != "" {
		args = append(args, "origin", cfg.Branch)
	}

	cmd := exec.CommandContext(ctx, gitBin, args...)
	cmd.Dir = cfg.RepoDir

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
				Err:       fmt.Errorf("git pull cancelled: %w", ctxErr),
				Retryable: false,
				Reason:    "context_cancelled",
			}
		}

		// git pull dengan exit code non-zero — bisa transient (network, conflict)
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return output.String(), &ScriptError{
				Err:       fmt.Errorf("git pull exited with code %d: %w", exitErr.ExitCode(), runErr),
				Retryable: true,
				Reason:    fmt.Sprintf("exit_code_%d", exitErr.ExitCode()),
			}
		}

		return output.String(), &ScriptError{
			Err:       fmt.Errorf("git pull failed: %w", runErr),
			Retryable: false,
			Reason:    "execution_error",
		}
	}

	return output.String(), nil
}
