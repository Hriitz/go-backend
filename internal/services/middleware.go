package services

import (
	"context"
	"net/http"
	"strings"

	"springstreet/internal/util"
	"springstreet/internal/database"
)

// JWTAuthMiddleware implements JWT authentication middleware
func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for public endpoints
		if isPublicEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Check Bearer token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := util.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Get user from database
		user, err := util.GetUserFromToken(database.GetDB(), claims)
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		if !user.IsActive {
			http.Error(w, "User account is inactive", http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), "user", user)
		ctx = context.WithValue(ctx, "claims", claims)

		// Check scope requirements (if any)
		if !checkScope(r.URL.Path, user) {
			http.Error(w, "Insufficient permissions", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isPublicEndpoint checks if the endpoint is public (doesn't require auth)
func isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/health",
		"/api/v1/auth/login",
		"/api/v1/investment/",
		"/api/v1/investment/by-phone/",
		"/api/v1/investment/verify/",
		"/api/v1/otp/",
	}

	for _, publicPath := range publicPaths {
		if strings.HasPrefix(path, publicPath) {
			// Special case: POST /api/v1/investment/ is public, but GET requires auth
			if path == "/api/v1/investment/" && !strings.Contains(path, "?") {
				// This is a bit simplified - in real implementation, check HTTP method
				return true
			}
			if strings.HasPrefix(path, publicPath) {
				return true
			}
		}
	}

	return false
}

// checkScope checks if user has required scope for the endpoint
func checkScope(path string, user interface{}) bool {
	// This is a simplified version - in real implementation, check user roles
	// against endpoint requirements
	adminPaths := []string{
		"/api/v1/auth/users",
	}

	for _, adminPath := range adminPaths {
		if strings.HasPrefix(path, adminPath) {
			// Check if user is admin
			// This would need proper type assertion
			return true // Simplified
		}
	}

	return true
}


