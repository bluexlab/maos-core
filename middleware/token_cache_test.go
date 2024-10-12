package middleware

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTokenFetcher is a mock implementation of the TokenFetcher function
type MockTokenFetcher struct {
	mock.Mock
}

func (m *MockTokenFetcher) Fetch(ctx context.Context, apiToken string) (*Token, error) {
	args := m.Called(ctx, apiToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Token), args.Error(1)
}

func TestNewApiTokenCache(t *testing.T) {
	mockFetcher := new(MockTokenFetcher)
	ttl := 10 * time.Second

	cache := NewApiTokenCache(mockFetcher.Fetch, ttl)

	assert.NotNil(t, cache)
	assert.NotNil(t, cache.cache)
	assert.Equal(t, ttl, cache.ttl)
}

func TestApiTokenCache_GetToken(t *testing.T) {
	ctx := context.Background()
	apiToken := "test-token"

	t.Run("Fetches and caches token", func(t *testing.T) {
		mockFetcher := new(MockTokenFetcher)
		cache := NewApiTokenCache(mockFetcher.Fetch, 10*time.Second)
		expectedToken := &Token{Id: "1", ActorId: 123}
		mockFetcher.On("Fetch", ctx, apiToken).Return(expectedToken, nil).Once()

		// First call, should fetch from the mock
		token := cache.GetToken(ctx, apiToken)
		assert.Equal(t, expectedToken, token)
		cache.cache.Wait() // Wait for cache to be ready

		// Second call, should retrieve from cache
		token = cache.GetToken(ctx, apiToken)
		assert.Equal(t, expectedToken, token)

		mockFetcher.AssertExpectations(t)
	})

	t.Run("Handles non-existent token", func(t *testing.T) {
		mockFetcher := new(MockTokenFetcher)
		cache := NewApiTokenCache(mockFetcher.Fetch, 10*time.Second)
		mockFetcher.On("Fetch", ctx, apiToken).Return(nil, nil).Once()

		token := cache.GetToken(ctx, apiToken)
		assert.Nil(t, token)

		mockFetcher.AssertExpectations(t)

		// Second call, it should return nil without calling the fetcher
		cache.Wait()
		token = cache.GetToken(ctx, apiToken)
		assert.Nil(t, token)
		mockFetcher.AssertExpectations(t)
	})

	t.Run("Handles fetch error", func(t *testing.T) {
		mockFetcher := new(MockTokenFetcher)
		cache := NewApiTokenCache(mockFetcher.Fetch, 10*time.Second)
		mockFetcher.On("Fetch", ctx, apiToken).Return(nil, errors.New("fetch error")).Once()

		token := cache.GetToken(ctx, apiToken)
		assert.Nil(t, token)

		mockFetcher.AssertExpectations(t)
	})

	t.Run("Token found in cache", func(t *testing.T) {
		mockFetcher := new(MockTokenFetcher)
		cache := NewApiTokenCache(mockFetcher.Fetch, 10*time.Second)
		expectedToken := &Token{
			ActorId:     1,
			QueueId:     2,
			ExpireAt:    time.Now().Unix() + 3600,
			Permissions: []string{"read", "write"},
		}

		cache.cache.Set(apiToken, expectedToken, 1)
		cache.cache.Wait() // Wait for cache to be ready

		result := cache.GetToken(ctx, apiToken)

		assert.Equal(t, expectedToken, result)
	})
}

func TestApiTokenCache_GetToken_Singleflight(t *testing.T) {
	mockFetcher := new(MockTokenFetcher)
	cache := NewApiTokenCache(mockFetcher.Fetch, 10*time.Second)

	ctx := context.Background()
	apiToken := "test-token-singleflight"

	// Set up the mock to return a token after a short delay
	expectedToken := &Token{
		ActorId:     5,
		QueueId:     6,
		ExpireAt:    time.Now().Unix() + 3600,
		Permissions: []string{"read"},
	}

	mockFetcher.On("Fetch", ctx, apiToken).
		Return(expectedToken, nil).
		After(100 * time.Millisecond).
		Once()

	// Number of concurrent requests
	numRequests := 10

	// Use a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup
	wg.Add(numRequests)

	// Channel to collect results
	resultChan := make(chan *Token, numRequests)

	// Start multiple goroutines to request the same token concurrently
	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			result := cache.GetToken(ctx, apiToken)
			resultChan <- result
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(resultChan)

	// Collect results
	var results []*Token
	for result := range resultChan {
		results = append(results, result)
	}

	// Assert that we got the correct number of results
	assert.Len(t, results, numRequests)

	// Assert that all results are the same and match the expected token
	for _, result := range results {
		assert.NotNil(t, result)
		assert.Equal(t, expectedToken, result)
	}

	// Verify that the accessor method was called only once
	mockFetcher.AssertExpectations(t)
}

func TestCreateCache(t *testing.T) {
	cache := createCache()

	assert.NotNil(t, cache)
	assert.IsType(t, &ristretto.Cache{}, cache)
}
