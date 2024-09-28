package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/samber/lo"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// DeploymentParams defines the parameters for a deployment
type DeploymentParams struct {
	Name          string
	Replicas      int32
	Labels        map[string]string
	Image         string
	EnvVars       map[string]string
	APIKey        string
	MemoryRequest string
	MemoryLimit   string
	HasService    bool
	ServicePort   int32
	HasIngress    bool
	IngressHost   string
	BodyLimit     string
}

type Secret struct {
	Name string
	Keys []string
}

// Controller defines the methods that a Controller should implement
type Controller interface {
	UpdateDeploymentSet(ctx context.Context, deploymentSet []DeploymentParams) error
	TriggerRollingRestart(ctx context.Context, deploymentName string) error
	ListSecrets(ctx context.Context) ([]Secret, error)
	UpdateSecret(ctx context.Context, secretName string, secretData map[string]string) error
	DeleteSecret(ctx context.Context, secretName string) error
}

// K8sController implements the Controller interface
type K8sController struct {
	clientset kubernetes.Interface
	namespace string
}

// NewK8sController creates a new Controller with a kubernetes clientset
func NewK8sController() (*K8sController, error) {
	config, err := getKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting kubernetes config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %v", err)
	}

	namespace, err := getCurrentNamespace()
	if err != nil {
		return nil, fmt.Errorf("error getting current namespace: %v", err)
	}

	return &K8sController{
		clientset: clientset,
		namespace: namespace,
	}, nil
}

// UpdateDeploymentSet updates the set of deployments
func (c *K8sController) UpdateDeploymentSet(ctx context.Context, deploymentSet []DeploymentParams) error {
	slog.Info("Updating deployment set", "deploymentSet", lo.Map(deploymentSet, func(d DeploymentParams, _ int) string { return d.Name }))

	existingDeployments, err := c.listExistingDeployments(ctx)
	if err != nil {
		return err
	}

	existingServices, err := c.listExistingServices(ctx)
	if err != nil {
		return err
	}

	existingIngresses, err := c.listExistingIngress(ctx)
	if err != nil {
		return err
	}

	for _, params := range deploymentSet {
		if err := c.processDeployment(ctx, params, existingDeployments); err != nil {
			return err
		}

		if err := c.processService(ctx, params, existingServices); err != nil {
			return err
		}

		if err := c.processIngress(ctx, params, existingIngresses); err != nil {
			return err
		}
	}

	if err := c.deleteObsoleteResources(ctx, existingDeployments, existingServices, existingIngresses); err != nil {
		return err
	}

	return nil
}

func (c *K8sController) processDeployment(ctx context.Context, params DeploymentParams, existingDeployments map[string]*apps.Deployment) error {
	if existingDeployment, exists := existingDeployments[params.Name]; exists {
		err := c.updateDeployment(ctx, existingDeployment, params)
		if err != nil {
			return fmt.Errorf("failed to update deployment %s: %v", params.Name, err)
		}
		delete(existingDeployments, params.Name)
	} else {
		_, err := c.createDeployment(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to create deployment %s: %v", params.Name, err)
		}
	}
	return nil
}

func (c *K8sController) processService(ctx context.Context, params DeploymentParams, existingServices map[string]*core.Service) error {
	if params.HasService {
		if existingService, exists := existingServices[params.Name]; exists {
			err := c.updateService(ctx, existingService, params)
			if err != nil {
				return fmt.Errorf("failed to update service %s: %v", params.Name, err)
			}
			delete(existingServices, params.Name)
		} else {
			err := c.createService(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to create service %s: %v", params.Name, err)
			}
		}
	}
	return nil
}

func (c *K8sController) processIngress(ctx context.Context, params DeploymentParams, existingIngresses map[string]*networking.Ingress) error {
	if params.HasIngress {
		if existingIngress, exists := existingIngresses[params.Name]; exists {
			err := c.updateIngress(ctx, existingIngress, params)
			if err != nil {
				return fmt.Errorf("failed to update ingress %s: %v", params.Name, err)
			}
			delete(existingIngresses, params.Name)
		} else {
			err := c.createIngress(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to create ingress %s: %v", params.Name, err)
			}
		}
	}
	return nil
}

func (c *K8sController) deleteObsoleteResources(ctx context.Context, existingDeployments map[string]*apps.Deployment, existingServices map[string]*core.Service, existingIngresses map[string]*networking.Ingress) error {
	if err := c.deleteObsoleteDeployments(ctx, existingDeployments); err != nil {
		return err
	}
	if err := c.deleteObsoleteServices(ctx, existingServices); err != nil {
		return err
	}
	if err := c.deleteObsoleteIngresses(ctx, existingIngresses); err != nil {
		return err
	}
	return nil
}

// TriggerRollingRestart triggers a rolling restart of the deployment
func (c *K8sController) TriggerRollingRestart(ctx context.Context, deploymentName string) error {
	slog.Info("Triggered rolling restart", "namespace", c.namespace, "deployment", deploymentName)
	deployment, err := c.clientset.AppsV1().Deployments(c.namespace).Get(ctx, deploymentName, meta.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %v", err)
	}

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

// ListSecrets lists all secrets in the namespace that are created by our application
func (c *K8sController) ListSecrets(ctx context.Context) ([]Secret, error) {
	secretList, err := c.clientset.CoreV1().Secrets(c.namespace).List(ctx, meta.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %v", err)
	}

	secrets := make([]Secret, 0, len(secretList.Items))
	for _, secret := range secretList.Items {
		keys := make([]string, 0, len(secret.Data))
		for k := range secret.Data {
			keys = append(keys, k)
		}
		secrets = append(secrets, Secret{
			Name: secret.Name,
			Keys: keys,
		})
	}

	return secrets, nil
}

func (c *K8sController) UpdateSecret(ctx context.Context, secretName string, secretData map[string]string) error {
	secret, err := c.clientset.CoreV1().Secrets(c.namespace).Get(ctx, secretName, meta.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return c.createSecret(ctx, secretName, secretData)
		}
		return fmt.Errorf("failed to get secret: %v", err)
	}

	return c.updateExistingSecret(ctx, secret, secretData)
}

func (c *K8sController) createSecret(ctx context.Context, secretName string, secretData map[string]string) error {
	slog.Info("Secret not found, creating", "name", secretName)
	newSecret := &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      secretName,
			Namespace: c.namespace,
			Labels: map[string]string{
				"created-by": "maos",
			},
		},
		StringData: secretData,
		Type:       core.SecretTypeOpaque,
	}
	_, err := c.clientset.CoreV1().Secrets(c.namespace).Create(ctx, newSecret, meta.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secret: %v", err)
	}
	return nil
}

func (c *K8sController) updateExistingSecret(ctx context.Context, secret *core.Secret, secretData map[string]string) error {
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	for k, v := range secretData {
		if v == "" {
			delete(secret.Data, k)
		} else {
			secret.Data[k] = []byte(v)
		}
	}

	_, err := c.clientset.CoreV1().Secrets(c.namespace).Update(ctx, secret, meta.UpdateOptions{
		FieldValidation: "Strict",
	})
	if err != nil {
		return fmt.Errorf("failed to update secret: %v", err)
	}

	return nil
}

func (c *K8sController) DeleteSecret(ctx context.Context, secretName string) error {
	err := c.clientset.CoreV1().Secrets(c.namespace).Delete(ctx, secretName, meta.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete secret: %v", err)
	}
	return nil
}

func (c *K8sController) listExistingDeployments(ctx context.Context) (map[string]*apps.Deployment, error) {
	deploymentList, err := c.clientset.AppsV1().Deployments(c.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list existing deployments: %v", err)
	}

	existingDeployments := make(map[string]*apps.Deployment)
	for i := range deploymentList.Items {
		deployment := &deploymentList.Items[i]
		existingDeployments[deployment.Name] = deployment
	}
	return existingDeployments, nil
}

func (c *K8sController) listExistingServices(ctx context.Context) (map[string]*core.Service, error) {
	serviceList, err := c.clientset.CoreV1().Services(c.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list existing services: %v", err)
	}

	existingServices := make(map[string]*core.Service)
	for i := range serviceList.Items {
		service := &serviceList.Items[i]
		existingServices[service.Name] = service
	}
	return existingServices, nil
}

func (c *K8sController) listExistingIngress(ctx context.Context) (map[string]*networking.Ingress, error) {
	ingressList, err := c.clientset.NetworkingV1().Ingresses(c.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list existing ingresses: %v", err)
	}

	existingIngresses := make(map[string]*networking.Ingress)
	for i := range ingressList.Items {
		ingress := &ingressList.Items[i]
		existingIngresses[ingress.Name] = ingress
	}
	return existingIngresses, nil
}

func (c *K8sController) createDeployment(ctx context.Context, params DeploymentParams) (*apps.Deployment, error) {
	slog.Info("Creating deployment", "name", params.Name)

	if err := c.checkDeploymentExists(ctx, params.Name); err != nil {
		return nil, err
	}

	sa, err := c.createServiceAccount(ctx, params.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create service account: %v", err)
	}

	if err := c.createOrUpdateApiKey(ctx, params); err != nil {
		return nil, err
	}

	deployment := c.createDeploymentStruct(params, sa)
	return c.clientset.AppsV1().Deployments(c.namespace).Create(ctx, deployment, meta.CreateOptions{})
}

func (c *K8sController) updateDeployment(ctx context.Context, existingDeployment *apps.Deployment, params DeploymentParams) error {
	slog.Info("Updating deployment", "name", existingDeployment.Name)
	if err := c.createOrUpdateApiKey(ctx, params); err != nil {
		return err
	}

	sa, err := c.clientset.CoreV1().ServiceAccounts(c.namespace).Get(ctx, existingDeployment.Spec.Template.Spec.ServiceAccountName, meta.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get service account: %v", err)
	}

	newDeployment := c.createDeploymentStruct(params, sa)
	existingDeployment.Spec = newDeployment.Spec
	existingDeployment.ObjectMeta.Labels = newDeployment.ObjectMeta.Labels

	_, err = c.clientset.AppsV1().Deployments(c.namespace).Update(ctx, existingDeployment, meta.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %v", err)
	}

	return nil
}

func (c *K8sController) deleteObsoleteDeployments(ctx context.Context, obsoleteDeployments map[string]*apps.Deployment) error {
	for name := range obsoleteDeployments {
		if err := c.deleteDeploymentResources(ctx, name); err != nil {
			slog.Error("Failed to delete deployment resources", "namespace", c.namespace, "name", name, "error", err)
		} else {
			slog.Info("Deleted deployment resources", "namespace", c.namespace, "name", name)
		}
	}
	return nil
}

func (c *K8sController) deleteDeploymentResources(ctx context.Context, name string) error {
	if err := c.clientset.AppsV1().Deployments(c.namespace).Delete(ctx, name, meta.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete deployment: %v", err)
	}

	secretName := fmt.Sprintf("%s-api-key", name)
	if err := c.clientset.CoreV1().Secrets(c.namespace).Delete(ctx, secretName, meta.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete secret: %v", err)
	}

	if err := c.clientset.CoreV1().ServiceAccounts(c.namespace).Delete(ctx, name, meta.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete service account: %v", err)
	}

	return nil
}

func (c *K8sController) checkDeploymentExists(ctx context.Context, name string) error {
	_, err := c.clientset.AppsV1().Deployments(c.namespace).Get(ctx, name, meta.GetOptions{})
	if err == nil {
		return fmt.Errorf("deployment %s already exists in namespace %s", name, c.namespace)
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("error checking deployment existence: %v", err)
	}
	return nil
}

func (c *K8sController) createServiceAccount(ctx context.Context, name string) (*core.ServiceAccount, error) {
	sa := &core.ServiceAccount{
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: c.namespace,
		},
	}
	return c.clientset.CoreV1().ServiceAccounts(c.namespace).Create(ctx, sa, meta.CreateOptions{})
}

func (c *K8sController) createOrUpdateApiKey(ctx context.Context, params DeploymentParams) error {
	slog.Info("Creating/updating secret", "name", params.Name)

	secretName := fmt.Sprintf("%s-api-key", params.Name)
	secret := &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      secretName,
			Namespace: c.namespace,
			Labels: map[string]string{
				"created-by": "maos-internal",
			},
		},
		StringData: map[string]string{
			"MAOS_API_KEY": params.APIKey,
		},
		Type: core.SecretTypeOpaque,
	}

	_, err := c.clientset.CoreV1().Secrets(c.namespace).Get(ctx, secretName, meta.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = c.clientset.CoreV1().Secrets(c.namespace).Create(ctx, secret, meta.CreateOptions{})
		}
	} else {
		_, err = c.clientset.CoreV1().Secrets(c.namespace).Update(ctx, secret, meta.UpdateOptions{})
	}

	if err != nil {
		return fmt.Errorf("failed to create/update secret: %v", err)
	}
	return nil
}

func (c *K8sController) createDeploymentStruct(params DeploymentParams, sa *core.ServiceAccount) *apps.Deployment {
	memoryRequest, _ := resource.ParseQuantity(params.MemoryRequest)
	memoryLimit, _ := resource.ParseQuantity(params.MemoryLimit)

	envVars := c.createEnvVars(params)

	if params.Labels == nil {
		slog.Error("Labels must not be nil", "name", params.Name)
		return nil
	}

	params.Labels["created-by"] = "maos"

	return &apps.Deployment{
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
}

func (c *K8sController) createServiceStruct(params DeploymentParams) *core.Service {
	serviceName := params.Name
	servicePort := params.ServicePort

	return &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:      serviceName,
			Namespace: c.namespace,
			Labels:    params.Labels,
		},
		Spec: core.ServiceSpec{
			Selector: params.Labels,
			Ports: []core.ServicePort{
				{
					Port:       servicePort,
					TargetPort: intstr.FromInt(int(servicePort)),
					Protocol:   core.ProtocolTCP,
				},
			},
			Type: core.ServiceTypeClusterIP,
		},
	}
}

func (c *K8sController) createIngressStruct(params DeploymentParams) *networking.Ingress {
	ingressName := params.Name

	return &networking.Ingress{
		ObjectMeta: meta.ObjectMeta{
			Name:      ingressName,
			Namespace: c.namespace,
			Labels:    params.Labels,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/proxy-body-size": params.BodyLimit,
			},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: params.IngressHost,
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path:     "/",
									PathType: lo.ToPtr(networking.PathTypePrefix),
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: params.Name,
											Port: networking.ServiceBackendPort{
												Number: params.ServicePort,
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
}

func (c *K8sController) createEnvVars(params DeploymentParams) []core.EnvVar {
	var envVars []core.EnvVar
	for key, value := range params.EnvVars {
		if strings.HasPrefix(value, "[[SECRET]]") {
			secretName, secretKey := parseSecretValue(value, key)
			envVars = append(envVars, createSecretEnvVar(key, secretName, secretKey))
		} else {
			envVars = append(envVars, core.EnvVar{Name: key, Value: value})
		}
	}

	secretName := fmt.Sprintf("%s-api-key", params.Name)
	envVars = append(envVars, createSecretEnvVar("MAOS_API_KEY", secretName, "MAOS_API_KEY"))

	return envVars
}

func (c *K8sController) updateService(ctx context.Context, existingService *core.Service, params DeploymentParams) error {
	slog.Info("Updating service", "name", existingService.Name)

	newService := c.createServiceStruct(params)
	existingService.Spec = newService.Spec
	existingService.ObjectMeta.Labels = newService.ObjectMeta.Labels

	_, err := c.clientset.CoreV1().Services(c.namespace).Update(ctx, existingService, meta.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update service: %v", err)
	}

	return nil
}

func (c *K8sController) updateIngress(ctx context.Context, existingIngress *networking.Ingress, params DeploymentParams) error {
	slog.Info("Updating ingress", "name", existingIngress.Name)

	newIngress := c.createIngressStruct(params)
	existingIngress.Spec = newIngress.Spec
	existingIngress.ObjectMeta.Labels = newIngress.ObjectMeta.Labels
	existingIngress.ObjectMeta.Annotations = newIngress.ObjectMeta.Annotations

	_, err := c.clientset.NetworkingV1().Ingresses(c.namespace).Update(ctx, existingIngress, meta.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ingress: %v", err)
	}

	return nil
}

func (c *K8sController) createService(ctx context.Context, params DeploymentParams) error {
	slog.Info("Creating service", "name", params.Name)

	service := c.createServiceStruct(params)
	_, err := c.clientset.CoreV1().Services(c.namespace).Create(ctx, service, meta.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}

	return nil
}

func (c *K8sController) createIngress(ctx context.Context, params DeploymentParams) error {
	slog.Info("Creating ingress", "name", params.Name)

	ingress := c.createIngressStruct(params)
	_, err := c.clientset.NetworkingV1().Ingresses(c.namespace).Create(ctx, ingress, meta.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ingress: %v", err)
	}

	return nil
}

func (c *K8sController) deleteObsoleteServices(ctx context.Context, obsoleteServices map[string]*core.Service) error {
	for name, service := range obsoleteServices {
		slog.Info("Deleting obsolete service", "name", name)
		err := c.clientset.CoreV1().Services(c.namespace).Delete(ctx, service.Name, meta.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			slog.Error("Failed to delete service", "name", name, "error", err)
		}
	}
	return nil
}

func (c *K8sController) deleteObsoleteIngresses(ctx context.Context, obsoleteIngresses map[string]*networking.Ingress) error {
	for name, ingress := range obsoleteIngresses {
		slog.Info("Deleting obsolete ingress", "name", name)
		err := c.clientset.NetworkingV1().Ingresses(c.namespace).Delete(ctx, ingress.Name, meta.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			slog.Error("Failed to delete ingress", "name", name, "error", err)
		}
	}
	return nil
}

func parseSecretValue(value, defaultKey string) (string, string) {
	secretName := strings.TrimPrefix(value, "[[SECRET]]")
	secretKey := defaultKey
	if strings.Contains(secretName, ":") {
		parts := strings.SplitN(secretName, ":", 2)
		secretName = parts[0]
		secretKey = parts[1]
	}
	return secretName, secretKey
}

func createSecretEnvVar(envName, secretName, secretKey string) core.EnvVar {
	return core.EnvVar{
		Name: envName,
		ValueFrom: &core.EnvVarSource{
			SecretKeyRef: &core.SecretKeySelector{
				LocalObjectReference: core.LocalObjectReference{
					Name: secretName,
				},
				Key: secretKey,
			},
		},
	}
}

func getKubernetesConfig() (*rest.Config, error) {
	if inKubernetes() {
		return rest.InClusterConfig()
	}

	kubeconfigPath := os.Getenv("KUBECONFIG_PATH")
	if kubeconfigPath == "" {
		return nil, fmt.Errorf("KUBECONFIG_PATH environment variable not set")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

func inKubernetes() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

func getCurrentNamespace() (string, error) {
	if !inKubernetes() {
		return "default", nil
	}

	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", fmt.Errorf("error reading namespace file: %v", err)
	}

	return strings.TrimSpace(string(data)), nil
}
