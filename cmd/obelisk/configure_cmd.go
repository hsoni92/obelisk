package main

import (
	"fmt"
	"os"
)

func cmdConfigure(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: obelisk configure deployment|github")
		os.Exit(1)
	}
	switch args[0] {
	case "deployment":
		reexecWithSudoForK3s()
		cmdConfigureDeployment(args[1:])
	case "github":
		cmdConfigureGitHub(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown configure target %q (try: deployment, github)\n", args[0])
		os.Exit(1)
	}
}

func cmdConfigureGitHub(args []string) {
	reexecWithSudoForK3s()
	cmdGitHubOAuthConfigure(args)
}
