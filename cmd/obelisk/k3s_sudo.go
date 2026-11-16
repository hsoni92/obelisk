package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/hsoni92/obelisk/internal/credentials"
)

// reexecWithSudoForK3s re-runs this command as root when the k3s kubeconfig
// is mode 0600. The GitHub wizard otherwise writes credentials under $HOME
// and kubectl cannot create Secret obelisk-github-oauth.
func reexecWithSudoForK3s() {
	if !credentials.K3sKubeconfigNeedsRoot() {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot find obelisk binary:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "k3s kubeconfig is root-only; re-running with sudo")
	cmd := exec.Command("sudo", append([]string{"--preserve-env", "--", exe}, os.Args[1:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			os.Exit(ee.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
