// Package prompt provides minimal interactive CLI input for operator wizards.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// IsInteractive reports whether stdin is a TTY (wizard can run).
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// ReadLine prints prompt and reads a line. Default is used when the user enters nothing.
func ReadLine(w io.Writer, r *bufio.Reader, prompt, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(w, "%s [%s]: ", prompt, def)
	} else {
		fmt.Fprintf(w, "%s: ", prompt)
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	return line, nil
}

// ReadSecret reads a line without echo when stdin is a TTY; otherwise reads a plain line.
func ReadSecret(w io.Writer, prompt string) (string, error) {
	fmt.Fprint(w, prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(w)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// Confirm asks y/N (default no when empty).
func Confirm(w io.Writer, r *bufio.Reader, question string, defaultYes bool) (bool, error) {
	def := "y/N"
	if defaultYes {
		def = "Y/n"
	}
	ans, err := ReadLine(w, r, question+" "+def, "")
	if err != nil {
		return false, err
	}
	if ans == "" {
		return defaultYes, nil
	}
	switch strings.ToLower(ans)[0] {
	case 'y':
		return true, nil
	case 'n':
		return false, nil
	default:
		return defaultYes, nil
	}
}