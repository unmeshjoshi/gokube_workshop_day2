package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gokube/pkg/api"
	"gokube/pkg/registry"
	"gokube/pkg/storage"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestNewScheduler(t *testing.T) {
	podRegistry := registry.NewPodRegistry(nil)
	nodeRegistry := registry.NewNodeRegistry(nil)

	scheduler := NewScheduler(podRegistry, nodeRegistry, 1*time.Second)

	assert.Equal(t, podRegistry, scheduler.podRegistry)
	assert.Equal(t, nodeRegistry, scheduler.nodeRegistry)
	assert.Equal(t, 1*time.Second, scheduler.schedulingRate)
}

func TestScheduler_Start(t *testing.T) {
	storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdClient *clientv3.Client) {
		// Create storage and registries
		etcdStorage := storage.NewEtcdStorage(etcdClient)
		podRegistry := registry.NewPodRegistry(etcdStorage)
		nodeRegistry := registry.NewNodeRegistry(etcdStorage)

		// Create scheduler
		scheduler := NewScheduler(podRegistry, nodeRegistry, 1*time.Minute)

		// Start scheduler
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go scheduler.Start(ctx)

		// Wait for scheduler to start
		time.Sleep(100 * time.Millisecond)

		//TODO: Add assertions to check if scheduler is running

		// Stop scheduler
		cancel()
	})
}

func TestScheduler_SchedulePendingPods(t *testing.T) {

	testCases := []struct {
		name              string
		nodes             []*api.Node
		pendingPods       []*api.Pod
		expectedScheduled int
	}{
		{
			name: "Schedule pending pods to available nodes",
			nodes: []*api.Node{
				{ObjectMeta: api.ObjectMeta{Name: "node1"}},
				{ObjectMeta: api.ObjectMeta{Name: "node2"}},
			},
			pendingPods: []*api.Pod{
				{
					ObjectMeta: api.ObjectMeta{Name: "pod1"},
					Spec: api.PodSpec{
						Containers: []api.Container{{Name: "container1", Image: "nginx:latest"}},
					},
					Status: api.PodPending,
				},
				{
					ObjectMeta: api.ObjectMeta{Name: "pod2"},
					Spec: api.PodSpec{
						Containers: []api.Container{{Name: "container2", Image: "redis:latest"}},
					},
					Status: api.PodPending,
				},
				{
					ObjectMeta: api.ObjectMeta{Name: "pod3"},
					Spec: api.PodSpec{
						Containers: []api.Container{{Name: "container3", Image: "mysql:5.7"}},
					},
					Status: api.PodPending,
				},
			},
			expectedScheduled: 3,
		},
		{
			name:  "No nodes available",
			nodes: []*api.Node{},
			pendingPods: []*api.Pod{
				{
					ObjectMeta: api.ObjectMeta{Name: "pod4"},
					Spec: api.PodSpec{
						Containers: []api.Container{{Name: "container4", Image: "busybox:latest"}},
					},
					Status: api.PodPending,
				},
			},
			expectedScheduled: 0,
		},
	}

	// Test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdClient *clientv3.Client) {
				etcdStorage := storage.NewEtcdStorage(etcdClient)
				podRegistry := registry.NewPodRegistry(etcdStorage)
				nodeRegistry := registry.NewNodeRegistry(etcdStorage)
				scheduler := NewScheduler(podRegistry, nodeRegistry, 1*time.Second)
				ctx := context.Background()

				// Create test nodes
				for _, node := range tc.nodes {
					err := nodeRegistry.CreateNode(ctx, node)
					require.NoErrorf(t, err, "Failed to create test node: %v", err)
				}

				// Create pending pods
				for _, pod := range tc.pendingPods {
					err := podRegistry.CreatePod(ctx, pod)
					require.NoErrorf(t, err, "Failed to create test pod: %v", err)
				}

				// Run scheduler
				err := scheduler.schedulePendingPods(ctx)
				if tc.expectedScheduled > 0 {
					require.NoErrorf(t, err, "Failed to schedule pending pods: %v", err)
					assert.Equalf(t, tc.expectedScheduled, len(tc.pendingPods), "Expected %d scheduled pods, but got %d", tc.expectedScheduled, len(tc.pendingPods))
				}

				// Check scheduled pods
				scheduledPods, err := podRegistry.ListPods(ctx)
				require.NoErrorf(t, err, "Failed to list pods: %v", err)

				scheduledCount := 0
				for _, pod := range scheduledPods {
					if pod.Status == api.PodScheduled && pod.NodeName != "" {
						scheduledCount++
					}
				}

				assert.Equal(t, tc.expectedScheduled, scheduledCount)
			})
		})
	}
}
