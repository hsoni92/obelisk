package controller

import (
	"context"
	"os/exec"
)

// commandRunner runs helm/kubectl; tests may replace it.
var commandRunner = func(ctx context.Context, name string, args ...string) (out []byte, err error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = kubeconfigEnv()
	return cmd.CombinedOutput()
}
