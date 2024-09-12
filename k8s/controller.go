package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// DeploymentParams struct
type DeploymentParams struct {
	Name          string
	Replicas      int32
	Labels        map[string]string
	Image         string
	EnvVars       map[string]string
	APIKey        string
	MemoryRequest string
	MemoryLimit   string
}

// Controller defines the methods that a Controller should implement
type Controller interface {
	UpdateDeploymentSet(ctx context.Context, deploymentSet []DeploymentParams) error
}

// K8sController implements the Controller interface
type K8sController struct {
	clientset *kubernetes.Clientset
	namespace string
}

// NewK8sController creates a new Controller with a kubernetes clientset
func NewK8sController() (*K8sController, error) {
	var config *rest.Config
	var err error

	// Check if running in-cluster
	if _, exists := os.LookupEnv("KUBERNETES_SERVICE_HOST"); exists {
		// Use in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("error creating in-cluster config: %v", err)
		}
	} else {
		// Use out-of-cluster config
		kubeconfigPath := os.Getenv("KUBECONFIG_PATH")
		if kubeconfigPath == "" {
			return nil, fmt.Errorf("KUBECONFIG_PATH environment variable not set")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("error building kubeconfig: %v", err)
		}
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %v", err)
	}

	// get the current namespace
	currentNamespace := "default"
	if inKubernetes() {
		currentNamespace, err = getCurrentNamespace()
		if err != nil {
			return nil, fmt.Errorf("error getting current namespace: %v", err)
		}
	}

	controller := &K8sController{
		clientset: clientset,
		namespace: currentNamespace,
	}

	return controller, nil
}

// UpdateDeploymentSet lists all deployments created by MAOS, removes the ones not in the given set,
// and creates or updates the ones in the given set
func (c *K8sController) UpdateDeploymentSet(ctx context.Context, deploymentSet []DeploymentParams) error {
	// List all deployments in the namespace
	existingDeployments, err := c.clientset.AppsV1().Deployments(c.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos", // Assuming we label MAOS-created deployments
	})
	if err != nil {
		return fmt.Errorf("failed to list existing deployments: %v", err)
	}

	// Create a map of existing deployments for quick lookup
	existingDeploymentMap := make(map[string]*apps.Deployment)
	for i := range existingDeployments.Items {
		deployment := &existingDeployments.Items[i]
		existingDeploymentMap[deployment.Name] = deployment
	}

	// Process the deployment set
	for _, params := range deploymentSet {
		if existingDeployment, exists := existingDeploymentMap[params.Name]; exists {
			// Update existing deployment
			err := c.updateDeployment(ctx, existingDeployment, params)
			if err != nil {
				return fmt.Errorf("failed to update deployment %s: %v", params.Name, err)
			}
			// Remove from map to track which deployments need to be deleted
			delete(existingDeploymentMap, params.Name)
		} else {
			// Create new deployment
			_, err := c.createDeployment(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to create deployment %s: %v", params.Name, err)
			}
		}
	}

	// Delete deployments that are not in the deployment set
	for name := range existingDeploymentMap {
		// Delete the deployment
		err := c.clientset.AppsV1().Deployments(c.namespace).Delete(ctx, name, meta.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete deployment %s: %v", name, err)
		}

		// Delete the associated service account
		err = c.clientset.CoreV1().ServiceAccounts(c.namespace).Delete(ctx, name, meta.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			slog.Error("failed to delete service account", "namespace", c.namespace, "name", name, "error", err)
		}

		slog.Info("Deleted deployment and service account", "namespace", c.namespace, "name", name)
	}

	return nil
}

// TriggerRollingRestart triggers a rolling restart of the deployment
func (c *K8sController) TriggerRollingRestart(ctx context.Context, deploymentName string) error {
	slog.Info("Triggering rolling restart", "namespace", c.namespace, "deployment", deploymentName)

	deployment, err := c.clientset.AppsV1().Deployments(c.namespace).Get(ctx, deploymentName, meta.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %v", err)
	}

	// Update an annotation to trigger the rolling restart
	if deployment.Spec.Template.ObjectMeta.Annotations == nil {
		deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = c.clientset.AppsV1().Deployments(c.namespace).Update(ctx, deployment, meta.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %v", err)
	}

	return nil
}

// createDeployment creates a new deployment with a service account if it doesn't exist
func (c *K8sController) createDeployment(ctx context.Context, params DeploymentParams) (*apps.Deployment, error) {
	slog.Info("Creating deployment", "namespace", c.namespace, "name", params.Name)

	// Check if deployment exists
	_, err := c.clientset.AppsV1().Deployments(c.namespace).Get(ctx, params.Name, meta.GetOptions{})
	if err == nil {
		return nil, fmt.Errorf("deployment %s already exists in namespace %s", params.Name, c.namespace)
	}
	if !errors.IsNotFound(err) {
		return nil, err
	}

	// Create service account
	slog.Info("Creating service account", "namespace", c.namespace, "name", params.Name)
	sa, err := c.createServiceAccount(ctx, params.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create service account: %v", err)
	}

	// Parse memory request and limit
	slog.Info("Parsing memory request and limit", "namespace", c.namespace, "name", params.Name)
	memoryRequest, err := resource.ParseQuantity(params.MemoryRequest)
	if err != nil {
		return nil, fmt.Errorf("invalid memory request: %v", err)
	}
	memoryLimit, err := resource.ParseQuantity(params.MemoryLimit)
	if err != nil {
		return nil, fmt.Errorf("invalid memory limit: %v", err)
	}

	// Convert map[string]string to []core.EnvVar
	var envVars []core.EnvVar
	for key, value := range params.EnvVars {
		envVars = append(envVars, core.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	// Add MAOS_API_KEY to envVars
	envVars = append(envVars, core.EnvVar{
		Name: "MAOS_API_KEY",
		ValueFrom: &core.EnvVarSource{
			SecretKeyRef: &core.SecretKeySelector{
				LocalObjectReference: core.LocalObjectReference{
					Name: "maos-api-key",
				},
				Key: "MAOS_API_KEY",
			},
		},
	})

	// Add the "created-by=maos" label
	if params.Labels == nil {
		params.Labels = make(map[string]string)
	}
	params.Labels["created-by"] = "maos"

	// Create deployment
	slog.Info("Creating deployment", "namespace", c.namespace, "name", params.Name)
	deployment := &apps.Deployment{
		ObjectMeta: meta.ObjectMeta{
			Name:      params.Name,
			Namespace: c.namespace,
			Labels:    params.Labels,
		},
		Spec: apps.DeploymentSpec{
			Replicas: &params.Replicas,
			Selector: &meta.LabelSelector{
				MatchLabels: params.Labels,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels: params.Labels,
				},
				Spec: core.PodSpec{
					ServiceAccountName: sa.Name,
					Containers: []core.Container{
						{
							Name:            params.Name,
							Image:           params.Image,
							ImagePullPolicy: core.PullIfNotPresent,
							Env:             envVars,
							Resources: core.ResourceRequirements{
								Requests: core.ResourceList{
									core.ResourceMemory: memoryRequest,
								},
								Limits: core.ResourceList{
									core.ResourceMemory: memoryLimit,
								},
							},
						},
					},
				},
			},
		},
	}

	return c.clientset.AppsV1().Deployments(c.namespace).Create(ctx, deployment, meta.CreateOptions{})
}

// createServiceAccount creates a new service account for the deployment
func (c *K8sController) createServiceAccount(ctx context.Context, name string) (*core.ServiceAccount, error) {
	sa := &core.ServiceAccount{
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: c.namespace,
		},
	}
	return c.clientset.CoreV1().ServiceAccounts(c.namespace).Create(ctx, sa, meta.CreateOptions{})
}

// updateDeployment updates an existing deployment with new parameters
func (c *K8sController) updateDeployment(ctx context.Context, existingDeployment *apps.Deployment, params DeploymentParams) error {
	// Update deployment spec
	existingDeployment.Spec.Replicas = &params.Replicas
	existingDeployment.Spec.Template.Spec.Containers[0].Image = params.Image

	// Update environment variables
	existingDeployment.Spec.Template.Spec.Containers[0].Env = []core.EnvVar{}
	for key, value := range params.EnvVars {
		existingDeployment.Spec.Template.Spec.Containers[0].Env = append(
			existingDeployment.Spec.Template.Spec.Containers[0].Env,
			core.EnvVar{Name: key, Value: value},
		)
	}

	// Update resource requirements
	memoryRequest, err := resource.ParseQuantity(params.MemoryRequest)
	if err != nil {
		return fmt.Errorf("invalid memory request: %v", err)
	}
	memoryLimit, err := resource.ParseQuantity(params.MemoryLimit)
	if err != nil {
		return fmt.Errorf("invalid memory limit: %v", err)
	}
	existingDeployment.Spec.Template.Spec.Containers[0].Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceMemory: memoryRequest,
		},
		Limits: core.ResourceList{
			core.ResourceMemory: memoryLimit,
		},
	}

	// Update the deployment
	_, err = c.clientset.AppsV1().Deployments(c.namespace).Update(ctx, existingDeployment, meta.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %v", err)
	}

	return nil
}

func inKubernetes() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

// getCurrentNamespace returns the current namespace of the running pod
func getCurrentNamespace() (string, error) {
	// Check if running in a Kubernetes cluster
	if _, exists := os.LookupEnv("KUBERNETES_SERVICE_HOST"); !exists {
		return "", fmt.Errorf("not running in a Kubernetes cluster")
	}

	// Read the namespace from the service account secret
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", fmt.Errorf("error reading namespace file: %v", err)
	}

	return strings.TrimSpace(string(data)), nil
}
