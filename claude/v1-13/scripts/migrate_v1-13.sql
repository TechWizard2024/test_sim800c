-- ============================================================
-- Migration v1-13 — MICRO-BLOC A1 : is_read SMS
-- Date : 26 Mai 2026
-- Description :
--   - Ajout colonne is_read (BOOLEAN DEFAULT FALSE) sur sms_messages
--   - Ajout index composite (module_id, is_read) pour GetUnreadSMSCount
-- Exécution :
--   C:\xampp\mysql\bin\mysql.exe -u root sim800c_manager_deepseekv1 < scripts\migrate_v1-13.sql
-- ============================================================

USE sim800c_manager_deepseekv1;

-- Ajouter la colonne is_read si elle n'existe pas encore
ALTER TABLE sms_messages
    ADD COLUMN IF NOT EXISTS is_read BOOLEAN NOT NULL DEFAULT FALSE;

-- Ajouter l'index pour accélérer les requêtes GetUnreadSMSCount
-- (utilisation de CREATE INDEX ... IF NOT EXISTS non supporté en MySQL < 8.0)
-- Script idempotent : DROP + CREATE pour compatibilité MySQL 5.7+
SET @indexExists = (
    SELECT COUNT(*)
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'sms_messages'
      AND index_name = 'idx_is_read'
);

SET @sql = IF(@indexExists = 0,
    'ALTER TABLE sms_messages ADD INDEX idx_is_read (module_id, is_read)',
    'SELECT 1 -- index idx_is_read already exists'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Tous les SMS existants conservent is_read = FALSE (non lus par défaut)
-- Optionnel : marquer tous les SMS sortants (direction='out') comme lus
UPDATE sms_messages SET is_read = TRUE WHERE direction = 'out' AND is_read = FALSE;

SELECT 'Migration v1-13 terminée avec succès.' AS status;
