package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMiddlewareWithStaticToken(t *testing.T) {
	// Switch to test mode so we don't get noisy output
	gin.SetMode(gin.TestMode)

	validToken := "my-secret-token"

	// Setup a dummy router with the middleware
	router := gin.New()
	router.Use(MiddlewareWithStaticToken(validToken))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Missing Authorization Header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Header Format (Missing Bearer)",
			authHeader:     "my-secret-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Token",
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Valid Token",
			authHeader:     "Bearer my-secret-token",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			router.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
