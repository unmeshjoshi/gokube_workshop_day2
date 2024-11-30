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

func TestCreatePod(t *testing.T) {
	t.Run("should create a new pod", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)

			RegisterPodRoutes(ws, handler)

			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Replicas: 1,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
				// Note: We don't set the Status field here, as it should be set by the server
			}

			body, _ := json.Marshal(pod)
			req := httptest.NewRequest("POST", "/api/v1/pods", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusCreated, resp.Code)

			var createdPod api.Pod
			err := json.Unmarshal(resp.Body.Bytes(), &createdPod)
			assert.NoError(t, err)

			require.NotEmpty(t, createdPod)

			assert.Equal(t, pod.Name, createdPod.Name)
			assert.Equal(t, pod.Spec.Replicas, createdPod.Spec.Replicas)
			assert.Equal(t, len(pod.Spec.Containers), len(createdPod.Spec.Containers))
			assert.Equal(t, pod.Spec.Containers[0].Image, createdPod.Spec.Containers[0].Image)

			// Check that the status is set to Unassigned
			assert.Equal(t, api.PodPending, createdPod.Status)
		})
	})

	t.Run("should return bad request for invalid pod", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)

			RegisterPodRoutes(ws, handler)

			invalidPod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "invalid-pod",
				},
				Spec: api.PodSpec{
					Replicas: -1, // Invalid value for Replicas
					Containers: []api.Container{
						{
							Name:  "",
							Image: "nginx:latest",
						},
					},
				},
			}

			body, _ := json.Marshal(invalidPod)
			req := httptest.NewRequest("POST", "/api/v1/pods", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code)
		})
	})

	t.Run("should return conflict for existing pod", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)
			ctx := context.Background()

			RegisterPodRoutes(ws, handler)

			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Replicas: 1,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			}

			err := podRegistry.CreatePod(ctx, pod)
			require.NoError(t, err)

			body, _ := json.Marshal(pod)
			req := httptest.NewRequest("POST", "/api/v1/pods", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusConflict, resp.Code)
		})
	})

	t.Run("should return internal server error for registry failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockStorage.NewMockStorage(ctrl)
		podRegistry := registry.NewPodRegistry(mockStore)
		handler := NewPodHandler(podRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterPodRoutes(ws, handler)

			mockStore.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))
			mockStore.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Replicas: 1,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
				// Note: We don't set the Status field here, as it should be set by the server
			}

			body, _ := json.Marshal(pod)
			req := httptest.NewRequest("POST", "/api/v1/pods", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	})
}

func TestListPods(t *testing.T) {
	t.Run("should list all pods", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)
			ctx := context.Background()

			RegisterPodRoutes(ws, handler)

			pod1 := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod-1",
				},
				Spec: api.PodSpec{
					Replicas: 1,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			}

			pod2 := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod-2",
				},
				Spec: api.PodSpec{
					Replicas: 2,
					Containers: []api.Container{
						{
							Name:  "redis",
							Image: "redis:latest",
						},
					},
				},
			}

			err := podRegistry.CreatePod(ctx, pod1)
			require.NoError(t, err)
			err = podRegistry.CreatePod(ctx, pod2)
			require.NoError(t, err)

			req := httptest.NewRequest("GET", "/api/v1/pods", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)

			var pods []api.Pod
			err = json.Unmarshal(resp.Body.Bytes(), &pods)
			assert.NoError(t, err)

			require.Len(t, pods, 2)
			assert.Equal(t, pod1.Name, pods[0].Name)
			assert.Equal(t, pod2.Name, pods[1].Name)
		})
	})

	t.Run("should return internal server error for registry failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockStorage.NewMockStorage(ctrl)
		podRegistry := registry.NewPodRegistry(mockStore)
		handler := NewPodHandler(podRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterPodRoutes(ws, handler)

			mockStore.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			req := httptest.NewRequest("GET", "/api/v1/pods", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	})
}

func TestGetPod(t *testing.T) {
	t.Run("should get existing pod", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)
			ctx := context.Background()

			RegisterPodRoutes(ws, handler)

			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Replicas: 1,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			}

			err := podRegistry.CreatePod(ctx, pod)
			require.NoError(t, err)

			req := httptest.NewRequest("GET", "/api/v1/pods/test-pod", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)

			var returnedPod api.Pod
			err = json.Unmarshal(resp.Body.Bytes(), &returnedPod)
			assert.NoError(t, err)

			assert.Equal(t, pod.Name, returnedPod.Name)
			assert.Equal(t, pod.Spec.Replicas, returnedPod.Spec.Replicas)
			assert.Equal(t, pod.Spec.Containers[0].Image, returnedPod.Spec.Containers[0].Image)
		})
	})

	t.Run("should return not found for non-existent pod", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)

			RegisterPodRoutes(ws, handler)

			req := httptest.NewRequest("GET", "/api/v1/pods/non-existent-pod", nil)
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
		podRegistry := registry.NewPodRegistry(mockStore)
		handler := NewPodHandler(podRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterPodRoutes(ws, handler)

			mockStore.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			req := httptest.NewRequest("GET", "/api/v1/pods/test-pod", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	})
}

func TestUpdatePod(t *testing.T) {
	t.Run("should update existing pod", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)
			ctx := context.Background()

			RegisterPodRoutes(ws, handler)

			// Create initial pod
			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Replicas: 1,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			}

			err := podRegistry.CreatePod(ctx, pod)
			require.NoError(t, err)

			// Update pod
			updatedPod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Replicas: 2,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.19",
						},
					},
				},
			}

			body, _ := json.Marshal(updatedPod)
			req := httptest.NewRequest("PUT", "/api/v1/pods/test-pod", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			require.Equal(t, http.StatusOK, resp.Code)

			var returnedPod api.Pod
			err = json.Unmarshal(resp.Body.Bytes(), &returnedPod)
			assert.NoError(t, err)
			assert.Equal(t, updatedPod.Spec.Replicas, returnedPod.Spec.Replicas)
			assert.Equal(t, updatedPod.Spec.Containers[0].Image, returnedPod.Spec.Containers[0].Image)
		})
	})

	t.Run("should return bad request when pod names don't match", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)
			ctx := context.Background()

			RegisterPodRoutes(ws, handler)

			// Create the initial pod first
			existingPod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Replicas: 1,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			}
			err := podRegistry.CreatePod(ctx, existingPod)
			require.NoError(t, err)

			// Try to update with a different name
			updatePod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "different-name",
				},
			}

			body, _ := json.Marshal(updatePod)
			req := httptest.NewRequest("PUT", "/api/v1/pods/test-pod", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code)
		})
	})

	t.Run("should return bad request for invalid pod", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)
			ctx := context.Background()

			RegisterPodRoutes(ws, handler)

			// Create initial pod
			initialPod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Replicas: 1,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			}
			err := podRegistry.CreatePod(ctx, initialPod)
			require.NoError(t, err)

			// Try to update with invalid pod
			invalidPod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Replicas: -1,
				},
			}

			body, _ := json.Marshal(invalidPod)
			req := httptest.NewRequest("PUT", "/api/v1/pods/test-pod", bytes.NewReader(body))
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
		podRegistry := registry.NewPodRegistry(mockStore)
		handler := NewPodHandler(podRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterPodRoutes(ws, handler)

			// Mock Get operation for the middleware
			existingPod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
			}
			mockStore.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).SetArg(2, *existingPod)
			mockStore.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Replicas: 1,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.19",
						},
					},
				},
			}

			body, _ := json.Marshal(pod)
			req := httptest.NewRequest("PUT", "/api/v1/pods/test-pod", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	})

	t.Run("should return not found for non-existent pod", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)

			RegisterPodRoutes(ws, handler)

			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "non-existent-pod",
				},
				Spec: api.PodSpec{
					Replicas: 1,
				},
			}

			body, _ := json.Marshal(pod)
			req := httptest.NewRequest("PUT", "/api/v1/pods/non-existent-pod", bytes.NewReader(body))
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusNotFound, resp.Code)
		})
	})
}

func TestDeletePod(t *testing.T) {
	t.Run("should delete existing pod", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)
			ctx := context.Background()

			RegisterPodRoutes(ws, handler)

			// Create a pod first
			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Replicas: 1,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			}

			err := podRegistry.CreatePod(ctx, pod)
			require.NoError(t, err)

			// Delete the pod
			req := httptest.NewRequest("DELETE", "/api/v1/pods/test-pod", nil)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusNoContent, resp.Code)

			// Verify pod is deleted
			_, err = podRegistry.GetPod(ctx, "test-pod")
			assert.Error(t, err)
		})
	})

	t.Run("should return internal server error for registry failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockStorage.NewMockStorage(ctrl)
		podRegistry := registry.NewPodRegistry(mockStore)
		handler := NewPodHandler(podRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterPodRoutes(ws, handler)

			// Mock the Get operation from the middleware
			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
			}
			mockStore.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).SetArg(2, *pod)
			mockStore.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			req := httptest.NewRequest("DELETE", "/api/v1/pods/test-pod", nil)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	})

	t.Run("should return not found for non-existent pod", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)

			RegisterPodRoutes(ws, handler)

			req := httptest.NewRequest("DELETE", "/api/v1/pods/non-existent-pod", nil)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusNotFound, resp.Code)
		})
	})
}

func TestListUnassignedPods(t *testing.T) {
	t.Run("should list all unassigned pods", func(t *testing.T) {
		withTestServer(t, func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			store := storage.NewEtcdStorage(etcdServer)
			podRegistry := registry.NewPodRegistry(store)
			handler := NewPodHandler(podRegistry)
			ctx := context.Background()

			RegisterPodRoutes(ws, handler)

			// Create unassigned pod
			unassignedPod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "unassigned-pod",
				},
				Spec: api.PodSpec{
					Replicas: 1,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
				Status: api.PodPending,
			}

			// Create assigned pod
			assignedPod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "assigned-pod",
				},
				Spec: api.PodSpec{
					Replicas: 1,
					Containers: []api.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
				Status: api.PodRunning,
			}

			err := podRegistry.CreatePod(ctx, unassignedPod)
			require.NoError(t, err)
			err = podRegistry.CreatePod(ctx, assignedPod)
			require.NoError(t, err)

			req := httptest.NewRequest("GET", "/api/v1/pods/unassigned", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)

			var pods []api.Pod
			err = json.Unmarshal(resp.Body.Bytes(), &pods)
			assert.NoError(t, err)

			require.Len(t, pods, 1)
			assert.Equal(t, unassignedPod.Name, pods[0].Name)
			assert.Equal(t, api.PodPending, pods[0].Status)
		})
	})

	t.Run("should return internal server error for registry failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockStorage.NewMockStorage(ctrl)
		podRegistry := registry.NewPodRegistry(mockStore)
		handler := NewPodHandler(podRegistry)

		withTestServer(t, func(_ *clientv3.Client, ws *restful.WebService, container *restful.Container) {
			RegisterPodRoutes(ws, handler)

			mockStore.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("simulated registry failure"))

			req := httptest.NewRequest("GET", "/api/v1/pods/unassigned", nil)
			req.Header.Set("Content-Type", restful.MIME_JSON)
			resp := httptest.NewRecorder()

			container.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	})
}
