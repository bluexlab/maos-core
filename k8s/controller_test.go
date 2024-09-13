package k8s

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestK8sController_UpdateDeploymentSet(t *testing.T) {
	// Create a fake clientset with an existing deployment
	existingServiceAccount := &core.ServiceAccount{
		ObjectMeta: meta.ObjectMeta{
			Name:      "existing-deployment",
			Namespace: "test-namespace",
		},
	}
	existingDeployment := &apps.Deployment{
		ObjectMeta: meta.ObjectMeta{
			Name:      "existing-deployment",
			Namespace: "test-namespace",
			Labels:    map[string]string{"created-by": "maos", "app": "existing-test"},
		},
		Spec: apps.DeploymentSpec{
			Replicas: lo.ToPtr(int32(1)),
			Template: core.PodTemplateSpec{
				Spec: core.PodSpec{
					ServiceAccountName: "existing-deployment",
					Containers: []core.Container{
						{
							Name:  "existing-deployment",
							Image: "existing-image:v1",
						},
					},
				},
			},
		},
	}
	existingSecret := &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      "existing-deployment-api-key",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"MAOS_API_KEY": []byte("existing-api-key"),
		},
	}
	clientset := fake.NewSimpleClientset(existingDeployment, existingSecret, existingServiceAccount)

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	// Test case
	ctx := context.Background()
	deploymentSet := []DeploymentParams{
		{
			Name:          "existing-deployment",
			Replicas:      2,
			Labels:        map[string]string{"app": "updated-test"},
			Image:         "updated-image:v2",
			EnvVars:       map[string]string{"UPDATED_ENV": "updated-value"},
			APIKey:        "updated-api-key",
			MemoryRequest: "256Mi",
			MemoryLimit:   "512Mi",
		},
		{
			Name:          "new-deployment",
			Replicas:      1,
			Labels:        map[string]string{"app": "new-test"},
			Image:         "new-image:v1",
			EnvVars:       map[string]string{"NEW_ENV": "new-value"},
			APIKey:        "new-api-key",
			MemoryRequest: "128Mi",
			MemoryLimit:   "256Mi",
		},
	}

	// Run the UpdateDeploymentSet method
	err := controller.UpdateDeploymentSet(ctx, deploymentSet)
	require.NoError(t, err)

	// Verify the existing deployment was updated
	updatedDeployment, err := clientset.AppsV1().Deployments("test-namespace").Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, updatedDeployment)
	assert.Equal(t, int32(2), *updatedDeployment.Spec.Replicas)
	assert.Equal(t, "updated-image:v2", updatedDeployment.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, "updated-test", updatedDeployment.Labels["app"])

	// Verify the existing secret was updated
	updatedSecret, err := clientset.CoreV1().Secrets("test-namespace").Get(ctx, "existing-deployment-api-key", meta.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, updatedSecret)
	assert.Equal(t, "updated-api-key", updatedSecret.StringData["MAOS_API_KEY"])

	// Verify the new deployment was created
	newDeployment, err := clientset.AppsV1().Deployments("test-namespace").Get(ctx, "new-deployment", meta.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, newDeployment)
	assert.Equal(t, int32(1), *newDeployment.Spec.Replicas)
	assert.Equal(t, "new-image:v1", newDeployment.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, "new-test", newDeployment.Labels["app"])

	// Verify the new secret was created
	newSecret, err := clientset.CoreV1().Secrets("test-namespace").Get(ctx, "new-deployment-api-key", meta.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, newSecret)
	assert.Equal(t, "new-api-key", newSecret.StringData["MAOS_API_KEY"])

	// Verify that only two deployments exist (no extra deployments)
	deploymentList, err := clientset.AppsV1().Deployments("test-namespace").List(ctx, meta.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, deploymentList.Items, 2)
}
func TestK8sController_UpdateDeploymentSet_EmptyCluster(t *testing.T) {
	// Create a new empty fake clientset
	clientset := fake.NewSimpleClientset()

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	// Test case
	ctx := context.Background()

	// Create a deployment set with one new deployment
	deploymentSet := []DeploymentParams{
		{
			Name:          "new-deployment",
			Replicas:      2,
			Labels:        map[string]string{"app": "test-app"},
			Image:         "test-image:v1",
			EnvVars:       map[string]string{"ENV_VAR": "value"},
			APIKey:        "test-api-key",
			MemoryRequest: "64Mi",
			MemoryLimit:   "128Mi",
		},
	}

	// Run the UpdateDeploymentSet method
	err := controller.UpdateDeploymentSet(ctx, deploymentSet)
	require.NoError(t, err)

	// Verify the new deployment was created
	newDeployment, err := clientset.AppsV1().Deployments("test-namespace").Get(ctx, "new-deployment", meta.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, newDeployment)
	assert.Equal(t, int32(2), *newDeployment.Spec.Replicas)
	assert.Equal(t, "test-image:v1", newDeployment.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, "test-app", newDeployment.Labels["app"])

	// Verify the new secret was created
	newSecret, err := clientset.CoreV1().Secrets("test-namespace").Get(ctx, "new-deployment-api-key", meta.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, newSecret)
	assert.Equal(t, "test-api-key", newSecret.StringData["MAOS_API_KEY"])

	// Verify that only one deployment exists
	deploymentList, err := clientset.AppsV1().Deployments("test-namespace").List(ctx, meta.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, deploymentList.Items, 1)

	// Verify that only one secret exists
	secretList, err := clientset.CoreV1().Secrets("test-namespace").List(ctx, meta.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, secretList.Items, 1)

	// Verify that a service account was created
	sa, err := clientset.CoreV1().ServiceAccounts("test-namespace").Get(ctx, "new-deployment", meta.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, sa)
}

func TestK8sController_TriggerRollingRestart(t *testing.T) {
	// Create a fake clientset with an existing deployment
	existingDeployment := &apps.Deployment{
		ObjectMeta: meta.ObjectMeta{
			Name:      "existing-deployment",
			Namespace: "test-namespace",
		},
		Spec: apps.DeploymentSpec{
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
	}
	clientset := fake.NewSimpleClientset(existingDeployment)

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	// Test case
	ctx := context.Background()

	// Trigger rolling restart
	err := controller.TriggerRollingRestart(ctx, "existing-deployment")
	require.NoError(t, err)

	// Verify the deployment was updated
	updatedDeployment, err := clientset.AppsV1().Deployments("test-namespace").Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, updatedDeployment)
	assert.NotEmpty(t, updatedDeployment.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"])
}

// Add more test functions for other methods as needed
