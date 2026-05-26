-- ============================================================
-- Migration v1-13 : ajout colonne is_read dans sms_messages
-- Compatible MySQL 5.7+ / MariaDB 10.4+
-- ============================================================
USE `sim800c_manager_deepseekv1`;

-- Ajouter is_read si absent
SET @col_exists = (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = 'sim800c_manager_deepseekv1'
    AND TABLE_NAME = 'sms_messages'
    AND COLUMN_NAME = 'is_read'
);

SET @sql = IF(@col_exists = 0,
  'ALTER TABLE sms_messages ADD COLUMN is_read BOOLEAN DEFAULT FALSE AFTER is_trash',
  'SELECT "is_read already exists" AS info'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- Ajouter index si absent
SET @idx_exists = (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
  WHERE TABLE_SCHEMA = 'sim800c_manager_deepseekv1'
    AND TABLE_NAME = 'sms_messages'
    AND INDEX_NAME = 'idx_is_read'
);
SET @sql2 = IF(@idx_exists = 0,
  'ALTER TABLE sms_messages ADD INDEX idx_is_read (module_id, is_read)',
  'SELECT "idx_is_read already exists" AS info'
);
PREPARE stmt2 FROM @sql2; EXECUTE stmt2; DEALLOCATE PREPARE stmt2;

SELECT 'Migration v1-13 appliquee avec succes' AS result;
