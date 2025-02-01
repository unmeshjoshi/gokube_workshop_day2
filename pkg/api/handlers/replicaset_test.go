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

func TestCreateReplicaset(t *testing.T) {
	t.Run("should create a new replicaset", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			replicasetRegistry := registry.NewReplicaSetRegistry(store)
			handler := NewReplicasetHandler(replicasetRegistry)

			RegisterReplicasetRoutes(ws, handler)

			replicaset := &api.ReplicaSet{
				ObjectMeta: api.ObjectMeta{
					Name: "nginx-rs",
				},
				Spec: api.ReplicaSetSpec{
					Replicas: 2,
					Selector: map[string]string{
						"name": "nginx-rs",
					},
					Template: api.PodTemplateSpec{
						ObjectMeta: api.ObjectMeta{
							Name: "nginx-rs",
						},
						Spec: api.PodSpec{
							Containers: []api.Container{
								{
									Name:  "nginx",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
			}

			body, _ := json.Marshal(replicaset)
			req := httptest.NewRequest("POST", "/api/v1/replicasets", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusCreated, resp.Code)

			var createdReplicaset api.ReplicaSet
			err := json.Unmarshal(resp.Body.Bytes(), &createdReplicaset)
			assert.NoError(t, err)

			require.NotEmpty(t, createdReplicaset)

			assert.Equal(t, replicaset.Name, createdReplicaset.Name)
			assert.Equal(t, replicaset.Spec.Replicas, createdReplicaset.Spec.Replicas)
			assert.Equal(t, len(replicaset.Spec.Template.Spec.Containers), len(createdReplicaset.Spec.Template.Spec.Containers))
			assert.Equal(t, replicaset.Spec.Template.Spec.Containers[0].Image, createdReplicaset.Spec.Template.Spec.Containers[0].Image)
		})
	})

	t.Run("should return internal server error for registry failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockStorage.NewMockStorage(ctrl)
		nodeRegistry := registry.NewReplicaSetRegistry(mockStore)
		handler := NewReplicasetHandler(nodeRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterReplicasetRoutes(ws, handler)

			mockStore.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))
			mockStore.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			replicaset := &api.ReplicaSet{
				ObjectMeta: api.ObjectMeta{
					Name: "nginx-rs",
				},
				Spec: api.ReplicaSetSpec{
					Replicas: 2,
					Selector: map[string]string{
						"name": "nginx-rs",
					},
					Template: api.PodTemplateSpec{
						ObjectMeta: api.ObjectMeta{
							Name: "nginx-rs",
						},
						Spec: api.PodSpec{
							Containers: []api.Container{
								{
									Name:  "nginx",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
			}

			body, _ := json.Marshal(replicaset)
			req := httptest.NewRequest("POST", "/api/v1/replicasets", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	})

	t.Run("should return conflict error when replicasets already exists", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			replicasetRegistry := registry.NewReplicaSetRegistry(store)
			handler := NewReplicasetHandler(replicasetRegistry)
			ctx := context.Background()

			RegisterReplicasetRoutes(ws, handler)

			// Create initial replicaset
			replicaset := &api.ReplicaSet{
				ObjectMeta: api.ObjectMeta{
					Name: "nginx-rs",
				},
				Spec: api.ReplicaSetSpec{
					Replicas: 2,
					Selector: map[string]string{
						"name": "nginx-rs",
					},
					Template: api.PodTemplateSpec{
						ObjectMeta: api.ObjectMeta{
							Name: "nginx-rs",
						},
						Spec: api.PodSpec{
							Containers: []api.Container{
								{
									Name:  "nginx",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
			}

			err := replicasetRegistry.Create(ctx, replicaset)
			require.NoError(t, err)

			// Try to create same replicaset again
			body, _ := json.Marshal(replicaset)
			req := httptest.NewRequest("POST", "/api/v1/replicasets", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusConflict, resp.Code)
		})
	})
}

func TestGetReplicaset(t *testing.T) {
	t.Run("should get existing replicaset", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			replicasetRegistry := registry.NewReplicaSetRegistry(store)
			handler := NewReplicasetHandler(replicasetRegistry)
			ctx := context.Background()

			RegisterReplicasetRoutes(ws, handler)

			replicaset := &api.ReplicaSet{
				ObjectMeta: api.ObjectMeta{
					Name: "nginx-rs",
				},
				Spec: api.ReplicaSetSpec{
					Replicas: 2,
					Selector: map[string]string{
						"name": "nginx-rs",
					},
					Template: api.PodTemplateSpec{
						ObjectMeta: api.ObjectMeta{
							Name: "nginx-rs",
						},
						Spec: api.PodSpec{
							Containers: []api.Container{
								{
									Name:  "nginx",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
			}

			err := replicasetRegistry.Create(ctx, replicaset)
			require.NoError(t, err)

			req := httptest.NewRequest("GET", "/api/v1/replicasets/nginx-rs", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)

			var returnedReplicaset api.ReplicaSet
			err = json.Unmarshal(resp.Body.Bytes(), &returnedReplicaset)
			assert.NoError(t, err)

			assert.Equal(t, replicaset.Name, returnedReplicaset.Name)
		})
	})

	t.Run("should return not found for non-existent replicaset", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			replicasetRegistry := registry.NewReplicaSetRegistry(store)
			handler := NewReplicasetHandler(replicasetRegistry)

			RegisterReplicasetRoutes(ws, handler)

			req := httptest.NewRequest("GET", "/api/v1/replicasets/non-existent-replicaset", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusNotFound, resp.Code)
		})
	})

	t.Run("should return not found for registry failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockStorage.NewMockStorage(ctrl)
		replicasetRegistry := registry.NewReplicaSetRegistry(mockStore)
		handler := NewReplicasetHandler(replicasetRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterReplicasetRoutes(ws, handler)

			mockStore.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			req := httptest.NewRequest("GET", "/api/v1/replicasets/test-replicaset", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusNotFound, resp.Code)
		})
	})
}

func TestUpdateReplicaset(t *testing.T) {
	t.Run("should update existing replicaset", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			replicasetRegistry := registry.NewReplicaSetRegistry(store)
			handler := NewReplicasetHandler(replicasetRegistry)
			ctx := context.Background()

			RegisterReplicasetRoutes(ws, handler)

			// Create initial replicaset
			replicaset := &api.ReplicaSet{
				ObjectMeta: api.ObjectMeta{
					Name: "nginx-rs",
				},
				Spec: api.ReplicaSetSpec{
					Replicas: 2,
					Selector: map[string]string{
						"name": "nginx-rs",
					},
					Template: api.PodTemplateSpec{
						ObjectMeta: api.ObjectMeta{
							Name: "nginx-rs",
						},
						Spec: api.PodSpec{
							Containers: []api.Container{
								{
									Name:  "nginx",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
			}

			err := replicasetRegistry.Create(ctx, replicaset)
			require.NoError(t, err)

			// Update replicaset
			updatedReplicaset := &api.ReplicaSet{
				ObjectMeta: api.ObjectMeta{
					Name: "nginx-rs",
				},
				Spec: api.ReplicaSetSpec{
					Replicas: 3,
					Selector: map[string]string{
						"name": "nginx-rs",
					},
					Template: api.PodTemplateSpec{
						ObjectMeta: api.ObjectMeta{
							Name: "nginx-rs",
						},
						Spec: api.PodSpec{
							Containers: []api.Container{
								{
									Name:  "nginx",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
			}

			body, _ := json.Marshal(updatedReplicaset)
			req := httptest.NewRequest("PUT", "/api/v1/replicasets/nginx-rs", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			require.Equal(t, http.StatusOK, resp.Code)

			var returnedReplicaset api.ReplicaSet
			err = json.Unmarshal(resp.Body.Bytes(), &returnedReplicaset)
			assert.NoError(t, err)
			assert.Equal(t, int32(3), returnedReplicaset.Spec.Replicas)
		})
	})

	t.Run("should return bad request when replicaset names don't match", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			replicasetRegistry := registry.NewReplicaSetRegistry(store)
			handler := NewReplicasetHandler(replicasetRegistry)
			ctx := context.Background()

			RegisterReplicasetRoutes(ws, handler)

			// Create the initial replicaset first
			existingReplicaset := &api.ReplicaSet{
				ObjectMeta: api.ObjectMeta{
					Name: "nginx-rs",
				},
				Spec: api.ReplicaSetSpec{
					Replicas: 2,
					Selector: map[string]string{
						"name": "nginx-rs",
					},
					Template: api.PodTemplateSpec{
						ObjectMeta: api.ObjectMeta{
							Name: "nginx-rs",
						},
						Spec: api.PodSpec{
							Containers: []api.Container{
								{
									Name:  "nginx",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
			}
			err := replicasetRegistry.Create(ctx, existingReplicaset)
			require.NoError(t, err)

			// Try to update with mismatched name
			updatedReplicaset := &api.ReplicaSet{
				ObjectMeta: api.ObjectMeta{
					Name: "nginx-rs-different-name",
				},
				Spec: api.ReplicaSetSpec{
					Replicas: 2,
					Selector: map[string]string{
						"name": "nginx-rs",
					},
					Template: api.PodTemplateSpec{
						ObjectMeta: api.ObjectMeta{
							Name: "nginx-rs",
						},
						Spec: api.PodSpec{
							Containers: []api.Container{
								{
									Name:  "nginx",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
			}

			body, _ := json.Marshal(updatedReplicaset)
			req := httptest.NewRequest("PUT", "/api/v1/replicasets/nginx-rs", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code)
		})
	})

}
