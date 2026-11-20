package peer

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const registryFile = "peer.registry.json"

type registryDisk struct {
	Peers map[string]PeerRecord `json:"peers,omitempty"`
}

// Registry holds admitted peers and their latest catalog snapshots.
type Registry struct {
	mu   sync.RWMutex
	path string
	data registryDisk
}

func registryPath(configPath string) string {
	return filepath.Join(config.CredentialsDir(configPath), registryFile)
}

// LoadRegistry reads peer state from disk.
func LoadRegistry(configPath string) (*Registry, error) {
	r := &Registry{path: registryPath(configPath), data: registryDisk{Peers: map[string]PeerRecord{}}}
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, fmt.Errorf("read peer registry: %w", err)
	}
	if err := json.Unmarshal(data, &r.data); err != nil {
		return nil, fmt.Errorf("parse peer registry: %w", err)
	}
	if r.data.Peers == nil {
		r.data.Peers = map[string]PeerRecord{}
	}
	return r, nil
}

func (r *Registry) save() error {
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}

func newToken() (plain string, hash string, err error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b[:])
	sum := sha256.Sum256([]byte(plain))
	hash = hex.EncodeToString(sum[:])
	return plain, hash, nil
}

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(plain)))
	return hex.EncodeToString(sum[:])
}

// ApplyAdvertisement validates the peer token and merges catalog data.
func (r *Registry) ApplyAdvertisement(req AdvertiseRequest, bearer string, autoAdmit bool) (AdvertiseResponse, error) {
	peerID := strings.TrimSpace(req.PeerID)
	if peerID == "" {
		return AdvertiseResponse{}, fmt.Errorf("peer_id required")
	}
	if strings.TrimSpace(req.TailnetHost) == "" {
		return AdvertiseResponse{}, fmt.Errorf("tailnet_host required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	rec, ok := r.data.Peers[peerID]
	issuedToken := ""
	if !ok || !rec.Admitted {
		if !autoAdmit {
			return AdvertiseResponse{Accepted: false, Message: "peer not admitted"}, nil
		}
		plain, h, err := newToken()
		if err != nil {
			return AdvertiseResponse{}, err
		}
		rec = PeerRecord{ID: peerID, Admitted: true, TokenHash: h}
		issuedToken = plain
	}
	if rec.TokenHash != "" && issuedToken == "" {
		if !subtleCompareBearer(bearer, rec.TokenHash) {
			if !autoAdmit {
				return AdvertiseResponse{}, fmt.Errorf("invalid peer token")
			}
			// Local lost peer.local.json; re-issue under auto_admit (OQ2).
			plain, h, err := newToken()
			if err != nil {
				return AdvertiseResponse{}, err
			}
			rec.TokenHash = h
			issuedToken = plain
		}
	}

	rec.MachineID = strings.TrimSpace(req.MachineID)
	rec.TailnetHost = strings.TrimSpace(req.TailnetHost)
	rec.LastSeen = time.Now().UTC()
	rec.Reachable = true
	if rec.Instances == nil {
		rec.Instances = map[string]AdvertisedInstance{}
	}
	for _, inst := range req.Instances {
		key := inst.Type + "/" + inst.ID
		rec.Instances[key] = inst
	}
	r.data.Peers[peerID] = rec
	if err := r.save(); err != nil {
		return AdvertiseResponse{}, err
	}
	resp := AdvertiseResponse{Accepted: true}
	if issuedToken != "" {
		resp.Token = issuedToken
		resp.Message = "peer admitted"
	}
	return resp, nil
}

func subtleCompareBearer(bearer, hash string) bool {
	bearer = strings.TrimSpace(bearer)
	if bearer == "" {
		return false
	}
	return hashToken(bearer) == hash
}

// ListPeers returns all peer records (newest last_seen first, then id).
func (r *Registry) ListPeers() []PeerRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]PeerRecord, 0, len(r.data.Peers))
	for _, p := range r.data.Peers {
		cp := p
		out = append(out, cp)
	}
	sort.Slice(out, func(i, j int) bool {
		ti, tj := out[i].LastSeen, out[j].LastSeen
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// FederatedTargets builds exposure routes for enabled advertised instances.
func (r *Registry) FederatedTargets(spec obelisk.Spec, staleAfter time.Duration) []FederatedTarget {
	if staleAfter <= 0 {
		staleAfter = 2 * time.Minute
	}
	now := time.Now().UTC()
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []FederatedTarget
	for peerID, rec := range r.data.Peers {
		if !rec.Admitted {
			continue
		}
		reachable := rec.Reachable && now.Sub(rec.LastSeen) <= staleAfter
		host := strings.TrimSpace(rec.TailnetHost)
		if host == "" {
			continue
		}
		for _, ad := range rec.Instances {
			if !ad.Enabled {
				continue
			}
			fedID := obelisk.FederatedInstanceID(peerID, ad.ID)
			inst := obelisk.ServiceInstance{
				Type:        ad.Type,
				ID:          fedID,
				Enabled:     true,
				DisplayName: ad.DisplayName,
				Credentials: ad.Credentials,
				Databases:   append([]string(nil), ad.Databases...),
			}
			nt, ntok := obelisk.NativeEdgeTypes[ad.Type]
			ht, htok := obelisk.HTTPEdgeTypes[ad.Type]
			if !ntok && !htok {
				continue
			}
			out = append(out, FederatedTarget{
				PeerID:       peerID,
				Reachable:    reachable,
				Inst:         inst,
				UpstreamHost: host,
				NativePort:   nt,
				HTTPPort:     ht,
			})
		}
	}
	return out
}

// AdmitPeer manually admits a pending peer and returns a new bearer token (OQ2 manual path).
func (r *Registry) AdmitPeer(peerID string) (string, error) {
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return "", fmt.Errorf("peer id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.data.Peers[peerID]
	if !ok {
		rec = PeerRecord{ID: peerID}
	}
	plain, h, err := newToken()
	if err != nil {
		return "", err
	}
	rec.Admitted = true
	rec.TokenHash = h
	r.data.Peers[peerID] = rec
	if err := r.save(); err != nil {
		return "", err
	}
	return plain, nil
}

// MarkStale sets reachable=false for peers not seen recently.
func (r *Registry) MarkStale(staleAfter time.Duration) {
	if staleAfter <= 0 {
		staleAfter = 2 * time.Minute
	}
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	changed := false
	for id, rec := range r.data.Peers {
		ok := now.Sub(rec.LastSeen) <= staleAfter
		if rec.Reachable != ok {
			rec.Reachable = ok
			r.data.Peers[id] = rec
			changed = true
		}
	}
	if changed {
		_ = r.save()
	}
}