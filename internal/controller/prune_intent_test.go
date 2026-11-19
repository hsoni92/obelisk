package controller

import "testing"

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