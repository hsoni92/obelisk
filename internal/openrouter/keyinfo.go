package openrouter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const keyInfoURL = "https://openrouter.ai/api/v1/key"

// KeyMetadata is returned from OpenRouter when validating a provisioning/management key.
type KeyMetadata struct {
	Fingerprint       string `json:"fingerprint"`
	KeyTail           string `json:"key_tail,omitempty"`
	UpstreamLabel     string `json:"upstream_label,omitempty"`
	CreatorUserID     string `json:"creator_user_id,omitempty"`
	WorkspaceID       string `json:"workspace_id,omitempty"`
	IsProvisioningKey bool   `json:"is_provisioning_key,omitempty"`
	IsManagementKey   bool   `json:"is_management_key,omitempty"`
}

type keyInfoResponse struct {
	Data struct {
		Label             string   `json:"label"`
		CreatorUserID     string   `json:"creator_user_id"`
		WorkspaceID       string   `json:"workspace_id"`
		IsProvisioningKey bool     `json:"is_provisioning_key"`
		IsManagementKey   bool     `json:"is_management_key"`
		Limit             *float64 `json:"limit"`
		Usage             *float64 `json:"usage"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// FingerprintKey returns a stable SHA-256 hex digest for duplicate detection (never log the raw key).
func FingerprintKey(apiKey string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(apiKey)))
	return hex.EncodeToString(sum[:])
}

// KeyTail returns the last 8 characters of the sk-or-v1 secret material for display.
func KeyTail(apiKey string) string {
	k := strings.TrimSpace(apiKey)
	const prefix = "sk-or-v1-"
	if !strings.HasPrefix(k, prefix) {
		if len(k) > 8 {
			return k[len(k)-8:]
		}
		return k
	}
	material := strings.TrimPrefix(k, prefix)
	if len(material) >= 8 {
		return material[len(material)-8:]
	}
	return material
}

// LookupKey validates the API key against OpenRouter and returns identity fields for the pool.
func LookupKey(apiKey string) (KeyMetadata, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return KeyMetadata{}, fmt.Errorf("api_key is empty")
	}
	req, err := http.NewRequest(http.MethodGet, keyInfoURL, nil)
	if err != nil {
		return KeyMetadata{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return KeyMetadata{}, fmt.Errorf("openrouter key lookup: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return KeyMetadata{}, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return KeyMetadata{}, fmt.Errorf("openrouter rejected key (HTTP 401)")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(body)
		var parsed keyInfoResponse
		if json.Unmarshal(body, &parsed) == nil && parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		return KeyMetadata{}, fmt.Errorf("openrouter key lookup HTTP %d: %s", resp.StatusCode, msg)
	}
	var parsed keyInfoResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return KeyMetadata{}, fmt.Errorf("parse openrouter key response: %w", err)
	}
	return KeyMetadata{
		Fingerprint:       FingerprintKey(apiKey),
		KeyTail:           KeyTail(apiKey),
		UpstreamLabel:     strings.TrimSpace(parsed.Data.Label),
		CreatorUserID:     strings.TrimSpace(parsed.Data.CreatorUserID),
		WorkspaceID:       strings.TrimSpace(parsed.Data.WorkspaceID),
		IsProvisioningKey: parsed.Data.IsProvisioningKey,
		IsManagementKey:   parsed.Data.IsManagementKey,
	}, nil
}
