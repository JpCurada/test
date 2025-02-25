// internal/api/handlers/auth.go
package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ISKOnnect/iskonnect-web/internal/config"
	"github.com/ISKOnnect/iskonnect-web/internal/email"
	"github.com/ISKOnnect/iskonnect-web/internal/models"
	"github.com/ISKOnnect/iskonnect-web/internal/utils"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	DB          *sql.DB
	Config      *config.Config
	UserModel   *models.UserModel
	EmailSender *email.Sender
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(db *sql.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		DB:          db,
		Config:      cfg,
		UserModel:   models.NewUserModel(db),
		EmailSender: email.NewSender(cfg.Email),
	}
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	StudentNumber  string `json:"student_number"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	StudentNumber string `json:"student_number"`
	Password      string `json:"password"`
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.StudentNumber == "" || req.FirstName == "" || req.LastName == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if req.Password != req.ConfirmPassword {
		http.Error(w, "Passwords do not match", http.StatusBadRequest)
		return
	}

	// Check if user already exists
	_, err := h.UserModel.GetByEmail(req.Email)
	if err == nil {
		http.Error(w, "Email already registered", http.StatusConflict)
		return
	} else if !errors.Is(err, errors.New("user not found")) {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if student number already exists
	_, err = h.UserModel.GetByStudentNumber(req.StudentNumber)
	if err == nil {
		http.Error(w, "Student number already registered", http.StatusConflict)
		return
	} else if !errors.Is(err, errors.New("user not found")) {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Hash the password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create user
	user := &models.User{
		ID:            req.StudentNumber,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Email:         req.Email,
		UserType:      models.UserTypeStudent,
		Points:        0,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.UserModel.Create(user, hashedPassword); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate verification token
	token, err := utils.GenerateRandomToken(32)
	if err != nil {
		http.Error(w, "Failed to generate verification token", http.StatusInternalServerError)
		return
	}

	// Store verification token in database
	// In a real app, you'd store the token with an expiry time
	
	// Send verification email
	if err := h.EmailSender.SendVerificationEmail(req.Email, token); err != nil {
		// Log the error but don't return it to the client
		// In a real app, you'd handle this more gracefully
		http.Error(w, "Registration successful, but failed to send verification email", http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Registration successful. Please check your email to verify your account.",
	})
}

// VerifyEmail handles email verification
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	// In a real app, you'd verify the token against what's stored in the database
	// and update the user's verified status
	// For simplicity, we'll just respond with success

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Email verified successfully. You can now log in.",
	})
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.StudentNumber == "" || req.Password == "" {
		http.Error(w, "Missing student number or password", http.StatusBadRequest)
		return
	}

	// Get user from database
	user, err := h.UserModel.GetByStudentNumber(req.StudentNumber)
	if err != nil {
		http.Error(w, "Invalid student number or password", http.StatusUnauthorized)
		return
	}

	// In a real app, you'd check if the email is verified
	// if !user.EmailVerified {
	//     http.Error(w, "Email not verified", http.StatusUnauthorized)
	//     return
	// }

	// Check password
	passwordHash, err := h.UserModel.GetPasswordHash(req.StudentNumber)
	if err != nil {
		http.Error(w, "Invalid student number or password", http.StatusUnauthorized)
		return
	}

	if err := utils.CheckPassword(passwordHash, req.Password); err != nil {
		http.Error(w, "Invalid student number or password", http.StatusUnauthorized)
		return
	}

	// Generate JWT tokens
	accessToken, err := utils.GenerateJWT(user, 24) // 24 hours
	if err != nil {
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := utils.GenerateJWT(user, 168) // 7 days
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Set cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		MaxAge:   24 * 3600, // 24 hours
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		MaxAge:   168 * 3600, // 7 days
		SameSite: http.SameSiteStrictMode,
	})

	// Return response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"id":         user.ID,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"email":      user.Email,
			"user_type":  user.UserType,
			"points":     user.Points,
		},
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		MaxAge:   -1,
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		MaxAge:   -1,
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "Refresh token not found", http.StatusUnauthorized)
		return
	}

	refreshToken := cookie.Value

	// Validate refresh token
	claims, err := utils.ValidateJWT(refreshToken)
	if err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Get user from database
	user, err := h.UserModel.GetByID(claims.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Generate new access token
	accessToken, err := utils.GenerateJWT(user, 24) // 24 hours
	if err != nil {
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		MaxAge:   24 * 3600, // 24 hours
		SameSite: http.SameSiteStrictMode,
	})

	// Return response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"access_token": accessToken,
	})
}

// ForgotPassword handles password reset requests
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// Check if user exists
	user, err := h.UserModel.GetByEmail(req.Email)
	if err != nil {
		// Don't reveal that the email doesn't exist
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "If your email is registered, you will receive a password reset code.",
		})
		return
	}

	// Generate OTP
	otp, err := utils.GenerateOTP(6)
	if err != nil {
		http.Error(w, "Failed to generate OTP", http.StatusInternalServerError)
		return
	}

	// In a real app, you'd store the OTP with an expiry time
	// For simplicity, we'll just send the email without storing the OTP

	// Send password reset email
	if err := h.EmailSender.SendPasswordResetEmail(user.Email, otp); err != nil {
		http.Error(w, "Failed to send password reset email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "If your email is registered, you will receive a password reset code.",
	})
}

// VerifyOTP verifies the OTP for password reset
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Email == "" || req.OTP == "" {
		http.Error(w, "Email and OTP are required", http.StatusBadRequest)
		return
	}

	// In a real app, you'd verify the OTP against what's stored in the database
	// For simplicity, we'll assume the OTP is correct

	// Generate a reset token
	resetToken, err := utils.GenerateRandomToken(32)
	if err != nil {
		http.Error(w, "Failed to generate reset token", http.StatusInternalServerError)
		return
	}

	// In a real app, you'd store the reset token with an expiry time
	// For simplicity, we'll just return the token

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"reset_token": resetToken,
	})
}

// ResetPassword resets the user's password
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		ResetToken  string `json:"reset_token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Email == "" || req.ResetToken == "" || req.NewPassword == "" {
		http.Error(w, "Email, reset token, and new password are required", http.StatusBadRequest)
		return
	}

	// In a real app, you'd verify the reset token against what's stored in the database
	// For simplicity, we'll assume the token is valid

	// Get user from database
	user, err := h.UserModel.GetByEmail(req.Email)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Hash the new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Update the user's password
	if err := h.UserModel.UpdatePassword(user.ID, hashedPassword); err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Password reset successfully. You can now log in with your new password.",
	})
}