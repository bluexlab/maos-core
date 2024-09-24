package suitestore

import (
	"context"
)

// ReferenceConfigSuite represents a collection of agent configurations
type AgentConfig struct {
	AgentName string            `json:"agent_name"`
	Configs   map[string]string `json:"configs"`
}

type ReferenceConfigSuite struct {
	SuiteName    string        `json:"suite_name"`
	ConfigSuites []AgentConfig `json:"config_suites"`
}

// SuiteStore defines an interface for reading and writing config suites to AWS S3
type SuiteStore interface {
	// WriteSuite writes the given config suite to store
	WriteSuite(ctx context.Context, suite []AgentConfig) error
}
