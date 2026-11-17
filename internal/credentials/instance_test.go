package credentials

import (
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestCredentialsComplete_RustFS(t *testing.T) {
	if credentialsComplete("rustfs", obelisk.InstanceCredentials{AccessKey: "a"}) {
		t.Fatal("expected incomplete without secret")
	}
	if !credentialsComplete("rustfs", obelisk.InstanceCredentials{AccessKey: "a", SecretKey: "b"}) {
		t.Fatal("expected complete")
	}
}