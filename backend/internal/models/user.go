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

// Badge represents a badge in the system
type Badge struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	ImageURL        string `json:"image_url"`
	RequirementPoints int    `json:"requirement_points"`
}

// AssignBadge assigns a badge to a user if not already assigned
func (m *UserModel) AssignBadge(userID string, badgeID int) error {
	query := `
		INSERT INTO user_badges (user_id, badge_id, awarded_date)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, badge_id) DO NOTHING
	`
	_, err := m.DB.Exec(query, userID, badgeID, time.Now())
	return err
}

// CheckAndAssignBadges checks for badges a user qualifies for and assigns them
func (m *UserModel) CheckAndAssignBadges(userID string) error {
	user, err := m.GetByID(userID)
	if err != nil {
		return err
	}

	// Get all badges the user doesn't yet have where points meet the requirement
	query := `
		SELECT b.id, b.name, b.description, b.image_url, b.requirement_points
		FROM badges b
		LEFT JOIN user_badges ub ON b.id = ub.badge_id AND ub.user_id = $1
		WHERE b.requirement_points <= $2 AND ub.badge_id IS NULL
	`
	rows, err := m.DB.Query(query, userID, user.Points)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var badge Badge
		err := rows.Scan(&badge.ID, &badge.Name, &badge.Description, &badge.ImageURL, &badge.RequirementPoints)
		if err != nil {
			return err
		}
		if err := m.AssignBadge(userID, badge.ID); err != nil {
			return err
		}
	}
	return rows.Err()
}

// IncrementPointsAndCheckBadges increments a user's points and assigns eligible badges
func (m *UserModel) IncrementPointsAndCheckBadges(userID string, points int) error {
	// Update points
	err := m.UpdatePoints(userID, points)
	if err != nil {
		return err
	}
	// Check and assign badges
	return m.CheckAndAssignBadges(userID)
}

// GetUserBadges retrieves all badges for a user
func (m *UserModel) GetUserBadges(userID string) ([]*Badge, error) {
	query := `
		SELECT b.id, b.name, b.description, b.image_url, b.requirement_points
		FROM badges b
		JOIN user_badges ub ON b.id = ub.badge_id
		WHERE ub.user_id = $1
	`
	rows, err := m.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var badges []*Badge
	for rows.Next() {
		var badge Badge
		err := rows.Scan(&badge.ID, &badge.Name, &badge.Description, &badge.ImageURL, &badge.RequirementPoints)
		if err != nil {
			return nil, err
		}
		badges = append(badges, &badge)
	}
	return badges, rows.Err()
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

// GetAll retrieves all users
func (m *UserModel) GetAll() ([]*User, error) {
	query := `
		SELECT id, first_name, last_name, email, user_type, points, email_verified, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`
	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
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
	return users, rows.Err()
}

// ReportModel handles database operations for reports
type Report struct {
	ID             int       `json:"id"`
	MaterialID     int       `json:"material_id"`
	ReporterID     string    `json:"reporter_id"`
	Reason         string    `json:"reason"`
	AdditionalInfo string    `json:"additional_info"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	ResolvedAt     time.Time `json:"resolved_at,omitempty"`
	ResolvedBy     string    `json:"resolved_by,omitempty"`
	ResolutionNotes string    `json:"resolution_notes,omitempty"`
}

func (m *UserModel) GetReports() ([]*Report, error) {
	query := `
		SELECT id, material_id, reporter_id, reason, additional_info, status, created_at, resolved_at, resolved_by, resolution_notes
		FROM reports
		ORDER BY created_at DESC
	`
	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*Report
	for rows.Next() {
		var r Report
		err := rows.Scan(
			&r.ID,
			&r.MaterialID,
			&r.ReporterID,
			&r.Reason,
			&r.AdditionalInfo,
			&r.Status,
			&r.CreatedAt,
			&r.ResolvedAt,
			&r.ResolvedBy,
			&r.ResolutionNotes,
		)
		if err != nil {
			return nil, err
		}
		reports = append(reports, &r)
	}
	return reports, rows.Err()
}

// ResolveReport resolves a report
func (m *UserModel) ResolveReport(reportID int, resolverID, resolutionNotes string, status string) error {
	query := `
		UPDATE reports
		SET status = $1, resolved_at = $2, resolved_by = $3, resolution_notes = $4
		WHERE id = $5
	`
	_, err := m.DB.Exec(query, status, time.Now(), resolverID, resolutionNotes, reportID)
	return err
}