package control

import (
	"fmt"
	"net"
	"sync"
)

// Node represents a peer in the mesh network.
type Node struct {
	ID        string `json:"id"`         // Unique identifier assigned at registration
	Name      string `json:"name"`       // Human-readable label for the node, e.g. "macbook"
	PublicKey string `json:"public_key"` // WireGuard public key, base64-encoded
	Endpoint  string `json:"endpoint"`   // Publicly reachable address, e.g. "1.2.3.4:51820"
	VirtualIP string `json:"virtual_ip"` // Virtual IP assigned by the control plane, e.g. "10.0.0.1"
}

// NodeStore is an in-memory registry of all nodes in the network.
// All methods are safe for concurrent use.
type NodeStore struct {
	mu          sync.RWMutex
	nodes       map[string]*Node // keyed by node ID
	byPublicKey map[string]*Node // keyed by WireGuard public key
	nextIP      int              // monotonically increasing counter for IP allocation
	subnet      string           // first three octets of the virtual subnet, e.g. "10.0.0"
}

// NewNodeStore returns an empty NodeStore ready for use.
//
// Virtual IP allocation uses the 10.0.0.0/24 subnet. This may conflict
// with existing local networks (e.g. home routers on 10.x.x.x).
// Consider a less common range like 100.64.0.0/10 (CGNAT space, used by
// Tailscale) if conflicts arise in production.
func NewNodeStore() *NodeStore {
	return &NodeStore{
		nodes:       make(map[string]*Node),
		byPublicKey: make(map[string]*Node),
		nextIP:      1,
		subnet:      "10.0.0",
	}
}

// allocateIP returns the next available virtual IP in the subnet.
// Must be called with s.mu held for writing.
func (s *NodeStore) allocateIP() (string, error) {
	if s.nextIP > 254 {
		return "", fmt.Errorf("virtual IP address pool exhausted: subnet %s.0/24 is full", s.subnet)
	}
	ip := fmt.Sprintf("%s.%d", s.subnet, s.nextIP)
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("allocated invalid IP address: %s", ip)
	}
	s.nextIP++
	return ip, nil
}

// Add registers a new node or updates the endpoint of an existing one.
// Uniqueness is enforced by public key: re-registering with the same key
// updates the endpoint and returns the previously assigned virtual IP.
func (s *NodeStore) Add(node *Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.byPublicKey[node.PublicKey]; ok {
		existing.Endpoint = node.Endpoint
		node.ID = existing.ID
		node.VirtualIP = existing.VirtualIP
		return nil
	}

	ip, err := s.allocateIP()
	if err != nil {
		return err
	}
	node.VirtualIP = ip
	s.nodes[node.ID] = node
	s.byPublicKey[node.PublicKey] = node
	return nil
}

// GetAll returns a snapshot of all registered nodes.
// Callers receive value copies; mutations to the returned slice
// do not affect the store's internal state.
func (s *NodeStore) GetAll() []Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Node, 0, len(s.nodes))
	for _, n := range s.nodes {
		result = append(result, *n)
	}
	return result
}
