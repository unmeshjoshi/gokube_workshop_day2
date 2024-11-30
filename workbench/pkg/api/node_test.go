package api

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestNodeValidation(t *testing.T) {
	tests := []struct {
		name    string
		node    Node
		wantErr error
	}{
		{
			name: "valid node with all required fields",
			node: Node{
				ObjectMeta: ObjectMeta{
					Name: "test-node",
				},
				Spec:   NodeSpec{},
				Status: NodeReady,
			},
			wantErr: nil,
		},
		{
			name: "valid node with not ready status",
			node: Node{
				ObjectMeta: ObjectMeta{
					Name: "test-node-not-ready",
				},
				Spec:   NodeSpec{},
				Status: NodeNotReady,
			},
			wantErr: nil,
		},
		{
			name: "node with empty name",
			node: Node{
				ObjectMeta: ObjectMeta{
					Name: "",
				},
				Spec:   NodeSpec{},
				Status: NodeReady,
			},
			wantErr: ErrInvalidNodeSpec,
		},
		{
			name: "node with missing name",
			node: Node{
				Spec:   NodeSpec{},
				Status: NodeReady,
			},
			wantErr: ErrInvalidNodeSpec,
		},
		{
			name: "node with memory pressure status",
			node: Node{
				ObjectMeta: ObjectMeta{
					Name: "test-node-memory-pressure",
				},
				Spec:   NodeSpec{},
				Status: NodeMemoryPressure,
			},
			wantErr: nil,
		},
		{
			name:    "empty node",
			node:    Node{},
			wantErr: ErrInvalidNodeSpec,
		},
	}

	validate := validator.New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Validate method
			err := tt.node.Validate()
			assert.Equal(t, tt.wantErr, err)

			// Test struct validation
			err = validate.Struct(tt.node)
			if tt.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
