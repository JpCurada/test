package models

import (
	"database/sql"
	"errors"
	"time"
)

type UserType string

const (
	UserTypeStudent UserType = "STUDENT"
	UserTypeAdmin   UserType = "ADMIN"
)

// User model represents a user in the system
type User struct {
	ID            string    `json:"id"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Email         string    `json:"email"`
	UserType      UserType  `json:"user_type"`
	Points        int       `json:"points"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UserModel handles database operations for users
type UserModel struct {
	DB *sql.DB
}

// NewUserModel creates a new UserModel
func NewUserModel(db *sql.DB) *UserModel {
	return &UserModel{DB: db}
}

// GetByID retrieves a user by ID
func (m *UserModel) GetByID(id string) (*User, error) {
	query := `
		SELECT id, first_name, last_name, email, user_type, points, email_verified, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := m.DB.QueryRow(query, id).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.UserType,
		&user.Points,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (m *UserModel) GetByEmail(email string) (*User, error) {
    query := `
        SELECT id, first_name, last_name, email, user_type, points, email_verified, created_at, updated_at
        FROM users
        WHERE email = $1
    `

    var user User
    err := m.DB.QueryRow(query, email).Scan(
        &user.ID,
        &user.FirstName,
        &user.LastName,
        &user.Email,
        &user.UserType,
        &user.Points,
        &user.EmailVerified,
        &user.CreatedAt,
        &user.UpdatedAt,
    )

    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, sql.ErrNoRows // Return sql.ErrNoRows directly
        }
        return nil, err
    }

    return &user, nil
}

func (m *UserModel) GetByStudentNumber(studentNumber string) (*User, error) {
    query := `
        SELECT id, first_name, last_name, email, user_type, points, email_verified, created_at, updated_at
        FROM users
        WHERE id = $1
    `

    var user User
    err := m.DB.QueryRow(query, studentNumber).Scan(
        &user.ID,
        &user.FirstName,
        &user.LastName,
        &user.Email,
        &user.UserType,
        &user.Points,
        &user.EmailVerified,
        &user.CreatedAt,
        &user.UpdatedAt,
    )

    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, sql.ErrNoRows // Return sql.ErrNoRows directly
        }
        return nil, err
    }

    return &user, nil
}

// Create inserts a new user into the database
func (m *UserModel) Create(user *User, passwordHash string) error {
	// Start transaction
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert user credentials
	credQuery := `
		INSERT INTO user_credentials (user_id, email, password_hash, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.Exec(credQuery, user.ID, user.Email, passwordHash, time.Now())
	if err != nil {
		return err
	}

	// Insert user details
	userQuery := `
		INSERT INTO users (id, first_name, last_name, email, user_type, points, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = tx.Exec(
		userQuery,
		user.ID,
		user.FirstName,
		user.LastName,
		user.Email,
		user.UserType,
		user.Points,
		user.EmailVerified,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return err
	}

	// Commit transaction
	return tx.Commit()
}

// Update updates user information
func (m *UserModel) Update(user *User) error {
	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, updated_at = $3
		WHERE id = $4
	`
	_, err := m.DB.Exec(query, user.FirstName, user.LastName, time.Now(), user.ID)
	return err
}

// UpdatePoints updates user points
func (m *UserModel) UpdatePoints(userID string, points int) error {
	query := `
		UPDATE users
		SET points = points + $1, updated_at = $2
		WHERE id = $3
	`
	_, err := m.DB.Exec(query, points, time.Now(), userID)
	return err
}

// GetLeaderboard retrieves the top users by points
func (m *UserModel) GetLeaderboard(limit int) ([]*User, error) {
	query := `
		SELECT id, first_name, last_name, email, user_type, points, email_verified, created_at, updated_at
		FROM users
		WHERE user_type = $1
		ORDER BY points DESC
		LIMIT $2
	`

	rows, err := m.DB.Query(query, UserTypeStudent, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&user.UserType,
			&user.Points,
			&user.EmailVerified,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// VerifyEmail updates a user's email verification status
func (m *UserModel) VerifyEmail(userID string) error {
	query := `
		UPDATE users
		SET email_verified = true, updated_at = $1
		WHERE id = $2
	`
	_, err := m.DB.Exec(query, time.Now(), userID)
	return err
}

// GetPasswordHash retrieves a user's password hash
func (m *UserModel) GetPasswordHash(studentNumber string) (string, error) {
	query := `
		SELECT password_hash
		FROM user_credentials
		WHERE user_id = $1
	`

	var passwordHash string
	err := m.DB.QueryRow(query, studentNumber).Scan(&passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("user not found")
		}
		return "", err
	}

	return passwordHash, nil
}

// UpdatePassword updates a user's password
func (m *UserModel) UpdatePassword(userID, passwordHash string) error {
	query := `
		UPDATE user_credentials
		SET password_hash = $1
		WHERE user_id = $2
	`
	_, err := m.DB.Exec(query, passwordHash, userID)
	return err
}

// StoreVerificationToken stores a verification token for a user
func (m *UserModel) StoreVerificationToken(userID string, token string, expiresAt time.Time) error {
    query := `
        INSERT INTO email_verifications (user_id, token, expires_at, created_at)
        VALUES ($1, $2, $3, $4)
    `
    _, err := m.DB.Exec(query, userID, token, expiresAt, time.Now())
    return err
}

// VerifyEmailToken verifies a verification token and returns the user ID
func (m *UserModel) VerifyEmailToken(token string) (string, error) {
    query := `
        SELECT user_id
        FROM email_verifications
        WHERE token = $1 AND expires_at > NOW()
    `
    var userID string
    err := m.DB.QueryRow(query, token).Scan(&userID)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return "", errors.New("invalid or expired token")
        }
        return "", err
    }
    return userID, nil
}

// DeleteVerificationToken deletes a verification token after use
func (m *UserModel) DeleteVerificationToken(token string) error {
    query := `
        DELETE FROM email_verifications
        WHERE token = $1
    `
    _, err := m.DB.Exec(query, token)
    return err
}