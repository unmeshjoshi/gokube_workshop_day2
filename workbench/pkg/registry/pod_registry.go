package registry

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"gokube/pkg/api"
	"gokube/pkg/storage"
)

const podPrefix = "/pods/"

var (
	ErrPodAlreadyExists = errors.New("pod already exists")
	ErrPodNotFound      = errors.New("pod not found")
	ErrListPodsFailed   = errors.New("failed to list pods")
	ErrPodInvalid       = errors.New("invalid pod")
)

// PodRegistry provides thread-safe operations for managing Pod objects in the storage.
type PodRegistry struct {
	storage storage.Storage
	mutex   sync.RWMutex
}

// NewPodRegistry creates a new PodRegistry with the given storage.
func NewPodRegistry(storage storage.Storage) *PodRegistry {
	return &PodRegistry{
		storage: storage,
	}
}

func (r *PodRegistry) generateKey(podName string) string {
	return fmt.Sprintf("%s%s", podPrefix, podName)
}

// CreatePod creates a new pod in the registry.
// It returns an error if the pod already exists or if the pod spec is invalid.
// If the pod status is not set, it defaults to api.PodPending.
func (r *PodRegistry) CreatePod(ctx context.Context, pod *api.Pod) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := r.generateKey(pod.Name)
	existingPod := &api.Pod{}
	err := r.storage.Get(ctx, key, existingPod)
	if err == nil {
		return fmt.Errorf("%w: %s", ErrPodAlreadyExists, pod.Name)
	}

	if pod.Status == "" {
		pod.Status = api.PodPending
	}

	// Validate Pod spec
	if err := pod.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrPodInvalid, err)
	}

	return r.storage.Create(ctx, key, pod)
}

// GetPod retrieves a Pod by its name from the registry.
// It returns the Pod object if found, otherwise it returns an error indicating that the Pod was not found.
func (r *PodRegistry) GetPod(ctx context.Context, name string) (*api.Pod, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	key := r.generateKey(name)
	pod := &api.Pod{}
	if err := r.storage.Get(ctx, key, pod); err != nil {
		switch {
		case errors.Is(err, storage.ErrNotFound):
			return nil, fmt.Errorf("%w: %s", ErrPodNotFound, name)
		default:
			return nil, fmt.Errorf("%w: failed to get pod: %v", ErrInternal, err)
		}
	}

	return pod, nil
}

// UpdatePod updates an existing Pod in the registry.
// It returns an error if the Pod spec is invalid.
func (r *PodRegistry) UpdatePod(ctx context.Context, pod *api.Pod) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := r.generateKey(pod.Name)

	// Validate Pod spec
	if err := pod.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrPodInvalid, err)
	}

	return r.storage.Update(ctx, key, pod)
}

// DeletePod removes a Pod from the registry by its name.
// It returns an error if the deletion fails.
func (r *PodRegistry) DeletePod(ctx context.Context, name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := r.generateKey(name)
	return r.storage.Delete(ctx, key)
}

// ListPods retrieves all Pods from the registry.
// It returns a slice of Pod objects and an error if the listing fails.
func (r *PodRegistry) ListPods(ctx context.Context) ([]*api.Pod, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var pods []*api.Pod
	if err := r.storage.List(ctx, podPrefix, &pods); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrListPodsFailed, err)
	}

	return pods, nil
}

// listPodsByStatus retrieves all Pods with a specific status from the registry.
// It returns a slice of Pod objects with the given status and an error if the listing fails.
func (r *PodRegistry) listPodsByStatus(ctx context.Context, status api.PodStatus) ([]*api.Pod, error) {
	pods, err := r.ListPods(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrListPodsFailed, err)
	}

	filteredPods := make([]*api.Pod, 0)
	for _, pod := range pods {
		if pod.Status == status {
			filteredPods = append(filteredPods, pod)
		}
	}

	return filteredPods, nil
}

// ListUnassignedPods retrieves all Pods with a status of PodPending from the registry.
// It returns a slice of unassigned Pod objects and an error if the listing fails.
func (r *PodRegistry) ListUnassignedPods(ctx context.Context) ([]*api.Pod, error) {
	return r.listPodsByStatus(ctx, api.PodPending)
}

// ListPendingPods retrieves all Pods with a status of PodPending from the registry.
// It returns a slice of pending Pod objects and an error if the listing fails.
func (r *PodRegistry) ListPendingPods(ctx context.Context) ([]*api.Pod, error) {
	return r.listPodsByStatus(ctx, api.PodPending)
}
