package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hsoni92/obelisk/pkg/obelisk"
	"gopkg.in/yaml.v3"
)

const DefaultPath = "/etc/obelisk/obelisk.yaml"

func Load(path string) (obelisk.Spec, error) {
	if path == "" {
		path = ResolvePath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && autoInitAllowed(path) {
			spec := obelisk.DefaultSpec(defaultProfileForPath(path))
			if saveErr := Save(path, spec); saveErr != nil {
				return obelisk.Spec{}, fmt.Errorf("create config %s: %w", path, saveErr)
			}
			return spec, nil
		}
		if os.IsNotExist(err) && path == DefaultPath {
			alt := ResolvePath()
			if alt != path {
				return Load(alt)
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
	obelisk.NormalizeOpenRouter(&spec)
	obelisk.NormalizeMail(&spec)
	obelisk.NormalizeInstances(&spec)
	if obelisk.EnsureMachineIdentity(&spec) {
		if saveErr := Save(path, spec); saveErr == nil {
			return spec, nil
		}
	}
	return spec, nil
}

func Save(path string, spec obelisk.Spec) error {
	if path == "" {
		path = ResolvePath()
	}
	if err := Validate(&spec); err != nil {
		return err
	}
	obelisk.NormalizeOpenRouter(&spec)
	obelisk.NormalizeMail(&spec)
	obelisk.NormalizeInstances(&spec)
	data, err := yaml.Marshal(&spec)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
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
	if spec.Deployment.Profile == "vps" && spec.Deployment.Tailscale.Enabled {
		return fmt.Errorf("tailscale cannot be enabled when deployment.profile is vps")
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
	return nil
}