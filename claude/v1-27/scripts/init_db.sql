-- ============================================================
-- SIM800C Supervisor — Script d'initialisation de la base de données
-- Version : v1-27 (synchronisée avec db.go après MICRO-BLOCS A1, C5)
-- Base de données : sim800c_manager_deepseekv1
-- Moteur : MariaDB 10.4+ / MySQL 5.7+
-- ============================================================

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
SET time_zone = "+00:00";
SET NAMES utf8mb4;

CREATE DATABASE IF NOT EXISTS `sim800c_manager_deepseekv1`
  CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

USE `sim800c_manager_deepseekv1`;

-- ============================================================
-- TABLE : audit_log
-- ============================================================
CREATE TABLE IF NOT EXISTS `audit_log` (
  `id`          INT(11)       NOT NULL AUTO_INCREMENT,
  `user_id`     VARCHAR(50)   DEFAULT NULL,
  `action`      VARCHAR(100)  NOT NULL,
  `target_type` VARCHAR(50)   DEFAULT NULL,
  `target_id`   INT(11)       DEFAULT NULL,
  `details`     LONGTEXT      CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `ip_address`  VARCHAR(45)   DEFAULT NULL,
  `created_at`  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_audit_user`    (`user_id`),
  KEY `idx_audit_created` (`created_at`),
  KEY `idx_audit_action`  (`action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ============================================================
-- TABLE : excel_versions
-- ============================================================
CREATE TABLE IF NOT EXISTS `excel_versions` (
  `id`               INT(11)      NOT NULL AUTO_INCREMENT,
  `filename`         VARCHAR(255) NOT NULL,
  `version_date`     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `created_by`       VARCHAR(50)  DEFAULT 'system',
  `new_codes_count`  INT(11)      DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_version_date` (`version_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ============================================================
-- TABLE : modules
-- ============================================================
CREATE TABLE IF NOT EXISTS `modules` (
  `id`           INT(11)      NOT NULL AUTO_INCREMENT,
  `com_port`     VARCHAR(10)  NOT NULL,
  `imei`         VARCHAR(15)  DEFAULT NULL,
  `phone_number` VARCHAR(20)  DEFAULT NULL,
  `carrier`      VARCHAR(50)  DEFAULT NULL,
  `status`       ENUM('connected','disconnected','error') DEFAULT 'disconnected',
  `last_seen`    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `created_at`   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_com_port` (`com_port`),
  KEY `idx_status`   (`status`),
  KEY `idx_com_port` (`com_port`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ============================================================
-- TABLE : sms_messages (avec colonne is_read — MICRO-BLOC A1)
-- ============================================================
CREATE TABLE IF NOT EXISTS `sms_messages` (
  `id`              INT(11)      NOT NULL AUTO_INCREMENT,
  `module_id`       INT(11)      NOT NULL,
  `sender_number`   VARCHAR(20)  DEFAULT NULL,
  `receiver_number` VARCHAR(20)  DEFAULT NULL,
  `message`         TEXT         NOT NULL,
  `direction`       ENUM('in','out') NOT NULL,
  `is_deleted`      TINYINT(1)   DEFAULT 0,
  `is_trash`        TINYINT(1)   DEFAULT 0,
  `is_read`         BOOLEAN      DEFAULT FALSE,
  `sms_index`       INT(11)      DEFAULT NULL,
  `received_at`     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_module_direction` (`module_id`, `direction`),
  KEY `idx_received_at`      (`received_at`),
  KEY `idx_is_trash`         (`is_trash`),
  KEY `idx_is_read`          (`module_id`, `is_read`),
  CONSTRAINT `sms_messages_ibfk_1`
    FOREIGN KEY (`module_id`) REFERENCES `modules` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ============================================================
-- TABLE : users
-- ============================================================
CREATE TABLE IF NOT EXISTS `users` (
  `id`            VARCHAR(36)  NOT NULL,
  `username`      VARCHAR(50)  NOT NULL,
  `password_hash` VARCHAR(255) NOT NULL,
  `role`          ENUM('admin','operator','viewer') DEFAULT 'viewer',
  `created_at`    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_username` (`username`),
  KEY `idx_users_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Compte admin par défaut : admin / admin123
INSERT IGNORE INTO `users` (`id`, `username`, `password_hash`, `role`)
VALUES ('admin-001', 'admin', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LXdeXkYrG9iKj5FJK', 'admin');

-- ============================================================
-- TABLE : ussd_favorites
-- ============================================================
CREATE TABLE IF NOT EXISTS `ussd_favorites` (
  `id`           INT(11)      NOT NULL AUTO_INCREMENT,
  `user_id`      VARCHAR(50)  NOT NULL,
  `ussd_code_id` INT(11)      DEFAULT NULL,
  `ussd_code`    VARCHAR(50)  NOT NULL,
  `carrier`      VARCHAR(50)  DEFAULT NULL,
  `operation`    VARCHAR(100) DEFAULT NULL,
  `created_at`   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_uf_user`    (`user_id`),
  KEY `idx_uf_carrier` (`carrier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ============================================================
-- TABLE : ussd_history
-- ============================================================
CREATE TABLE IF NOT EXISTS `ussd_history` (
  `id`          INT(11)      NOT NULL AUTO_INCREMENT,
  `module_id`   INT(11)      NOT NULL,
  `ussd_code`   VARCHAR(50)  NOT NULL,
  `input_data`  TEXT         DEFAULT NULL,
  `output_data` TEXT         DEFAULT NULL,
  `status`      ENUM('success','error','timeout') NOT NULL,
  `duration_ms` INT(11)      DEFAULT NULL,
  `executed_by` VARCHAR(50)  DEFAULT 'system',
  `executed_at` TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_uh_module`      (`module_id`),
  KEY `idx_uh_executed_at` (`executed_at`),
  KEY `idx_uh_status`      (`status`),
  CONSTRAINT `ussd_history_ibfk_1`
    FOREIGN KEY (`module_id`) REFERENCES `modules` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ============================================================
-- TABLE : dial_plan
-- ============================================================
CREATE TABLE IF NOT EXISTS `dial_plan` (
  `id`            INT(11)      NOT NULL AUTO_INCREMENT,
  `country_code`  VARCHAR(5)   NOT NULL,
  `country_name`  VARCHAR(100) NOT NULL,
  `calling_code`  VARCHAR(10)  NOT NULL,
  `number_length` INT(11)      NOT NULL DEFAULT 10,
  `operator`      VARCHAR(100) NOT NULL,
  `prefix`        VARCHAR(10)  NOT NULL,
  `is_active`     TINYINT(1)   DEFAULT 1,
  `created_at`    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_country_operator_prefix` (`country_code`, `operator`, `prefix`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Données CI par défaut
INSERT IGNORE INTO `dial_plan`
  (`country_code`, `country_name`, `calling_code`, `number_length`, `operator`, `prefix`, `is_active`)
VALUES
  ('CI', 'Côte d''Ivoire', '+225', 10, 'Orange CI',     '07', 1),
  ('CI', 'Côte d''Ivoire', '+225', 10, 'MTN CI',        '05', 1),
  ('CI', 'Côte d''Ivoire', '+225', 10, 'Moov Africa CI','01', 1);

-- ============================================================
-- TABLE : app_settings (persistance paramètres application)
-- ============================================================
CREATE TABLE IF NOT EXISTS `app_settings` (
  `setting_key`   VARCHAR(100) NOT NULL,
  `setting_value` TEXT         NOT NULL,
  `updated_at`    TIMESTAMP    DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`setting_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- TABLE : signal_log (historique qualité signal — MICRO-BLOC C5)
-- ============================================================
CREATE TABLE IF NOT EXISTS `signal_log` (
  `id`             INT(11)      NOT NULL AUTO_INCREMENT,
  `module_id`      INT(11)      NOT NULL,
  `csq`            INT(11)      NOT NULL COMMENT 'Valeur brute AT+CSQ (0-31)',
  `rssi`           FLOAT        NOT NULL COMMENT 'RSSI en dBm (-113 + 2*csq)',
  `network_status` VARCHAR(20)  DEFAULT NULL,
  `logged_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_sl_module`    (`module_id`),
  KEY `idx_sl_logged_at` (`logged_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ============================================================
-- LOG : initialisation
-- ============================================================
INSERT INTO `audit_log` (`user_id`, `action`, `details`)
VALUES (NULL, 'database_initialized', '{"version": "1.27.0", "script": "init_db.sql"}');

