package control

import (
	"fmt"
	"sync"
	"testing"
)

func TestNodeStore_Add_firstNodeGetsFirstIP(t *testing.T) {
	store := NewNodeStore()
	node := &Node{ID: "1", PublicKey: "key1"}

	if err := store.Add(node); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.VirtualIP != "10.0.0.1" {
		t.Errorf("got VirtualIP %q, want %q", node.VirtualIP, "10.0.0.1")
	}
}

func TestNodeStore_Add_incrementsIPForEachNode(t *testing.T) {
	store := NewNodeStore()
	node1 := &Node{ID: "1", PublicKey: "key1"}
	node2 := &Node{ID: "2", PublicKey: "key2"}

	if err := store.Add(node1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := store.Add(node2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if node1.VirtualIP != "10.0.0.1" {
		t.Errorf("node1: got %q, want %q", node1.VirtualIP, "10.0.0.1")
	}
	if node2.VirtualIP != "10.0.0.2" {
		t.Errorf("node2: got %q, want %q", node2.VirtualIP, "10.0.0.2")
	}
}

func TestNodeStore_Add_deduplicatesByPublicKey(t *testing.T) {
	store := NewNodeStore()

	node1 := &Node{ID: "1", PublicKey: "key1", Endpoint: "1.2.3.4:51820"}
	if err := store.Add(node1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assignedIP := node1.VirtualIP

	// Re-register same public key with a new endpoint (simulates roaming).
	node2 := &Node{ID: "2", PublicKey: "key1", Endpoint: "5.6.7.8:51820"}
	if err := store.Add(node2); err != nil {
		t.Fatalf("unexpected error on re-registration: %v", err)
	}

	if node2.VirtualIP != assignedIP {
		t.Errorf("re-registration got VirtualIP %q, want original %q", node2.VirtualIP, assignedIP)
	}

	nodes := store.GetAll()
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	if nodes[0].Endpoint != "5.6.7.8:51820" {
		t.Errorf("endpoint not updated: got %q, want %q", nodes[0].Endpoint, "5.6.7.8:51820")
	}
}

func TestNodeStore_Add_poolExhaustion(t *testing.T) {
	store := NewNodeStore()

	for i := 0; i < 254; i++ {
		node := &Node{
			ID:        fmt.Sprintf("id-%d", i),
			PublicKey: fmt.Sprintf("key-%d", i),
		}
		if err := store.Add(node); err != nil {
			t.Fatalf("unexpected error at node %d: %v", i, err)
		}
	}

	overflow := &Node{ID: "overflow", PublicKey: "key-overflow"}
	if err := store.Add(overflow); err == nil {
		t.Error("expected error when pool is exhausted, got nil")
	}
}

func TestNodeStore_GetAll_emptyStoreReturnsEmptySlice(t *testing.T) {
	store := NewNodeStore()
	nodes := store.GetAll()

	if nodes == nil {
		t.Error("GetAll on empty store returned nil, want empty slice")
	}
	if len(nodes) != 0 {
		t.Errorf("got %d nodes, want 0", len(nodes))
	}
}

func TestNodeStore_GetAll_returnsSnapshot(t *testing.T) {
	store := NewNodeStore()
	if err := store.Add(&Node{ID: "1", PublicKey: "key1", Endpoint: "original"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshot := store.GetAll()
	snapshot[0].Endpoint = "mutated"

	fresh := store.GetAll()
	if fresh[0].Endpoint != "original" {
		t.Errorf("store was mutated through snapshot: got %q, want %q", fresh[0].Endpoint, "original")
	}
}

func TestNodeStore_concurrent(t *testing.T) {
	store := NewNodeStore()
	const goroutines = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			if err := store.Add(&Node{
				ID:        fmt.Sprintf("id-%d", i),
				PublicKey: fmt.Sprintf("key-%d", i),
			}); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			store.GetAll()
		}()
	}

	wg.Wait()
}
