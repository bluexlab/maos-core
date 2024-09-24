package admin

import (
	"fmt"
	"regexp"
	"strconv"
)

var KubeConfigsWithDefault = map[string]string{
	"KUBE_DOCKER_IMAGE":   "",
	"KUBE_REPLICAS":       "1",
	"KUBE_CPU_REQUEST":    "500m",
	"KUBE_CPU_LIMIT":      "500m",
	"KUBE_MEMORY_REQUEST": "100Mi",
	"KUBE_MEMORY_LIMIT":   "100Mi",
}

func InsertMissingKubeConfigsWithDefault(content map[string]string) {
	for kubeConfig, defaultValue := range KubeConfigsWithDefault {
		if _, found := content[kubeConfig]; !found {
			content[kubeConfig] = defaultValue
		}
	}
}

func ValidateKubeConfig(content map[string]string) error {
	for kubeConfig, value := range content {
		switch kubeConfig {
		case "KUBE_DOCKER_IMAGE":
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
		case "KUBE_MEMORY_REQUEST", "KUBE_MEMORY_LIMIT":
			if value != "" && !isValidMemoryResourceQuantity(value) {
				return fmt.Errorf("invalid %s: %s", kubeConfig, value)
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
