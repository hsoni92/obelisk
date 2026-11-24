package controller

import "testing"

func TestMergePodSummaries(t *testing.T) {
	got := mergePodSummaries(
		PodSummary{Ready: 1, Total: 1},
		PodSummary{Ready: 0, Total: 1, PrimaryReason: "Pending"},
	)
	if got.Ready != 1 || got.Total != 2 {
		t.Fatalf("ready/total=%d/%d", got.Ready, got.Total)
	}
	if got.PrimaryReason != "Pending" {
		t.Fatalf("reason=%q", got.PrimaryReason)
	}
}

func TestPodsForReleasePlatformAlias(t *testing.T) {
	if PlatformRelease != "obelisk-platform" {
		t.Fatal("unexpected platform release constant")
	}
}
