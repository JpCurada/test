package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID            int       `json:"id"`
	StudentNumber string    `json:"student_number,omitempty"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Email         string    `json:"email"`
	IsStudent     bool      `json:"is_student"`
	Points        int       `json:"points"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type UserModel struct {
	db *sql.DB
}

func NewUserModel(db *sql.DB) *UserModel {
	return &UserModel{db: db}
}

func (m *UserModel) Create(tx *sql.Tx, user *User) error {
	query := `
		INSERT INTO users (id, student_number, first_name, last_name, email, is_student, points, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := tx.Exec(query, user.ID, user.StudentNumber, user.FirstName, user.LastName, user.Email, user.IsStudent, user.Points, user.EmailVerified, user.CreatedAt, user.UpdatedAt)
	return err
}

func (m *UserModel) GetByID(id int) (*User, error) {
	query := `
		SELECT id, student_number, first_name, last_name, email, is_student, points, email_verified, created_at, updated_at
		FROM users WHERE id = $1
	`
	var user User
	err := m.db.QueryRow(query, id).Scan(&user.ID, &user.StudentNumber, &user.FirstName, &user.LastName, &user.Email, &user.IsStudent, &user.Points, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	return &user, err
}

func (m *UserModel) GetByEmail(email string) (*User, error) {
	query := `
		SELECT id, student_number, first_name, last_name, email, is_student, points, email_verified, created_at, updated_at
		FROM users WHERE email = $1
	`
	var user User
	err := m.db.QueryRow(query, email).Scan(&user.ID, &user.StudentNumber, &user.FirstName, &user.LastName, &user.Email, &user.IsStudent, &user.Points, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	return &user, err
}

func (m *UserModel) GetByStudentNumber(studentNumber string) (*User, error) {
	query := `
		SELECT id, student_number, first_name, last_name, email, is_student, points, email_verified, created_at, updated_at
		FROM users WHERE student_number = $1
	`
	var user User
	err := m.db.QueryRow(query, studentNumber).Scan(&user.ID, &user.StudentNumber, &user.FirstName, &user.LastName, &user.Email, &user.IsStudent, &user.Points, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	return &user, err
}

func (m *UserModel) Update(user *User) error {
	query := `
		UPDATE users SET first_name = $1, last_name = $2, updated_at = $3
		WHERE id = $4
	`
	_, err := m.db.Exec(query, user.FirstName, user.LastName, time.Now(), user.ID)
	return err
}

func (m *UserModel) Delete(id int) error {
	_, err := m.db.Exec("DELETE FROM users WHERE id = $1", id)
	return err
}

func (m *UserModel) GetAll() ([]*User, error) {
	query := `
		SELECT id, student_number, first_name, last_name, email, is_student, points, email_verified, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.StudentNumber, &u.FirstName, &u.LastName, &u.Email, &u.IsStudent, &u.Points, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (m *UserModel) GetLeaderboard(limit int) ([]*User, error) {
	query := `
		SELECT id, student_number, first_name, last_name, email, is_student, points, email_verified, created_at, updated_at
		FROM users WHERE is_student = true
		ORDER BY points DESC LIMIT $1
	`
	rows, err := m.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.StudentNumber, &u.FirstName, &u.LastName, &u.Email, &u.IsStudent, &u.Points, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (m *UserModel) IncrementPointsAndCheckBadges(userID, points int) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := "UPDATE users SET points = points + $1, updated_at = $2 WHERE id = $3 RETURNING points"
	var newPoints int
	if err := tx.QueryRow(query, points, time.Now(), userID).Scan(&newPoints); err != nil {
		return err
	}

	badgesQuery := `
		SELECT id FROM badges
		WHERE points_required <= $1
		AND id NOT IN (SELECT badge_id FROM user_badges WHERE user_id = $2)
	`
	rows, err := tx.Query(badgesQuery, newPoints, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var badgeID int
		if err := rows.Scan(&badgeID); err != nil {
			return err
		}
		_, err := tx.Exec("INSERT INTO user_badges (user_id, badge_id, awarded_at) VALUES ($1, $2, $3)", userID, badgeID, time.Now())
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (m *UserModel) VerifyEmail(userID int) error {
	_, err := m.db.Exec("UPDATE users SET email_verified = true, updated_at = $1 WHERE id = $2", time.Now(), userID)
	return err
}

func (m *UserModel) GetPasswordHash(userID int) (string, error) {
	var hash string
	err := m.db.QueryRow("SELECT password_hash FROM user_credentials WHERE id = $1", userID).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", sql.ErrNoRows
	}
	return hash, err
}

func (m *UserModel) UpdatePassword(userID int, hash string) error {
	_, err := m.db.Exec("UPDATE user_credentials SET password_hash = $1 WHERE id = $2", hash, userID)
	return err
}

func (m *UserModel) StoreVerificationToken(tx *sql.Tx, userID int, token string, expiresAt time.Time) error {
	_, err := tx.Exec("INSERT INTO email_verifications (user_id, token, expires_at, created_at) VALUES ($1, $2, $3, $4)", userID, token, expiresAt, time.Now())
	return err
}

func (m *UserModel) VerifyEmailToken(token string) (int, error) {
	var userID int
	err := m.db.QueryRow("SELECT user_id FROM email_verifications WHERE token = $1 AND expires_at > NOW()", token).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, sql.ErrNoRows
	}
	return userID, err
}

func (m *UserModel) DeleteVerificationToken(token string) error {
	_, err := m.db.Exec("DELETE FROM email_verifications WHERE token = $1", token)
	return err
}

func (m *UserModel) StoreOTP(userID int, otp string, expiresAt time.Time) error {
	_, err := m.db.Exec("INSERT INTO reset_tokens (user_id, token, expires_at, created_at) VALUES ($1, $2, $3, $4)", userID, otp, expiresAt, time.Now())
	return err
}

func (m *UserModel) VerifyOTP(userID int, otp string) error {
	var id int
	err := m.db.QueryRow("SELECT id FROM reset_tokens WHERE user_id = $1 AND token = $2 AND expires_at > NOW()", userID, otp).Scan(&id)
	if err == sql.ErrNoRows {
		return sql.ErrNoRows
	}
	if err != nil {
		return err
	}
	_, err = m.db.Exec("DELETE FROM reset_tokens WHERE id = $1", id)
	return err
}

func (m *UserModel) StoreResetToken(userID int, token string, expiresAt time.Time) error {
	_, err := m.db.Exec("INSERT INTO reset_tokens (user_id, token, expires_at, created_at) VALUES ($1, $2, $3, $4)", userID, token, expiresAt, time.Now())
	return err
}

func (m *UserModel) VerifyResetToken(userID int, token string) error {
	var id int
	err := m.db.QueryRow("SELECT id FROM reset_tokens WHERE user_id = $1 AND token = $2 AND expires_at > NOW()", userID, token).Scan(&id)
	if err == sql.ErrNoRows {
		return sql.ErrNoRows
	}
	return err
}

func (m *UserModel) DeleteResetToken(userID int, token string) error {
	_, err := m.db.Exec("DELETE FROM reset_tokens WHERE user_id = $1 AND token = $2", userID, token)
	return err
}
