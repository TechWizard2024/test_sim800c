-- ============================================================
-- Migration v1-25 : création table signal_log (MICRO-BLOC C5)
-- Compatible MySQL 5.7+ / MariaDB 10.4+
-- ============================================================
USE `sim800c_manager_deepseekv1`;

CREATE TABLE IF NOT EXISTS `signal_log` (
  `id`             INT(11)     NOT NULL AUTO_INCREMENT,
  `module_id`      INT(11)     NOT NULL,
  `csq`            INT(11)     NOT NULL COMMENT 'Valeur brute AT+CSQ (0-31)',
  `rssi`           FLOAT       NOT NULL COMMENT 'RSSI en dBm = -113 + 2*csq',
  `network_status` VARCHAR(20) DEFAULT NULL,
  `logged_at`      DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_sl_module`    (`module_id`),
  KEY `idx_sl_logged_at` (`logged_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

SELECT 'Migration v1-25 appliquee avec succes' AS result;
