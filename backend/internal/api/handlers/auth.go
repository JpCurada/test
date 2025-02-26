package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ISKOnnect/iskonnect-web/internal/config"
	"github.com/ISKOnnect/iskonnect-web/internal/email"
	"github.com/ISKOnnect/iskonnect-web/internal/models"
	"github.com/ISKOnnect/iskonnect-web/internal/utils"
)

type AuthHandler struct {
	db          *sql.DB
	cfg         *config.Config
	userModel   *models.UserModel
	emailSender *email.Sender
}

func NewAuthHandler(db *sql.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		db:          db,
		cfg:         cfg,
		userModel:   models.NewUserModel(db),
		emailSender: email.NewSender(cfg.Email),
	}
}

type RegisterRequest struct {
	StudentNumber   string `json:"student_number"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type LoginRequest struct {
	StudentNumber string `json:"student_number"`
	Password      string `json:"password"`
}

func isValidStudentNumber(sn string) bool {
	return regexp.MustCompile(`^\d{4}-\d{5}-[A-Z]{2}-\d$`).MatchString(sn)
}

func isValidEmail(email string) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`).MatchString(email)
}

func isValidPassword(password string) bool {
	return len(password) >= 8 && regexp.MustCompile(`[A-Z]`).MatchString(password) &&
		regexp.MustCompile(`[a-z]`).MatchString(password) &&
		regexp.MustCompile(`[0-9]`).MatchString(password) &&
		regexp.MustCompile(`[!@#$%^&*]`).MatchString(password)
}

func isValidName(name string) bool {
	trimmed := strings.TrimSpace(name)
	return len(trimmed) >= 2 && len(trimmed) <= 50 && regexp.MustCompile(`^[a-zA-Z\s-]+$`).MatchString(trimmed)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if !isValidStudentNumber(req.StudentNumber) {
		http.Error(w, "Invalid student number format (e.g., 2023-00239-MN-0)", http.StatusBadRequest)
		return
	}
	if !isValidName(req.FirstName) || !isValidName(req.LastName) {
		http.Error(w, "Names must be 2-50 letters", http.StatusBadRequest)
		return
	}
	if !isValidEmail(req.Email) {
		http.Error(w, "Invalid email", http.StatusBadRequest)
		return
	}
	if !isValidPassword(req.Password) {
		http.Error(w, "Password must be 8+ chars with uppercase, lowercase, number, and special char", http.StatusBadRequest)
		return
	}
	if req.Password != req.ConfirmPassword {
		http.Error(w, "Passwords do not match", http.StatusBadRequest)
		return
	}

	if _, err := h.userModel.GetByEmail(req.Email); err == nil {
		http.Error(w, "Email already registered", http.StatusConflict)
		return
	}
	if _, err := h.userModel.GetByStudentNumber(req.StudentNumber); err == nil {
		http.Error(w, "Student number already registered", http.StatusConflict)
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		log.Printf("Hash failed: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var userID int
	err = tx.QueryRow(`
		INSERT INTO user_credentials (email, password_hash, created_at)
		VALUES ($1, $2, $3) RETURNING id`,
		strings.ToLower(req.Email), hashedPassword, time.Now(),
	).Scan(&userID)
	if err != nil {
		log.Printf("Credential insert failed: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	user := &models.User{
		ID:            userID,
		StudentNumber: req.StudentNumber,
		FirstName:     strings.TrimSpace(req.FirstName),
		LastName:      strings.TrimSpace(req.LastName),
		Email:         strings.ToLower(req.Email),
		IsStudent:     true,
		Points:        0,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := h.userModel.Create(tx, user); err != nil {
		log.Printf("User insert failed: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	token, err := utils.GenerateRandomToken(32)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if err := h.userModel.StoreVerificationToken(tx, userID, token, time.Now().Add(24*time.Hour)); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := h.emailSender.SendVerificationEmail(req.Email, token); err != nil {
		log.Printf("Email send failed: %v", err)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Registered. Verify your email."})
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	userID, err := h.userModel.VerifyEmailToken(token)
	if err != nil {
		http.Error(w, "Invalid or expired token", http.StatusBadRequest)
		return
	}

	if err := h.userModel.VerifyEmail(userID); err != nil {
		http.Error(w, "Verification failed", http.StatusInternalServerError)
		return
	}

	if err := h.userModel.DeleteVerificationToken(token); err != nil {
		log.Printf("Token delete failed: %v", err)
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Email verified"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if !isValidStudentNumber(req.StudentNumber) {
		http.Error(w, "Invalid student number", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "Password required", http.StatusBadRequest)
		return
	}

	user, err := h.userModel.GetByStudentNumber(req.StudentNumber)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !user.EmailVerified {
		http.Error(w, "Email not verified", http.StatusUnauthorized)
		return
	}

	hash, err := h.userModel.GetPasswordHash(user.ID)
	if err != nil || utils.CheckPassword(hash, req.Password) != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	accessToken, err := utils.GenerateJWT(user, h.cfg.JWT.Secret, 24)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	refreshToken, err := utils.GenerateJWT(user, h.cfg.JWT.Secret, 168)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.Server.Environment == "production",
		MaxAge:   24 * 3600,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.Server.Environment == "production",
		MaxAge:   168 * 3600,
		SameSite: http.SameSiteStrictMode,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":          user,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out"})
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "No refresh token", http.StatusUnauthorized)
		return
	}

	claims, err := utils.ValidateJWT(cookie.Value, h.cfg.JWT.Secret)
	if err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	user, err := h.userModel.GetByID(claims.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	accessToken, err := utils.GenerateJWT(user, h.cfg.JWT.Secret, 24)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.Server.Environment == "production",
		MaxAge:   24 * 3600,
		SameSite: http.SameSiteStrictMode,
	})
	json.NewEncoder(w).Encode(map[string]string{"access_token": accessToken})
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if !isValidEmail(req.Email) {
		http.Error(w, "Invalid email", http.StatusBadRequest)
		return
	}

	user, err := h.userModel.GetByEmail(req.Email)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"message": "If email exists, reset OTP sent"})
		return
	}

	otp, err := utils.GenerateOTP(6)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := h.userModel.StoreOTP(user.ID, otp, time.Now().Add(15*time.Minute)); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := h.emailSender.SendPasswordResetEmail(req.Email, otp); err != nil {
		log.Printf("Email send failed: %v", err)
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "If email exists, reset OTP sent"})
}

func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if !isValidEmail(req.Email) || len(req.OTP) != 6 || !regexp.MustCompile(`^\d{6}$`).MatchString(req.OTP) {
		http.Error(w, "Invalid email or OTP", http.StatusBadRequest)
		return
	}

	user, err := h.userModel.GetByEmail(req.Email)
	if err != nil {
		http.Error(w, "Invalid OTP", http.StatusBadRequest)
		return
	}

	if err := h.userModel.VerifyOTP(user.ID, req.OTP); err != nil {
		http.Error(w, "Invalid or expired OTP", http.StatusBadRequest)
		return
	}

	resetToken, err := utils.GenerateRandomToken(32)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := h.userModel.StoreResetToken(user.ID, resetToken, time.Now().Add(15*time.Minute)); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"reset_token": resetToken})
}

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

	if !isValidEmail(req.Email) || req.ResetToken == "" || !isValidPassword(req.NewPassword) {
		http.Error(w, "Invalid email, token, or password", http.StatusBadRequest)
		return
	}

	user, err := h.userModel.GetByEmail(req.Email)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if err := h.userModel.VerifyResetToken(user.ID, req.ResetToken); err != nil {
		http.Error(w, "Invalid or expired token", http.StatusBadRequest)
		return
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := h.userModel.UpdatePassword(user.ID, hash); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	if err := h.userModel.DeleteResetToken(user.ID, req.ResetToken); err != nil {
		log.Printf("Token delete failed: %v", err)
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Password reset successful"})
}