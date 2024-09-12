package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertMissingKubeConfigsWithDefault(t *testing.T) {
	t.Run("Insert missing configs", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS": "2",
		}

		InsertMissingKubeConfigsWithDefault(content)

		assert.Equal(t, "2", content["KUBE_REPLICAS"])
		assert.Equal(t, "", content["KUBE_DOCKER_IMAGE"])
		assert.Equal(t, "500m", content["KUBE_CPU_REQUEST"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_REQUEST"])
		assert.Equal(t, "500m", content["KUBE_CPU_LIMIT"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_LIMIT"])
	})

	t.Run("Don't overwrite existing configs", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS":    "3",
			"KUBE_CPU_REQUEST": "200m",
		}

		InsertMissingKubeConfigsWithDefault(content)

		assert.Equal(t, "3", content["KUBE_REPLICAS"])
		assert.Equal(t, "200m", content["KUBE_CPU_REQUEST"])
		assert.Equal(t, "", content["KUBE_DOCKER_IMAGE"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_REQUEST"])
		assert.Equal(t, "500m", content["KUBE_CPU_LIMIT"])
		assert.Equal(t, "100Mi", content["KUBE_MEMORY_LIMIT"])
	})
}

func TestValidateKubeConfig(t *testing.T) {
	t.Run("Valid config", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS":       "2",
			"KUBE_DOCKER_IMAGE":   "myregistry.com/myimage:latest",
			"KUBE_CPU_REQUEST":    "200m",
			"KUBE_MEMORY_REQUEST": "256Mi",
			"KUBE_CPU_LIMIT":      "500m",
			"KUBE_MEMORY_LIMIT":   "512Mi",
		}

		err := ValidateKubeConfig(content)
		assert.NoError(t, err)
	})

	t.Run("Invalid KUBE_REPLICAS", func(t *testing.T) {
		content := map[string]string{
			"KUBE_REPLICAS": "invalid",
		}

		err := ValidateKubeConfig(content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid replicas")
	})

	t.Run("Invalid KUBE_DOCKER_IMAGE", func(t *testing.T) {
		content := map[string]string{
			"KUBE_DOCKER_IMAGE": "invalid image",
		}

		err := ValidateKubeConfig(content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid docker image")
	})

	t.Run("Invalid KUBE_CPU_REQUEST", func(t *testing.T) {
		content := map[string]string{
			"KUBE_CPU_REQUEST": "invalid",
		}

		err := ValidateKubeConfig(content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid KUBE_CPU_REQUEST")
	})

	t.Run("Invalid KUBE_MEMORY_REQUEST", func(t *testing.T) {
		content := map[string]string{
			"KUBE_MEMORY_REQUEST": "invalid",
		}

		err := ValidateKubeConfig(content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid KUBE_MEMORY_REQUEST")
	})

	t.Run("Empty values are valid", func(t *testing.T) {
		content := map[string]string{
			"KUBE_DOCKER_IMAGE":   "",
			"KUBE_CPU_REQUEST":    "",
			"KUBE_MEMORY_REQUEST": "",
			"KUBE_CPU_LIMIT":      "",
			"KUBE_MEMORY_LIMIT":   "",
		}

		err := ValidateKubeConfig(content)
		assert.NoError(t, err)
	})
}
