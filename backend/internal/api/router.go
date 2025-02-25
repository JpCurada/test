// internal/api/router.go
package api

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ISKOnnect/iskonnect-web/internal/api/handlers"
	apiMiddleware "github.com/ISKOnnect/iskonnect-web/internal/api/middleware"
	"github.com/ISKOnnect/iskonnect-web/internal/config"
	"github.com/ISKOnnect/iskonnect-web/internal/models"
)

// New sets up the HTTP router
func New(db *sql.DB, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not to exceed 12 hours
	}))

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, cfg)
	// TODO: Initialize other handlers

	// Create auth middleware with the JWT secret
	authMiddleware := apiMiddleware.NewAuthMiddleware(cfg.JWT.Secret)

	// Public routes
	r.Route("/api", func(r chi.Router) {
		// Auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Get("/verify-email", authHandler.VerifyEmail)
			r.Post("/login", authHandler.Login)
			r.Post("/logout", authHandler.Logout)
			r.Post("/refresh", authHandler.RefreshToken)
			r.Post("/forgot-password", authHandler.ForgotPassword)
			r.Post("/verify-otp", authHandler.VerifyOTP)
			r.Post("/reset-password", authHandler.ResetPassword)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			// Apply authentication middleware
			r.Use(authMiddleware.Authenticate)

			// User routes
			r.Route("/users", func(r chi.Router) {
				r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Current user details"))
				})

				r.Put("/me", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Update current user"))
				})

				r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("User profile"))
				})

				r.Get("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Leaderboard"))
				})
			})

			// Materials routes
			r.Route("/materials", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("List materials"))
				})

				r.Post("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Create material"))
				})

				r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Get material"))
				})

				r.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Update material"))
				})

				r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Delete material"))
				})

				r.Post("/{id}/vote", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Vote for material"))
				})

				r.Post("/{id}/bookmark", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Bookmark material"))
				})

				r.Post("/{id}/report", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Report material"))
				})
			})

			// Admin routes
			r.Route("/admin", func(r chi.Router) {
				// Apply admin role middleware
				r.Use(authMiddleware.RequireRole(models.UserTypeAdmin))

				r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Admin dashboard"))
				})

				r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("List all users"))
				})

				r.Delete("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Delete user"))
				})

				r.Get("/reports", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("List all reports"))
				})

				r.Put("/reports/{id}", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Resolve report"))
				})
			})
		})
	})

	return r
}