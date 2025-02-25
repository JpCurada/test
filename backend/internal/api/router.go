package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

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
		MaxAge:           300,
	}))

	// Initialize handlers and models
	authHandler := handlers.NewAuthHandler(db, cfg)
	userModel := models.NewUserModel(db)
	materialModel := models.NewMaterialModel(db)
	authMiddleware := apiMiddleware.NewAuthMiddleware(cfg.JWT.Secret)

	// Public routes
	r.Route("/api", func(r chi.Router) {
		// Auth routes (unchanged)
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
			r.Use(authMiddleware.Authenticate)

			// User routes (unchanged)
			r.Route("/users", func(r chi.Router) {
				r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
					userID := r.Context().Value("user_id").(string)
					user, err := userModel.GetByID(userID)
					if err != nil {
						http.Error(w, "User not found", http.StatusNotFound)
						return
					}
					json.NewEncoder(w).Encode(user)
				})

				r.Put("/me", func(w http.ResponseWriter, r *http.Request) {
					userID := r.Context().Value("user_id").(string)
					var updates struct {
						FirstName string `json:"first_name"`
						LastName  string `json:"last_name"`
					}
					if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
						http.Error(w, "Invalid request", http.StatusBadRequest)
						return
					}
					user, err := userModel.GetByID(userID)
					if err != nil {
						http.Error(w, "User not found", http.StatusNotFound)
						return
					}
					user.FirstName = updates.FirstName
					user.LastName = updates.LastName
					if err := userModel.Update(user); err != nil {
						http.Error(w, "Failed to update user", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(user)
				})

				r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
					id := chi.URLParam(r, "id")
					user, err := userModel.GetByID(id)
					if err != nil {
						http.Error(w, "User not found", http.StatusNotFound)
						return
					}
					json.NewEncoder(w).Encode(user)
				})

				r.Get("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
					users, err := userModel.GetLeaderboard(10)
					if err != nil {
						http.Error(w, "Failed to fetch leaderboard", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(users)
				})
			})

			// Materials routes
			r.Route("/materials", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					materials, err := materialModel.List()
					if err != nil {
						http.Error(w, "Failed to list materials", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(materials)
				})

				r.Post("/", func(w http.ResponseWriter, r *http.Request) {
					userID := r.Context().Value("user_id").(string)
					var material models.Material
					if err := json.NewDecoder(r.Body).Decode(&material); err != nil {
						http.Error(w, "Invalid request", http.StatusBadRequest)
						return
					}
					material.UploaderID = userID
					if err := materialModel.Create(&material); err != nil {
						http.Error(w, "Failed to create material", http.StatusInternalServerError)
						return
					}

					// Increment user's points by 5 and check for badges
					if err := userModel.IncrementPointsAndCheckBadges(userID, 5); err != nil {
						http.Error(w, "Failed to update user points or badges", http.StatusInternalServerError)
						return
					}

					// Fetch updated user info (optional, for response)
					user, err := userModel.GetByID(userID)
					if err != nil {
						http.Error(w, "Failed to fetch updated user", http.StatusInternalServerError)
						return
					}

					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"material": material,
						"user":     user,
					})
				})

				r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
					id, _ := strconv.Atoi(chi.URLParam(r, "id"))
					material, err := materialModel.GetByID(id)
					if err != nil {
						http.Error(w, "Material not found", http.StatusNotFound)
						return
					}
					json.NewEncoder(w).Encode(material)
				})

				r.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
					id, _ := strconv.Atoi(chi.URLParam(r, "id"))
					userID := r.Context().Value("user_id").(string)
					material, err := materialModel.GetByID(id)
					if err != nil {
						http.Error(w, "Material not found", http.StatusNotFound)
						return
					}
					if material.UploaderID != userID {
						http.Error(w, "Unauthorized", http.StatusForbidden)
						return
					}
					var updates models.Material
					if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
						http.Error(w, "Invalid request", http.StatusBadRequest)
						return
					}
					material.Title = updates.Title
					material.Description = updates.Description
					material.Subject = updates.Subject
					material.College = updates.College
					material.Course = updates.Course
					material.FileURL = updates.FileURL
					material.Filename = updates.Filename
					if err := materialModel.Update(material); err != nil {
						http.Error(w, "Failed to update material", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(material)
				})

				r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
					id, _ := strconv.Atoi(chi.URLParam(r, "id"))
					userID := r.Context().Value("user_id").(string)
					material, err := materialModel.GetByID(id)
					if err != nil {
						http.Error(w, "Material not found", http.StatusNotFound)
						return
					}
					if material.UploaderID != userID {
						http.Error(w, "Unauthorized", http.StatusForbidden)
						return
					}
					if err := materialModel.Delete(id); err != nil {
						http.Error(w, "Failed to delete material", http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusNoContent)
				})

				r.Post("/{id}/vote", func(w http.ResponseWriter, r *http.Request) {
					id, _ := strconv.Atoi(chi.URLParam(r, "id"))
					userID := r.Context().Value("user_id").(string)
					var vote struct {
						VoteType string `json:"vote_type"`
					}
					if err := json.NewDecoder(r.Body).Decode(&vote); err != nil {
						http.Error(w, "Invalid request", http.StatusBadRequest)
						return
					}
					if vote.VoteType != "UPVOTE" && vote.VoteType != "DOWNVOTE" {
						http.Error(w, "Invalid vote type", http.StatusBadRequest)
						return
					}
					if err := materialModel.Vote(id, userID, vote.VoteType); err != nil {
						http.Error(w, "Failed to vote", http.StatusInternalServerError)
						return
					}
					material, err := materialModel.GetByID(id)
					if err != nil {
						http.Error(w, "Material not found", http.StatusNotFound)
						return
					}
					json.NewEncoder(w).Encode(material)
				})

				r.Post("/{id}/bookmark", func(w http.ResponseWriter, r *http.Request) {
					id, _ := strconv.Atoi(chi.URLParam(r, "id"))
					userID := r.Context().Value("user_id").(string)
					if err := materialModel.Bookmark(id, userID); err != nil {
						http.Error(w, "Failed to bookmark", http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]string{"message": "Material bookmarked"})
				})

				r.Post("/{id}/report", func(w http.ResponseWriter, r *http.Request) {
					id, _ := strconv.Atoi(chi.URLParam(r, "id"))
					userID := r.Context().Value("user_id").(string)
					var report struct {
						Reason         string `json:"reason"`
						AdditionalInfo string `json:"additional_info"`
					}
					if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
						http.Error(w, "Invalid request", http.StatusBadRequest)
						return
					}
					if err := materialModel.Report(id, userID, report.Reason, report.AdditionalInfo); err != nil {
						http.Error(w, "Failed to report material", http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]string{"message": "Material reported"})
				})
			})

			// Admin routes (unchanged)
			r.Route("/admin", func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(models.UserTypeAdmin))

				r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
					users, err := userModel.GetAll()
					if err != nil {
						http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
						return
					}
					reports, err := userModel.GetReports()
					if err != nil {
						http.Error(w, "Failed to fetch reports", http.StatusInternalServerError)
						return
					}
					materials, err := materialModel.List()
					if err != nil {
						http.Error(w, "Failed to fetch materials", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(map[string]interface{}{
						"users":     users,
						"reports":   reports,
						"materials": materials,
					})
				})

				r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
					users, err := userModel.GetAll()
					if err != nil {
						http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(users)
				})

				r.Delete("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
					id := chi.URLParam(r, "id")
					_, err := userModel.GetByID(id)
					if err != nil {
						http.Error(w, "User not found", http.StatusNotFound)
						return
					}
					query := `DELETE FROM users WHERE id = $1`
					_, err = db.Exec(query, id)
					if err != nil {
						http.Error(w, "Failed to delete user", http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusNoContent)
				})

				r.Get("/reports", func(w http.ResponseWriter, r *http.Request) {
					reports, err := userModel.GetReports()
					if err != nil {
						http.Error(w, "Failed to fetch reports", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(reports)
				})

				r.Put("/reports/{id}", func(w http.ResponseWriter, r *http.Request) {
					id, _ := strconv.Atoi(chi.URLParam(r, "id"))
					userID := r.Context().Value("user_id").(string)
					var resolution struct {
						Status          string `json:"status"`
						ResolutionNotes string `json:"resolution_notes"`
					}
					if err := json.NewDecoder(r.Body).Decode(&resolution); err != nil {
						http.Error(w, "Invalid request", http.StatusBadRequest)
						return
					}
					if resolution.Status != "RESOLVED" && resolution.Status != "DISMISSED" {
						http.Error(w, "Invalid status", http.StatusBadRequest)
						return
					}
					if err := userModel.ResolveReport(id, userID, resolution.ResolutionNotes, resolution.Status); err != nil {
						http.Error(w, "Failed to resolve report", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(map[string]string{"message": "Report resolved"})
				})
			})
		})
	})

	return r
}