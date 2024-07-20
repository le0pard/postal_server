package cmd

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type TokenVerificationFunc func(string, *gin.Context) bool

type Handler func(*gin.Context)

func Middleware(tokenVerificationFunc TokenVerificationFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// Authorization header is missing in HTTP request
		if authHeader == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		authTokens := strings.Split(authHeader, " ")

		// The value of authorization header is invalid
		// It should start with "Bearer ", then the token value
		if len(authTokens) != 2 || strings.ToLower(authTokens[0]) != "bearer" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Check token value is valid or not
		if !tokenVerificationFunc(authTokens[1], c) {
			return
		}

		// Everything looks fine, process next action
		c.Next()
	}
}

func MiddlewareWithStaticToken(token string) gin.HandlerFunc {
	return Middleware(func(s string, c *gin.Context) bool {
		if s != token {
			c.AbortWithStatus(http.StatusUnauthorized)
			return false
		}
		return true
	})
}
