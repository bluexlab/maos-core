package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthTokenFetcher is a mock implementation of TokenFetcher
type MockAuthTokenFetcher struct {
	mock.Mock
}

func (m *MockAuthTokenFetcher) FetchToken(ctx context.Context, apiToken string) (*Token, error) {
	args := m.Called(ctx, apiToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Token), args.Error(1)
}

func TestNewBearerAuthMiddleware(t *testing.T) {
	mockFetcher := new(MockAuthTokenFetcher)
	cacheTTL := 5 * time.Minute

	middleware := NewBearerAuthMiddleware(mockFetcher.FetchToken, cacheTTL)

	t.Run("Missing Authorization Header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler := middleware(func(ctx context.Context, w http.ResponseWriter, r *http.Request, args interface{}) (interface{}, error) {
			return nil, nil
		}, "test")

		result, err := handler(context.Background(), w, req, nil)

		assert.Nil(t, result)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, `{"error":"missing authorization header"}`, strings.TrimSpace(w.Body.String()))
	})

	t.Run("Invalid Authorization Header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Invalid Token")
		w := httptest.NewRecorder()

		handler := middleware(func(ctx context.Context, w http.ResponseWriter, r *http.Request, args interface{}) (interface{}, error) {
			return nil, nil
		}, "test")

		result, err := handler(context.Background(), w, req, nil)

		assert.Nil(t, result)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, `{"error":"invalid authorization header"}`, strings.TrimSpace(w.Body.String()))
	})

	t.Run("Invalid Token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalidtoken")
		w := httptest.NewRecorder()

		mockFetcher.On("FetchToken", mock.Anything, "invalidtoken").Return(nil, nil)

		handler := middleware(func(ctx context.Context, w http.ResponseWriter, r *http.Request, args interface{}) (interface{}, error) {
			return nil, nil
		}, "test")

		result, err := handler(context.Background(), w, req, nil)

		assert.Nil(t, result)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, `{"error":"invalid token"}`, strings.TrimSpace(w.Body.String()))
	})

	t.Run("Valid Token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer validtoken")
		w := httptest.NewRecorder()

		validToken := &Token{
			Id:          "valid-id",
			AgentId:     123,
			QueueId:     456,
			ExpireAt:    time.Now().Add(1 * time.Hour).Unix(),
			Permissions: []string{"read", "write"},
		}

		mockFetcher.On("FetchToken", mock.Anything, "validtoken").Return(validToken, nil)

		var capturedToken *Token
		handler := middleware(func(ctx context.Context, w http.ResponseWriter, r *http.Request, args interface{}) (interface{}, error) {
			capturedToken = ctx.Value(TokenContextKey).(*Token)
			return "success", nil
		}, "test")

		result, err := handler(context.Background(), w, req, nil)

		assert.Equal(t, "success", result)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, validToken, capturedToken)
	})
}
