package admin

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var KubeConfigsWithDefault = map[string]string{
	"KUBE_DOCKER_IMAGE":      "",
	"KUBE_IMAGE_PULL_SECRET": "",
	"KUBE_REPLICAS":          "1",
	"KUBE_CPU_REQUEST":       "10m",
	"KUBE_CPU_LIMIT":         "500m",
	"KUBE_MEMORY_REQUEST":    "100Mi",
	"KUBE_MEMORY_LIMIT":      "100Mi",
}

var KubeConfigsWithDefaultForService = map[string]string{
	"KUBE_DOCKER_IMAGE":      "",
	"KUBE_IMAGE_PULL_SECRET": "",
	"KUBE_REPLICAS":          "1",
	"KUBE_CPU_REQUEST":       "10m",
	"KUBE_CPU_LIMIT":         "500m",
	"KUBE_MEMORY_REQUEST":    "100Mi",
	"KUBE_MEMORY_LIMIT":      "100Mi",
	"KUBE_SERVICE_PORT":      "3000",
	"KUBE_SERVICE_NAME":      "",
}

var KubeConfigsWithDefaultForPortal = map[string]string{
	"KUBE_DOCKER_IMAGE":       "",
	"KUBE_IMAGE_PULL_SECRET":  "",
	"KUBE_REPLICAS":           "1",
	"KUBE_CPU_REQUEST":        "10m",
	"KUBE_CPU_LIMIT":          "500m",
	"KUBE_MEMORY_REQUEST":     "100Mi",
	"KUBE_MEMORY_LIMIT":       "100Mi",
	"KUBE_SERVICE_PORT":       "3000",
	"KUBE_SERVICE_NAME":       "",
	"KUBE_INGRESS_HOST":       "portal.example.com",
	"KUBE_INGRESS_BODY_LIMIT": "11m",
}

var KubeConfigsWithDefaultForMigratable = map[string]string{
	"KUBE_MIGRATE_DOCKER_IMAGE":      "",
	"KUBE_MIGRATE_IMAGE_PULL_SECRET": "",
	"KUBE_MIGRATE_COMMAND":           "",
	"KUBE_MIGRATE_MEMORY_REQUEST":    "100Mi",
	"KUBE_MIGRATE_MEMORY_LIMIT":      "100Mi",
}

func InsertMissingKubeConfigsWithDefault(content map[string]string, role string, migratable bool) {
	defaultConfig := KubeConfigsWithDefault
	switch role {
	case "portal":
		defaultConfig = KubeConfigsWithDefaultForPortal
	case "service":
		defaultConfig = KubeConfigsWithDefaultForService
	}

	for kubeConfig, defaultValue := range defaultConfig {
		if _, found := content[kubeConfig]; !found {
			content[kubeConfig] = defaultValue
		}
	}
	if migratable {
		for kubeConfig, defaultValue := range KubeConfigsWithDefaultForMigratable {
			if _, found := content[kubeConfig]; !found {
				content[kubeConfig] = defaultValue
			}
		}
	}
}

func ValidateKubeConfig(content map[string]string, role string, migratable bool) error {
	for kubeConfig, value := range content {
		switch kubeConfig {
		case "KUBE_DOCKER_IMAGE", "KUBE_MIGRATE_DOCKER_IMAGE":
			if value != "" && !isValidDockerImage(value) {
				return fmt.Errorf("invalid docker image: %s", value)
			}
		case "KUBE_REPLICAS":
			replicas, err := strconv.Atoi(value)
			if err != nil || replicas < 1 {
				return fmt.Errorf("invalid replicas: %s, must be a number >= 1", value)
			}
		case "KUBE_CPU_REQUEST", "KUBE_CPU_LIMIT":
			if value != "" && !isValidCPUResourceQuantity(value) {
				return fmt.Errorf("invalid %s: %s", kubeConfig, value)
			}
		case "KUBE_MEMORY_REQUEST", "KUBE_MEMORY_LIMIT", "KUBE_MIGRATE_MEMORY_REQUEST", "KUBE_MIGRATE_MEMORY_LIMIT":
			if value != "" && !isValidMemoryResourceQuantity(value) {
				return fmt.Errorf("invalid %s: %s", kubeConfig, value)
			}
		case "KUBE_SERVICE_PORT":
			if role == "portal" || role == "service" {
				ports := strings.Split(value, ",")
				for _, portStr := range ports {
					port, err := strconv.Atoi(strings.TrimSpace(portStr))
					if err != nil || port < 1 || port > 65535 {
						return fmt.Errorf("invalid service port: %s, must be a number between 1 and 65535", portStr)
					}
				}
			}
		case "KUBE_INGRESS_HOST":
			if role == "portal" || role == "service" {
				if value == "" {
					return fmt.Errorf("KUBE_INGRESS_HOST is required for portal and service")
				}
				// Basic domain name validation
				if !regexp.MustCompile(`^([a-zA-Z0-9]+(-[a-zA-Z0-9]+)*\.)+[a-zA-Z]{2,}$`).MatchString(value) {
					return fmt.Errorf("invalid ingress host: %s", value)
				}
			}
		case "KUBE_INGRESS_BODY_LIMIT":
			if role == "portal" || role == "service" {
				if value == "" {
					return fmt.Errorf("KUBE_INGRESS_BODY_LIMIT is required for portal and service")
				}
				// Validate body limit format (e.g., "10m", "1G")
				if !regexp.MustCompile(`^[0-9]+[kKmMgG]$`).MatchString(value) {
					return fmt.Errorf("invalid ingress body limit: %s, must be in format like '10m' or '1G'", value)
				}
			}
		}

		requiredConfigs := []string{"KUBE_DOCKER_IMAGE", "KUBE_MEMORY_REQUEST", "KUBE_MEMORY_LIMIT", "KUBE_CPU_REQUEST", "KUBE_CPU_LIMIT"}
		if role == "service" {
			requiredConfigs = append(requiredConfigs, "KUBE_SERVICE_PORT")
		}
		if role == "portal" {
			requiredConfigs = append(requiredConfigs, "KUBE_INGRESS_HOST", "KUBE_INGRESS_BODY_LIMIT")
		}
		if migratable {
			requiredConfigs = append(requiredConfigs, "KUBE_MIGRATE_DOCKER_IMAGE", "KUBE_MIGRATE_MEMORY_REQUEST", "KUBE_MIGRATE_MEMORY_LIMIT")
		}

		for _, requiredConfig := range requiredConfigs {
			if content[requiredConfig] == "" {
				return fmt.Errorf("missing required config: %s", requiredConfig)
			}
		}
	}
	return nil
}

func isValidDockerImage(image string) bool {
	// Basic Docker image validation regex
	// This regex allows for:
	// - Optional registry domain with port
	// - Image name
	// - Optional tag or digest
	regex := `^((?:[a-zA-Z0-9\-\.]+(?::[0-9]+)?/)?)([a-zA-Z0-9\._\-]+)(?::([\w\.\-]+))?(?:@sha256:[a-fA-F0-9]{64})?$`
	return regexp.MustCompile(regex).MatchString(image)
}

func isValidCPUResourceQuantity(quantity string) bool {
	// Regex for CPU resource quantity
	// Allows for:
	// - Numbers with optional decimal point
	// - Optional suffixes: m (millicpu)
	regex := `^([0-9]+(\.[0-9]+)?)(m)?$`
	return regexp.MustCompile(regex).MatchString(quantity)
}

func isValidMemoryResourceQuantity(quantity string) bool {
	// Regex for Memory resource quantity
	// Allows for:
	// - Numbers with optional decimal point
	// - Optional suffixes: Ki, Mi, Gi, Ti, Pi, Ei
	regex := `^([0-9]+(\.[0-9]+)?)(([KMGTPE]i)|[kMGTP])?$`
	return regexp.MustCompile(regex).MatchString(quantity)
}
