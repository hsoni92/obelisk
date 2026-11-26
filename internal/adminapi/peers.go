package adminapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/peer"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type peerResignRequest struct {
	PeerID string `json:"peer_id"`
}

func (s *Server) handlePeerResign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !obelisk.IsGateway(spec) {
		http.Error(w, "not a gateway appliance", http.StatusForbidden)
		return
	}
	var req peerResignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	peerID := strings.TrimSpace(req.PeerID)
	if peerID == "" {
		http.Error(w, "peer_id required", http.StatusBadRequest)
		return
	}
	bearer := ""
	if h := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(h), "bearer ") {
		bearer = strings.TrimSpace(h[7:])
	}
	reg, err := peer.LoadRegistry(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := reg.RemovePeerWithToken(peerID, bearer); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	rec := s.reconciler(cfg)
	_ = rec.ReconcileExposurePeers(r.Context(), cfg)
	writeJSON(w, map[string]any{"ok": true, "peer_id": peerID})
}

func (s *Server) handlePeerAdvertise(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !obelisk.IsGateway(spec) {
		http.Error(w, "not a gateway appliance", http.StatusForbidden)
		return
	}
	var req peer.AdvertiseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.EqualFold(strings.TrimSpace(req.DeploymentRole), obelisk.DeploymentRoleGateway) {
		http.Error(w, "gateway appliances cannot register as peers (OQ3)", http.StatusBadRequest)
		return
	}
	bearer := ""
	if h := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(h), "bearer ") {
		bearer = strings.TrimSpace(h[7:])
	}
	reg, err := peer.LoadRegistry(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	autoAdmit := obelisk.PeerGatewayAutoAdmit(spec)
	resp, err := reg.ApplyAdvertisement(req, bearer, autoAdmit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if resp.Accepted {
		resp.EdgeBaseDomain = obelisk.PublicEdgeBaseDomain()
		rec := s.reconciler(cfg)
		_ = rec.ReconcileExposurePeers(r.Context(), cfg)
	}
	writeJSON(w, resp)
}

func (s *Server) handleDeletePeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "peer id required", http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !obelisk.IsGateway(spec) {
		http.Error(w, "not a gateway appliance", http.StatusForbidden)
		return
	}
	reg, err := peer.LoadRegistry(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := reg.RemovePeer(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rec := s.reconciler(cfg)
	_ = rec.ReconcileExposurePeers(r.Context(), cfg)
	writeJSON(w, map[string]any{"ok": true, "peer_id": id})
}

func (s *Server) handleAdmitPeer(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "peer id required", http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !obelisk.IsGateway(spec) {
		http.Error(w, "not a gateway appliance", http.StatusForbidden)
		return
	}
	reg, err := peer.LoadRegistry(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	token, err := reg.AdmitPeer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"peer_id": id, "token": token})
}

func (s *Server) handleListPeers(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	reg, err := peer.LoadRegistry(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"peers": reg.ListPeers()})
}
