package credentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const adminAllowlistFile = "admin-github-allowlist.json"

// ErrLoginNotAllowed is returned when GitHub sign-in succeeded but the login is not on the appliance allowlist.
var ErrLoginNotAllowed = errors.New("github login is not allowed to use this admin UI")

// AdminGitHubAllowlist lists GitHub logins permitted to sign in via the admin UI.
// When the file is missing, all GitHub logins are allowed (legacy installs).
// When the file exists, only listed logins may authenticate; an empty list denies all UI logins.
type AdminGitHubAllowlist struct {
	Logins []string `json:"logins"`
}

// ReadAdminGitHubAllowlist loads the allowlist from the credentials directory.
func ReadAdminGitHubAllowlist(credentialsDir string) (AdminGitHubAllowlist, bool, error) {
	path := filepath.Join(credentialsDir, adminAllowlistFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return AdminGitHubAllowlist{}, false, nil
		}
		return AdminGitHubAllowlist{}, false, err
	}
	var list AdminGitHubAllowlist
	if err := json.Unmarshal(data, &list); err != nil {
		return AdminGitHubAllowlist{}, false, fmt.Errorf("parse %s: %w", adminAllowlistFile, err)
	}
	list.Logins = normalizeAllowlistLogins(list.Logins)
	return list, true, nil
}

// WriteAdminGitHubAllowlist persists the allowlist with mode 0600.
func WriteAdminGitHubAllowlist(credentialsDir string, list AdminGitHubAllowlist) (string, error) {
	if err := os.MkdirAll(credentialsDir, 0o750); err != nil {
		return "", err
	}
	list.Logins = normalizeAllowlistLogins(list.Logins)
	path := filepath.Join(credentialsDir, adminAllowlistFile)
	data, err := json.Marshal(list)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// RemoveAdminGitHubAllowlist deletes the allowlist file so any GitHub login may sign in again.
func RemoveAdminGitHubAllowlist(credentialsDir string) error {
	path := filepath.Join(credentialsDir, adminAllowlistFile)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// LoginPermitted returns nil when login may use GitHub OAuth for the admin UI.
func LoginPermitted(credentialsDir, login string) error {
	list, enforced, err := ReadAdminGitHubAllowlist(credentialsDir)
	if err != nil {
		return err
	}
	if !enforced {
		return nil
	}
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" {
		return ErrLoginNotAllowed
	}
	for _, allowed := range list.Logins {
		if allowed == login {
			return nil
		}
	}
	return ErrLoginNotAllowed
}

func normalizeAllowlistLogins(logins []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range logins {
		l := strings.ToLower(strings.TrimSpace(raw))
		if l == "" {
			continue
		}
		if _, ok := seen[l]; ok {
			continue
		}
		seen[l] = struct{}{}
		out = append(out, l)
	}
	return out
}

// AddAdminGitHubLogin appends a login to the allowlist (creates the file if missing).
func AddAdminGitHubLogin(credentialsDir, login string) (AdminGitHubAllowlist, error) {
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" {
		return AdminGitHubAllowlist{}, fmt.Errorf("login is required")
	}
	list, _, err := ReadAdminGitHubAllowlist(credentialsDir)
	if err != nil {
		return AdminGitHubAllowlist{}, err
	}
	for _, existing := range list.Logins {
		if existing == login {
			return list, nil
		}
	}
	list.Logins = append(list.Logins, login)
	if _, err := WriteAdminGitHubAllowlist(credentialsDir, list); err != nil {
		return AdminGitHubAllowlist{}, err
	}
	return list, nil
}

// RemoveAdminGitHubLogin removes a login from the allowlist.
func RemoveAdminGitHubLogin(credentialsDir, login string) (AdminGitHubAllowlist, bool, error) {
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" {
		return AdminGitHubAllowlist{}, false, fmt.Errorf("login is required")
	}
	list, hadFile, err := ReadAdminGitHubAllowlist(credentialsDir)
	if err != nil {
		return AdminGitHubAllowlist{}, false, err
	}
	if !hadFile {
		return AdminGitHubAllowlist{}, false, nil
	}
	next := list.Logins[:0]
	removed := false
	for _, existing := range list.Logins {
		if existing == login {
			removed = true
			continue
		}
		next = append(next, existing)
	}
	list.Logins = next
	if _, err := WriteAdminGitHubAllowlist(credentialsDir, list); err != nil {
		return AdminGitHubAllowlist{}, false, err
	}
	return list, removed, nil
}