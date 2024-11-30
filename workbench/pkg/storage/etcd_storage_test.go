package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type TestObject struct {
	Name string `json:"name"`
}

func TestEtcdStorage_Create(t *testing.T) {
	TestWithEmbeddedEtcd(t, func(t *testing.T, cli *clientv3.Client) {
		storage := NewEtcdStorage(cli)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		obj := &TestObject{Name: "test-value"}
		err := storage.Create(ctx, "test-key", obj)
		assert.NoError(t, err)

		var retrievedObj TestObject
		err = storage.Get(ctx, "test-key", &retrievedObj)
		assert.NoError(t, err)

		assert.Equal(t, "test-value", retrievedObj.Name)
	})
}

func TestEtcdStorage_Update(t *testing.T) {
	TestWithEmbeddedEtcd(t, func(t *testing.T, cli *clientv3.Client) {
		storage := NewEtcdStorage(cli)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		obj := &TestObject{Name: "test-value"}
		err := storage.Create(ctx, "test-key", obj)
		assert.NoError(t, err)

		updatedObj := &TestObject{Name: "updated-value"}
		err = storage.Update(ctx, "test-key", updatedObj)
		assert.NoError(t, err)

		var retrievedObj TestObject
		err = storage.Get(ctx, "test-key", &retrievedObj)
		assert.NoError(t, err)

		assert.Equal(t, "updated-value", retrievedObj.Name)
	})
}

func TestEtcdStorage_Delete(t *testing.T) {
	TestWithEmbeddedEtcd(t, func(t *testing.T, cli *clientv3.Client) {
		storage := NewEtcdStorage(cli)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		obj := &TestObject{Name: "test-value"}
		err := storage.Create(ctx, "test-key", obj)
		assert.NoError(t, err)

		err = storage.Delete(ctx, "test-key")
		assert.NoError(t, err)

		var retrievedObj TestObject
		err = storage.Get(ctx, "test-key", &retrievedObj)
		assert.Error(t, err)
	})
}

func TestEtcdStorage_List(t *testing.T) {
	TestWithEmbeddedEtcd(t, func(t *testing.T, cli *clientv3.Client) {
		storage := NewEtcdStorage(cli)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		obj1 := &TestObject{Name: "value1"}
		err := storage.Create(ctx, "/prefix/key1", obj1)
		assert.NoError(t, err)

		obj2 := &TestObject{Name: "value2"}
		err = storage.Create(ctx, "/prefix/key2", obj2)
		assert.NoError(t, err)

		var list []*TestObject
		err = storage.List(ctx, "/prefix/", &list)
		assert.NoError(t, err)

		assert.Len(t, list, 2)
		assert.ElementsMatch(t, []*TestObject{obj1, obj2}, list)
	})
}

func TestWatch(t *testing.T) {
	TestWithEmbeddedEtcd(t, func(t *testing.T, cli *clientv3.Client) {
		watchKey := "/watch-test/key"
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		watchChan := cli.Watch(ctx, watchKey)

		go func() {
			time.Sleep(1 * time.Second)
			_, err := cli.Put(ctx, watchKey, "initial-value")
			assert.NoError(t, err)

			time.Sleep(1 * time.Second)
			_, err = cli.Put(ctx, watchKey, "updated-value")
			assert.NoError(t, err)

			time.Sleep(1 * time.Second)
			_, err = cli.Delete(ctx, watchKey)
			assert.NoError(t, err)
		}()

		expectedEvents := []struct {
			Type  mvccpb.Event_EventType
			Value string
		}{
			{mvccpb.PUT, "initial-value"},
			{mvccpb.PUT, "updated-value"},
			{mvccpb.DELETE, ""},
		}

		for _, expected := range expectedEvents {
			select {
			case watchResp := <-watchChan:
				assert.Len(t, watchResp.Events, 1)

				ev := watchResp.Events[0]
				assert.Equal(t, expected.Type, ev.Type)
				assert.Equal(t, expected.Value, string(ev.Kv.Value))
			case <-ctx.Done():
				t.Fatalf("Watch timed out")
			}
		}
	})
}
