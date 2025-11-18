package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteRequestSuccess(t *testing.T) {
	// Create a test server
	expectedBody := "test response body"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	}))
	defer server.Close()
	
	proxy := NewProxy()
	body, status, err := proxy.ExecuteRequest(server.URL)
	
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, expectedBody, string(body))
}

func TestExecuteRequestDifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus int
	}{
		{"OK", http.StatusOK, http.StatusOK},
		{"Created", http.StatusCreated, http.StatusCreated},
		{"Not Found", http.StatusNotFound, http.StatusNotFound},
		{"Server Error", http.StatusInternalServerError, http.StatusInternalServerError},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()
			
			proxy := NewProxy()
			_, status, err := proxy.ExecuteRequest(server.URL)
			
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestExecuteRequestInvalidURL(t *testing.T) {
	proxy := NewProxy()
	_, _, err := proxy.ExecuteRequest("http://invalid-url-that-does-not-exist:9999")
	
	assert.Error(t, err)
}

func TestExecuteRequestLargeResponse(t *testing.T) {
	// Create a large response body
	largeBody := make([]byte, 1024*1024) // 1MB
	for i := range largeBody {
		largeBody[i] = 'A'
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largeBody)
	}))
	defer server.Close()
	
	proxy := NewProxy()
	body, status, err := proxy.ExecuteRequest(server.URL)
	
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, len(largeBody), len(body))
}

func TestExecuteRequestWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a GET request
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	proxy := NewProxy()
	_, status, err := proxy.ExecuteRequest(server.URL)
	
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
}
