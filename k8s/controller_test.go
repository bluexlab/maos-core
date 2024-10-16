package k8s

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
			Labels:    map[string]string{"created-by": "maos", "app": "existing-deployment", "component": "existing-test"},
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
			Labels:        map[string]string{"component": "updated-test"},
			Image:         "updated-image:v2",
			EnvVars:       map[string]string{"UPDATED_ENV": "updated-value"},
			APIKey:        "updated-api-key",
			MemoryRequest: "256Mi",
			MemoryLimit:   "512Mi",
		},
		{
			Name:          "new-deployment",
			Replicas:      1,
			Labels:        map[string]string{"component": "new-test"},
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
	require.Equal(t, "updated-test", updatedDeployment.Labels["component"])

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
	require.Equal(t, "new-test", newDeployment.Labels["component"])

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
			Labels:        map[string]string{"component": "test-app"},
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
	require.Equal(t, "test-app", newDeployment.Labels["component"])

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

// TestK8sController_UpdateDeploymentSet_HasService tests the scenario where hasService is true
func TestK8sController_UpdateDeploymentSet_HasService_WithEmptyCluster(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	ctx := context.Background()

	// Create a deployment set with a service
	deploymentSet := []DeploymentParams{
		{
			Name:          "existing-deployment",
			Replicas:      2,
			Labels:        map[string]string{"component": "test-app"},
			Image:         "test-image:v1",
			EnvVars:       map[string]string{"ENV_VAR": "value"},
			APIKey:        "test-api-key",
			MemoryRequest: "64Mi",
			MemoryLimit:   "128Mi",
			HasService:    true, // Set hasService to true
			ServicePorts:  []int32{8080},
			BodyLimit:     "1Mi", // Set bodyLimit
		},
	}

	// Run the UpdateDeploymentSet method
	err := controller.UpdateDeploymentSet(ctx, deploymentSet)
	require.NoError(t, err)

	// Verify the new deployment was created
	newDeployment, err := clientset.AppsV1().Deployments("test-namespace").Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, newDeployment)
	require.Equal(t, int32(2), *newDeployment.Spec.Replicas)
	require.Equal(t, "test-image:v1", newDeployment.Spec.Template.Spec.Containers[0].Image)
	require.Equal(t, "test-app", newDeployment.Labels["component"])
	// verify service was updated
	updatedService, err := clientset.CoreV1().Services("test-namespace").Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, updatedService)
	require.Equal(t, int32(8080), updatedService.Spec.Ports[0].Port)
}

func TestK8sController_UpdateDeploymentSet_HasService(t *testing.T) {
	// Create a fake clientset with an existing service and deployment
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
			Labels:    map[string]string{"created-by": "maos", "app": "existing-deployment", "component": "existing-test"},
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
	existingService := &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:      "existing-service",
			Namespace: "test-namespace",
			Labels:    map[string]string{"created-by": "maos", "app": "existing-deployment", "component": "existing-test"},
		},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"app": "test-app"},
			Ports: []core.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
	}
	clientset := fake.NewSimpleClientset(existingServiceAccount, existingDeployment, existingService)

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	ctx := context.Background()

	// Create a deployment set with a service
	deploymentSet := []DeploymentParams{
		{
			Name:          "existing-deployment",
			Replicas:      2,
			Labels:        map[string]string{"component": "test-app"},
			Image:         "test-image:v1",
			EnvVars:       map[string]string{"ENV_VAR": "value"},
			APIKey:        "test-api-key",
			MemoryRequest: "64Mi",
			MemoryLimit:   "128Mi",
			HasService:    true, // Set hasService to true
			ServicePorts:  []int32{8080, 8081},
			BodyLimit:     "1Mi", // Set bodyLimit
		},
	}

	// Run the UpdateDeploymentSet method
	err := controller.UpdateDeploymentSet(ctx, deploymentSet)
	require.NoError(t, err)

	// Verify the new deployment was created
	newDeployment, err := clientset.AppsV1().Deployments("test-namespace").Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, newDeployment)
	require.Equal(t, int32(2), *newDeployment.Spec.Replicas)
	require.Equal(t, "test-image:v1", newDeployment.Spec.Template.Spec.Containers[0].Image)
	require.Equal(t, "test-app", newDeployment.Labels["component"])
	// verify service was updated
	updatedService, err := clientset.CoreV1().Services("test-namespace").Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, updatedService)
	require.Equal(t, int32(8080), updatedService.Spec.Ports[0].Port)
	require.Equal(t, int32(8080), updatedService.Spec.Ports[0].TargetPort.IntVal)
	require.Equal(t, int32(8081), updatedService.Spec.Ports[1].Port)
	require.Equal(t, int32(8081), updatedService.Spec.Ports[1].TargetPort.IntVal)
}

// TestK8sController_UpdateDeploymentSet_HasServiceAndIngress tests the scenario where both hasService and hasIngress are true
func TestK8sController_UpdateDeploymentSet_HasServiceAndIngress_WithEmptyCluster(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	ctx := context.Background()

	// Create a deployment set with a service and ingress
	deploymentSet := []DeploymentParams{
		{
			Name:          "existing-deployment",
			Replicas:      2,
			Labels:        map[string]string{"component": "test-app"},
			Image:         "test-image:v1",
			EnvVars:       map[string]string{"ENV_VAR": "value"},
			APIKey:        "test-api-key",
			MemoryRequest: "64Mi",
			MemoryLimit:   "128Mi",
			HasService:    true,
			ServicePorts:  []int32{8080, 8081},
			HasIngress:    true,
			IngressHost:   "example.com",
			BodyLimit:     "1Mi",
		},
	}

	// Run the UpdateDeploymentSet method
	err := controller.UpdateDeploymentSet(ctx, deploymentSet)
	require.NoError(t, err)

	// Verify the new deployment was created
	newDeployment, err := clientset.AppsV1().Deployments("test-namespace").Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, newDeployment)
	require.Equal(t, int32(2), *newDeployment.Spec.Replicas)
	require.Equal(t, "test-image:v1", newDeployment.Spec.Template.Spec.Containers[0].Image)
	require.Equal(t, "test-app", newDeployment.Labels["component"])

	// Verify the new service was created
	newService, err := clientset.CoreV1().Services("test-namespace").Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, newService)
	require.Equal(t, int32(8080), newService.Spec.Ports[0].Port)

	// Verify the new ingress was created
	newIngress, err := clientset.NetworkingV1().Ingresses("test-namespace").Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, newIngress)
	require.Equal(t, "example.com", newIngress.Spec.Rules[0].Host)
	require.Equal(t, int32(8080), newIngress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number)
}

func TestK8sController_UpdateDeploymentSet_HasServiceAndIngress(t *testing.T) {
	// Create a fake clientset with an existing service and ingress
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
			Labels:    map[string]string{"created-by": "maos", "app": "existing-deployment", "component": "existing-test"},
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
	existingService := &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:      "existing-deployment",
			Namespace: "test-namespace",
			Labels:    map[string]string{"created-by": "maos", "app": "existing-deployment", "component": "existing-test"},
		},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"component": "existing-test"},
			Ports: []core.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
	}
	existingIngress := &networking.Ingress{
		ObjectMeta: meta.ObjectMeta{
			Name:      "existing-deployment",
			Namespace: "test-namespace",
			Labels:    map[string]string{"created-by": "maos", "app": "existing-deployment", "component": "existing-test"},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "old-example.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path:     "/",
									PathType: lo.ToPtr(networking.PathTypePrefix),
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: "existing-deployment",
											Port: networking.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	clientset := fake.NewSimpleClientset(existingServiceAccount, existingDeployment, existingService, existingIngress)

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	ctx := context.Background()

	// Create a deployment set with a service and ingress
	deploymentSet := []DeploymentParams{
		{
			Name:          "existing-deployment",
			Replicas:      2,
			Labels:        map[string]string{"component": "test-app"},
			Image:         "test-image:v1",
			EnvVars:       map[string]string{"ENV_VAR": "value"},
			APIKey:        "test-api-key",
			MemoryRequest: "64Mi",
			MemoryLimit:   "128Mi",
			HasService:    true,
			ServicePorts:  []int32{8080, 8081},
			HasIngress:    true,
			IngressHost:   "example.com",
			BodyLimit:     "1Mi",
		},
	}

	// Run the UpdateDeploymentSet method
	err := controller.UpdateDeploymentSet(ctx, deploymentSet)
	require.NoError(t, err)

	// Verify the new deployment was updated
	newDeployment, err := clientset.AppsV1().Deployments("test-namespace").Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, newDeployment)
	require.Equal(t, int32(2), *newDeployment.Spec.Replicas)
	require.Equal(t, "test-image:v1", newDeployment.Spec.Template.Spec.Containers[0].Image)
	require.Equal(t, "test-app", newDeployment.Labels["component"])

	// Verify the new service was updated
	newService, err := clientset.CoreV1().Services("test-namespace").Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, newService)
	require.Equal(t, int32(8080), newService.Spec.Ports[0].Port)
	require.Equal(t, int32(8080), newService.Spec.Ports[0].TargetPort.IntVal)
	require.Equal(t, int32(8081), newService.Spec.Ports[1].Port)
	require.Equal(t, int32(8081), newService.Spec.Ports[1].TargetPort.IntVal)

	// Verify the new ingress was updated
	newIngress, err := clientset.NetworkingV1().Ingresses("test-namespace").Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, newIngress)
	require.Equal(t, "example.com", newIngress.Spec.Rules[0].Host)
	require.Equal(t, int32(8080), newIngress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number)
}

func TestK8sController_UpdateDeploymentSet_WithExistingServiceAndIngressAndUpdateToBlankSet(t *testing.T) {
	// Create a fake clientset with an existing service and ingress
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
			Labels:    map[string]string{"created-by": "maos", "app": "existing-deployment", "component": "existing-test"},
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
	existingService := &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:      "existing-deployment",
			Namespace: "test-namespace",
			Labels:    map[string]string{"created-by": "maos", "app": "existing-deployment", "component": "existing-test"},
		},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"component": "existing-test"},
			Ports: []core.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
	}
	existingIngress := &networking.Ingress{
		ObjectMeta: meta.ObjectMeta{
			Name:      "existing-deployment",
			Namespace: "test-namespace",
			Labels:    map[string]string{"created-by": "maos", "app": "existing-deployment", "component": "existing-test"},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "old-example.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path:     "/",
									PathType: lo.ToPtr(networking.PathTypePrefix),
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: "existing-deployment",
											Port: networking.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	clientset := fake.NewSimpleClientset(existingServiceAccount, existingDeployment, existingService, existingIngress)

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	ctx := context.Background()

	// Create a deployment set with a service and ingress
	deploymentSet := []DeploymentParams{}

	// Run the UpdateDeploymentSet method
	err := controller.UpdateDeploymentSet(ctx, deploymentSet)
	require.NoError(t, err)

	// Verify the there is no deployment
	deploymentList, err := clientset.AppsV1().Deployments(controller.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	require.NoError(t, err)
	require.Empty(t, deploymentList.Items)

	// Verify the there is no service
	serviceList, err := clientset.CoreV1().Services(controller.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	require.NoError(t, err)
	require.Empty(t, serviceList.Items)

	// Verify the there is no ingress
	ingressList, err := clientset.NetworkingV1().Ingresses(controller.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	require.NoError(t, err)
	require.Empty(t, ingressList.Items)
}

func TestK8sController_UpdateDeploymentSet_WithExistingServiceAndIngressAndUpdateToNoServiceAndIngress(t *testing.T) {
	// Create a fake clientset with an existing service and ingress
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
			Labels:    map[string]string{"created-by": "maos", "app": "existing-deployment", "component": "existing-test"},
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
	existingService := &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:      "existing-deployment",
			Namespace: "test-namespace",
			Labels:    map[string]string{"created-by": "maos", "app": "existing-deployment", "component": "existing-test"},
		},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"component": "existing-test"},
			Ports: []core.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
	}
	existingIngress := &networking.Ingress{
		ObjectMeta: meta.ObjectMeta{
			Name:      "existing-deployment",
			Namespace: "test-namespace",
			Labels:    map[string]string{"created-by": "maos", "app": "existing-deployment", "component": "existing-test"},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "old-example.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path:     "/",
									PathType: lo.ToPtr(networking.PathTypePrefix),
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: "existing-deployment",
											Port: networking.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	clientset := fake.NewSimpleClientset(existingServiceAccount, existingDeployment, existingService, existingIngress)

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	ctx := context.Background()

	// Create a deployment set with a service and ingress
	deploymentSet := []DeploymentParams{
		{
			Name:          "existing-deployment",
			Replicas:      2,
			Labels:        map[string]string{"component": "test-app"},
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

	// Verify the deployment was updated
	newDeployment, err := clientset.AppsV1().Deployments(controller.namespace).Get(ctx, "existing-deployment", meta.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, newDeployment)
	require.Equal(t, int32(2), *newDeployment.Spec.Replicas)
	require.Equal(t, "test-image:v1", newDeployment.Spec.Template.Spec.Containers[0].Image)
	require.Equal(t, "test-app", newDeployment.Labels["component"])

	// Verify the there is no service
	serviceList, err := clientset.CoreV1().Services(controller.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	require.NoError(t, err)
	require.Empty(t, serviceList.Items)

	// Verify the there is no ingress
	ingressList, err := clientset.NetworkingV1().Ingresses(controller.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	require.NoError(t, err)
	require.Empty(t, ingressList.Items)
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

func TestK8sController_RunMigrations_Success(t *testing.T) {
	// Create a fake clientset
	clientset := fake.NewSimpleClientset()

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	ctx := context.Background()

	// Create migration params
	migrations := []MigrationParams{
		{
			Serial:        1,
			Name:          "migration1",
			Image:         "migration-image:v1",
			EnvVars:       map[string]string{"ENV_VAR": "value"},
			Command:       []string{"run", "migration"},
			MemoryRequest: "64Mi",
			MemoryLimit:   "128Mi",
		},
	}

	go func() {
		time.Sleep(300 * time.Millisecond)
		// Verify that the job was created
		slog.Info("---- list jobs")
		jobs, err := clientset.BatchV1().Jobs("test-namespace").List(ctx, meta.ListOptions{})
		slog.Info("---- after list jobs", "jobs", jobs, "err", err)
		require.NoError(t, err)
		require.Len(t, jobs.Items, 1)
		require.Equal(t, "migration-migration1-1", jobs.Items[0].Name)

		// Simulate job completion
		job := &jobs.Items[0]
		job.Status.Succeeded = 1
		slog.Info("---- before job status", "status", job.Status)
		job, err = clientset.BatchV1().Jobs("test-namespace").UpdateStatus(ctx, job, meta.UpdateOptions{})
		slog.Info("---- job status", "status", job.Status)
		require.NoError(t, err)
	}()

	// Run the migrations
	failures, err := controller.RunMigrations(ctx, migrations)

	// Assert no errors and no failures
	require.NoError(t, err)
	require.Empty(t, failures)
}

func TestK8sController_RunMigrations_Failure(t *testing.T) {
	// Create a fake clientset
	clientset := fake.NewSimpleClientset()

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	ctx := context.Background()

	// Create migration params
	migrations := []MigrationParams{
		{
			Serial:        1,
			Name:          "failed-migration",
			Image:         "migration-image:v1",
			EnvVars:       map[string]string{"ENV_VAR": "value"},
			Command:       []string{"run", "migration"},
			MemoryRequest: "64Mi",
			MemoryLimit:   "128Mi",
		},
	}

	go func() {
		time.Sleep(300 * time.Millisecond)
		// Verify that the job was created
		jobs, err := clientset.BatchV1().Jobs("test-namespace").List(ctx, meta.ListOptions{})
		require.NoError(t, err)
		require.Len(t, jobs.Items, 1)
		require.Equal(t, "migration-failed-migration-1", jobs.Items[0].Name)

		// Simulate job completion
		job := &jobs.Items[0]
		job.Status.Failed = 1
		_, err = clientset.BatchV1().Jobs("test-namespace").UpdateStatus(ctx, job, meta.UpdateOptions{})
		require.NoError(t, err)
	}()

	// Run the migrations
	failures, err := controller.RunMigrations(ctx, migrations)

	// Assert errors and failures
	require.Error(t, err)
	require.Contains(t, err.Error(), "migration failed")
	require.Len(t, failures, 1)
	require.Contains(t, failures, "migration-failed-migration-1")
}

func TestK8sController_RunMigrations_Timeout(t *testing.T) {
	// Create a fake clientset
	clientset := fake.NewSimpleClientset()

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create migration params
	migrations := []MigrationParams{
		{
			Serial:        1,
			Name:          "timeout-migration",
			Image:         "migration-image:v1",
			EnvVars:       map[string]string{"ENV_VAR": "value"},
			Command:       []string{"run", "migration"},
			MemoryRequest: "64Mi",
			MemoryLimit:   "128Mi",
		},
	}

	// Run the migrations
	failures, err := controller.RunMigrations(ctx, migrations)

	// Assert context deadline exceeded error
	require.Error(t, err)
	require.Equal(t, context.DeadlineExceeded, err)
	require.Nil(t, failures)

	// Verify that the job was created
	jobs, err := clientset.BatchV1().Jobs("test-namespace").List(context.Background(), meta.ListOptions{})
	require.NoError(t, err)
	require.Len(t, jobs.Items, 1)
	require.Equal(t, "migration-timeout-migration-1", jobs.Items[0].Name)
}

func TestK8sController_RunMigrations_MultipleJobs(t *testing.T) {
	// Create a fake clientset
	clientset := fake.NewSimpleClientset()

	// Create a K8sController with the fake clientset
	controller := &K8sController{
		clientset: clientset,
		namespace: "test-namespace",
	}

	ctx := context.Background()

	// Create migration params for multiple jobs
	migrations := []MigrationParams{
		{
			Serial:        168,
			Name:          "migration1",
			Image:         "migration-image:v1",
			EnvVars:       map[string]string{"ENV_VAR": "value1"},
			Command:       []string{"run", "migration1"},
			MemoryRequest: "64Mi",
			MemoryLimit:   "128Mi",
		},
		{
			Serial:        168,
			Name:          "migration2",
			Image:         "migration-image:v2",
			EnvVars:       map[string]string{"ENV_VAR": "value2"},
			Command:       []string{"run", "migration2"},
			MemoryRequest: "128Mi",
			MemoryLimit:   "256Mi",
		},
	}

	go func() {
		time.Sleep(300 * time.Millisecond)

		jobs, err := clientset.BatchV1().Jobs("test-namespace").List(ctx, meta.ListOptions{})
		require.NoError(t, err)
		require.Len(t, jobs.Items, 2)
		require.Equal(t, "migration-migration1-168", jobs.Items[0].Name)
		require.Equal(t, "migration-migration2-168", jobs.Items[1].Name)

		// Simulate job completion
		for _, job := range jobs.Items {
			job.Status.Succeeded = 1
			_, err = clientset.BatchV1().Jobs("test-namespace").UpdateStatus(ctx, &job, meta.UpdateOptions{})
			require.NoError(t, err)
		}
	}()

	// Run the migrations
	failures, err := controller.RunMigrations(ctx, migrations)

	// Assert no errors and no failures
	require.NoError(t, err)
	require.Empty(t, failures)
}

func TestSerializeSecretsToJSON(t *testing.T) {
	// Create a fake clientset
	clientset := fake.NewSimpleClientset()

	// Create a test controller
	controller := &K8sController{
		clientset: clientset,
		namespace: "default",
	}

	// Create some test secrets
	testSecrets := []core.Secret{
		{
			ObjectMeta: meta.ObjectMeta{
				Name: "secret1",
				Labels: map[string]string{
					"created-by": "maos",
				},
			},
			Data: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			},
		},
		{
			ObjectMeta: meta.ObjectMeta{
				Name: "secret2",
				Labels: map[string]string{
					"created-by": "maos",
				},
			},
			Data: map[string][]byte{
				"key3": []byte("value3"),
			},
		},
	}

	// Add test secrets to the fake clientset
	for _, secret := range testSecrets {
		_, err := clientset.CoreV1().Secrets("default").Create(context.Background(), &secret, meta.CreateOptions{})
		require.NoError(t, err)
	}

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicKey := &privateKey.PublicKey
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	require.NoError(t, err)

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	// Call SerializeSecretsToJSON
	result, err := controller.SerializeSecretsToJSON(context.Background(), string(publicKeyPEM))
	require.NoError(t, err)

	// Parse the JWE
	var allKeyAlgorithms = []jose.KeyAlgorithm{
		jose.RSA1_5,
		jose.RSA_OAEP,
		jose.RSA_OAEP_256,
		jose.ECDH_ES,
		jose.ECDH_ES_A128KW,
		jose.ECDH_ES_A192KW,
		jose.ECDH_ES_A256KW,
	}
	var allContentEncryptions = []jose.ContentEncryption{
		jose.A128GCM,
		jose.A192GCM,
		jose.A256GCM,
	}

	jwe, err := jose.ParseEncrypted(result, allKeyAlgorithms, allContentEncryptions)
	require.NoError(t, err)

	// Decrypt the JWE
	decrypted, err := jwe.Decrypt(privateKey)
	require.NoError(t, err)

	// Unmarshal the decrypted data
	var secretsData []SecretData
	err = json.Unmarshal(decrypted, &secretsData)
	require.NoError(t, err)

	// Assert the decrypted data matches the original secrets
	assert.Len(t, secretsData, len(testSecrets))
	for _, secret := range secretsData {
		originalSecret, found := lo.Find(testSecrets, func(s core.Secret) bool {
			return s.Name == secret.Name
		})
		assert.True(t, found)
		for key, value := range secret.Data {
			decodedValue, err := base64.StdEncoding.DecodeString(value)
			require.NoError(t, err)
			assert.Equal(t, string(originalSecret.Data[key]), string(decodedValue))
		}
	}
}
