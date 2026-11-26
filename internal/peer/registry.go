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

const (
	// AdvertiseInterval is how often a local posts its catalog to the gateway.
	AdvertiseInterval = 30 * time.Second
	// HeartbeatStaleAfter is the age of LastSeen after which a peer is no longer
	// considered connected. Every staleness check must use this constant.
	HeartbeatStaleAfter = 2 * time.Minute
)

type registryDisk struct {
	Peers map[string]PeerRecord `json:"peers,omitempty"`
}

// Registry holds admitted peers and their latest catalog snapshots.
//
// One Registry exists per registry file per process (see LoadRegistry), so
// every handler and the reconcile loop share one lock and one in-memory copy.
// That removes the lost-update race where a stale copy saved after a heartbeat
// rolled LastSeen back.
type Registry struct {
	mu   sync.RWMutex
	path string
	data registryDisk
}

var (
	sharedRegistriesMu sync.Mutex
	sharedRegistries   = map[string]*Registry{}
)

func registryPath(configPath string) string {
	return filepath.Join(config.CredentialsDir(configPath), registryFile)
}

// LoadRegistry returns the process-wide registry for configPath, reading the
// file from disk only the first time. If that first read fails, nothing is
// cached and the error is returned, so callers must not treat it as an empty set.
func LoadRegistry(configPath string) (*Registry, error) {
	path := registryPath(configPath)
	sharedRegistriesMu.Lock()
	defer sharedRegistriesMu.Unlock()
	if r, ok := sharedRegistries[path]; ok {
		return r, nil
	}
	r, err := readRegistryFile(path)
	if err != nil {
		return nil, err
	}
	sharedRegistries[path] = r
	return r, nil
}

func readRegistryFile(path string) (*Registry, error) {
	r := &Registry{path: path, data: registryDisk{Peers: map[string]PeerRecord{}}}
	data, err := os.ReadFile(path)
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

// save writes the registry atomically: a unique temp file, fsync, then rename.
// Callers must hold r.mu.
func (r *Registry) save() error {
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, registryFile+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	cleanup := func() { _ = os.Remove(tmp) }
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		cleanup()
		return err
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		cleanup()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		cleanup()
		return err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmp, r.path); err != nil {
		cleanup()
		return err
	}
	return nil
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
	if req.NativeBackhaulPort > 0 {
		rec.NativeBackhaulPort = req.NativeBackhaulPort
	}
	if req.HTTPBackhaulPort > 0 {
		rec.HTTPBackhaulPort = req.HTTPBackhaulPort
	}
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
	now := time.Now().UTC()
	out := make([]PeerRecord, 0, len(r.data.Peers))
	for _, p := range r.data.Peers {
		cp := p
		// Heartbeat freshness is derived at read time, never persisted.
		cp.Reachable = heartbeatFresh(cp.LastSeen, now, HeartbeatStaleAfter)
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
		staleAfter = HeartbeatStaleAfter
	}
	now := time.Now().UTC()
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []FederatedTarget
	for peerID, rec := range r.data.Peers {
		if !rec.Admitted {
			continue
		}
		reachable := heartbeatFresh(rec.LastSeen, now, staleAfter)
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
			_, ntok := obelisk.NativeEdgeTypes[ad.Type]
			_, htok := obelisk.HTTPEdgeTypes[ad.Type]
			if !ntok && !htok {
				continue
			}
			nativePort := rec.NativeBackhaulPort
			if nativePort <= 0 {
				nativePort = obelisk.NativeEdgeTLSPortNumber()
			}
			httpPort := rec.HTTPBackhaulPort
			if httpPort <= 0 {
				httpPort = obelisk.AdminHTTPPort()
			}
			ft := FederatedTarget{
				PeerID:       peerID,
				Reachable:    reachable,
				Inst:         inst,
				UpstreamHost: host,
			}
			if ntok {
				ft.NativePort = nativePort
			}
			if htok {
				ft.HTTPPort = httpPort
			}
			out = append(out, ft)
		}
	}
	return out
}

// RemovePeer deletes a peer and its catalog snapshot from the gateway registry.
func (r *Registry) RemovePeer(peerID string) error {
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return fmt.Errorf("peer id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data.Peers[peerID]; !ok {
		return fmt.Errorf("peer not found")
	}
	delete(r.data.Peers, peerID)
	return r.save()
}

// RemovePeerWithToken deletes the peer when bearer matches its admission token.
func (r *Registry) RemovePeerWithToken(peerID, bearer string) error {
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return fmt.Errorf("peer id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.data.Peers[peerID]
	if !ok {
		return fmt.Errorf("peer not found")
	}
	if rec.TokenHash == "" || !subtleCompareBearer(bearer, rec.TokenHash) {
		return fmt.Errorf("invalid peer token")
	}
	delete(r.data.Peers, peerID)
	return r.save()
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

// heartbeatFresh reports whether a peer has advertised within staleAfter.
// A zero LastSeen (never advertised) is never fresh.
func heartbeatFresh(lastSeen, now time.Time, staleAfter time.Duration) bool {
	if lastSeen.IsZero() {
		return false
	}
	age := now.Sub(lastSeen)
	if age < 0 {
		// Clock stepped backwards: treat as fresh, the advertise just happened.
		return true
	}
	return age <= staleAfter
}
