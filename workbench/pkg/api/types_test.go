package api

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestContainerValidation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name      string
		container Container
		wantErr   string
	}{
		{
			name: "valid container with required fields",
			container: Container{
				Name:  "nginx-container",
				Image: "nginx:latest",
			},
			wantErr: "",
		},
		{
			name: "missing container name",
			container: Container{
				Image: "nginx:latest",
			},
			wantErr: "Key: 'Container.Name' Error:Field validation for 'Name' failed on the 'required' tag",
		},
		{
			name: "missing container image",
			container: Container{
				Name: "nginx-container",
			},
			wantErr: "Key: 'Container.Image' Error:Field validation for 'Image' failed on the 'required' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.container)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
func TestObjectMetaValidation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name       string
		objectMeta ObjectMeta
		wantErr    string
	}{
		{
			name: "valid ObjectMeta with required fields",
			objectMeta: ObjectMeta{
				Name: "test-object",
			},
			wantErr: "",
		},
		{
			name: "valid ObjectMeta with all fields",
			objectMeta: ObjectMeta{
				Name:      "test-object",
				Namespace: "default",
				UID:       "123e4567-e89b-12d3-a456-426614174000",
			},
			wantErr: "",
		},
		{
			name:       "missing name field",
			objectMeta: ObjectMeta{},
			wantErr:    "Key: 'ObjectMeta.Name' Error:Field validation for 'Name' failed on the 'required' tag",
		},
		{
			name: "empty name field",
			objectMeta: ObjectMeta{
				Name: "",
			},
			wantErr: "Key: 'ObjectMeta.Name' Error:Field validation for 'Name' failed on the 'required' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.objectMeta)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
