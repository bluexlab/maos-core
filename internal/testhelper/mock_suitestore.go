package testhelper

import (
	"context"
	"sync"

	"gitlab.com/navyx/ai/maos/maos-core/internal/suitestore"
)

// MockSuiteStore is a mock implementation of the SuiteStore interface
type MockSuiteStore struct {
	mu     sync.Mutex
	suites []suitestore.ReferenceConfigSuite
}

// NewMockSuiteStore creates a new instance of MockSuiteStore
func NewMockSuiteStore() *MockSuiteStore {
	return &MockSuiteStore{
		suites: make([]suitestore.ReferenceConfigSuite, 0),
	}
}

// ReadSuites implements the SuiteStore interface
func (m *MockSuiteStore) ReadSuites(ctx context.Context) ([]suitestore.ReferenceConfigSuite, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.suites, nil
}

// WriteSuite implements the SuiteStore interface
func (m *MockSuiteStore) WriteSuite(ctx context.Context, suite []suitestore.ActorConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.suites = append(m.suites, suitestore.ReferenceConfigSuite{
		SuiteName:    "MockSuite",
		ConfigSuites: suite,
	})
	return nil
}

// ClearSuites clears all stored suites (helper method for testing)
func (m *MockSuiteStore) ClearSuites() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.suites = make([]suitestore.ReferenceConfigSuite, 0)
}
