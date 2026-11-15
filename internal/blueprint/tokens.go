package blueprint

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
)

// Token is the GitHub OAuth credential. It is never written to obelisk.yaml.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	Login        string    `json:"login,omitempty"`
}

// TokenStore persists tokens in a mode-0600 file and, inside a cluster,
// mirrors them to Secret obelisk-blueprint-github.
type TokenStore struct {
	path string
	mu   sync.Mutex
}

func NewTokenStore(configPath string) *TokenStore {
	return &TokenStore{path: filepath.Join(config.CredentialsDir(configPath), TokenSecretName+".json")}
}

func (s *TokenStore) Load() (Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tok, err := readTokenFile(s.path)
	if err == nil && tok.AccessToken != "" {
		return tok, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return Token{}, err
	}
	tok, kerr := readTokenSecret()
	if kerr != nil || tok.AccessToken == "" {
		if err != nil {
			return Token{}, err
		}
		return Token{}, os.ErrNotExist
	}
	_ = writeTokenFile(s.path, tok)
	return tok, nil
}

func (s *TokenStore) Save(tok Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := writeTokenFile(s.path, tok); err != nil {
		return err
	}
	if err := writeTokenSecret(tok); err != nil {
		// Local file remains the process copy when the API server is unreachable.
		return nil
	}
	return nil
}

func readTokenFile(path string) (Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Token{}, err
	}
	var tok Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return Token{}, fmt.Errorf("parse github token: %w", err)
	}
	return tok, nil
}

func writeTokenFile(path string, tok Token) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

type clusterCreds struct {
	host string
	ns   string
	ca   *x509.CertPool
	tok  string
}

func inCluster() (clusterCreds, bool) {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return clusterCreds{}, false
	}
	raw, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return clusterCreds{}, false
	}
	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return clusterCreds{}, false
	}
	caPEM, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return clusterCreds{}, false
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return clusterCreds{}, false
	}
	return clusterCreds{host: "https://" + host + ":" + port, ns: string(ns), ca: pool, tok: string(raw)}, true
}

func kubeClient(c clusterCreds) *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: c.ca, MinVersion: tls.VersionTLS12},
		},
	}
}

func writeTokenSecret(tok Token) error {
	c, ok := inCluster()
	if !ok {
		return nil
	}
	body := map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata":   map[string]string{"name": TokenSecretName},
		"type":       "Opaque",
		"stringData": map[string]string{
			"access_token":  tok.AccessToken,
			"refresh_token": tok.RefreshToken,
			"token_type":    tok.TokenType,
			"expiry":        tok.Expiry.UTC().Format(time.RFC3339),
			"scope":         tok.Scope,
			"login":         tok.Login,
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/secrets/%s", c.host, c.ns, TokenSecretName)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := kubeClient(c).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusConflict {
		req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/namespaces/%s/secrets", c.host, c.ns), bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.tok)
		req.Header.Set("Content-Type", "application/json")
		resp2, err := kubeClient(c).Do(req)
		if err != nil {
			return err
		}
		defer resp2.Body.Close()
		if resp2.StatusCode >= 300 {
			b, _ := io.ReadAll(io.LimitReader(resp2.Body, 512))
			return fmt.Errorf("create github secret: %s", b)
		}
		return nil
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("update github secret: %s", b)
	}
	return nil
}

func readTokenSecret() (Token, error) {
	c, ok := inCluster()
	if !ok {
		return Token{}, os.ErrNotExist
	}
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/secrets/%s", c.host, c.ns, TokenSecretName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.tok)
	resp, err := kubeClient(c).Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Token{}, os.ErrNotExist
	}
	if resp.StatusCode >= 300 {
		return Token{}, fmt.Errorf("read github secret: %s", resp.Status)
	}
	var obj struct {
		Data map[string]string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		return Token{}, err
	}
	dec := func(k string) string {
		raw, err := decodeB64(obj.Data[k])
		if err != nil {
			return ""
		}
		return string(raw)
	}
	tok := Token{
		AccessToken:  dec("access_token"),
		RefreshToken: dec("refresh_token"),
		TokenType:    dec("token_type"),
		Scope:        dec("scope"),
		Login:        dec("login"),
	}
	if exp := dec("expiry"); exp != "" {
		tok.Expiry, _ = time.Parse(time.RFC3339, exp)
	}
	return tok, nil
}

func decodeB64(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(s)
}
