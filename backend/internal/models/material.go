package models

import (
	"database/sql"
	"errors"
	"time"
)

// Material represents a study material in the system
type Material struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Subject     string    `json:"subject"`
	College     string    `json:"college"`
	Course      string    `json:"course"`
	FileURL     string    `json:"file_url"`
	Filename    string    `json:"filename"`
	UploaderID  string    `json:"uploader_id"`
	UploadDate  time.Time `json:"upload_date"`
	VoteCount   int       `json:"vote_count,omitempty"` // Calculated field
}

// MaterialModel handles database operations for materials
type MaterialModel struct {
	DB *sql.DB
}

// NewMaterialModel creates a new MaterialModel
func NewMaterialModel(db *sql.DB) *MaterialModel {
	return &MaterialModel{DB: db}
}

// Create inserts a new material into the database
func (m *MaterialModel) Create(material *Material) error {
	query := `
		INSERT INTO materials (title, description, subject, college, course, file_url, filename, uploader_id, upload_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	err := m.DB.QueryRow(
		query,
		material.Title,
		material.Description,
		material.Subject,
		material.College,
		material.Course,
		material.FileURL,
		material.Filename,
		material.UploaderID,
		time.Now(),
	).Scan(&material.ID)
	return err
}

// GetByID retrieves a material by ID with vote count
func (m *MaterialModel) GetByID(id int) (*Material, error) {
	query := `
		SELECT m.id, m.title, m.description, m.subject, m.college, m.course, m.file_url, m.filename, m.uploader_id, m.upload_date,
		       (SELECT COUNT(*) FILTER (WHERE vote_type = 'UPVOTE') - COUNT(*) FILTER (WHERE vote_type = 'DOWNVOTE') FROM votes WHERE material_id = m.id) AS vote_count
		FROM materials m
		WHERE m.id = $1
	`
	var material Material
	err := m.DB.QueryRow(query, id).Scan(
		&material.ID,
		&material.Title,
		&material.Description,
		&material.Subject,
		&material.College,
		&material.Course,
		&material.FileURL,
		&material.Filename,
		&material.UploaderID,
		&material.UploadDate,
		&material.VoteCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("material not found")
		}
		return nil, err
	}
	return &material, nil
}

// List retrieves all materials with vote counts
func (m *MaterialModel) List() ([]*Material, error) {
	query := `
		SELECT m.id, m.title, m.description, m.subject, m.college, m.course, m.file_url, m.filename, m.uploader_id, m.upload_date,
		       (SELECT COUNT(*) FILTER (WHERE vote_type = 'UPVOTE') - COUNT(*) FILTER (WHERE vote_type = 'DOWNVOTE') FROM votes WHERE material_id = m.id) AS vote_count
		FROM materials m
		ORDER BY m.upload_date DESC
	`
	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materials []*Material
	for rows.Next() {
		var m Material
		err := rows.Scan(
			&m.ID,
			&m.Title,
			&m.Description,
			&m.Subject,
			&m.College,
			&m.Course,
			&m.FileURL,
			&m.Filename,
			&m.UploaderID,
			&m.UploadDate,
			&m.VoteCount,
		)
		if err != nil {
			return nil, err
		}
		materials = append(materials, &m)
	}
	return materials, rows.Err()
}

// Update updates a material
func (m *MaterialModel) Update(material *Material) error {
	query := `
		UPDATE materials
		SET title = $1, description = $2, subject = $3, college = $4, course = $5, file_url = $6, filename = $7
		WHERE id = $8
	`
	_, err := m.DB.Exec(
		query,
		material.Title,
		material.Description,
		material.Subject,
		material.College,
		material.Course,
		material.FileURL,
		material.Filename,
		material.ID,
	)
	return err
}

// Delete deletes a material
func (m *MaterialModel) Delete(id int) error {
	query := `DELETE FROM materials WHERE id = $1`
	_, err := m.DB.Exec(query, id)
	return err
}

// Vote adds or updates a vote for a material
func (m *MaterialModel) Vote(materialID int, userID string, voteType string) error {
	query := `
		INSERT INTO votes (material_id, user_id, vote_type, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (material_id, user_id)
		DO UPDATE SET vote_type = $3, created_at = $4
	`
	_, err := m.DB.Exec(query, materialID, userID, voteType, time.Now())
	return err
}

// Bookmark adds a bookmark for a material
func (m *MaterialModel) Bookmark(materialID int, userID string) error {
	query := `
		INSERT INTO bookmarks (material_id, user_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (material_id, user_id) DO NOTHING
	`
	_, err := m.DB.Exec(query, materialID, userID, time.Now())
	return err
}

// Report reports a material
func (m *MaterialModel) Report(materialID int, reporterID, reason, additionalInfo string) error {
	query := `
		INSERT INTO reports (material_id, reporter_id, reason, additional_info, status, created_at)
		VALUES ($1, $2, $3, $4, 'PENDING', $5)
	`
	_, err := m.DB.Exec(query, materialID, reporterID, reason, additionalInfo, time.Now())
	return err
}
