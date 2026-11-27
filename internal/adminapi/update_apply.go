package adminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const (
	updateJobLabel   = "app.kubernetes.io/name=obelisk-update"
	updateJobTimeout = 30 * time.Minute
)

// updateApply is the state of the in-cluster update Job, as reported to the UI.
type updateApply struct {
	Phase       string `json:"phase"` // "running", "succeeded" or "failed"
	TargetBuild int    `json:"target_build,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	LogTail     string `json:"log_tail,omitempty"`
}

// runKubectl is replaced in tests.
var runKubectl = func(ctx context.Context, stdin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	if err != nil {
		return out.Bytes(), fmt.Errorf("kubectl %s: %w: %s", args[0], err, strings.TrimSpace(out.String()))
	}
	return out.Bytes(), nil
}

// updateJobManifest builds a one-shot privileged Job that enters the host namespaces and runs
// `obelisk update --to build-N` there. It must be a Job, not a goroutine: the update replaces
// this very controller pod, so the work has to outlive it.
func updateJobManifest(ns, image, root string, target int) (name string, manifest string) {
	name = fmt.Sprintf("obelisk-update-%d-%d", target, time.Now().Unix())
	manifest = fmt.Sprintf(`apiVersion: batch/v1
kind: Job
metadata:
  name: %[1]s
  namespace: %[2]s
  labels:
    app.kubernetes.io/name: obelisk-update
  annotations:
    obelisk.dev/target-build: "%[5]d"
spec:
  backoffLimit: 0
  activeDeadlineSeconds: %[6]d
  ttlSecondsAfterFinished: 86400
  template:
    metadata:
      labels:
        app.kubernetes.io/name: obelisk-update
    spec:
      hostPID: true
      restartPolicy: Never
      containers:
        - name: update
          image: %[3]s
          imagePullPolicy: IfNotPresent
          securityContext:
            privileged: true
          command:
            - nsenter
            - -t
            - "1"
            - -m
            - -u
            - -i
            - -n
            - -p
            - --
            - env
            - OBELISK_ROOT=%[4]s
            - /usr/local/bin/obelisk
            - update
            - --to
            - build-%[5]d
`, name, ns, image, root, target, int(updateJobTimeout.Seconds()))
	return name, manifest
}

type kubeJobList struct {
	Items []struct {
		Metadata struct {
			Name              string            `json:"name"`
			CreationTimestamp string            `json:"creationTimestamp"`
			Annotations       map[string]string `json:"annotations"`
		} `json:"metadata"`
		Status struct {
			Active    int `json:"active"`
			Succeeded int `json:"succeeded"`
			Failed    int `json:"failed"`
		} `json:"status"`
	} `json:"items"`
}

// currentUpdateApply reports the newest update Job, or nil when none exists.
func currentUpdateApply(ctx context.Context) *updateApply {
	out, err := runKubectl(ctx, "", "get", "jobs", "-n", instanceNamespace(), "-l", updateJobLabel, "-o", "json")
	if err != nil {
		return nil
	}
	var list kubeJobList
	if json.Unmarshal(out, &list) != nil || len(list.Items) == 0 {
		return nil
	}
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[i].Metadata.CreationTimestamp > list.Items[j].Metadata.CreationTimestamp
	})
	j := list.Items[0]
	a := &updateApply{Phase: "running", StartedAt: j.Metadata.CreationTimestamp}
	fmt.Sscanf(j.Metadata.Annotations["obelisk.dev/target-build"], "%d", &a.TargetBuild)
	switch {
	case j.Status.Succeeded > 0:
		a.Phase = "succeeded"
	case j.Status.Failed > 0:
		a.Phase = "failed"
		if logs, lerr := runKubectl(ctx, "", "logs", "-n", instanceNamespace(), "job/"+j.Metadata.Name, "--tail=15"); lerr == nil {
			a.LogTail = strings.TrimSpace(string(logs))
		}
	}
	return a
}

// handleUpdateApply serves POST /api/v1/update/apply: it starts the update on the host.
func (s *Server) handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if cur := currentUpdateApply(ctx); cur != nil && cur.Phase == "running" {
		http.Error(w, "an update is already running", http.StatusConflict)
		return
	}
	st := s.checkUpdate(true)
	if !st.UpdateAvailable {
		http.Error(w, "no update is available", http.StatusConflict)
		return
	}
	image := strings.TrimSpace(os.Getenv("OBELISK_CONTROLLER_IMAGE"))
	if image == "" {
		http.Error(w, "controller image is unknown; update from the appliance with: "+st.Command, http.StatusServiceUnavailable)
		return
	}
	name, manifest := updateJobManifest(instanceNamespace(), image, s.HelmRoot, st.LatestBuild)
	if _, err := runKubectl(ctx, manifest, "apply", "-f", "-"); err != nil {
		http.Error(w, "could not start the update: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.recordAudit("Update", "started", fmt.Sprintf("Updating build %d → %d (%s)", st.InstalledBuild, st.LatestBuild, name))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(updateApply{Phase: "running", TargetBuild: st.LatestBuild})
}
