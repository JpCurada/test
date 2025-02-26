package models

import (
	"database/sql"
	"time"
)

type Material struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Subject     string    `json:"subject"`
	College     string    `json:"college"`
	Course      string    `json:"course"`
	FileURL     string    `json:"file_url"`
	Filename    string    `json:"filename"`
	UploaderID  int       `json:"uploader_id"`
	UploadDate  time.Time `json:"upload_date"`
	VoteCount   int       `json:"vote_count"`
}

type MaterialModel struct {
	db *sql.DB
}

func NewMaterialModel(db *sql.DB) *MaterialModel {
	return &MaterialModel{db: db}
}

func (m *MaterialModel) Create(material *Material) error {
	query := `
		INSERT INTO materials (title, description, subject, college, course, file_url, filename, uploader_id, upload_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	return m.db.QueryRow(query, material.Title, material.Description, material.Subject, material.College, material.Course, material.FileURL, material.Filename, material.UploaderID, time.Now()).Scan(&material.ID)
}

func (m *MaterialModel) GetByID(id int) (*Material, error) {
	query := `
		SELECT id, title, description, subject, college, course, file_url, filename, uploader_id, upload_date,
		       COALESCE((
		           SELECT SUM(CASE WHEN vote_type = 'UPVOTE' THEN 1 ELSE -1 END)
		           FROM votes WHERE material_id = materials.id
		       ), 0) AS vote_count
		FROM materials WHERE id = $1
	`
	var mat Material
	err := m.db.QueryRow(query, id).Scan(&mat.ID, &mat.Title, &mat.Description, &mat.Subject, &mat.College, &mat.Course, &mat.FileURL, &mat.Filename, &mat.UploaderID, &mat.UploadDate, &mat.VoteCount)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	return &mat, err
}

func (m *MaterialModel) List() ([]*Material, error) {
	query := `
		SELECT id, title, description, subject, college, course, file_url, filename, uploader_id, upload_date,
		       COALESCE((
		           SELECT SUM(CASE WHEN vote_type = 'UPVOTE' THEN 1 ELSE -1 END)
		           FROM votes WHERE material_id = materials.id
		       ), 0) AS vote_count
		FROM materials ORDER BY upload_date DESC
	`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materials []*Material
	for rows.Next() {
		var m Material
		if err := rows.Scan(&m.ID, &m.Title, &m.Description, &m.Subject, &m.College, &m.Course, &m.FileURL, &m.Filename, &m.UploaderID, &m.UploadDate, &m.VoteCount); err != nil {
			return nil, err
		}
		materials = append(materials, &m)
	}
	return materials, rows.Err()
}

func (m *MaterialModel) Update(material *Material) error {
	query := `
		UPDATE materials SET title = $1, description = $2, subject = $3, college = $4, course = $5, file_url = $6, filename = $7
		WHERE id = $8
	`
	_, err := m.db.Exec(query, material.Title, material.Description, material.Subject, material.College, material.Course, material.FileURL, material.Filename, material.ID)
	return err
}

func (m *MaterialModel) Delete(id int) error {
	_, err := m.db.Exec("DELETE FROM materials WHERE id = $1", id)
	return err
}

func (m *MaterialModel) Vote(materialID, userID int, voteType string) error {
	query := `
		INSERT INTO votes (material_id, user_id, vote_type, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (material_id, user_id) DO UPDATE SET vote_type = $3, created_at = $4
	`
	_, err := m.db.Exec(query, materialID, userID, voteType, time.Now())
	return err
}

func (m *MaterialModel) Bookmark(materialID, userID int) error {
	query := `
		INSERT INTO bookmarks (material_id, user_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (material_id, user_id) DO NOTHING
	`
	_, err := m.db.Exec(query, materialID, userID, time.Now())
	return err
}

func (m *MaterialModel) GetBookmarks(userID int) ([]*Material, error) {
	query := `
		SELECT m.id, m.title, m.description, m.subject, m.college, m.course, m.file_url, m.filename, m.uploader_id, m.upload_date,
		       COALESCE((
		           SELECT SUM(CASE WHEN vote_type = 'UPVOTE' THEN 1 ELSE -1 END)
		           FROM votes WHERE material_id = m.id
		       ), 0) AS vote_count
		FROM materials m
		JOIN bookmarks b ON m.id = b.material_id
		WHERE b.user_id = $1
		ORDER BY b.created_at DESC
	`
	rows, err := m.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []*Material
	for rows.Next() {
		var m Material
		if err := rows.Scan(&m.ID, &m.Title, &m.Description, &m.Subject, &m.College, &m.Course, &m.FileURL, &m.Filename, &m.UploaderID, &m.UploadDate, &m.VoteCount); err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, &m)
	}
	return bookmarks, rows.Err()
}
