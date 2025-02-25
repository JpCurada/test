// internal/api/middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ISKOnnect/iskonnect-web/internal/models"
	"github.com/ISKOnnect/iskonnect-web/internal/utils"
)

// AuthMiddleware authenticates users
type AuthMiddleware struct{}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{}
}

// Authenticate authenticates a request
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from request
		token := extractToken(r)
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := utils.ValidateJWT(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "user_type", claims.UserType)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole requires a specific user role
func (m *AuthMiddleware) RequireRole(role models.UserType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user type from context
			userType, ok := r.Context().Value("user_type").(models.UserType)
			if !ok || userType != role {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// extractToken extracts the JWT token from the request
func extractToken(r *http.Request) string {
	// First, try to get the token from the Authorization header
	bearerToken := r.Header.Get("Authorization")
	if bearerToken != "" && strings.HasPrefix(bearerToken, "Bearer ") {
		return strings.TrimPrefix(bearerToken, "Bearer ")
	}

	// If not in the header, try to get it from the cookie
	cookie, err := r.Cookie("access_token")
	if err == nil {
		return cookie.Value
	}

	return ""
}