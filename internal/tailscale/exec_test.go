package tailscale

import (
	"errors"
	"testing"
)

func TestSudoNeedsPassword(t *testing.T) {
	err := errors.New(`exit status 1: sudo: a password is required`)
	if !sudoNeedsPassword(nil, err) {
		t.Fatal("expected password required detection")
	}
}

func TestNeedsTailscaleElevation(t *testing.T) {
	err := errors.New(`exit status 1: Access denied: checkprefs access denied`)
	out := []byte(`Use 'sudo tailscale up'`)
	if !needsTailscaleElevation(out, err) {
		t.Fatal("expected elevation for checkprefs")
	}
	if needsTailscaleElevation(nil, nil) {
		t.Fatal("nil err should not need elevation")
	}
}
