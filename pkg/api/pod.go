package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	ErrInvalidPodSpec = errors.New("invalid pod spec")
)

type PodSpec struct {
	Containers []Container `json:"containers" validate:"required,dive,required"`
	Replicas   int32       `json:"replicas" validate:"gte=0"`
}

type Pod struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       PodSpec   `json:"spec" validate:"required"`
	NodeName   string    `json:"nodeName,omitempty"`
	Status     PodStatus `json:"status"`
	// Add other fields as needed
}

// Validate validates the PodSpec of the Pod.
func (p *Pod) Validate() error {
	validate := validator.New()
	if err := validate.Struct(p); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPodSpec, err)
	}

	return nil
}

// IsActive checks if the pod is active.
func (p *Pod) IsActive() bool {
	return p.Status != PodFailed //even succeeded pods should be considered active? or else controller keeps on creating pods
}

func IsPodActiveAndOwnedBy(pod *Pod, meta *ObjectMeta) bool {
	// Check if the pod name contains the ReplicaSet name (ownership)
	return IsOwnedBy(pod, meta) && pod.IsActive()
}

func IsOwnedBy(pod *Pod, meta *ObjectMeta) bool {
	return strings.HasPrefix(pod.Name, meta.Name)
}
