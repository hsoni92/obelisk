package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/internal/credentials"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type statusReport struct {
	AdminURL      string                   `json:"admin_url"`
	AdminAPI      string                   `json:"admin_api"`
	Profile       string                   `json:"profile"`
	MachineID     string                   `json:"machine_id"`
	Reconcile     string                   `json:"reconcile"`
	Blueprint     blueprintStatus          `json:"blueprint"`
	Backup        backupStatus             `json:"backup"`
	Platform      []platformComponent      `json:"platform"`
	Instances     []instanceStatus         `json:"instances"`
}

type blueprintStatus struct {
	Repository string `json:"repository,omitempty"`
	Branch     string `json:"branch,omitempty"`
	Generation int    `json:"generation,omitempty"`
	Sync       string `json:"sync,omitempty"`
	Login      string `json:"login,omitempty"`
}

type backupStatus struct {
	DestinationsEnabled int    `json:"destinations_enabled"`
	PolicySchedule      string `json:"policy_schedule,omitempty"`
}

type platformComponent struct {
	Name  string `json:"name"`
	State string `json:"state"`
	Hint  string `json:"hint,omitempty"`
}

type instanceStatus struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
	State   string `json:"state"`
}

func cmdStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	cfgPath := fs.String("config", config.ResolvePath(), "config path")
	helmRoot := fs.String("helm-root", config.ResolveHelmRoot(), "helm charts root")
	asJSON := fs.Bool("json", false, "emit JSON instead of a table")
	_ = fs.Parse(args)

	cfg := config.EnsureReadablePath(*cfgPath)
	spec, err := config.Load(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()
	report, err := buildStatusReport(ctx, cfg, *helmRoot, spec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		return
	}
	printStatusReport(report)
}

func buildStatusReport(ctx context.Context, cfg, helmRoot string, spec obelisk.Spec) (statusReport, error) {
	adminURL := credentials.DefaultAdminPublicURL(ctx)
	if adminURL == "" {
		adminURL = fmt.Sprintf("http://127.0.0.1:%s", obelisk.AdminHTTPPortString())
	}
	rec := controller.Reconciler{ConfigPath: cfg, HelmRoot: helmRoot, Namespace: namespaceFromEnv()}
	states := rec.ObserveServiceStates(ctx, spec)
	reconcile := "cluster"
	if rec.SimulateMode() {
		reconcile = "off"
	}

	gh := spec.Blueprint.GitHub
	bp := blueprintStatus{
		Repository: gh.Repository,
		Branch:     gh.Branch,
		Generation: gh.Generation,
		Sync:       string(gh.SyncStatus),
		Login:      gh.OAuthLogin,
	}

	destEnabled := 0
	for _, d := range spec.Backup.Destinations {
		if d.Enabled {
			destEnabled++
		}
	}
	sched := spec.Backup.Policy.Schedule
	if sched == "" {
		sched = "default"
	}

	instOut := make([]instanceStatus, 0, len(spec.Instances))
	for _, inst := range spec.Instances {
		st := states[inst.StateKey()]
		if st == "" {
			st = "unknown"
		}
		name := inst.DisplayName
		if name == "" {
			name = inst.ID
		}
		instOut = append(instOut, instanceStatus{
			Name:    name,
			Type:    inst.Type,
			ID:      inst.ID,
			Enabled: inst.Enabled,
			State:   st,
		})
	}

	return statusReport{
		AdminURL:  adminURL,
		AdminAPI:  strings.TrimRight(adminURL, "/") + "/api/v1",
		Profile:   spec.Deployment.Profile,
		MachineID: spec.BackupBucketName(),
		Reconcile: reconcile,
		Blueprint: bp,
		Backup: backupStatus{
			DestinationsEnabled: destEnabled,
			PolicySchedule:      sched,
		},
		Platform:  platformComponents(ctx, rec.SimulateMode()),
		Instances: instOut,
	}, nil
}

func platformComponents(ctx context.Context, simulate bool) []platformComponent {
	if simulate {
		return []platformComponent{
			{Name: "Kong", State: "n/a", Hint: "reconcile off"},
			{Name: "Controller", State: "n/a", Hint: "reconcile off"},
			{Name: "Admin UI", State: "n/a", Hint: "reconcile off"},
		}
	}
	names := []struct {
		label string
		dep   string
	}{
		{"Kong", "obelisk-kong"},
		{"Controller", "obelisk-controller"},
		{"Admin UI", "obelisk-admin-ui"},
	}
	var out []platformComponent
	for _, row := range names {
		state, hint := deploymentHealth(ctx, row.dep)
		out = append(out, platformComponent{Name: row.label, State: state, Hint: hint})
	}
	return out
}

func deploymentHealth(ctx context.Context, name string) (state, hint string) {
	ns := namespaceFromEnv()
	if _, err := exec.LookPath("kubectl"); err != nil {
		return "unknown", "kubectl unavailable"
	}
	kc := kubeconfigPath()
	args := []string{"get", "deployment", name, "-n", ns,
		"-o", "jsonpath={.status.readyReplicas}/{.status.replicas}:{.status.conditions[?(@.type==\"Available\")].status}"}
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	if kc != "" {
		cmd.Env = append(os.Environ(), "KUBECONFIG="+kc)
	}
	raw, err := cmd.Output()
	if err != nil {
		if strings.Contains(string(raw), "NotFound") {
			return "not installed", ""
		}
		return "unknown", "cannot read deployment"
	}
	parts := strings.SplitN(strings.TrimSpace(string(raw)), ":", 2)
	scale := parts[0]
	available := ""
	if len(parts) > 1 {
		available = parts[1]
	}
	ready, desired, _ := strings.Cut(scale, "/")
	if available == "True" && ready != "" && ready == desired && desired != "0" {
		return "running", ""
	}
	if ready == "0" && desired == "0" {
		return "stopped", ""
	}
	if available == "True" && ready == desired {
		return "running", ""
	}
	return "starting", fmt.Sprintf("ready %s/%s", ready, desired)
}

func printStatusReport(r statusReport) {
	fmt.Printf("Admin UI:  %s\n", r.AdminURL)
	fmt.Printf("Admin API: %s\n", r.AdminAPI)
	fmt.Printf("Profile:   %s\n", r.Profile)
	fmt.Printf("Machine:   %s\n", r.MachineID)
	fmt.Printf("Reconcile: %s\n", r.Reconcile)
	if r.Blueprint.Repository != "" {
		fmt.Printf("Blueprint: %s", r.Blueprint.Sync)
		if r.Blueprint.Generation > 0 {
			fmt.Printf(" (gen %d)", r.Blueprint.Generation)
		}
		fmt.Printf(" — %s", r.Blueprint.Repository)
		if r.Blueprint.Branch != "" {
			fmt.Printf("@%s", r.Blueprint.Branch)
		}
		if r.Blueprint.Login != "" {
			fmt.Printf(" [%s]", r.Blueprint.Login)
		}
		fmt.Println()
	}
	fmt.Printf("Backup:    %d destination(s) enabled · schedule %s\n",
		r.Backup.DestinationsEnabled, r.Backup.PolicySchedule)

	fmt.Println("\nPlatform")
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, p := range r.Platform {
		line := fmt.Sprintf("%s\t%s", p.Name, p.State)
		if p.Hint != "" {
			line += "\t(" + p.Hint + ")"
		}
		fmt.Fprintln(tw, line)
	}
	_ = tw.Flush()

	if len(r.Instances) == 0 {
		fmt.Println("\nServices:  none configured")
		return
	}
	fmt.Println("\nServices")
	tw = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tTYPE\tENABLED\tSTATE")
	for _, inst := range r.Instances {
		en := "no"
		if inst.Enabled {
			en = "yes"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", inst.Name, inst.Type, en, inst.State)
	}
	_ = tw.Flush()
}

func namespaceFromEnv() string {
	if ns := strings.TrimSpace(os.Getenv("OBELISK_NAMESPACE")); ns != "" {
		return ns
	}
	return "obelisk"
}

func kubeconfigPath() string {
	if kc := strings.TrimSpace(os.Getenv("KUBECONFIG")); kc != "" {
		return kc
	}
	if _, err := os.Stat("/etc/rancher/k3s/k3s.yaml"); err == nil {
		return "/etc/rancher/k3s/k3s.yaml"
	}
	return ""
}