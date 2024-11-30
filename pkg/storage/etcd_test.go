package storage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartEmbeddedEtcd(t *testing.T) {
	etcd, port, err := StartEmbeddedEtcd()
	defer StopEmbeddedEtcd(etcd)
	assert.NoError(t, err, "Failed to start embedded etcd")

	assert.NotNil(t, etcd, "Expected etcd instance, got nil")
	assert.NotEqual(t, 0, port, "Expected non-zero port, got 0")
}

func TestPickAvailableRandomPort(t *testing.T) {
	port, err := PickAvailableRandomPort()
	assert.NoError(t, err, "Failed to pick available random port")
	assert.NotEqual(t, 0, port, "Expected non-zero port, got 0")
}

func TestStopEmbeddedEtcd(t *testing.T) {
	etcd, _, err := StartEmbeddedEtcd()
	assert.NoError(t, err, "Failed to start embedded etcd")

	StopEmbeddedEtcd(etcd)

	_, err = os.Stat(etcd.Config().Dir)
	assert.True(t, os.IsNotExist(err), "Expected data directory to be removed, but it still exists")
}
