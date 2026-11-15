package blueprint

import (
	"encoding/json"

	"github.com/hsoni92/obelisk/internal/backup"
	"github.com/hsoni92/obelisk/internal/mail"
	"github.com/hsoni92/obelisk/internal/openrouter"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// CollectSecrets reads sidecar files and instance credential fields for secrets.enc.
func CollectSecrets(configPath string, spec obelisk.Spec) (Secrets, error) {
	sec := Secrets{Version: 1, Instances: map[string]obelisk.InstanceCredentials{}}
	for _, inst := range spec.Instances {
		sec.Instances[inst.ID] = inst.Credentials
	}
	if dest, err := backup.LoadDestinationsCredentials(configPath); err != nil {
		return Secrets{}, err
	} else if len(dest.Destinations) > 0 {
		raw, err := json.Marshal(dest)
		if err != nil {
			return Secrets{}, err
		}
		sec.BackupDestinations = raw
	}
	if legacy, ok, err := backup.LoadCredentials(configPath); err != nil {
		return Secrets{}, err
	} else if ok {
		raw, err := json.Marshal(legacy)
		if err != nil {
			return Secrets{}, err
		}
		sec.BackupLegacy = raw
	}
	if or, ok, err := openrouter.LoadCredentials(configPath); err != nil {
		return Secrets{}, err
	} else if ok {
		raw, err := json.Marshal(or)
		if err != nil {
			return Secrets{}, err
		}
		sec.OpenRouter = raw
	}
	if m, ok, err := mail.LoadCredentials(configPath); err != nil {
		return Secrets{}, err
	} else if ok {
		raw, err := json.Marshal(m)
		if err != nil {
			return Secrets{}, err
		}
		sec.Mail = raw
	}
	return sec, nil
}

// WriteSidecars restores credential files from a decrypted bundle.
func WriteSidecars(configPath string, sec Secrets) error {
	if len(sec.BackupDestinations) > 0 {
		var dest backup.DestinationsCredentialsFile
		if err := json.Unmarshal(sec.BackupDestinations, &dest); err != nil {
			return err
		}
		if err := backup.SaveDestinationsCredentials(configPath, dest); err != nil {
			return err
		}
	}
	if len(sec.BackupLegacy) > 0 {
		var legacy backup.RemoteCredentials
		if err := json.Unmarshal(sec.BackupLegacy, &legacy); err != nil {
			return err
		}
		if legacy.AccessKeyID != "" && legacy.SecretAccessKey != "" {
			if err := backup.SaveCredentials(configPath, legacy); err != nil {
				return err
			}
		}
	}
	if len(sec.OpenRouter) > 0 {
		var or openrouter.CredentialsFile
		if err := json.Unmarshal(sec.OpenRouter, &or); err != nil {
			return err
		}
		if err := openrouter.SaveCredentials(configPath, or); err != nil {
			return err
		}
	}
	if len(sec.Mail) > 0 {
		var m mail.CredentialsFile
		if err := json.Unmarshal(sec.Mail, &m); err != nil {
			return err
		}
		if err := mail.SaveCredentials(configPath, m); err != nil {
			return err
		}
	}
	return nil
}

// MergeInstanceSecrets copies sealed credentials back onto the public spec.
func MergeInstanceSecrets(spec *obelisk.Spec, sec Secrets) {
	if len(sec.Instances) == 0 {
		return
	}
	for i := range spec.Instances {
		cred, ok := sec.Instances[spec.Instances[i].ID]
		if ok {
			spec.Instances[i].Credentials = cred
		}
	}
}
