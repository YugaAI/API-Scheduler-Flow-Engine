package action

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// ScriptError merepresentasikan error dari eksekusi action dengan metadata retryability.
// Diimplementasikan oleh semua action — bukan hanya run_script.
type ScriptError struct {
	Err       error
	Retryable bool
	Reason    string
}

func (e *ScriptError) Error() string {
	return fmt.Sprintf("action error [retryable=%v, reason=%s]: %v", e.Retryable, e.Reason, e.Err)
}

func (e *ScriptError) Unwrap() error {
	return e.Err
}

// IsRetryable mengecek apakah error dari action.Execute() adalah retryable.
// Exported — dipakai oleh executor_service di application layer.
// Single source of truth untuk retryability di seluruh codebase.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// ScriptError adalah sumber of truth utama — prioritas tertinggi
	var scriptErr *ScriptError
	if errors.As(err, &scriptErr) {
		return scriptErr.Retryable
	}

	// Fallback untuk error yang tidak dibungkus ScriptError
	if errors.Is(err, exec.ErrNotFound) {
		return false
	}
	if errors.Is(err, os.ErrPermission) {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return true
}

// resolveShell mendeteksi shell yang tersedia di environment secara berurutan.
// Single source of truth — dipakai oleh semua action yang butuh shell execution.
// Menghindari hardcode "sh" yang tidak tersedia di distroless/scratch images.
func resolveShell() (string, error) {
	candidates := []string{"/bin/bash", "/bin/sh", "/bin/ash", "/usr/bin/bash", "/usr/bin/sh"}
	for _, shell := range candidates {
		if _, err := os.Stat(shell); err == nil {
			return shell, nil
		}
	}

	// Fallback via PATH — untuk environment non-standard
	if path, err := exec.LookPath("bash"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("sh"); err == nil {
		return path, nil
	}

	return "", &ScriptError{
		Err:       errors.New("no shell available in container environment (tried: bash, sh, ash)"),
		Retryable: false,
		Reason:    "no_shell_available",
	}
}

// resolveBinary mencari binary di absolute path candidates atau via PATH.
// Dipakai oleh git_pull, docker_build, docker_push untuk validasi binary sebelum exec.
func resolveBinary(name string) (string, error) {
	// Cek absolute path dulu — lebih reliable di container
	candidates := map[string][]string{
		"git":    {"/usr/bin/git", "/usr/local/bin/git", "/bin/git"},
		"docker": {"/usr/bin/docker", "/usr/local/bin/docker"},
	}

	if paths, ok := candidates[name]; ok {
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	// Fallback via PATH
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	return "", &ScriptError{
		Err:       fmt.Errorf("binary %q not found in PATH or known locations", name),
		Retryable: false,
		Reason:    "binary_not_found",
	}
}
