package controller

import (
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestOrphanGraceAndDeleteIntent(t *testing.T) {
	release := "obelisk-redis-test"
	resetOrphanGrace(release)
	if orphanGraceReached(release) {
		t.Fatal("expected grace not met on first scan")
	}
	if orphanGraceReached(release) {
		t.Fatal("expected grace not met on second scan")
	}
	if !orphanGraceReached(release) {
		t.Fatal("expected grace after third scan")
	}
	MarkReleaseDeleteIntent(release)
	if !consumeDeleteIntent(release) {
		t.Fatal("expected delete intent")
	}
	if consumeDeleteIntent(release) {
		t.Fatal("intent consumed once")
	}
}

func TestGraceAllowedForSpec(t *testing.T) {
	if graceAllowedForSpec(obelisk.Spec{}) {
		t.Fatal("empty spec must not allow grace-based pruning")
	}
	if !graceAllowedForSpec(obelisk.Spec{Instances: []obelisk.ServiceInstance{{ID: "a", Type: "redis"}}}) {
		t.Fatal("spec with instances allows grace pruning")
	}
}
