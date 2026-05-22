-- Ajout de la table users
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('admin', 'operator', 'viewer') DEFAULT 'viewer',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Ajout de l'utilisateur admin par défaut
INSERT INTO users (id, username, password_hash, role) 
VALUES ('admin-001', 'admin', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LXdeXkYrG9iKj5FJK', 'admin')
ON DUPLICATE KEY UPDATE id=id;

-- Ajout de la colonne details_json si elle n'existe pas dans audit_log
ALTER TABLE audit_log MODIFY details JSON;

-- Ajout de l'index sur users
CREATE INDEX idx_users_username ON users(username);

-- Ajout de l'index sur audit_log
CREATE INDEX idx_audit_user ON audit_log(user_id);
CREATE INDEX idx_audit_created ON audit_log(created_at);