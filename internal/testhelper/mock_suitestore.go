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
	synced bool
}

// NewMockSuiteStore creates a new instance of MockSuiteStore
func NewMockSuiteStore() *MockSuiteStore {
	return &MockSuiteStore{
		suites: make([]suitestore.ReferenceConfigSuite, 0),
		synced: false,
	}
}

// ReadSuites implements the SuiteStore interface
func (m *MockSuiteStore) ReadSuites(ctx context.Context) ([]suitestore.ReferenceConfigSuite, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.suites, nil
}

func (m *MockSuiteStore) IsSynced() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.synced
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

// SyncSuites implements the SuiteStore interface
func (m *MockSuiteStore) SyncSuites(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.synced = true
	return nil
}

// ClearSuites clears all stored suites (helper method for testing)
func (m *MockSuiteStore) ClearSuites() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.suites = make([]suitestore.ReferenceConfigSuite, 0)
	m.synced = false
}
