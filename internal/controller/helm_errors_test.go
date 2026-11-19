package controller

import (
	"errors"
	"testing"
)

func TestHelmReleaseNotFound(t *testing.T) {
	tests := []struct {
		out string
		err error
		want bool
	}{
		{out: "", err: nil, want: false},
		{out: "Error: release: not found\n", err: errors.New("exit status 1"), want: true},
		{out: "", err: errors.New("helm status foo: exit status 1"), want: false},
		{out: "release foo not found in namespace", err: errors.New("exit status 1"), want: true},
	}
	for _, tt := range tests {
		got := HelmReleaseNotFound([]byte(tt.out), tt.err)
		if got != tt.want {
			t.Errorf("HelmReleaseNotFound(%q, %v) = %v, want %v", tt.out, tt.err, got, tt.want)
		}
	}
}