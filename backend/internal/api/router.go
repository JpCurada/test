package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/ISKOnnect/iskonnect-web/internal/api/handlers"
	apiMiddleware "github.com/ISKOnnect/iskonnect-web/internal/api/middleware"
	"github.com/ISKOnnect/iskonnect-web/internal/config"
	"github.com/ISKOnnect/iskonnect-web/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware" // Aliased as middleware for chi middleware
	"github.com/go-chi/cors"
)

func New(db *sql.DB, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	// Use chi middleware directly
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Update for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	authHandler := handlers.NewAuthHandler(db, cfg)
	userModel := models.NewUserModel(db)
	materialModel := models.NewMaterialModel(db)
	authMiddleware := apiMiddleware.NewAuthMiddleware(cfg.JWT.Secret) // Use aliased apiMiddleware

	r.Route("/api", func(r chi.Router) {
		// Public routes
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

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			// User routes (all users)
			r.Get("/users/me", func(w http.ResponseWriter, r *http.Request) {
				userID := r.Context().Value("user_id").(int)
				user, err := userModel.GetByID(userID)
				if err != nil {
					http.Error(w, "User not found", http.StatusNotFound)
					return
				}
				json.NewEncoder(w).Encode(user)
			})

			r.Put("/users/me", func(w http.ResponseWriter, r *http.Request) {
				userID := r.Context().Value("user_id").(int)
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
				user.FirstName = strings.TrimSpace(updates.FirstName)
				user.LastName = strings.TrimSpace(updates.LastName)
				if err := userModel.Update(user); err != nil {
					http.Error(w, "Update failed", http.StatusInternalServerError)
					return
				}
				json.NewEncoder(w).Encode(user)
			})

			// Student-only routes
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireStudent)

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
						userID := r.Context().Value("user_id").(int)
						// Verify user exists before proceeding
						user, err := userModel.GetByID(userID)
						if err != nil {
							http.Error(w, "User not found", http.StatusNotFound)
							return
						}

						var material models.Material
						if err := json.NewDecoder(r.Body).Decode(&material); err != nil {
							http.Error(w, "Invalid request", http.StatusBadRequest)
							return
						}
						if err := validateMaterial(material); err != nil {
							http.Error(w, err.Error(), http.StatusBadRequest)
							return
						}
						material.UploaderID = userID
						if err := materialModel.Create(&material); err != nil {
							http.Error(w, "Create failed", http.StatusInternalServerError)
							return
						}
						if err := userModel.IncrementPointsAndCheckBadges(userID, 5); err != nil {
							http.Error(w, fmt.Sprintf("Points update failed: %v", err), http.StatusInternalServerError)
							return
						}
						user, err = userModel.GetByID(userID) // Refresh user data after points update
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
						id, err := strconv.Atoi(chi.URLParam(r, "id"))
						if err != nil || id <= 0 {
							http.Error(w, "Invalid ID", http.StatusBadRequest)
							return
						}
						material, err := materialModel.GetByID(id)
						if err != nil {
							http.Error(w, "Material not found", http.StatusNotFound)
							return
						}
						json.NewEncoder(w).Encode(material)
					})

					r.Post("/{id}/vote", func(w http.ResponseWriter, r *http.Request) {
						id, err := strconv.Atoi(chi.URLParam(r, "id"))
						if err != nil || id <= 0 {
							http.Error(w, "Invalid ID", http.StatusBadRequest)
							return
						}
						userID := r.Context().Value("user_id").(int)
						var vote struct {
							VoteType string `json:"vote_type"`
						}
						if err := json.NewDecoder(r.Body).Decode(&vote); err != nil {
							http.Error(w, "Invalid request", http.StatusBadRequest)
							return
						}
						vote.VoteType = strings.ToUpper(vote.VoteType)
						if vote.VoteType != "UPVOTE" && vote.VoteType != "DOWNVOTE" {
							http.Error(w, "Invalid vote type", http.StatusBadRequest)
							return
						}
						if err := materialModel.Vote(id, userID, vote.VoteType); err != nil {
							http.Error(w, "Vote failed", http.StatusInternalServerError)
							return
						}
						material, _ := materialModel.GetByID(id)
						json.NewEncoder(w).Encode(material)
					})

					r.Post("/{id}/bookmark", func(w http.ResponseWriter, r *http.Request) {
						id, err := strconv.Atoi(chi.URLParam(r, "id"))
						if err != nil || id <= 0 {
							http.Error(w, "Invalid ID", http.StatusBadRequest)
							return
						}
						userID := r.Context().Value("user_id").(int)
						if err := materialModel.Bookmark(id, userID); err != nil {
							http.Error(w, "Bookmark failed", http.StatusInternalServerError)
							return
						}
						w.WriteHeader(http.StatusCreated)
						json.NewEncoder(w).Encode(map[string]string{"message": "Bookmarked"})
					})
				})

				r.Get("/materials/bookmarks", func(w http.ResponseWriter, r *http.Request) {
					userID := r.Context().Value("user_id").(int)
					bookmarks, err := materialModel.GetBookmarks(userID)
					if err != nil {
						http.Error(w, "Failed to get bookmarks", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(bookmarks)
				})

				r.Get("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
					limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
					if limit <= 0 {
						limit = 10
					}
					users, err := userModel.GetLeaderboard(limit)
					if err != nil {
						http.Error(w, "Failed to get leaderboard", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(users)
				})
			})

			// Admin-only routes
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireAdmin)

				r.Get("/admin/users", func(w http.ResponseWriter, r *http.Request) {
					users, err := userModel.GetAll()
					if err != nil {
						http.Error(w, "Failed to get users", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(users)
				})

				r.Delete("/admin/users/{id}", func(w http.ResponseWriter, r *http.Request) {
					id, err := strconv.Atoi(chi.URLParam(r, "id"))
					if err != nil || id <= 0 {
						http.Error(w, "Invalid ID", http.StatusBadRequest)
						return
					}
					if err := userModel.Delete(id); err != nil {
						http.Error(w, "Delete failed", http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusNoContent)
				})

				r.Get("/admin/materials", func(w http.ResponseWriter, r *http.Request) {
					materials, err := materialModel.List()
					if err != nil {
						http.Error(w, "Failed to list materials", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(materials)
				})

				r.Put("/admin/materials/{id}", func(w http.ResponseWriter, r *http.Request) {
					id, err := strconv.Atoi(chi.URLParam(r, "id"))
					if err != nil || id <= 0 {
						http.Error(w, "Invalid ID", http.StatusBadRequest)
						return
					}
					var material models.Material
					if err := json.NewDecoder(r.Body).Decode(&material); err != nil {
						http.Error(w, "Invalid request", http.StatusBadRequest)
						return
					}
					if err := validateMaterial(material); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					material.ID = id
					if err := materialModel.Update(&material); err != nil {
						http.Error(w, "Update failed", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(material)
				})

				r.Delete("/admin/materials/{id}", func(w http.ResponseWriter, r *http.Request) {
					id, err := strconv.Atoi(chi.URLParam(r, "id"))
					if err != nil || id <= 0 {
						http.Error(w, "Invalid ID", http.StatusBadRequest)
						return
					}
					if err := materialModel.Delete(id); err != nil {
						http.Error(w, "Delete failed", http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusNoContent)
				})
			})
		})
	})

	return r
}

func validateMaterial(m models.Material) error {
	if strings.TrimSpace(m.Title) == "" || len(m.Title) > 100 {
		return errors.New("title must be 1-100 characters")
	}
	if strings.TrimSpace(m.Description) == "" || len(m.Description) > 500 {
		return errors.New("description must be 1-500 characters")
	}
	if strings.TrimSpace(m.Subject) == "" || len(m.Subject) > 50 {
		return errors.New("subject must be 1-50 characters")
	}
	if strings.TrimSpace(m.College) == "" || len(m.College) > 50 {
		return errors.New("college must be 1-50 characters")
	}
	if strings.TrimSpace(m.Course) == "" || len(m.Course) > 50 {
		return errors.New("course must be 1-50 characters")
	}
	if !regexp.MustCompile(`^https?://`).MatchString(m.FileURL) {
		return errors.New("invalid file URL")
	}
	return nil
}
