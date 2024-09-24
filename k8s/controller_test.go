package k8s

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	require.NotNil(t, updatedDeployment)
	require.Equal(t, int32(2), *updatedDeployment.Spec.Replicas)
	require.Equal(t, "updated-image:v2", updatedDeployment.Spec.Template.Spec.Containers[0].Image)
	require.Equal(t, "updated-test", updatedDeployment.Labels["app"])

	// Verify the existing secret was updated
	updatedSecret, err := clientset.CoreV1().Secrets("test-namespace").Get(ctx, "existing-deployment-api-key", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, updatedSecret)
	require.Equal(t, "updated-api-key", updatedSecret.StringData["MAOS_API_KEY"])

	// Verify the new deployment was created
	newDeployment, err := clientset.AppsV1().Deployments("test-namespace").Get(ctx, "new-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, newDeployment)
	require.Equal(t, int32(1), *newDeployment.Spec.Replicas)
	require.Equal(t, "new-image:v1", newDeployment.Spec.Template.Spec.Containers[0].Image)
	require.Equal(t, "new-test", newDeployment.Labels["app"])

	// Verify the new secret was created
	newSecret, err := clientset.CoreV1().Secrets("test-namespace").Get(ctx, "new-deployment-api-key", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, newSecret)
	require.Equal(t, "new-api-key", newSecret.StringData["MAOS_API_KEY"])

	// Verify that only two deployments exist (no extra deployments)
	deploymentList, err := clientset.AppsV1().Deployments("test-namespace").List(ctx, meta.ListOptions{})
	require.NoError(t, err)
	require.Len(t, deploymentList.Items, 2)
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
	require.NotNil(t, newDeployment)
	require.Equal(t, int32(2), *newDeployment.Spec.Replicas)
	require.Equal(t, "test-image:v1", newDeployment.Spec.Template.Spec.Containers[0].Image)
	require.Equal(t, "test-app", newDeployment.Labels["app"])

	// Verify the new secret was created
	newSecret, err := clientset.CoreV1().Secrets("test-namespace").Get(ctx, "new-deployment-api-key", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, newSecret)
	require.Equal(t, "test-api-key", newSecret.StringData["MAOS_API_KEY"])

	// Verify that only one deployment exists
	deploymentList, err := clientset.AppsV1().Deployments("test-namespace").List(ctx, meta.ListOptions{})
	require.NoError(t, err)
	require.Len(t, deploymentList.Items, 1)

	// Verify that only one secret exists
	secretList, err := clientset.CoreV1().Secrets("test-namespace").List(ctx, meta.ListOptions{})
	require.NoError(t, err)
	require.Len(t, secretList.Items, 1)

	// Verify that a service account was created
	sa, err := clientset.CoreV1().ServiceAccounts("test-namespace").Get(ctx, "new-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, sa)
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
	require.NotNil(t, updatedDeployment)
	require.NotEmpty(t, updatedDeployment.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"])
}

// Add more test functions for other methods as needed

func TestK8sController_ListSecrets(t *testing.T) {
	// Create a fake clientset with existing secrets
	secret1 := &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      "secret1",
			Namespace: "test-namespace",
			Labels:    map[string]string{"created-by": "maos"},
		},
		Data: map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		},
	}
	secret2 := &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      "secret2",
			Namespace: "test-namespace",
			Labels:    map[string]string{"created-by": "maos"},
		},
		Data: map[string][]byte{
			"key3": []byte("value3"),
		},
	}
	secretOther := &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      "other-secret",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"key4": []byte("value4"),
		},
	}
	clientset := fake.NewSimpleClientset(secret1, secret2, secretOther)

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	// Test case
	ctx := context.Background()

	// List secrets
	secrets, err := controller.ListSecrets(ctx)
	require.NoError(t, err)

	// Verify the secrets list
	require.Len(t, secrets, 3)

	// Create a map of secrets for easier requireion
	secretMap := make(map[string]Secret)
	for _, s := range secrets {
		secretMap[s.Name] = s
	}

	// Assert secret1
	require.Contains(t, secretMap, "secret1")
	require.ElementsMatch(t, []string{"key1", "key2"}, secretMap["secret1"].Keys)

	// Assert secret2
	require.Contains(t, secretMap, "secret2")
	require.ElementsMatch(t, []string{"key3"}, secretMap["secret2"].Keys)

	// Assert other-secret is not included
	require.Contains(t, secretMap, "other-secret")
}

func TestK8sController_UpdateSecret(t *testing.T) {
	// Create a fake clientset
	clientset := fake.NewSimpleClientset()

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	ctx := context.Background()

	// Test case 1: Update an existing secret
	existingSecret := &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"created-by": "maos",
			},
		},
		StringData: map[string]string{
			"existing-key": "existing-value",
		},
	}
	_, err := clientset.CoreV1().Secrets("test-namespace").Create(ctx, existingSecret, meta.CreateOptions{})
	require.NoError(t, err)

	// Update the existing secret
	err = controller.UpdateSecret(ctx, "existing-secret", map[string]string{
		"existing-key": "updated-value",
		"new-key":      "new-value",
	})
	require.NoError(t, err)

	// Verify the secret was updated
	updatedSecret, err := clientset.CoreV1().Secrets("test-namespace").Get(ctx, "existing-secret", meta.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "updated-value", string(updatedSecret.Data["existing-key"]))
	require.Equal(t, "new-value", string(updatedSecret.Data["new-key"]))

	// Test case 2: Create a new secret
	err = controller.UpdateSecret(ctx, "new-secret", map[string]string{
		"key1": "value1",
		"key2": "value2",
	})
	require.NoError(t, err)

	// Verify the new secret was created
	newSecret, err := clientset.CoreV1().Secrets("test-namespace").Get(ctx, "new-secret", meta.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "value1", newSecret.StringData["key1"])
	require.Equal(t, "value2", newSecret.StringData["key2"])
	require.Equal(t, "maos", newSecret.Labels["created-by"])
}

func TestDeleteSecret(t *testing.T) {
	// Create a fake clientset
	clientset := fake.NewSimpleClientset()

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	ctx := context.Background()

	// Create a test secret
	testSecret := &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"created-by": "maos",
			},
		},
		StringData: map[string]string{
			"key1": "value1",
		},
	}
	_, err := clientset.CoreV1().Secrets("test-namespace").Create(ctx, testSecret, meta.CreateOptions{})
	require.NoError(t, err)

	// Test case 1: Delete an existing secret
	err = controller.DeleteSecret(ctx, "test-secret")
	require.NoError(t, err)

	// Verify the secret was deleted
	_, err = clientset.CoreV1().Secrets("test-namespace").Get(ctx, "test-secret", meta.GetOptions{})
	require.True(t, errors.IsNotFound(err), "Expected secret to be deleted")

	// Test case 2: Try to delete a non-existent secret
	err = controller.DeleteSecret(ctx, "non-existent-secret")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to delete secret")
}
