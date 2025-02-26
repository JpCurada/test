CREATE TABLE user_credentials (
    id SERIAL PRIMARY KEY,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id INTEGER PRIMARY KEY REFERENCES user_credentials(id),
    student_number VARCHAR(20) UNIQUE,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    is_student BOOLEAN NOT NULL DEFAULT TRUE,
    points INTEGER NOT NULL DEFAULT 0,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_student_number ON users(student_number);
CREATE INDEX idx_users_points ON users(points);

CREATE TABLE materials (
    id SERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    subject VARCHAR(50) NOT NULL,
    college VARCHAR(50) NOT NULL,
    course VARCHAR(50) NOT NULL,
    file_url VARCHAR(255) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    uploader_id INTEGER NOT NULL REFERENCES users(id),
    upload_date TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_materials_uploader_id ON materials(uploader_id);
CREATE INDEX idx_materials_upload_date ON materials(upload_date);

CREATE TABLE votes (
    id SERIAL PRIMARY KEY,
    material_id INTEGER NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    vote_type VARCHAR(10) NOT NULL CHECK (vote_type IN ('UPVOTE', 'DOWNVOTE')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (material_id, user_id)
);

CREATE TABLE badges (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    image_url VARCHAR(255) NOT NULL,
    points_required INTEGER NOT NULL
);

CREATE TABLE user_badges (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_id INTEGER NOT NULL REFERENCES badges(id) ON DELETE CASCADE,
    awarded_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, badge_id)
);

CREATE TABLE bookmarks (
    id SERIAL PRIMARY KEY,
    material_id INTEGER NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (material_id, user_id)
);

CREATE TABLE email_verifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE reset_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO badges (name, description, image_url, points_required) VALUES
('Freshie', 'First contribution!', '/badges/freshie.png', 0),
('Scholar', 'Solid contributor!', '/badges/scholar.png', 50),
('Elite', 'Top-tier contributor!', '/badges/elite.png', 100);