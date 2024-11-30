package registry

import (
	"context"
	"errors"
	"fmt"
	"testing"

	mockStorage "gokube/mocks/pkg/storage"
	"gokube/pkg/api"
	"gokube/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestNewPodRegistry(t *testing.T) {
	storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
		etcdStorage := storage.NewEtcdStorage(etcdServer)
		registry := NewPodRegistry(etcdStorage)

		assert.NotNil(t, registry)
		assert.Equal(t, etcdStorage, registry.storage)
	})
}

func TestPodRegistry_GetPod(t *testing.T) {
	t.Run("should return pod if it exists", func(t *testing.T) {
		storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
			etcdStorage := storage.NewEtcdStorage(etcdServer)
			registry := NewPodRegistry(etcdStorage)
			ctx := context.Background()

			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
						},
					},
					Replicas: 3,
				},
				Status: api.PodPending,
			}

			err := registry.CreatePod(ctx, pod)
			require.NoError(t, err)

			// Test GetPod
			retrievedPod, err := registry.GetPod(ctx, "test-pod")
			require.NoError(t, err)

			// Verify pod name and status
			assert.Equal(t, "test-pod", retrievedPod.Name)
			assert.Equal(t, api.PodPending, retrievedPod.Status)

			// Verify pod spec
			assert.Len(t, retrievedPod.Spec.Containers, 1)
			assert.Equal(t, "nginx:latest", retrievedPod.Spec.Containers[0].Image)
			assert.Equal(t, int32(3), retrievedPod.Spec.Replicas)
		})
	})

	t.Run("should return error if pod does not exist", func(t *testing.T) {
		storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
			etcdStorage := storage.NewEtcdStorage(etcdServer)
			registry := NewPodRegistry(etcdStorage)
			ctx := context.Background()

			_, err := registry.GetPod(ctx, "non-existent-pod")
			assert.ErrorIs(t, err, ErrPodNotFound)
			assert.EqualError(t, err, "pod not found: non-existent-pod")
		})
	})

	t.Run("should return error if storage returns ErrInternal", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mStorage := mockStorage.NewMockStorage(ctrl)
		registry := NewPodRegistry(mStorage)
		ctx := context.Background()

		mStorage.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(fmt.Errorf("storage error"))

		_, err := registry.GetPod(ctx, "invalid-pod")
		assert.ErrorIs(t, err, ErrInternal)
	})
}

func TestPodRegistry_CreatePod(t *testing.T) {
	t.Run("should create pod", func(t *testing.T) {
		storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
			etcdStorage := storage.NewEtcdStorage(etcdServer)
			registry := NewPodRegistry(etcdStorage)
			ctx := context.Background()

			// Test Create
			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name: "test-container", Image: "nginx:latest",
						},
					},
					Replicas: 3,
				},
				Status: api.PodPending,
			}

			err := registry.CreatePod(ctx, pod)
			require.NoError(t, err)

			// Verify pod was created
			_, err = registry.GetPod(ctx, "test-pod")
			require.NoError(t, err)
		})
	})

	t.Run("should fail to create pod with the same name", func(t *testing.T) {
		storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
			etcdStorage := storage.NewEtcdStorage(etcdServer)
			registry := NewPodRegistry(etcdStorage)
			ctx := context.Background()

			// Create the first pod
			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "duplicate-pod",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
						},
					},
					Replicas: 3,
				},
				Status: api.PodPending,
			}

			err := registry.CreatePod(ctx, pod)
			require.NoError(t, err)

			// Attempt to create another pod with the same name
			err = registry.CreatePod(ctx, pod)
			assert.ErrorIs(t, err, ErrPodAlreadyExists)
			assert.EqualError(t, err, "pod already exists: duplicate-pod")
		})
	})

	t.Run("should set default status when pod status is not provided", func(t *testing.T) {
		storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
			etcdStorage := storage.NewEtcdStorage(etcdServer)
			registry := NewPodRegistry(etcdStorage)
			ctx := context.Background()

			// Create a pod without specifying the status
			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "no-status-pod",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
						},
					},
					Replicas: 3,
				},
			}

			err := registry.CreatePod(ctx, pod)
			require.NoError(t, err)

			// Verify pod was created with default status
			retrievedPod, err := registry.GetPod(ctx, "no-status-pod")
			require.NoError(t, err)
			assert.Equal(t, api.PodPending, retrievedPod.Status)
		})
	})

	t.Run("should validate pod spec", func(t *testing.T) {
		storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
			etcdStorage := storage.NewEtcdStorage(etcdServer)
			registry := NewPodRegistry(etcdStorage)
			ctx := context.Background()

			// Create a pod with an invalid spec
			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "invalid-spec-pod",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "test-container",
							Image: "", // Invalid because image is empty
						},
					},
					Replicas: 3,
				},
			}

			err := registry.CreatePod(ctx, pod)
			assert.ErrorIs(t, err, ErrPodInvalid)
		})
	})
}

func TestPodRegistry_UpdatePod(t *testing.T) {
	t.Run("should update pod status", func(t *testing.T) {
		storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
			etcdStorage := storage.NewEtcdStorage(etcdServer)
			registry := NewPodRegistry(etcdStorage)
			ctx := context.Background()

			pod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "test-pod",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name: "test-container", Image: "nginx:latest",
						},
					},
					Replicas: 3,
				},
				Status: api.PodPending,
			}

			err := registry.CreatePod(ctx, pod)
			require.NoError(t, err)

			// Update pod status
			pod.Status = api.PodRunning
			err = registry.UpdatePod(ctx, pod)
			require.NoError(t, err)

			// Verify updated status
			retrievedPod, err := registry.GetPod(ctx, "test-pod")
			require.NoError(t, err)
			assert.Equal(t, api.PodRunning, retrievedPod.Status)
		})
	})
	t.Run("should validate pod spec on update", func(t *testing.T) {
		storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
			etcdStorage := storage.NewEtcdStorage(etcdServer)
			registry := NewPodRegistry(etcdStorage)
			ctx := context.Background()

			validPod := &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "valid-pod",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
						},
					},
					Replicas: 3,
				},
				Status: api.PodPending,
			}

			err := registry.CreatePod(ctx, validPod)
			require.NoError(t, err)

			// Update pod with invalid spec
			validPod.Spec.Containers[0].Image = "" // Invalid because image is empty
			err = registry.UpdatePod(ctx, validPod)
			assert.ErrorIs(t, err, ErrPodInvalid)
		})
	})
}

func TestPodRegistry_DeletePod(t *testing.T) {
	storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
		etcdStorage := storage.NewEtcdStorage(etcdServer)
		registry := NewPodRegistry(etcdStorage)
		ctx := context.Background()

		pod := &api.Pod{
			ObjectMeta: api.ObjectMeta{
				Name: "test-pod",
			},
			Spec: api.PodSpec{
				Containers: []api.Container{
					{
						Name:  "test-container",
						Image: "nginx:latest",
					},
				},
				Replicas: 3,
			},
			Status: api.PodPending,
		}

		err := registry.CreatePod(ctx, pod)
		require.NoError(t, err)

		err = registry.DeletePod(ctx, "test-pod")
		require.NoError(t, err)

		_, err = registry.GetPod(ctx, "test-pod")
		assert.Error(t, err)
	})
}

func TestPodRegistry_ListPods(t *testing.T) {
	storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
		etcdStorage := storage.NewEtcdStorage(etcdServer)
		registry := NewPodRegistry(etcdStorage)
		ctx := context.Background()

		// Test cases

		// Test ListPods
		pod1 := &api.Pod{
			ObjectMeta: api.ObjectMeta{
				Name: "test-pod-1",
			},
			Spec: api.PodSpec{
				Containers: []api.Container{
					{
						Name:  "test-container-1",
						Image: "nginx:latest",
					},
				},
				Replicas: 3,
			},
			Status: api.PodPending,
		}

		pod2 := &api.Pod{
			ObjectMeta: api.ObjectMeta{
				Name: "test-pod-2",
			},
			Spec: api.PodSpec{
				Containers: []api.Container{
					{
						Name:  "test-container-2",
						Image: "nginx:latest",
					},
				},
				Replicas: 3,
			},
			Status: api.PodRunning,
		}

		err := registry.CreatePod(ctx, pod1)
		require.NoError(t, err)

		err = registry.CreatePod(ctx, pod2)
		require.NoError(t, err)

		pods, err := registry.ListPods(ctx)
		require.NoError(t, err)
		require.Len(t, pods, 2)

		// Verify pod names
		assert.Equal(t, "test-pod-1", pods[0].Name)
		assert.Equal(t, "test-pod-2", pods[1].Name)
	})

	t.Run("should handle error returned by the storage provider", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mStorage := mockStorage.NewMockStorage(ctrl)
		registry := NewPodRegistry(mStorage)
		ctx := context.Background()

		mStorage.EXPECT().List(ctx, podPrefix, gomock.Any()).Return(errors.New("failed to list pods"))

		pods, err := registry.ListPods(ctx)

		assert.ErrorIs(t, err, ErrListPodsFailed, "Expected error when listing pods")
		assert.Nil(t, pods, "Expected nil list of pods")
	})
}

func TestPodRegistry_ListPendingPods(t *testing.T) {
	t.Run("should list pending pods", func(t *testing.T) {
		testCases := []struct {
			name                string
			podsToCreate        []*api.Pod
			expectedPendingPods int
		}{
			{
				name: "no pending pods",
				podsToCreate: []*api.Pod{
					{ObjectMeta: api.ObjectMeta{Name: "pod1"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodRunning},
					{ObjectMeta: api.ObjectMeta{Name: "pod2"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodRunning},
				},
				expectedPendingPods: 0,
			},
			{
				name: "some pending pods",
				podsToCreate: []*api.Pod{
					{ObjectMeta: api.ObjectMeta{Name: "pod3"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodPending},
					{ObjectMeta: api.ObjectMeta{Name: "pod4"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodRunning},
					{ObjectMeta: api.ObjectMeta{Name: "pod5"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodPending},
				},
				expectedPendingPods: 2,
			},
			{
				name: "all pending pods",
				podsToCreate: []*api.Pod{
					{ObjectMeta: api.ObjectMeta{Name: "pod6"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodPending},
					{ObjectMeta: api.ObjectMeta{Name: "pod7"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodPending},
				},
				expectedPendingPods: 2,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
					etcdStorage := storage.NewEtcdStorage(etcdServer)
					registry := NewPodRegistry(etcdStorage)
					ctx := context.Background()

					// Create test pods
					for _, pod := range tc.podsToCreate {
						if err := registry.CreatePod(ctx, pod); err != nil {
							t.Fatalf("Failed to create test pod: %v", err)
						}
					}

					// Call ListPendingPods
					pods, err := registry.ListPendingPods(ctx)
					require.NoError(t, err)

					assert.Equal(t, tc.expectedPendingPods, len(pods))
				})
			})
		}
	})

	t.Run("should handle error returned by the storage provider", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mStorage := mockStorage.NewMockStorage(ctrl)
		registry := NewPodRegistry(mStorage)
		ctx := context.Background()

		mStorage.EXPECT().List(ctx, podPrefix, gomock.Any()).Return(errors.New("failed to list pods"))

		pods, err := registry.ListPendingPods(ctx)

		assert.ErrorIs(t, err, ErrListPodsFailed, "Expected error when listing pods")
		assert.Nil(t, pods, "Expected nil list of pods")
	})
}

func TestPodRegistry_ListUnassignedPods(t *testing.T) {
	t.Run("should list unassigned pods", func(t *testing.T) {
		testCases := []struct {
			name                   string
			podsToCreate           []*api.Pod
			expectedUnassignedPods int
		}{
			{
				name: "no unassigned pods",
				podsToCreate: []*api.Pod{
					{ObjectMeta: api.ObjectMeta{Name: "pod1"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodRunning},
					{ObjectMeta: api.ObjectMeta{Name: "pod2"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodRunning},
				},
				expectedUnassignedPods: 0,
			},
			{
				name: "some unassigned pods",
				podsToCreate: []*api.Pod{
					{ObjectMeta: api.ObjectMeta{Name: "pod3"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodPending},
					{ObjectMeta: api.ObjectMeta{Name: "pod4"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodRunning},
					{ObjectMeta: api.ObjectMeta{Name: "pod5"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodPending},
				},
				expectedUnassignedPods: 2,
			},
			{
				name: "all unassigned pods",
				podsToCreate: []*api.Pod{
					{ObjectMeta: api.ObjectMeta{Name: "pod6"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodPending},
					{ObjectMeta: api.ObjectMeta{Name: "pod7"},
						Spec:   api.PodSpec{Containers: []api.Container{{Name: "test-container2", Image: "nginx"}}},
						Status: api.PodPending},
				},
				expectedUnassignedPods: 2,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
					etcdStorage := storage.NewEtcdStorage(etcdServer)
					registry := NewPodRegistry(etcdStorage)
					ctx := context.Background()

					// Create test pods
					for _, pod := range tc.podsToCreate {
						if err := registry.CreatePod(ctx, pod); err != nil {
							t.Fatalf("Failed to create test pod: %v", err)
						}
					}

					// Call ListUnassignedPods
					pods, err := registry.ListUnassignedPods(ctx)
					require.NoError(t, err)

					assert.Equal(t, tc.expectedUnassignedPods, len(pods))
				})
			})
		}
	})

	t.Run("should handle error returned by the storage provider", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mStorage := mockStorage.NewMockStorage(ctrl)
		registry := NewPodRegistry(mStorage)
		ctx := context.Background()

		mStorage.EXPECT().List(ctx, podPrefix, gomock.Any()).Return(errors.New("failed to list pods"))

		pods, err := registry.ListUnassignedPods(ctx)

		assert.ErrorIs(t, err, ErrListPodsFailed, "Expected error when listing pods")
		assert.Nil(t, pods, "Expected nil list of pods")
	})
}
