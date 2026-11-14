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
	Ready int
	Total int
	Err   error
}

func (r *Reconciler) namespace() string {
	if r.Namespace != "" {
		return r.Namespace
	}
	return "obelisk"
}

func (r *Reconciler) HelmReleaseDetail(ctx context.Context, release string) HelmReleaseDetail {
	if r.simulateMode() {
		return HelmReleaseDetail{Status: "deployed", Revision: 1, Notes: "simulate mode"}
	}
	ns := r.namespace()
	cmd := exec.CommandContext(ctx, "helm", "status", release, "-n", ns, "-o", "json")
	cmd.Env = kubeconfigEnv()
	raw, err := cmd.Output()
	if err != nil {
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
	if r.simulateMode() {
		return PodSummary{Ready: 1, Total: 1}
	}
	ns := r.namespace()
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods",
		"-n", ns,
		"-l", fmt.Sprintf("app=%s", release),
		"-o", "json")
	cmd.Env = kubeconfigEnv()
	raw, err := cmd.Output()
	if err != nil {
		return PodSummary{Err: err}
	}
	var doc struct {
		Items []struct {
			Status struct {
				Phase string `json:"phase"`
				ContainerStatuses []struct {
					Ready bool `json:"ready"`
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
					break
				}
			}
			if allReady {
				out.Ready++
			}
		}
	}
	return out
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}