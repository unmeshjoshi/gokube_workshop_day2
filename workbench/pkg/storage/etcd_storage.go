package storage

import (
	"context"
	"fmt"
	"reflect"

	"gokube/pkg/runtime"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdStorage implements the Storage interface using etcd
type EtcdStorage struct {
	client *clientv3.Client
}

// NewEtcdStorage creates a new EtcdStorage
func NewEtcdStorage(client *clientv3.Client) *EtcdStorage {
	return &EtcdStorage{client: client}
}

var (
	ErrEncoding   = fmt.Errorf("error encoding object")
	ErrDecoding   = fmt.Errorf("error decoding object")
	ErrNotFound   = fmt.Errorf("object not found")
	ErrEtcdClient = fmt.Errorf("etcd client error")
)

func (s *EtcdStorage) Create(ctx context.Context, key string, obj runtime.Object) error {
	data, err := runtime.Encode(obj)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEncoding, err)
	}

	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEtcdClient, err)
	}
	return nil
}

func (s *EtcdStorage) Get(ctx context.Context, key string, obj runtime.Object) error {
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEtcdClient, err)
	}

	if len(resp.Kvs) == 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, key)
	}

	if err := runtime.Decode(resp.Kvs[0].Value, obj); err != nil {
		return fmt.Errorf("%w: %v", ErrDecoding, err)
	}
	return nil
}

func (s *EtcdStorage) Update(ctx context.Context, key string, obj runtime.Object) error {
	data, err := runtime.Encode(obj)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEncoding, err)
	}

	if _, err = s.client.Put(ctx, key, string(data)); err != nil {
		return fmt.Errorf("%w: %v", ErrEtcdClient, err)
	}
	return nil
}

func (s *EtcdStorage) Delete(ctx context.Context, key string) error {
	if _, err := s.client.Delete(ctx, key); err != nil {
		return fmt.Errorf("%w: %v", ErrEtcdClient, err)
	}

	return nil
}

func (s *EtcdStorage) List(ctx context.Context, prefix string, listObj interface{}) error {
	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEtcdClient, err)
	}

	listValue := reflect.ValueOf(listObj)
	if listValue.Kind() != reflect.Ptr || listValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("listObj must be a pointer to a slice")
	}

	sliceValue := listValue.Elem()
	elementType := sliceValue.Type().Elem()

	for _, kv := range resp.Kvs {
		obj := reflect.New(elementType.Elem()).Interface().(runtime.Object)
		if err := runtime.Decode(kv.Value, obj); err != nil {
			return fmt.Errorf("%w: %v", ErrDecoding, err)
		}
		sliceValue = reflect.Append(sliceValue, reflect.ValueOf(obj))
	}

	listValue.Elem().Set(sliceValue)
	return nil
}

func (s *EtcdStorage) DeletePrefix(ctx context.Context, prefix string) error {
	if _, err := s.client.Delete(ctx, prefix, clientv3.WithPrefix()); err != nil {
		return fmt.Errorf("%w: %v", ErrEtcdClient, err)
	}

	return nil
}
