package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/hsoni92/obelisk/pkg/obelisk"
	"gopkg.in/yaml.v3"
)

const DefaultPath = "/etc/obelisk/obelisk.yaml"

// Options controls bootstrap identity when loading config.
type Options struct {
	// AdoptMachineID is applied when the file has no machine.id, before a new id is minted.
	AdoptMachineID string
}

func Load(path string) (obelisk.Spec, error) {
	return load(path, Options{})
}

// LoadOptions reads config and, when AdoptMachineID is set, applies that id
// before a new machine id would be minted (blueprint adopt on an empty install).
func LoadOptions(path string, opt Options) (obelisk.Spec, error) {
	return load(path, opt)
}

func load(path string, opt Options) (obelisk.Spec, error) {
	if path == "" {
		path = ResolvePath()
	}
	path = ActiveConfigPath(path)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && autoInitAllowed(path) {
			spec := obelisk.DefaultSpec(defaultProfileForPath(path))
			if opt.AdoptMachineID != "" {
				spec.Machine.ID = opt.AdoptMachineID
			}
			if saveErr := SaveQuiet(path, spec); saveErr != nil {
				return obelisk.Spec{}, fmt.Errorf("create config %s: %w", path, saveErr)
			}
			return spec, nil
		}
		if os.IsNotExist(err) && path == DefaultPath {
			alt := ResolvePath()
			if alt != path {
				return load(alt, opt)
			}
			return obelisk.Spec{}, fmt.Errorf("read config %s: %w\n\nOn a host: run install/bootstrap.sh or:\n  sudo obelisk init-config --profile vps --path %s\n\nFor local UI dev: make dev-api", path, err, DefaultPath)
		}
		return obelisk.Spec{}, fmt.Errorf("read config %s: %w", path, err)
	}
	var spec obelisk.Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return obelisk.Spec{}, fmt.Errorf("parse yaml: %w", err)
	}
	if err := Validate(&spec); err != nil {
		return obelisk.Spec{}, err
	}
	normalize(&spec)
	if opt.AdoptMachineID != "" && spec.Machine.ID == "" {
		spec.Machine.ID = opt.AdoptMachineID
	}
	if obelisk.EnsureMachineIdentity(&spec) {
		if saveErr := SaveQuiet(path, spec); saveErr == nil {
			return spec, nil
		}
	}
	return spec, nil
}

// Save writes spec and notifies the blueprint sync hook.
func Save(path string, spec obelisk.Spec) error {
	return write(path, spec, true)
}

// SaveQuiet writes spec without enqueueing a blueprint push.
// Used for sync metadata, machine-id minting, and restore/import.
func SaveQuiet(path string, spec obelisk.Spec) error {
	return write(path, spec, false)
}

func write(path string, spec obelisk.Spec, notify bool) error {
	if path == "" {
		path = ResolvePath()
	}
	path = WriteConfigPath(path)
	if err := Validate(&spec); err != nil {
		return err
	}
	normalize(&spec)
	data, err := yaml.Marshal(&spec)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	perm := os.FileMode(0o755)
	if dir == filepath.Dir(WritableApplianceConfigPath) {
		perm = 0o700
	}
	if err := os.MkdirAll(dir, perm); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	if notify {
		saveHookMu.RLock()
		fn := saveHook
		saveHookMu.RUnlock()
		if fn != nil {
			fn(path)
		}
	}
	return nil
}

func normalize(spec *obelisk.Spec) {
	obelisk.NormalizeOpenRouter(spec)
	obelisk.NormalizeMail(spec)
	obelisk.NormalizeInstances(spec)
	obelisk.NormalizeBackup(spec)
	obelisk.NormalizeBlueprint(spec)
	obelisk.NormalizeDeployment(spec)
	obelisk.NormalizePeerGateway(spec)
	spec.Projects = obelisk.NormalizeProjects(spec.Projects)
}

var (
	saveHookMu sync.RWMutex
	saveHook   func(path string)
)

// SetAfterSave registers a hook invoked after a successful operator save.
// Restore, import, and sync metadata writes use SaveQuiet and do not call it.
func SetAfterSave(fn func(path string)) {
	saveHookMu.Lock()
	saveHook = fn
	saveHookMu.Unlock()
}

func Validate(spec *obelisk.Spec) error {
	switch spec.Deployment.Profile {
	case "vps", "local", "":
		if spec.Deployment.Profile == "" {
			spec.Deployment.Profile = "vps"
		}
	default:
		return fmt.Errorf("deployment.profile must be vps or local")
	}
	if spec.Deployment.Profile == "vps" && spec.Deployment.Tailscale.Enabled && !obelisk.IsGateway(*spec) {
		return fmt.Errorf("tailscale cannot be enabled when deployment.profile is vps (use deployment.role: gateway for mesh backhaul)")
	}
	if obelisk.IsGateway(*spec) && !spec.Deployment.Tailscale.Enabled {
		return fmt.Errorf("deployment.role gateway requires tailscale.enabled")
	}
	if obelisk.IsGateway(*spec) {
		mode := obelisk.PeerGatewayTLSMode(*spec)
		if mode != obelisk.PeerGatewayTLSPassthrough {
			return fmt.Errorf("peer_gateway.tls_mode must be %q in v1", obelisk.PeerGatewayTLSPassthrough)
		}
	}
	if spec.Deployment.Profile == "local" && spec.Exposure.Funnel.RustFSS3.Enabled && !spec.Deployment.Tailscale.Enabled {
		return fmt.Errorf("funnel requires tailscale.enabled on local profile")
	}
	switch spec.Hardware.Tier {
	case "floor", "comfort", "":
		if spec.Hardware.Tier == "" {
			spec.Hardware.Tier = "floor"
		}
	default:
		return fmt.Errorf("hardware.tier must be floor or comfort")
	}
	if spec.Machine.ID != "" {
		if err := obelisk.ValidateMachineID(spec.Machine.ID); err != nil {
			return err
		}
	}
	for _, port := range []int{obelisk.AdminHTTPPort(), obelisk.NativeEdgeTLSPortNumber()} {
		if label, conflict := obelisk.ReservedHostPortConflict(port); conflict {
			return fmt.Errorf("exposure port %d (%s) is reserved; Obelisk never binds host 80/443", port, label)
		}
	}
	return obelisk.ValidateBackupPolicy(*spec)
}
