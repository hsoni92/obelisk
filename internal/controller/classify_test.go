package controller

import (
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestPodIssueReason(t *testing.T) {
	code, _ := podIssueReason(PodSummary{PrimaryReason: "CrashLoopBackOff"})
	if code != ReasonPodCrashLoop {
		t.Fatal(code)
	}
	code, _ = podIssueReason(PodSummary{PrimaryReason: "ImagePullBackOff"})
	if code != ReasonPodImagePull {
		t.Fatal(code)
	}
}

func TestInstanceDesiredHashDiffersOnEnabled(t *testing.T) {
	inst := obelisk.ServiceInstance{Type: "redis", ID: "a", Enabled: true}
	off := inst
	off.Enabled = false
	if instanceDesiredHash(false, false, inst) == instanceDesiredHash(false, false, off) {
		t.Fatal("expected different hash")
	}
}