package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	mockStorage "gokube/mocks/pkg/storage"
	"gokube/pkg/api"
	"gokube/pkg/registry"
	"gokube/pkg/storage"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/mock/gomock"
)

func TestCreateNode(t *testing.T) {
	t.Run("should create a new node", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			nodeRegistry := registry.NewNodeRegistry(store)
			handler := NewNodeHandler(nodeRegistry)

			RegisterNodeRoutes(ws, handler)

			node := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node",
				},
				Spec: api.NodeSpec{},
			}

			body, _ := json.Marshal(node)
			req := httptest.NewRequest("POST", "/api/v1/nodes", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusCreated, resp.Code)

			var createdNode api.Node
			err := json.Unmarshal(resp.Body.Bytes(), &createdNode)
			assert.NoError(t, err)

			require.NotEmpty(t, createdNode)

			assert.Equal(t, node.Name, createdNode.Name)
		})
	})

	t.Run("should return bad request for invalid node", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			nodeRegistry := registry.NewNodeRegistry(store)
			handler := NewNodeHandler(nodeRegistry)

			RegisterNodeRoutes(ws, handler)

			invalidNode := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "",
				},
				Spec: api.NodeSpec{},
			}

			body, _ := json.Marshal(invalidNode)
			req := httptest.NewRequest("POST", "/api/v1/nodes", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code)
		})
	})

	t.Run("should return internal server error for registry failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockStorage.NewMockStorage(ctrl)
		nodeRegistry := registry.NewNodeRegistry(mockStore)
		handler := NewNodeHandler(nodeRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterNodeRoutes(ws, handler)

			mockStore.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))
			mockStore.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			node := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node",
				},
				Spec: api.NodeSpec{},
			}

			body, _ := json.Marshal(node)
			req := httptest.NewRequest("POST", "/api/v1/nodes", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	})

	t.Run("should return conflict error when node already exists", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			nodeRegistry := registry.NewNodeRegistry(store)
			handler := NewNodeHandler(nodeRegistry)
			ctx := context.Background()

			RegisterNodeRoutes(ws, handler)

			// Create initial node
			node := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node",
				},
				Spec: api.NodeSpec{},
			}

			err := nodeRegistry.CreateNode(ctx, node)
			require.NoError(t, err)

			// Try to create same node again
			body, _ := json.Marshal(node)
			req := httptest.NewRequest("POST", "/api/v1/nodes", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusConflict, resp.Code)
		})
	})
}

func TestGetNode(t *testing.T) {
	t.Run("should get existing node", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			nodeRegistry := registry.NewNodeRegistry(store)
			handler := NewNodeHandler(nodeRegistry)
			ctx := context.Background()

			RegisterNodeRoutes(ws, handler)

			node := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node",
				},
				Spec: api.NodeSpec{},
			}

			err := nodeRegistry.CreateNode(ctx, node)
			require.NoError(t, err)

			req := httptest.NewRequest("GET", "/api/v1/nodes/test-node", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)

			var returnedNode api.Node
			err = json.Unmarshal(resp.Body.Bytes(), &returnedNode)
			assert.NoError(t, err)

			assert.Equal(t, node.Name, returnedNode.Name)
		})
	})

	t.Run("should return not found for non-existent node", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			nodeRegistry := registry.NewNodeRegistry(store)
			handler := NewNodeHandler(nodeRegistry)

			RegisterNodeRoutes(ws, handler)

			req := httptest.NewRequest("GET", "/api/v1/nodes/non-existent-node", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusNotFound, resp.Code)
		})
	})

	t.Run("should return internal server error for registry failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockStorage.NewMockStorage(ctrl)
		nodeRegistry := registry.NewNodeRegistry(mockStore)
		handler := NewNodeHandler(nodeRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterNodeRoutes(ws, handler)

			mockStore.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			req := httptest.NewRequest("GET", "/api/v1/nodes/test-node", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	})
}

func TestUpdateNode(t *testing.T) {
	t.Run("should update existing node", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			nodeRegistry := registry.NewNodeRegistry(store)
			handler := NewNodeHandler(nodeRegistry)
			ctx := context.Background()

			RegisterNodeRoutes(ws, handler)

			// Create initial node
			node := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node",
				},
				Spec: api.NodeSpec{},
			}

			err := nodeRegistry.CreateNode(ctx, node)
			require.NoError(t, err)

			// Update node
			updatedNode := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node",
				},
				Spec: api.NodeSpec{
					Unschedulable: true,
				},
			}

			body, _ := json.Marshal(updatedNode)
			req := httptest.NewRequest("PUT", "/api/v1/nodes/test-node", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			require.Equal(t, http.StatusOK, resp.Code)

			var returnedNode api.Node
			err = json.Unmarshal(resp.Body.Bytes(), &returnedNode)
			assert.NoError(t, err)
			assert.True(t, returnedNode.Spec.Unschedulable)
		})
	})

	t.Run("should return bad request when node names don't match", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			nodeRegistry := registry.NewNodeRegistry(store)
			handler := NewNodeHandler(nodeRegistry)
			ctx := context.Background()

			RegisterNodeRoutes(ws, handler)

			// Create the initial node first
			existingNode := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node",
				},
				Spec: api.NodeSpec{},
			}
			err := nodeRegistry.CreateNode(ctx, existingNode)
			require.NoError(t, err)

			// Try to update with mismatched name
			nodeUpdate := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "different-name",
				},
			}

			body, _ := json.Marshal(nodeUpdate)
			req := httptest.NewRequest("PUT", "/api/v1/nodes/test-node", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code)
		})
	})

	t.Run("should return bad request for invalid node", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			nodeRegistry := registry.NewNodeRegistry(store)
			handler := NewNodeHandler(nodeRegistry)
			ctx := context.Background()

			RegisterNodeRoutes(ws, handler)

			// Create initial node
			node := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node",
				},
				Spec: api.NodeSpec{},
			}
			err := nodeRegistry.CreateNode(ctx, node)
			require.NoError(t, err)

			// Try to update with invalid node
			invalidNode := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "",
				},
				Spec: api.NodeSpec{},
			}

			body, _ := json.Marshal(invalidNode)
			req := httptest.NewRequest("PUT", "/api/v1/nodes/test-node", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code)
		})
	})

	t.Run("should return internal server error for registry failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockStorage.NewMockStorage(ctrl)
		nodeRegistry := registry.NewNodeRegistry(mockStore)
		handler := NewNodeHandler(nodeRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterNodeRoutes(ws, handler)

			mockStore.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			node := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node",
				},
				Spec: api.NodeSpec{},
			}

			body, _ := json.Marshal(node)
			req := httptest.NewRequest("PUT", "/api/v1/nodes/test-node", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	})

	t.Run("should return not found for non-existent node", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			nodeRegistry := registry.NewNodeRegistry(store)
			handler := NewNodeHandler(nodeRegistry)

			RegisterNodeRoutes(ws, handler)

			node := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "non-existent-node",
				},
				Spec: api.NodeSpec{},
			}

			body, _ := json.Marshal(node)
			req := httptest.NewRequest("PUT", "/api/v1/nodes/non-existent-node", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusNotFound, resp.Code)
		})
	})
}

func TestDeleteNode(t *testing.T) {
	t.Run("should delete existing node", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			nodeRegistry := registry.NewNodeRegistry(store)
			handler := NewNodeHandler(nodeRegistry)
			ctx := context.Background()

			RegisterNodeRoutes(ws, handler)

			// Create a node first
			node := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node",
				},
				Spec: api.NodeSpec{},
			}

			err := nodeRegistry.CreateNode(ctx, node)
			require.NoError(t, err)

			// Delete the node
			req := httptest.NewRequest("DELETE", "/api/v1/nodes/test-node", nil)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusNoContent, resp.Code)

			// Verify node is deleted
			_, err = nodeRegistry.GetNode(ctx, "test-node")
			assert.Error(t, err)
		})
	})

	t.Run("should return internal server error for registry failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockStorage.NewMockStorage(ctrl)
		nodeRegistry := registry.NewNodeRegistry(mockStore)
		handler := NewNodeHandler(nodeRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterNodeRoutes(ws, handler)

			node := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node",
				},
			}
			mockStore.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).SetArg(2, *node)
			mockStore.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			req := httptest.NewRequest("DELETE", "/api/v1/nodes/test-node", nil)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	})

	t.Run("should return not found for non-existent node", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			nodeRegistry := registry.NewNodeRegistry(store)
			handler := NewNodeHandler(nodeRegistry)

			RegisterNodeRoutes(ws, handler)

			req := httptest.NewRequest("DELETE", "/api/v1/nodes/non-existent-node", nil)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusNotFound, resp.Code)
		})
	})
}

func TestListNodes(t *testing.T) {
	t.Run("should list all nodes", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			nodeRegistry := registry.NewNodeRegistry(store)
			handler := NewNodeHandler(nodeRegistry)
			ctx := context.Background()

			RegisterNodeRoutes(ws, handler)

			node1 := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node-1",
				},
				Spec: api.NodeSpec{},
			}

			node2 := &api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "test-node-2",
				},
				Spec: api.NodeSpec{},
			}

			err := nodeRegistry.CreateNode(ctx, node1)
			require.NoError(t, err)
			err = nodeRegistry.CreateNode(ctx, node2)
			require.NoError(t, err)

			req := httptest.NewRequest("GET", "/api/v1/nodes", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)

			var nodes []api.Node
			err = json.Unmarshal(resp.Body.Bytes(), &nodes)
			assert.NoError(t, err)

			require.Len(t, nodes, 2)
			assert.Equal(t, node1.Name, nodes[0].Name)
			assert.Equal(t, node2.Name, nodes[1].Name)
		})
	})

	t.Run("should return internal server error for registry failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockStorage.NewMockStorage(ctrl)
		nodeRegistry := registry.NewNodeRegistry(mockStore)
		handler := NewNodeHandler(nodeRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterNodeRoutes(ws, handler)

			mockStore.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			req := httptest.NewRequest("GET", "/api/v1/nodes", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	})
}
