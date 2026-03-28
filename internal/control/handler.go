package control

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// RegisterRequest is the payload a node sends when joining the network.
type RegisterRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
	// Endpoint is the node's publicly reachable address (e.g. "1.2.3.4:51820").
	// It is optional at registration time: a node may not yet know its public IP
	// until it completes STUN discovery. The client is expected to update this
	// field after the endpoint is resolved.
	Endpoint string `json:"endpoint"`
}

// RegisterResponse is returned to the node after successful registration.
type RegisterResponse struct {
	ID        string `json:"id"`
	VirtualIP string `json:"virtual_ip"`
}

// Handler holds the dependencies for all HTTP handlers.
type Handler struct {
	store *NodeStore
}

// NewHandler returns a Handler backed by the given NodeStore.
func NewHandler(store *NodeStore) *Handler {
	return &Handler{store: store}
}

// RegisterNode handles POST /api/v1/nodes.
// It registers a new node and returns its assigned virtual IP.
func (h *Handler) RegisterNode(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024) // 4 KB limit

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.PublicKey == "" {
		http.Error(w, "name and public_key are required", http.StatusBadRequest)
		return
	}

	if err := validatePublicKey(req.PublicKey); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	node := &Node{
		ID:        uuid.New().String(),
		Name:      req.Name,
		PublicKey: req.PublicKey,
		Endpoint:  req.Endpoint,
	}

	if err := h.store.Add(node); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(RegisterResponse{
		ID:        node.ID,
		VirtualIP: node.VirtualIP,
	}); err != nil {
		log.Printf("failed to write register response: %v", err)
	}
}

// validatePublicKey checks that key is a valid WireGuard public key:
// base64-encoded 32-byte Curve25519 public key.
func validatePublicKey(key string) error {
	b, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return fmt.Errorf("public_key is not valid base64: %w", err)
	}
	if len(b) != 32 {
		return fmt.Errorf("public_key must be 32 bytes, got %d", len(b))
	}
	return nil
}

// ListNodes handles GET /api/v1/nodes.
// It returns all registered nodes in the network.
func (h *Handler) ListNodes(w http.ResponseWriter, r *http.Request) {
	nodes := h.store.GetAll()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string][]Node{"nodes": nodes}); err != nil {
		log.Printf("failed to write list response: %v", err)
	}
}
