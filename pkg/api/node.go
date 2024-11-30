package api

import "github.com/go-playground/validator/v10"

// Node is a simplified representation of a Kubernetes Node
type Node struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       NodeSpec   `json:"spec,omitempty"`
	Status     NodeStatus `json:"status,omitempty"`
}

// Validate checks if the Node configuration is valid
func (n *Node) Validate() error {
	validate := validator.New()
	if err := validate.Struct(n); err != nil {
		return ErrInvalidNodeSpec
	}

	return nil
}
