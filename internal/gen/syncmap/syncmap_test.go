package syncmap_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/simulot/immich-go/internal/gen/syncmap"
)

func TestStoreAndLoad(t *testing.T) {
	m := syncmap.New[string, int]()
	m.Store("a", 1)
	v, ok := m.Load("a")
	if !ok || v != 1 {
		t.Errorf("expected 1, got %v", v)
	}
}

func TestLoadOrStore(t *testing.T) {
	m := syncmap.New[string, int]()
	v, loaded := m.LoadOrStore("a", 10)
	if loaded || v != 10 {
		t.Errorf("expected to store new key 'a' -> 10")
	}
	v, loaded = m.LoadOrStore("a", 20)
	if !loaded || v != 10 {
		t.Errorf("expected 'a' to be loaded with value 10")
	}
	v, loaded = m.LoadOrStore("a", 10)
	if !loaded || v != 10 {
		t.Errorf("expected to load existing key 'a' -> 10")
	}
}

func TestKeys(t *testing.T) {
	m := syncmap.New[string, bool]()
	m.Store("x", true)
	m.Store("y", false)
	keys := m.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestCompareAndSwap(t *testing.T) {
	m := syncmap.New[string, int]()
	m.Store("x", 5)
	swapped := m.CompareAndSwap("x", 5, 10)
	if !swapped {
		t.Error("expected CompareAndSwap to succeed")
	}
	v, _ := m.Load("x")
	if v != 10 {
		t.Errorf("expected 'x' value to be 10, got %v", v)
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := syncmap.New[string, int]()
	var wg sync.WaitGroup
	const goroutines = 50
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			m.Store(fmt.Sprintf("key%d", idx), idx)
		}(i)
	}
	wg.Wait()
	for i := 0; i < goroutines; i++ {
		v, ok := m.Load(fmt.Sprintf("key%d", i))
		if !ok {
			t.Errorf("expected 'key%d' to be present", i)
		}
		if v != i {
			t.Errorf("unexpected value %v stored under 'key%d'", v, i)
		}
	}
}
