package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertMissingKubeConfigsWithDefault(t *testing.T) {
	t.Run("Insert missing configs for agent", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS": "2",
		}

		InsertMissingKubeConfigsWithDefault(content, "agent", false)

		assert.Equal(t, "2", content["KUBE_REPLICAS"])
		assert.Contains(t, content, "KUBE_DOCKER_IMAGE")
		assert.Equal(t, "10m", content["KUBE_CPU_REQUEST"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_REQUEST"])
		assert.Equal(t, "500m", content["KUBE_CPU_LIMIT"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_LIMIT"])
	})

	t.Run("Don't overwrite existing configs for agent", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS":    "3",
			"KUBE_CPU_REQUEST": "200m",
		}

		InsertMissingKubeConfigsWithDefault(content, "agent", false)

		assert.Equal(t, "3", content["KUBE_REPLICAS"])
		assert.Equal(t, "200m", content["KUBE_CPU_REQUEST"])
		assert.Contains(t, content, "KUBE_DOCKER_IMAGE")
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_REQUEST"])
		assert.Equal(t, "500m", content["KUBE_CPU_LIMIT"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_LIMIT"])
	})

	t.Run("Insert missing configs for portal", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS": "1",
		}

		InsertMissingKubeConfigsWithDefault(content, "portal", false)

		assert.Equal(t, "1", content["KUBE_REPLICAS"])
		assert.Contains(t, content, "KUBE_DOCKER_IMAGE")
		assert.Equal(t, "10m", content["KUBE_CPU_REQUEST"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_REQUEST"])
		assert.Equal(t, "500m", content["KUBE_CPU_LIMIT"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_LIMIT"])
		assert.Equal(t, "3000", content["KUBE_SERVICE_PORT"])
		assert.Equal(t, "11m", content["KUBE_INGRESS_BODY_LIMIT"])
		assert.Equal(t, "portal.example.com", content["KUBE_INGRESS_HOST"])
	})

	t.Run("Insert missing configs for service", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS": "3",
		}

		InsertMissingKubeConfigsWithDefault(content, "service", false)

		assert.Equal(t, "3", content["KUBE_REPLICAS"])
		assert.Contains(t, content, "KUBE_DOCKER_IMAGE")
		assert.Equal(t, "10m", content["KUBE_CPU_REQUEST"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_REQUEST"])
		assert.Equal(t, "500m", content["KUBE_CPU_LIMIT"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_LIMIT"])
		assert.Equal(t, "3000", content["KUBE_SERVICE_PORT"])
	})

	t.Run("Insert missing configs for migratable service", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS": "3",
		}

		InsertMissingKubeConfigsWithDefault(content, "service", true)

		assert.Equal(t, "3", content["KUBE_REPLICAS"])
		assert.Contains(t, content, "KUBE_DOCKER_IMAGE")
		assert.Equal(t, "10m", content["KUBE_CPU_REQUEST"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_REQUEST"])
		assert.Equal(t, "500m", content["KUBE_CPU_LIMIT"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_LIMIT"])
		assert.Equal(t, "3000", content["KUBE_SERVICE_PORT"])
		assert.Contains(t, content, "KUBE_MIGRATE_DOCKER_IMAGE")
		assert.Contains(t, content, "KUBE_MIGRATE_COMMAND")
		assert.Equal(t, "100Mi", content["KUBE_MIGRATE_MEMORY_REQUEST"])
		assert.Equal(t, "100Mi", content["KUBE_MIGRATE_MEMORY_LIMIT"])
	})
}

func TestValidateKubeConfig(t *testing.T) {
	t.Run("Valid config for agent", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS":       "2",
			"KUBE_DOCKER_IMAGE":   "myregistry.com/myimage:latest",
			"KUBE_CPU_REQUEST":    "10m",
			"KUBE_MEMORY_REQUEST": "256Mi",
			"KUBE_CPU_LIMIT":      "500m",
			"KUBE_MEMORY_LIMIT":   "512Mi",
		}

		err := ValidateKubeConfig(content, "agent", false)
		assert.NoError(t, err)
	})

	t.Run("Valid config for portal", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS":           "1",
			"KUBE_DOCKER_IMAGE":       "myregistry.com/portal:latest",
			"KUBE_CPU_REQUEST":        "1000m",
			"KUBE_MEMORY_REQUEST":     "200Mi",
			"KUBE_CPU_LIMIT":          "1000m",
			"KUBE_MEMORY_LIMIT":       "200Mi",
			"KUBE_INGRESS_HOST":       "portal.example.com",
			"KUBE_INGRESS_BODY_LIMIT": "11m",
		}

		err := ValidateKubeConfig(content, "portal", false)
		assert.NoError(t, err)
	})

	t.Run("Valid config for service", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS":       "3",
			"KUBE_DOCKER_IMAGE":   "myregistry.com/service:latest",
			"KUBE_CPU_REQUEST":    "250m",
			"KUBE_MEMORY_REQUEST": "50Mi",
			"KUBE_CPU_LIMIT":      "250m",
			"KUBE_MEMORY_LIMIT":   "50Mi",
			"KUBE_SERVICE_PORT":   "3000,3001",
			"KUBE_SERVICE_NAME":   "my-service",
		}

		err := ValidateKubeConfig(content, "service", false)
		assert.NoError(t, err)
	})

	t.Run("Invalid KUBE_REPLICAS", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS": "invalid",
		}

		err := ValidateKubeConfig(content, "agent", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid replicas")
	})

	t.Run("Invalid KUBE_DOCKER_IMAGE", func(t *testing.T) {
		content := map[string]string{
			"KUBE_DOCKER_IMAGE": "invalid image",
		}

		err := ValidateKubeConfig(content, "agent", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid docker image")
	})

	t.Run("Invalid KUBE_CPU_REQUEST", func(t *testing.T) {
		content := map[string]string{
			"KUBE_CPU_REQUEST": "invalid",
		}

		err := ValidateKubeConfig(content, "agent", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid KUBE_CPU_REQUEST")
	})

	t.Run("Invalid KUBE_MEMORY_REQUEST", func(t *testing.T) {
		content := map[string]string{
			"KUBE_MEMORY_REQUEST": "invalid",
		}

		err := ValidateKubeConfig(content, "agent", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid KUBE_MEMORY_REQUEST")
	})

	t.Run("Missing required configs", func(t *testing.T) {
		content := map[string]string{
			"KUBE_DOCKER_IMAGE":   "",
			"KUBE_CPU_REQUEST":    "",
			"KUBE_MEMORY_REQUEST": "",
			"KUBE_CPU_LIMIT":      "",
			"KUBE_MEMORY_LIMIT":   "",
		}

		err := ValidateKubeConfig(content, "agent", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required config")
	})

	t.Run("Valid config for migratable service", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS":               "3",
			"KUBE_DOCKER_IMAGE":           "myregistry.com/service:latest",
			"KUBE_CPU_REQUEST":            "250m",
			"KUBE_MEMORY_REQUEST":         "50Mi",
			"KUBE_CPU_LIMIT":              "250m",
			"KUBE_MEMORY_LIMIT":           "50Mi",
			"KUBE_SERVICE_PORT":           "3000",
			"KUBE_MIGRATE_DOCKER_IMAGE":   "myregistry.com/migrate:latest",
			"KUBE_MIGRATE_COMMAND":        "migrate.sh",
			"KUBE_MIGRATE_MEMORY_REQUEST": "100Mi",
			"KUBE_MIGRATE_MEMORY_LIMIT":   "100Mi",
		}

		err := ValidateKubeConfig(content, "service", true)
		assert.NoError(t, err)
	})

	t.Run("Invalid migratable config - missing KUBE_MIGRATE_DOCKER_IMAGE", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS":       "3",
			"KUBE_DOCKER_IMAGE":   "myregistry.com/service:latest",
			"KUBE_CPU_REQUEST":    "250m",
			"KUBE_MEMORY_REQUEST": "50Mi",
			"KUBE_CPU_LIMIT":      "250m",
			"KUBE_MEMORY_LIMIT":   "50Mi",
			"KUBE_SERVICE_PORT":   "3000",
			// "KUBE_MIGRATE_DOCKER_IMAGE" is missing
			"KUBE_MIGRATE_COMMAND":        "migrate.sh",
			"KUBE_MIGRATE_MEMORY_REQUEST": "100Mi",
			"KUBE_MIGRATE_MEMORY_LIMIT":   "100Mi",
		}

		err := ValidateKubeConfig(content, "service", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required config: KUBE_MIGRATE_DOCKER_IMAGE")
	})

	t.Run("Invalid migratable config - invalid KUBE_MIGRATE_MEMORY_REQUEST", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS":               "3",
			"KUBE_DOCKER_IMAGE":           "myregistry.com/service:latest",
			"KUBE_CPU_REQUEST":            "250m",
			"KUBE_MEMORY_REQUEST":         "50Mi",
			"KUBE_CPU_LIMIT":              "250m",
			"KUBE_MEMORY_LIMIT":           "50Mi",
			"KUBE_SERVICE_PORT":           "3000",
			"KUBE_MIGRATE_DOCKER_IMAGE":   "myregistry.com/migrate:latest",
			"KUBE_MIGRATE_COMMAND":        "migrate.sh",
			"KUBE_MIGRATE_MEMORY_REQUEST": "invalid",
			"KUBE_MIGRATE_MEMORY_LIMIT":   "100Mi",
		}

		err := ValidateKubeConfig(content, "service", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid KUBE_MIGRATE_MEMORY_REQUEST")
	})
}
