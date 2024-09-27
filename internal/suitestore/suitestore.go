package suitestore

import (
	"context"
)

// ReferenceConfigSuite represents a collection of actor configurations
type ActorConfig struct {
	ActorName string            `json:"actor_name"`
	Configs   map[string]string `json:"configs"`
}

type ReferenceConfigSuite struct {
	SuiteName    string        `json:"suite_name"`
	ConfigSuites []ActorConfig `json:"config_suites"`
}

// SuiteStore defines an interface for reading and writing config suites to AWS S3
type SuiteStore interface {
	// WriteSuite writes the given config suite to store
	WriteSuite(ctx context.Context, suite []ActorConfig) error
}
