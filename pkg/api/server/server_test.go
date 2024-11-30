package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockStorage "gokube/mocks/pkg/storage"
	"gokube/pkg/storage"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/mock/gomock"
)

func TestNewAPIServer(t *testing.T) {
	t.Run("should create new API server instance", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockStorage.NewMockStorage(ctrl)
		server := NewAPIServer(mockStore)

		assert.NotNil(t, server)
		assert.NotNil(t, server.nodeRegistry)
		assert.NotNil(t, server.podRegistry)
	})
}

func TestAPIServer_Start(t *testing.T) {
	t.Run("should start server and handle requests", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client) {
			store := storage.NewEtcdStorage(etcdServer)
			server := NewAPIServer(store)

			// Start server in a goroutine
			go func() {
				err := server.Start(":0")
				require.NoError(t, err)
			}()

			// Give the server time to start
			time.Sleep(100 * time.Millisecond)
		})
	})

	t.Run("should handle healthz endpoint", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client) {
			store := storage.NewEtcdStorage(etcdServer)
			server := NewAPIServer(store)

			req := httptest.NewRequest("GET", "/api/v1/healthz", nil)
			resp := httptest.NewRecorder()

			container := server.createTestContainer()
			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
		})
	})
}

func TestAPIServer_RegisterRoutes(t *testing.T) {
	t.Run("should register all routes correctly", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client) {
			store := storage.NewEtcdStorage(etcdServer)
			server := NewAPIServer(store)
			container := server.createTestContainer()

			routes := container.RegisteredWebServices()[0].Routes()
			expectedRoutes := map[string]bool{
				"/api/v1/pods:POST":           true, // Create pod
				"/api/v1/pods:GET":            true, // List pods
				"/api/v1/pods/{name}:GET":     true, // Get pod
				"/api/v1/pods/{name}:PUT":     true, // Get pod
				"/api/v1/pods/{name}:DELETE":  true, // Delete pod
				"/api/v1/pods/unassigned:GET": true, // List unassigned pods
				"/api/v1/nodes:POST":          true, // Create node
				"/api/v1/nodes:GET":           true, // List nodes
				"/api/v1/nodes/{name}:GET":    true, // Get node
				"/api/v1/nodes/{name}:PUT":    true, // Get node
				"/api/v1/nodes/{name}:DELETE": true, // Delete node
				"/api/v1/healthz:GET":         true, // Health check
			}

			foundRoutes := make(map[string]bool)
			for _, route := range routes {
				key := route.Path + ":" + route.Method
				foundRoutes[key] = true
			}

			// Verify all expected routes are registered
			for routeKey := range expectedRoutes {
				assert.True(t, foundRoutes[routeKey], "Route %s should be registered", routeKey)
			}
		})
	})
}

// Helper function to create a test container
func (s *APIServer) createTestContainer() *restful.Container {
	container := restful.NewContainer()
	s.registerRoutes(container)
	return container
}

// Helper function to set up a test environment with etcd
func withTestServer(t *testing.T, fn func(*clientv3.Client)) {
	// Set up etcd client for testing
	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:2379"},
	})
	require.NoError(t, err)
	defer client.Close()

	fn(client)
}
