package controller

import (
	"context"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type ClassifyResult struct {
	State      string
	ReasonCode string
	Detail     string
}

func (r *Reconciler) classifyInstance(ctx context.Context, inst obelisk.ServiceInstance) ClassifyResult {
	if r.reconcileDisabled() {
		if inst.Enabled {
			return ClassifyResult{State: "starting", ReasonCode: ReasonInitialInstall, Detail: "reconcile disabled"}
		}
		return ClassifyResult{State: "stopped"}
	}
	if !inst.Enabled {
		return r.classifyDisabled(ctx, inst)
	}
	return r.classifyEnabled(ctx, inst)
}

func (r *Reconciler) classifyDisabled(ctx context.Context, inst obelisk.ServiceInstance) ClassifyResult {
	release := inst.ReleaseName()
	pods := r.PodsForRelease(ctx, release)
	if pods.Err != nil {
		return ClassifyResult{State: "degraded", ReasonCode: ReasonPodNotReady, Detail: pods.Err.Error()}
	}
	if pods.Total > 0 {
		detail := pods.PrimaryReason
		if detail == "" {
			detail = "workload still draining"
		}
		return ClassifyResult{State: "stopping", ReasonCode: ReasonStoppingDrain, Detail: detail}
	}
	st, err := r.helmReleaseStatus(ctx, release)
	if err != nil {
		return ClassifyResult{State: "degraded", Detail: err.Error()}
	}
	if st == "" {
		return ClassifyResult{State: "stopped"}
	}
	switch st {
	case "pending-install", "pending-upgrade", "pending-rollback", "uninstalling":
		return ClassifyResult{State: "stopping", ReasonCode: ReasonHelmPendingUpgrade, Detail: st}
	case "failed":
		return ClassifyResult{State: "degraded", ReasonCode: ReasonHelmFailed, Detail: st}
	default:
		return ClassifyResult{State: "stopped"}
	}
}

func (r *Reconciler) classifyEnabled(ctx context.Context, inst obelisk.ServiceInstance) ClassifyResult {
	release := inst.ReleaseName()
	st, err := r.helmReleaseStatus(ctx, release)
	if err != nil {
		return ClassifyResult{State: "degraded", Detail: err.Error()}
	}
	if st == "" {
		return ClassifyResult{State: "starting", ReasonCode: ReasonReleaseMissing, Detail: "Initial install pending"}
	}
	switch st {
	case "failed", "uninstalling":
		return ClassifyResult{State: "degraded", ReasonCode: ReasonHelmFailed, Detail: st}
	case "pending-install":
		return ClassifyResult{State: "starting", ReasonCode: ReasonHelmPendingInstall, Detail: st}
	case "pending-upgrade", "pending-rollback":
		return ClassifyResult{State: "starting", ReasonCode: ReasonHelmPendingUpgrade, Detail: st}
	case "deployed":
		return r.classifyDeployed(ctx, release)
	default:
		return ClassifyResult{State: "starting", ReasonCode: ReasonHelmPendingInstall, Detail: st}
	}
}

func (r *Reconciler) classifyDeployed(ctx context.Context, release string) ClassifyResult {
	pods := r.PodsForRelease(ctx, release)
	if pods.Err != nil {
		return ClassifyResult{State: "degraded", ReasonCode: ReasonPodNotReady, Detail: pods.Err.Error()}
	}
	if pods.Total == 0 {
		return ClassifyResult{State: "starting", ReasonCode: ReasonPodNotReady, Detail: "no workload pods"}
	}
	if pods.Ready < pods.Total {
		code, detail := podIssueReason(pods)
		return ClassifyResult{State: "starting", ReasonCode: code, Detail: detail}
	}
	return ClassifyResult{State: "running"}
}

func podIssueReason(pods PodSummary) (code, detail string) {
	detail = pods.PrimaryReason
	if detail == "" {
		detail = "pods not ready"
	}
	switch pods.PrimaryReason {
	case "CrashLoopBackOff", "CreateContainerConfigError", "RunContainerError":
		return ReasonPodCrashLoop, detail
	case "ImagePullBackOff", "ErrImagePull", "InvalidImageName":
		return ReasonPodImagePull, detail
	case "Pending":
		if pods.Detail != "" {
			return ReasonPodPendingPVC, pods.Detail
		}
		return ReasonPodNotReady, detail
	default:
		return ReasonPodNotReady, detail
	}
}
