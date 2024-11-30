package api

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestPodSpecValidation(t *testing.T) {
	validate := validator.New()

	t.Run("should validate PodSpec with required fields", func(t *testing.T) {
		podSpec := PodSpec{
			Containers: []Container{
				{
					Name:  "nginx-container",
					Image: "nginx:latest",
				},
			},
			Replicas: 3,
		}

		err := validate.Struct(podSpec)
		assert.NoError(t, err)
	})

	t.Run("should fail validation if containers are missing", func(t *testing.T) {
		podSpec := PodSpec{
			Replicas: 3,
		}

		err := validate.Struct(podSpec)
		assert.Error(t, err)
		assert.EqualError(t, err, "Key: 'PodSpec.Containers' Error:Field validation for 'Containers' failed on the 'required' tag")
	})

	t.Run("should fail validation if container image is missing", func(t *testing.T) {
		podSpec := PodSpec{
			Containers: []Container{
				{
					Name: "nginx-container",
				},
			},
			Replicas: 3,
		}

		err := validate.Struct(podSpec)
		assert.Error(t, err)
		assert.EqualError(t, err, "Key: 'PodSpec.Containers[0].Image' Error:Field validation for 'Image' failed on the 'required' tag")
	})

	t.Run("should fail validation if replicas is negative", func(t *testing.T) {
		podSpec := PodSpec{
			Containers: []Container{
				{
					Name:  "nginx-container",
					Image: "nginx:latest",
				},
			},
			Replicas: -1,
		}

		err := validate.Struct(podSpec)
		assert.Error(t, err)
		assert.EqualError(t, err, "Key: 'PodSpec.Replicas' Error:Field validation for 'Replicas' failed on the 'gte' tag")
	})
}

func TestPodValidation(t *testing.T) {
	validate := validator.New()

	t.Run("should validate Pod with required fields", func(t *testing.T) {
		pod := Pod{
			ObjectMeta: ObjectMeta{
				Name: "test-pod",
			},
			Spec: PodSpec{
				Containers: []Container{
					{
						Name:  "nginx-container",
						Image: "nginx:latest",
					},
				},
				Replicas: 3,
			},
			Status: PodPending,
		}

		err := validate.Struct(pod)
		assert.NoError(t, err)
	})

	t.Run("should fail validation if spec is missing", func(t *testing.T) {
		pod := Pod{
			ObjectMeta: ObjectMeta{
				Name: "test-pod",
			},
			Status: PodPending,
		}

		err := validate.Struct(pod)
		assert.Error(t, err)
		assert.EqualError(t, err, "Key: 'Pod.Spec.Containers' Error:Field validation for 'Containers' failed on the 'required' tag")
	})
}

func TestPodIsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   PodStatus
		expected bool
	}{
		{
			name:     "Pod is active when status is PodPending",
			status:   PodPending,
			expected: true,
		},
		{
			name:     "Pod is active when status is PodRunning",
			status:   PodRunning,
			expected: true,
		},
		{
			name:     "Pod is active when status is PodSucceeded",
			status:   PodSucceeded,
			expected: true,
		},
		{
			name:     "Pod is not active when status is PodFailed",
			status:   PodFailed,
			expected: false,
		},
		{
			name:     "Pod is active when status is PodScheduled",
			status:   PodScheduled,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := Pod{
				Status: tt.status,
			}
			assert.Equal(t, tt.expected, pod.IsActive())
		})
	}
}

func TestIsPodActiveAndOwnedBy(t *testing.T) {
	tests := []struct {
		name     string
		pod      Pod
		meta     ObjectMeta
		expected bool
	}{
		{
			name: "Pod is active and owned by ReplicaSet",
			pod: Pod{
				ObjectMeta: ObjectMeta{
					Name: "replicaset-12345-pod",
				},
				Status: PodRunning,
			},
			meta: ObjectMeta{
				Name: "replicaset-12345",
			},
			expected: true,
		},
		{
			name: "Pod is not active but owned by ReplicaSet",
			pod: Pod{
				ObjectMeta: ObjectMeta{
					Name: "replicaset-12345-pod",
				},
				Status: PodFailed,
			},
			meta: ObjectMeta{
				Name: "replicaset-12345",
			},
			expected: false,
		},
		{
			name: "Pod is active but not owned by ReplicaSet",
			pod: Pod{
				ObjectMeta: ObjectMeta{
					Name: "other-replicaset-12345-pod",
				},
				Status: PodRunning,
			},
			meta: ObjectMeta{
				Name: "replicaset-12345",
			},
			expected: false,
		},
		{
			name: "Pod is not active and not owned by ReplicaSet",
			pod: Pod{
				ObjectMeta: ObjectMeta{
					Name: "other-replicaset-12345-pod",
				},
				Status: PodFailed,
			},
			meta: ObjectMeta{
				Name: "replicaset-12345",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsPodActiveAndOwnedBy(&tt.pod, &tt.meta))
		})
	}
}

func TestIsOwnedBy(t *testing.T) {
	tests := []struct {
		name     string
		pod      Pod
		meta     ObjectMeta
		expected bool
	}{
		{
			name: "Pod is owned by ReplicaSet",
			pod: Pod{
				ObjectMeta: ObjectMeta{
					Name: "replicaset-12345-pod",
				},
			},
			meta: ObjectMeta{
				Name: "replicaset-12345",
			},
			expected: true,
		},
		{
			name: "Pod is not owned by ReplicaSet",
			pod: Pod{
				ObjectMeta: ObjectMeta{
					Name: "other-replicaset-12345-pod",
				},
			},
			meta: ObjectMeta{
				Name: "replicaset-12345",
			},
			expected: false,
		},
		{
			name: "Pod name does not contain ReplicaSet name",
			pod: Pod{
				ObjectMeta: ObjectMeta{
					Name: "pod",
				},
			},
			meta: ObjectMeta{
				Name: "replicaset-12345",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsOwnedBy(&tt.pod, &tt.meta))
		})
	}
}
