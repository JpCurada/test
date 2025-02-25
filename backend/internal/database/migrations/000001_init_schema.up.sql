-- internal/database/migrations/000001_init_schema.up.sql
CREATE TABLE user_credentials (
    user_id VARCHAR(20) PRIMARY KEY,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id VARCHAR(20) PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    user_type VARCHAR(20) NOT NULL CHECK (user_type IN ('STUDENT', 'ADMIN')),
    points INTEGER NOT NULL DEFAULT 0,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (id) REFERENCES user_credentials(user_id)
);

CREATE TABLE materials (
    id SERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    subject VARCHAR(50) NOT NULL,
    college VARCHAR(50) NOT NULL,
    course VARCHAR(50) NOT NULL,
    file_url VARCHAR(255) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    uploader_id VARCHAR(20) NOT NULL,
    upload_date TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (uploader_id) REFERENCES users(id)
);

CREATE TABLE votes (
    id SERIAL PRIMARY KEY,
    material_id INTEGER NOT NULL,
    user_id VARCHAR(20) NOT NULL,
    vote_type VARCHAR(10) NOT NULL CHECK (vote_type IN ('UPVOTE', 'DOWNVOTE')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(material_id, user_id)
);

CREATE TABLE badges (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    image_url VARCHAR(255) NOT NULL,
    requirement_points INTEGER NOT NULL
);

CREATE TABLE user_badges (
    user_id VARCHAR(20) NOT NULL,
    badge_id INTEGER NOT NULL,
    awarded_date TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (badge_id) REFERENCES badges(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, badge_id)
);

CREATE TABLE bookmarks (
    id SERIAL PRIMARY KEY,
    material_id INTEGER NOT NULL,
    user_id VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(material_id, user_id)
);

CREATE TABLE reports (
    id SERIAL PRIMARY KEY,
    material_id INTEGER NOT NULL,
    reporter_id VARCHAR(20) NOT NULL,
    reason VARCHAR(100) NOT NULL,
    additional_info TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'RESOLVED', 'DISMISSED')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP,
    resolved_by VARCHAR(20),
    resolution_notes TEXT,
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE CASCADE,
    FOREIGN KEY (reporter_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (resolved_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE email_verifications (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(20) NOT NULL,
    token VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE password_resets (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(20) NOT NULL,
    otp VARCHAR(10) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Insert default badges
INSERT INTO badges (name, description, image_url, requirement_points) VALUES
('Freshie Fighter', 'You have started your journey as a contributor!', '/badges/freshie-fighter.png', 0),
('Dean''s Defender', 'You have become a respected contributor!', '/badges/deans-defender.png', 50),
('Supreme ISKOlar', 'You are among the elite contributors!', '/badges/supreme-iskolar.png', 100);