package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const PlatformRelease = "obelisk-platform"

// HelmReleaseDetail from `helm status -o json`.
type HelmReleaseDetail struct {
	Status     string
	Revision   int
	Notes      string
	Description string
	Err        error
}

// PodSummary from kubectl pod list for a release label.
type PodSummary struct {
	Ready         int
	Total         int
	Err           error
	PrimaryReason string
	Detail        string
}

func (r *Reconciler) namespace() string {
	if r.Namespace != "" {
		return r.Namespace
	}
	return "obelisk"
}

func (r *Reconciler) HelmReleaseDetail(ctx context.Context, release string) HelmReleaseDetail {
	if r.reconcileDisabled() {
		return HelmReleaseDetail{Status: "", Notes: "reconcile disabled or cluster tools unavailable"}
	}
	ns := r.namespace()
	cmd := exec.CommandContext(ctx, "helm", "status", release, "-n", ns, "-o", "json")
	cmd.Env = kubeconfigEnv()
	raw, err := cmd.CombinedOutput()
	if err != nil {
		if HelmReleaseNotFound(raw, err) {
			return HelmReleaseDetail{}
		}
		return HelmReleaseDetail{Err: fmt.Errorf("helm status %s: %w", release, err)}
	}
	var doc struct {
		Version int `json:"version"`
		Info    struct {
			Status      string `json:"status"`
			Notes       string `json:"notes"`
			Description string `json:"description"`
		} `json:"info"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return HelmReleaseDetail{Err: err}
	}
	return HelmReleaseDetail{
		Status:      strings.ToLower(doc.Info.Status),
		Revision:    doc.Version,
		Notes:       truncate(doc.Info.Notes, 240),
		Description: truncate(doc.Info.Description, 160),
	}
}

func (r *Reconciler) PodsForRelease(ctx context.Context, release string) PodSummary {
	if r.reconcileDisabled() {
		return PodSummary{}
	}
	ns := r.namespace()
	raw, err := commandRunner(ctx, "kubectl", "get", "pods",
		"-n", ns,
		"-l", fmt.Sprintf("app=%s", release),
		"-o", "json")
	if err != nil {
		return PodSummary{Err: err}
	}
	var doc struct {
		Items []struct {
			Status struct {
				Phase             string `json:"phase"`
				ContainerStatuses []struct {
					Ready bool `json:"ready"`
					State struct {
						Waiting *struct {
							Reason  string `json:"reason"`
							Message string `json:"message"`
						} `json:"waiting"`
						Terminated *struct {
							Reason  string `json:"reason"`
							Message string `json:"message"`
						} `json:"terminated"`
					} `json:"state"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return PodSummary{Err: err}
	}
	out := PodSummary{Total: len(doc.Items)}
	for _, pod := range doc.Items {
		if pod.Status.Phase == "Running" {
			allReady := len(pod.Status.ContainerStatuses) > 0
			for _, c := range pod.Status.ContainerStatuses {
				if !c.Ready {
					allReady = false
					if out.PrimaryReason == "" {
						out.PrimaryReason, out.Detail = firstPodIssue(c)
					}
					break
				}
			}
			if allReady {
				out.Ready++
			}
		} else if out.PrimaryReason == "" {
			out.PrimaryReason = pod.Status.Phase
			if pod.Status.Phase == "Pending" {
				for _, c := range pod.Status.ContainerStatuses {
					if r, d := firstPodIssue(c); r != "" {
						out.PrimaryReason, out.Detail = r, d
						break
					}
				}
			}
		}
	}
	return out
}

func firstPodIssue(c struct {
	Ready bool `json:"ready"`
	State struct {
		Waiting *struct {
			Reason  string `json:"reason"`
			Message string `json:"message"`
		} `json:"waiting"`
		Terminated *struct {
			Reason  string `json:"reason"`
			Message string `json:"message"`
		} `json:"terminated"`
	} `json:"state"`
}) (reason, detail string) {
	if c.State.Waiting != nil && c.State.Waiting.Reason != "" {
		return c.State.Waiting.Reason, c.State.Waiting.Message
	}
	if c.State.Terminated != nil && c.State.Terminated.Reason != "" {
		return c.State.Terminated.Reason, c.State.Terminated.Message
	}
	return "", ""
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}