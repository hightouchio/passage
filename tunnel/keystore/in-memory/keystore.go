package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"sync"
)

// InMemory is a keystore that stores all keys in process memory. It is useful for tests.
type InMemory struct {
	keys map[uuid.UUID][]byte
	mux  sync.RWMutex
}

func New() *InMemory {
	return &InMemory{keys: make(map[uuid.UUID][]byte)}
}

func (p *InMemory) Get(ctx context.Context, id uuid.UUID) ([]byte, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	key, ok := p.keys[id]
	if !ok {
		return []byte{}, fmt.Errorf("key does not exist")
	}
	return key, nil
}

func (p *InMemory) Set(ctx context.Context, id uuid.UUID, contents []byte) error {
	p.mux.Lock()
	defer p.mux.Unlock()

	p.keys[id] = contents
	return nil
}

func (p *InMemory) Delete(ctx context.Context, id uuid.UUID) error {
	p.mux.Lock()
	defer p.mux.Unlock()

	delete(p.keys, id)
	return nil
}
