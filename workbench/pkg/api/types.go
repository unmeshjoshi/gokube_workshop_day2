package api

import (
	"errors"
	"time"
)

type PodStatus string

const (
	// PodPending means the pod has been accepted by the system, but one or more of the containers
	// has not been started. This includes time before being bound to a node, as well as time spent
	// pulling images onto the host.
	PodPending PodStatus = "Pending"

	// PodRunning means the pod has been bound to a node and all of the containers have been started.
	// At least one container is still running or is in the process of being restarted.
	PodRunning PodStatus = "Running"

	// PodSucceeded means that all containers in the pod have voluntarily terminated
	// with a container exit code of 0, and the system is not going to restart any of these containers.
	PodSucceeded PodStatus = "Succeeded"

	// PodFailed means that all containers in the pod have terminated, and at least one container has
	// terminated in a failure (exited with a non-zero exit code or was stopped by the system).
	PodFailed PodStatus = "Failed"

	//TODO: Kubernetes separates PodPhase and PodCondition. We have simplified to have a single pod status.
	PodScheduled PodStatus = "Scheduled"
)

var (
	ErrInvalidNodeSpec = errors.New("invalid node spec")
)

type Container struct {
	Name  string `json:"name" validate:"required"`
	Image string `json:"image" validate:"required"`
}

// ObjectMeta is minimal metadata that all persisted resources must have
type ObjectMeta struct {
	Name              string    `json:"name" validate:"required"`
	Namespace         string    `json:"namespace,omitempty"`
	UID               string    `json:"uid,omitempty"`
	ResourceVersion   string    `json:"resourceVersion,omitempty"`
	CreationTimestamp time.Time `json:"creationTimestamp,omitempty"`
}

// NodeSpec describes the basic attributes of a node
type NodeSpec struct {
	Unschedulable bool   `json:"unschedulable,omitempty"`
	ProviderID    string `json:"providerID,omitempty"`
}

type NodeStatus string

// Define some constants for NodeConditionType and ConditionStatus
const (
	NodeNotReady       NodeStatus = "NotReady"
	NodeReady          NodeStatus = "Ready"
	NodeMemoryPressure NodeStatus = "MemoryPressure"
	NodeDiskPressure   NodeStatus = "DiskPressure"
)

// ReplicaSet represents the configuration of a ReplicaSet
type ReplicaSet struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       ReplicaSetSpec   `json:"spec"`
	Status     ReplicaSetStatus `json:"status,omitempty"`
}

// ReplicaSetSpec is the specification of a ReplicaSet
type ReplicaSetSpec struct {
	Replicas int32             `json:"replicas"`
	Selector map[string]string `json:"selector"`
	Template PodTemplateSpec   `json:"template"`
}

// PodTemplateSpec describes the data a pod should have when created from a template
type PodTemplateSpec struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       PodSpec `json:"spec"`
}

// ReplicaSetStatus represents the current status of a ReplicaSet
type ReplicaSetStatus struct {
	Replicas             int32 `json:"replicas"`
	FullyLabeledReplicas int32 `json:"fullyLabeledReplicas,omitempty"`
	ReadyReplicas        int32 `json:"readyReplicas,omitempty"`
	AvailableReplicas    int32 `json:"availableReplicas,omitempty"`
}
