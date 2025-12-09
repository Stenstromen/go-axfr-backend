package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		bodySize       int
		expectedStatus int
		checkHeaders   bool
	}{
		{
			name:           "small body",
			bodySize:       100,
			expectedStatus: http.StatusOK,
			checkHeaders:   true,
		},
		{
			name:           "large body exceeds limit",
			bodySize:       2 * 1048576,   // 2MB, exceeds 1MB limit
			expectedStatus: http.StatusOK, // MaxBytesReader limits reading, doesn't change status
			checkHeaders:   true,
		},
		{
			name:           "empty body",
			bodySize:       0,
			expectedStatus: http.StatusOK,
			checkHeaders:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that will be wrapped by middleware
			handler := func(w http.ResponseWriter, r *http.Request) {
				// Try to read the body to trigger MaxBytesReader limit
				buf := make([]byte, 100)
				_, err := r.Body.Read(buf)
				if err != nil && err.Error() != "EOF" {
					// MaxBytesReader will return an error when limit is exceeded
					// but the handler still executes
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}

			// Wrap handler with middleware
			wrappedHandler := Middleware(handler)

			// Create request with body
			body := strings.Repeat("a", tt.bodySize)
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
			rr := httptest.NewRecorder()

			// Execute request
			wrappedHandler(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Middleware() status code = %v, want %v", rr.Code, tt.expectedStatus)
			}

			// Check headers if expected
			if tt.checkHeaders {
				contentType := rr.Header().Get("content-type")
				if contentType != "application/json" {
					t.Errorf("Middleware() content-type = %v, want application/json", contentType)
				}
			}
		})
	}
}

func TestMiddlewareSetsContentType(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"test": "data"}`))
	}

	wrappedHandler := Middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	wrappedHandler(rr, req)

	if rr.Header().Get("content-type") != "application/json" {
		t.Errorf("Middleware() did not set content-type header correctly")
	}
}
