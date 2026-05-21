-- Script d'initialisation de la base de données SIM800C Supervisor

CREATE DATABASE IF NOT EXISTS sim800c_manager_deepseekv1
CHARACTER SET utf8mb4 
COLLATE utf8mb4_unicode_ci;

USE sim800c_manager_deepseekv1;

-- Tables principales
CREATE TABLE IF NOT EXISTS modules (
    id INT AUTO_INCREMENT PRIMARY KEY,
    com_port VARCHAR(10) NOT NULL UNIQUE,
    imei VARCHAR(15),
    phone_number VARCHAR(20),
    carrier VARCHAR(50),
    status ENUM('connected', 'disconnected', 'error') DEFAULT 'disconnected',
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_com_port (com_port)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ussd_history (
    id INT AUTO_INCREMENT PRIMARY KEY,
    module_id INT NOT NULL,
    ussd_code VARCHAR(50) NOT NULL,
    input_data TEXT,
    output_data TEXT,
    status ENUM('success', 'error', 'timeout') NOT NULL,
    duration_ms INT,
    executed_by VARCHAR(50) DEFAULT 'system',
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE,
    INDEX idx_module (module_id),
    INDEX idx_executed_at (executed_at),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS sms_messages (
    id INT AUTO_INCREMENT PRIMARY KEY,
    module_id INT NOT NULL,
    sender_number VARCHAR(20),
    receiver_number VARCHAR(20),
    message TEXT NOT NULL,
    direction ENUM('in', 'out') NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE,
    is_trash BOOLEAN DEFAULT FALSE,
    sms_index INT,
    received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE,
    INDEX idx_module_direction (module_id, direction),
    INDEX idx_received_at (received_at),
    INDEX idx_is_trash (is_trash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS audit_log (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(50),
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),
    target_id INT,
    details JSON,
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user (user_id),
    INDEX idx_created_at (created_at),
    INDEX idx_action (action)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS excel_versions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    version_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(50) DEFAULT 'system',
    new_codes_count INT DEFAULT 0,
    INDEX idx_version_date (version_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ussd_favorites (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    ussd_code_id INT,
    ussd_code VARCHAR(50) NOT NULL,
    carrier VARCHAR(50),
    operation VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user (user_id),
    INDEX idx_carrier (carrier)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Insérer un utilisateur admin par défaut (mot de passe: admin123)
INSERT INTO audit_log (action, details) VALUES ('database_initialized', '{"version": "1.0.0"}');

-- Vérifier l'installation
SELECT 'Database initialized successfully' AS status;