package k8s

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/samber/lo"
	apps "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

// MigrationParams defines the parameters for a Kubernetes migration
type MigrationParams struct {
	Serial           int64             // Batch number of the migration
	Name             string            // Name of the migration
	Image            string            // Name with tag of the Docker image to use
	ImagePullSecrets string            // Name of the secret containing image pull credentials
	EnvVars          map[string]string // Environment variables to pass to the container for the migration
	Command          []string          // Command to run for the migration
	MemoryRequest    string            // Memory request for the container
	MemoryLimit      string            // Memory limit for the container
}

// DeploymentParams defines the parameters for a Kubernetes deployment
type DeploymentParams struct {
	Name             string            // Name of the deployment
	Replicas         int32             // Number of replicas to create
	Labels           map[string]string // Labels to apply to the deployment
	Image            string            // Name with tag of the Docker image to use
	ImagePullSecrets string            // Name of the secret containing image pull credentials
	LaunchCommand    []string          // Command to run the deployment
	EnvVars          map[string]string // Environment variables to pass to the container
	APIKey           string            // API key to use for the deployment
	MemoryRequest    string            // Memory request for the container
	MemoryLimit      string            // Memory limit for the container
	CPURequest       string            // CPU request for the container
	CPULimit         string            // CPU limit for the container
	Command          []string          // Command to run the deployment

	// Service-related fields
	HasService   bool    // Whether to create a service for the deployment
	ServicePorts []int32 // Ports for the service
	ServiceName  string  // Name of the service. Optional. If empty, a name will be generated.

	// Ingress-related fields
	HasIngress  bool   // Whether to create an ingress for the deployment
	IngressHost string // Host for the ingress
	BodyLimit   string // Body size limit for the ingress
}

type Secret struct {
	Name string
	Keys []string
}

// PodWithMetrics represents a Pod with its associated metrics
type PodWithMetrics struct {
	Pod     core.Pod
	Metrics *metricsv1beta1.PodMetrics
}

// Controller defines the methods that a Controller should implement
type Controller interface {
	UpdateDeploymentSet(ctx context.Context, deploymentSet []DeploymentParams) error
	TriggerRollingRestart(ctx context.Context, deploymentName string) error
	ListSecrets(ctx context.Context) ([]Secret, error)
	UpdateSecret(ctx context.Context, secretName string, secretData map[string]string) error
	DeleteSecret(ctx context.Context, secretName string) error
	ListRunningPodsWithMetrics(ctx context.Context) ([]PodWithMetrics, error)
	RunMigrations(ctx context.Context, migrations []MigrationParams) (map[string]interface{}, error)
}

// K8sController implements the Controller interface
type K8sController struct {
	clientset     kubernetes.Interface
	metricsClient metrics.Interface
	namespace     string
	config        *rest.Config
}

type resourceSet struct {
	deployments map[string]*apps.Deployment
	services    map[string]*core.Service
	ingresses   map[string]*networking.Ingress
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

	metricsClient, err := metrics.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating metrics client: %v", err)
	}

	namespace, err := getCurrentNamespace()
	if err != nil {
		return nil, fmt.Errorf("error getting current namespace: %v", err)
	}

	return &K8sController{
		clientset:     clientset,
		metricsClient: metricsClient,
		namespace:     namespace,
		config:        config,
	}, nil
}

// UpdateDeploymentSet updates the set of deployments
func (c *K8sController) UpdateDeploymentSet(ctx context.Context, deploymentSet []DeploymentParams) error {
	slog.Info("Updating deployment set", "deploymentSet", lo.Map(deploymentSet, func(d DeploymentParams, _ int) string { return d.Name }))

	existingResources, err := c.listExistingResources(ctx)
	if err != nil {
		return err
	}

	obsoleteResources, err := c.processDeploymentSet(ctx, deploymentSet, existingResources)
	if err != nil {
		return err
	}

	if err := c.deleteObsoleteResources(ctx, obsoleteResources); err != nil {
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

// ListRunningPodsWithMetrics lists all running pods with metrics
func (c *K8sController) ListRunningPodsWithMetrics(ctx context.Context) ([]PodWithMetrics, error) {
	podList, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}

	podMetrics, err := c.metricsClient.MetricsV1beta1().PodMetricses(c.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %v", err)
	}

	podsWithMetrics := make([]PodWithMetrics, 0, len(podList.Items))
	for _, pod := range podList.Items {
		if pod.Status.Phase == core.PodRunning {
			metrics, found := lo.Find(podMetrics.Items, func(m metricsv1beta1.PodMetrics) bool {
				return m.Name == pod.Name
			})

			if found {
				podsWithMetrics = append(podsWithMetrics, PodWithMetrics{
					Pod:     pod,
					Metrics: &metrics,
				})
			}
		}
	}

	return podsWithMetrics, nil
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

// UpdateSecret updates or creates a secret
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

// DeleteSecret deletes a secret
func (c *K8sController) DeleteSecret(ctx context.Context, secretName string) error {
	err := c.clientset.CoreV1().Secrets(c.namespace).Delete(ctx, secretName, meta.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete secret: %v", err)
	}
	return nil
}

func (c *K8sController) RunMigrations(ctx context.Context, migrations []MigrationParams) (map[string]interface{}, error) {
	slog.Info("Running migrations", "count", len(migrations))

	jobs := make(map[string]*batchv1.Job)
	for _, migration := range migrations {
		job, err := c.createMigrationJob(ctx, migration)
		if err != nil {
			return nil, fmt.Errorf("failed to create migration job: %v", err)
		}
		jobs[job.Name] = job
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	failures := make(map[string]interface{})
	var lastLogs map[string]interface{}

	for len(jobs) > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			for jobName := range jobs {
				slog.Info("Checking job status", "job", jobName)
				updatedJob, err := c.clientset.BatchV1().Jobs(c.namespace).Get(ctx, jobName, meta.GetOptions{})
				if err != nil {
					slog.Error("Failed to get job status", "job", jobName, "error", err)
					continue
				}

				statusJson, err := json.Marshal(updatedJob.Status)
				if err != nil {
					slog.Error("Failed to marshal job status", "job", jobName, "error", err)
				} else {
					slog.Info("Job status", "job", jobName, "status", string(statusJson))
				}

				logs, err := c.collectJobPodsLogs(ctx, jobName)
				if err != nil {
					slog.Error("Failed to collect job logs", "job", jobName, "error", err)
				} else {
					slog.Info("Job logs", "job", jobName, "logs", logs)
				}
				if len(logs) > 0 {
					lastLogs = logs
				}

				if updatedJob.Status.Succeeded > 0 {
					slog.Info("Migration job completed successfully", "job", jobName)
					delete(jobs, jobName)
				} else if updatedJob.Status.Failed > 0 {
					slog.Error("Migration job failed", "job", jobName)
					slog.Info("Job logs", "job", jobName, "logs", lastLogs)
					failures[jobName] = lastLogs
					delete(jobs, jobName)
				}
			}
		}
	}

	if len(failures) > 0 {
		return failures, fmt.Errorf("migration failed")
	}

	return nil, nil
}

func (c *K8sController) createMigrationJob(ctx context.Context, migration MigrationParams) (*batchv1.Job, error) {
	jobName := fmt.Sprintf("migration-%s-%d", migration.Name, migration.Serial)
	slog.Info("Creating migration job", "name", jobName)

	job := &batchv1.Job{
		ObjectMeta: meta.ObjectMeta{
			Name:      jobName,
			Namespace: c.namespace,
			Labels: map[string]string{
				"created-by": "maos",
				"type":       "migration",
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: lo.ToPtr(int32(600)),
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels: map[string]string{
						"created-by": "maos",
						"type":       "migration",
					},
				},
				Spec: core.PodSpec{
					RestartPolicy: core.RestartPolicyOnFailure,
					Containers: []core.Container{
						{
							Name:            "migration",
							Image:           migration.Image,
							ImagePullPolicy: core.PullAlways,
							Env:             c.createMigrationEnvVars(migration),
							Command:         migration.Command,
							Resources: core.ResourceRequirements{
								Requests: core.ResourceList{
									core.ResourceMemory: resource.MustParse(migration.MemoryRequest),
								},
								Limits: core.ResourceList{
									core.ResourceMemory: resource.MustParse(migration.MemoryLimit),
								},
							},
						},
					},
				},
			},
			BackoffLimit: lo.ToPtr(int32(3)),
		},
	}

	if migration.ImagePullSecrets != "" {
		job.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{
			{Name: migration.ImagePullSecrets},
		}
	}

	createdJob, err := c.clientset.BatchV1().Jobs(c.namespace).Create(ctx, job, meta.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migration job: %v", err)
	}

	return createdJob, nil
}

func (c *K8sController) createMigrationEnvVars(migration MigrationParams) []core.EnvVar {
	var envVars []core.EnvVar
	for key, value := range migration.EnvVars {
		if strings.HasPrefix(value, "[[SECRET]]") {
			secretName, secretKey := parseSecretValue(value, key)
			envVars = append(envVars, createSecretEnvVar(key, secretName, secretKey))
		} else {
			envVars = append(envVars, core.EnvVar{Name: key, Value: value})
		}
	}
	return envVars
}

func (c *K8sController) collectJobPodsLogs(ctx context.Context, jobName string) (map[string]interface{}, error) {
	pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, meta.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil {
		slog.Error("Failed to list pods for job", "job", jobName, "error", err)
		return nil, fmt.Errorf("failed to list pods for job %s: %v", jobName, err)
	}

	slog.Info("Found pods for job", "job", jobName, "pods", len(pods.Items))
	logs := make(map[string]interface{})

	for _, pod := range pods.Items {
		logs["Message"] = pod.Status.Message
		logs["Reason"] = pod.Status.Reason
		logs["ContainerStatuses"] = pod.Status.ContainerStatuses
		logs["Conditions"] = pod.Status.Conditions

		req := c.clientset.CoreV1().Pods(c.namespace).GetLogs(pod.Name, &core.PodLogOptions{})
		podLogs, err := req.Stream(ctx)
		if err != nil {
			slog.Error("Failed to get pod logs", "pod", pod.Name, "error", err)
			continue
		}
		defer podLogs.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		if err != nil {
			slog.Error("Failed to read pod logs", "pod", pod.Name, "error", err)
			continue
		}

		logs["PodLogs"] = buf.String()
	}

	return logs, nil
}

func (c *K8sController) listExistingResources(ctx context.Context) (*resourceSet, error) {
	deploymentList, err := c.clientset.AppsV1().Deployments(c.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list existing deployments: %v", err)
	}

	serviceList, err := c.clientset.CoreV1().Services(c.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list existing services: %v", err)
	}

	ingressList, err := c.clientset.NetworkingV1().Ingresses(c.namespace).List(ctx, meta.ListOptions{
		LabelSelector: "created-by=maos",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list existing ingresses: %v", err)
	}

	existingResources := &resourceSet{
		deployments: lo.SliceToMap(deploymentList.Items, func(deployment apps.Deployment) (string, *apps.Deployment) {
			return deployment.Name, &deployment
		}),
		services: lo.SliceToMap(serviceList.Items, func(service core.Service) (string, *core.Service) {
			return service.Name, &service
		}),
		ingresses: lo.SliceToMap(ingressList.Items, func(ingress networking.Ingress) (string, *networking.Ingress) {
			return ingress.Name, &ingress
		}),
	}

	return existingResources, nil
}

func (c *K8sController) processDeploymentSet(ctx context.Context, deploymentSet []DeploymentParams, existingResources *resourceSet) (*resourceSet, error) {
	// Clone existing resources using lo for shallow cloning
	clonedResources := resourceSet{
		deployments: lo.Assign(map[string]*apps.Deployment{}, existingResources.deployments),
		services:    lo.Assign(map[string]*core.Service{}, existingResources.services),
		ingresses:   lo.Assign(map[string]*networking.Ingress{}, existingResources.ingresses),
	}

	for _, params := range deploymentSet {
		// Update or create the deployment
		if existingDeployment, exists := existingResources.deployments[params.Name]; exists {
			if err := c.updateDeployment(ctx, existingDeployment, params); err != nil {
				return nil, fmt.Errorf("failed to update deployment %s: %v", params.Name, err)
			}
			delete(clonedResources.deployments, params.Name)
		} else {
			if _, err := c.createDeployment(ctx, params); err != nil {
				return nil, fmt.Errorf("failed to create deployment %s: %v", params.Name, err)
			}
		}

		// Update or create the service
		if params.HasService {
			serviceName := lo.Ternary(params.ServiceName != "", params.ServiceName, params.Name)
			if existingService, exists := existingResources.services[serviceName]; exists {
				if err := c.updateService(ctx, existingService, params); err != nil {
					return nil, fmt.Errorf("failed to update service %s: %v", params.Name, err)
				}
				delete(clonedResources.services, serviceName)
			} else {
				if err := c.createService(ctx, params); err != nil {
					return nil, fmt.Errorf("failed to create service %s: %v", params.Name, err)
				}
			}
		}

		if params.HasIngress {
			if existingIngress, exists := existingResources.ingresses[params.Name]; exists {
				if err := c.updateIngress(ctx, existingIngress, params); err != nil {
					return nil, fmt.Errorf("failed to update ingress %s: %v", params.Name, err)
				}
				delete(clonedResources.ingresses, params.Name)
			} else {
				if err := c.createIngress(ctx, params); err != nil {
					return nil, fmt.Errorf("failed to create ingress %s: %v", params.Name, err)
				}
			}
		}
	}
	return &clonedResources, nil
}

func (c *K8sController) deleteObsoleteResources(ctx context.Context, obsoleteResources *resourceSet) error {
	for name := range obsoleteResources.deployments {
		if err := c.deleteDeploymentResources(ctx, name); err != nil {
			slog.Error("Failed to delete deployment resources", "namespace", c.namespace, "name", name, "error", err)
		} else {
			slog.Info("Deleted deployment resources", "namespace", c.namespace, "name", name)
		}
	}

	for name, service := range obsoleteResources.services {
		slog.Info("Deleting obsolete service", "name", name)
		err := c.clientset.CoreV1().Services(c.namespace).Delete(ctx, service.Name, meta.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			slog.Error("Failed to delete service", "name", name, "error", err)
		}
	}

	for name, ingress := range obsoleteResources.ingresses {
		slog.Info("Deleting obsolete ingress", "name", name)
		err := c.clientset.NetworkingV1().Ingresses(c.namespace).Delete(ctx, ingress.Name, meta.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			slog.Error("Failed to delete ingress", "name", name, "error", err)
		}
	}

	return nil
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
	// Check if the service account already exists
	existingSA, err := c.clientset.CoreV1().ServiceAccounts(c.namespace).Get(ctx, name, meta.GetOptions{})
	if err == nil {
		// Service account already exists, return it
		return existingSA, nil
	}

	// If the error is not "NotFound", return the error
	if !errors.IsNotFound(err) {
		return nil, fmt.Errorf("error checking service account existence: %v", err)
	}

	// Service account doesn't exist, create it
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
	cpuRequest, _ := resource.ParseQuantity(params.CPURequest)
	cpuLimit, _ := resource.ParseQuantity(params.CPULimit)
	envVars := c.createEnvVars(params)

	if params.Labels == nil {
		slog.Error("Labels must not be nil", "name", params.Name)
		return nil
	}

	params.Labels["created-by"] = "maos"

	var imagePullSecrets []core.LocalObjectReference
	if params.ImagePullSecrets != "" {
		imagePullSecrets = []core.LocalObjectReference{
			{
				Name: params.ImagePullSecrets,
			},
		}
	}

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
			Strategy: apps.DeploymentStrategy{
				Type: apps.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &apps.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 2},
				},
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
							ImagePullPolicy: core.PullAlways,
							Env:             envVars,
							Command:         params.LaunchCommand,
							Resources: core.ResourceRequirements{
								Requests: core.ResourceList{
									core.ResourceMemory: memoryRequest,
									core.ResourceCPU:    cpuRequest,
								},
								Limits: core.ResourceList{
									core.ResourceMemory: memoryLimit,
									core.ResourceCPU:    cpuLimit,
								},
							},
						},
					},
					ImagePullSecrets: imagePullSecrets,
				},
			},
		},
	}
}

func (c *K8sController) createServiceStruct(params DeploymentParams) *core.Service {
	servicePorts := params.ServicePorts

	return &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:      lo.Ternary(params.ServiceName != "", params.ServiceName, params.Name),
			Namespace: c.namespace,
			Labels:    params.Labels,
		},
		Spec: core.ServiceSpec{
			Selector: params.Labels,
			Ports: lo.Map(servicePorts, func(port int32, _ int) core.ServicePort {
				return core.ServicePort{
					Name:       lo.Ternary(len(servicePorts) > 1, fmt.Sprintf("port-%d", port), ""),
					Port:       port,
					TargetPort: intstr.FromInt(int(port)),
					Protocol:   core.ProtocolTCP,
				}
			}),
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
												Number: params.ServicePorts[0],
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
	envVars = append(envVars, core.EnvVar{Name: "MAOS_CREATED_AT", Value: fmt.Sprintf("%d", time.Now().UnixMilli())})

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

// SecretData represents the structure of a secret with its data
type SecretData struct {
	Name string            `json:"name"`
	Data map[string]string `json:"data"`
}

// SerializeSecretsToJSON reads all secrets and serializes them to a JSON string
func (c *K8sController) SerializeSecretsToJSON(ctx context.Context, publicKey string) (string, error) {
	secretList, err := c.clientset.CoreV1().Secrets(c.namespace).List(ctx, meta.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list secrets: %v", err)
	}

	secretsData := make([]SecretData, 0, len(secretList.Items))
	for _, secret := range secretList.Items {
		data := make(map[string]string)
		for key, value := range secret.Data {
			data[key] = base64.StdEncoding.EncodeToString(value)
		}
		secretsData = append(secretsData, SecretData{
			Name: secret.Name,
			Data: data,
		})
	}

	jsonData, err := json.Marshal(secretsData)
	if err != nil {
		return "", fmt.Errorf("failed to serialize secrets to JSON: %v", err)
	}

	// Parse the RSA public key
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block containing the public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %v", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("not an RSA public key")
	}

	// Create a JWE encrypter
	encrypter, err := jose.NewEncrypter(
		jose.A256GCM,
		jose.Recipient{Algorithm: jose.RSA_OAEP_256, Key: rsaPub},
		(&jose.EncrypterOptions{}).WithType("JWE").WithContentType("application/json"),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create JWE encrypter: %v", err)
	}

	// Encrypt the JSON data
	jwe, err := encrypter.Encrypt(jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt data: %v", err)
	}

	// Serialize the JWE
	serialized, err := jwe.CompactSerialize()
	if err != nil {
		return "", fmt.Errorf("failed to serialize JWE: %v", err)
	}

	slog.Info("Serialized secrets", "data", serialized)
	return serialized, nil
}
