// Package cli provides a Go wrapper around the vietcli bash dispatcher.
// Every module calls the CLI for system-level operations (user management,
// plan provisioning, domain setup, etc.) rather than touching the DB directly.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Executor runs vietcli commands from Go with a configurable path and timeout.
type Executor struct {
	Path    string        // path to the vietcli dispatcher binary
	Timeout time.Duration // per-command deadline (default 30s)
}

// New returns an Executor. If path is empty, VIETCLI_PATH is checked, then
// VIETCLI_DEV_DIR/vietcli (for local dev), then /usr/local/vietcli/vietcli.
func New(path string) *Executor {
	if path == "" {
		path = os.Getenv("VIETCLI_PATH")
	}
	if path == "" {
		if d := os.Getenv("VIETCLI_DEV_DIR"); d != "" {
			path = d + "/vietcli"
		}
	}
	if path == "" {
		path = "/usr/local/vietcli/vietcli"
	}
	return &Executor{Path: path, Timeout: 30 * time.Second}
}

// Run executes "vietcli <command> [args...]" and returns stdout trimmed.
// stderr is included in the error message on failure.
func (e *Executor) Run(command string, args ...string) (string, error) {
	return e.RunContext(context.Background(), command, args...)
}

// RunContext is like Run but honours the caller's context deadline.
func (e *Executor) RunContext(ctx context.Context, command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, e.Timeout)
	defer cancel()

	all := append([]string{command}, args...)
	cmd := exec.CommandContext(ctx, "bash", append([]string{e.Path}, all...)...)

	// Forward the SQLite dev env so the CLI can reach the panel database.
	if _, ok := os.LookupEnv("VIETCLI_DB_SQLITE"); ok {
		cmd.Env = append(os.Environ(), "VIETCLI_DB_SQLITE="+os.Getenv("VIETCLI_DB_SQLITE"))
	}
	// Point the dispatcher at the dev scripts dir when set.
	if d := os.Getenv("VIETCLI_DEV_DIR"); d != "" {
		cmd.Env = append(cmd.Env, "VIETCLI_DEV_DIR="+d)
	}

	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		return "", fmt.Errorf("vietcli %s: %w%s", strings.Join(all, " "), err, stderr)
	}
	return strings.TrimSpace(string(out)), nil
}