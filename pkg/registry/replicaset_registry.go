package registry

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"gokube/pkg/api"
	"gokube/pkg/storage"
)

const (
	replicaSetPrefix = "/replicasets"
)

var (
	ErrReplicaSetExists   = errors.New("replicaset already exists")
	ErrReplicaSetNotFound = errors.New("replicaset not found")
	ErrListReplicaSets    = errors.New("error listing replicasets")
)

type ReplicaSetRegistry struct {
	storage storage.Storage
	mutex   sync.RWMutex
}

func NewReplicaSetRegistry(storage storage.Storage) *ReplicaSetRegistry {
	return &ReplicaSetRegistry{
		storage: storage,
	}
}

func (r *ReplicaSetRegistry) generateKey(name string) string {
	return fmt.Sprintf("%s/%s", replicaSetPrefix, name)
}

func (r *ReplicaSetRegistry) Create(ctx context.Context, rs *api.ReplicaSet) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := r.generateKey(rs.Name)

	// Check if ReplicaSet already exists
	existingRS := &api.ReplicaSet{}
	if err := r.storage.Get(ctx, key, existingRS); err == nil {
		return fmt.Errorf("%w: %s", ErrReplicaSetExists, rs.Name)
	}

	// Store the ReplicaSet
	return r.storage.Create(ctx, key, rs)
}

func (r *ReplicaSetRegistry) Get(ctx context.Context, name string) (*api.ReplicaSet, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	key := r.generateKey(name)
	rs := &api.ReplicaSet{}
	if err := r.storage.Get(ctx, key, rs); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrReplicaSetNotFound, name)
	}

	return rs, nil
}

func (r *ReplicaSetRegistry) Update(ctx context.Context, rs *api.ReplicaSet) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := r.generateKey(rs.Name)

	// Check if ReplicaSet exists
	existingRS := &api.ReplicaSet{}
	if err := r.storage.Get(ctx, key, existingRS); err != nil {
		return fmt.Errorf("%w: %s", ErrReplicaSetNotFound, rs.Name)
	}

	// Update the ReplicaSet
	return r.storage.Update(ctx, key, rs)
}

func (r *ReplicaSetRegistry) Delete(ctx context.Context, name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := r.generateKey(name)
	return r.storage.Delete(ctx, key)
}

func (r *ReplicaSetRegistry) List(ctx context.Context) ([]*api.ReplicaSet, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var replicaSets []*api.ReplicaSet

	if err := r.storage.List(ctx, replicaSetPrefix, &replicaSets); err != nil {
		return nil, fmt.Errorf("%w", ErrListReplicaSets)
	}

	return replicaSets, nil
}
