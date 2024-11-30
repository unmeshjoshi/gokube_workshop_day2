package storage

import (
	"context"

	"gokube/pkg/runtime"
)

// Storage defines the interface for data storage operations
//
//go:generate $PROJECT_HOME/bin/mock mocks/pkg/storage
type Storage interface {
	Create(ctx context.Context, key string, obj runtime.Object) error
	Get(ctx context.Context, key string, obj runtime.Object) error
	Update(ctx context.Context, key string, obj runtime.Object) error
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) error
	List(ctx context.Context, prefix string, listObj interface{}) error
}
