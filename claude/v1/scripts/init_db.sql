-- phpMyAdmin SQL Dump
-- version 5.2.1
-- https://www.phpmyadmin.net/
--
-- Hôte : 127.0.0.1
-- Généré le : mar. 26 mai 2026 à 21:07
-- Version du serveur : 10.4.32-MariaDB
-- Version de PHP : 8.2.12

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- Base de données : `sim800c_manager_deepseekv1`
--

-- --------------------------------------------------------

--
-- Structure de la table `audit_log`
--

CREATE TABLE `audit_log` (
  `id` int(11) NOT NULL,
  `user_id` varchar(50) DEFAULT NULL,
  `action` varchar(100) NOT NULL,
  `target_type` varchar(50) DEFAULT NULL,
  `target_id` int(11) DEFAULT NULL,
  `details` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`details`)),
  `ip_address` varchar(45) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Déchargement des données de la table `audit_log`
--

INSERT INTO `audit_log` (`id`, `user_id`, `action`, `target_type`, `target_id`, `details`, `ip_address`, `created_at`) VALUES
(1, NULL, 'database_initialized', NULL, NULL, '{\"version\": \"1.0.0\"}', NULL, '2026-05-21 13:00:37'),
(2, NULL, 'database_initialized', NULL, NULL, '{\"version\": \"1.0.0\"}', NULL, '2026-05-21 13:48:31'),
(3, NULL, 'database_initialized', NULL, NULL, '{\"version\": \"1.0.0\"}', NULL, '2026-05-21 14:06:29'),
(4, NULL, 'database_initialized', NULL, NULL, '{\"version\": \"1.0.0\"}', NULL, '2026-05-21 14:21:58'),
(5, NULL, 'database_initialized', NULL, NULL, '{\"version\": \"1.0.0\"}', NULL, '2026-05-21 14:45:17');

-- --------------------------------------------------------

--
-- Structure de la table `dial_plan`
--

CREATE TABLE `dial_plan` (
  `id` int(11) NOT NULL,
  `country_code` varchar(5) NOT NULL,
  `country_name` varchar(100) NOT NULL,
  `calling_code` varchar(10) NOT NULL,
  `number_length` int(11) NOT NULL DEFAULT 10,
  `operator` varchar(100) NOT NULL,
  `prefix` varchar(10) NOT NULL,
  `is_active` tinyint(1) DEFAULT 1,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Déchargement des données de la table `dial_plan`
--

INSERT INTO `dial_plan` (`id`, `country_code`, `country_name`, `calling_code`, `number_length`, `operator`, `prefix`, `is_active`, `created_at`) VALUES
(1, 'CI', 'C??te d\'Ivoire', '+225', 10, 'Orange CI', '07', 1, '2026-05-24 00:52:29'),
(2, 'CI', 'C??te d\'Ivoire', '+225', 10, 'MTN CI', '05', 1, '2026-05-24 00:52:29'),
(3, 'CI', 'C??te d\'Ivoire', '+225', 10, 'Moov Africa CI', '01', 1, '2026-05-24 00:52:29');

-- --------------------------------------------------------

--
-- Structure de la table `excel_versions`
--

CREATE TABLE `excel_versions` (
  `id` int(11) NOT NULL,
  `filename` varchar(255) NOT NULL,
  `version_date` timestamp NOT NULL DEFAULT current_timestamp(),
  `created_by` varchar(50) DEFAULT 'system',
  `new_codes_count` int(11) DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Structure de la table `modules`
--

CREATE TABLE `modules` (
  `id` int(11) NOT NULL,
  `com_port` varchar(10) NOT NULL,
  `imei` varchar(15) DEFAULT NULL,
  `phone_number` varchar(20) DEFAULT NULL,
  `carrier` varchar(50) DEFAULT NULL,
  `status` enum('connected','disconnected','error') DEFAULT 'disconnected',
  `last_seen` timestamp NOT NULL DEFAULT current_timestamp(),
  `created_at` timestamp NOT NULL DEFAULT current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Structure de la table `sms_messages`
--

CREATE TABLE `sms_messages` (
  `id` int(11) NOT NULL,
  `module_id` int(11) NOT NULL,
  `sender_number` varchar(20) DEFAULT NULL,
  `receiver_number` varchar(20) DEFAULT NULL,
  `message` text NOT NULL,
  `direction` enum('in','out') NOT NULL,
  `is_deleted` tinyint(1) DEFAULT 0,
  `is_trash` tinyint(1) DEFAULT 0,
  `sms_index` int(11) DEFAULT NULL,
  `received_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `is_read` tinyint(1) NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Structure de la table `users`
--

CREATE TABLE `users` (
  `id` varchar(36) NOT NULL,
  `username` varchar(50) NOT NULL,
  `password_hash` varchar(255) NOT NULL,
  `role` enum('admin','operator','viewer') DEFAULT 'viewer',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

--
-- Déchargement des données de la table `users`
--

INSERT INTO `users` (`id`, `username`, `password_hash`, `role`, `created_at`) VALUES
('admin-001', 'admin', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LXdeXkYrG9iKj5FJK', 'admin', '2026-05-21 20:05:38');

-- --------------------------------------------------------

--
-- Structure de la table `ussd_favorites`
--

CREATE TABLE `ussd_favorites` (
  `id` int(11) NOT NULL,
  `user_id` varchar(50) NOT NULL,
  `ussd_code_id` int(11) DEFAULT NULL,
  `ussd_code` varchar(50) NOT NULL,
  `carrier` varchar(50) DEFAULT NULL,
  `operation` varchar(100) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Structure de la table `ussd_history`
--

CREATE TABLE `ussd_history` (
  `id` int(11) NOT NULL,
  `module_id` int(11) NOT NULL,
  `ussd_code` varchar(50) NOT NULL,
  `input_data` text DEFAULT NULL,
  `output_data` text DEFAULT NULL,
  `status` enum('success','error','timeout') NOT NULL,
  `duration_ms` int(11) DEFAULT NULL,
  `executed_by` varchar(50) DEFAULT 'system',
  `executed_at` timestamp NOT NULL DEFAULT current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Index pour les tables déchargées
--

--
-- Index pour la table `audit_log`
--
ALTER TABLE `audit_log`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_user` (`user_id`),
  ADD KEY `idx_created_at` (`created_at`),
  ADD KEY `idx_action` (`action`),
  ADD KEY `idx_audit_user` (`user_id`),
  ADD KEY `idx_audit_created` (`created_at`);

--
-- Index pour la table `dial_plan`
--
ALTER TABLE `dial_plan`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `uq_country_operator_prefix` (`country_code`,`operator`,`prefix`);

--
-- Index pour la table `excel_versions`
--
ALTER TABLE `excel_versions`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_version_date` (`version_date`);

--
-- Index pour la table `modules`
--
ALTER TABLE `modules`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `com_port` (`com_port`),
  ADD KEY `idx_status` (`status`),
  ADD KEY `idx_com_port` (`com_port`);

--
-- Index pour la table `sms_messages`
--
ALTER TABLE `sms_messages`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_module_direction` (`module_id`,`direction`),
  ADD KEY `idx_received_at` (`received_at`),
  ADD KEY `idx_is_trash` (`is_trash`),
  ADD KEY `idx_is_read` (`module_id`,`is_read`);

--
-- Index pour la table `users`
--
ALTER TABLE `users`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `username` (`username`),
  ADD KEY `idx_users_username` (`username`);

--
-- Index pour la table `ussd_favorites`
--
ALTER TABLE `ussd_favorites`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_user` (`user_id`),
  ADD KEY `idx_carrier` (`carrier`);

--
-- Index pour la table `ussd_history`
--
ALTER TABLE `ussd_history`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_module` (`module_id`),
  ADD KEY `idx_executed_at` (`executed_at`),
  ADD KEY `idx_status` (`status`);

--
-- AUTO_INCREMENT pour les tables déchargées
--

--
-- AUTO_INCREMENT pour la table `audit_log`
--
ALTER TABLE `audit_log`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=6;

--
-- AUTO_INCREMENT pour la table `dial_plan`
--
ALTER TABLE `dial_plan`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=4;

--
-- AUTO_INCREMENT pour la table `excel_versions`
--
ALTER TABLE `excel_versions`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT pour la table `modules`
--
ALTER TABLE `modules`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT pour la table `sms_messages`
--
ALTER TABLE `sms_messages`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT pour la table `ussd_favorites`
--
ALTER TABLE `ussd_favorites`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT pour la table `ussd_history`
--
ALTER TABLE `ussd_history`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

--
-- Contraintes pour les tables déchargées
--

--
-- Contraintes pour la table `sms_messages`
--
ALTER TABLE `sms_messages`
  ADD CONSTRAINT `sms_messages_ibfk_1` FOREIGN KEY (`module_id`) REFERENCES `modules` (`id`) ON DELETE CASCADE;

--
-- Contraintes pour la table `ussd_history`
--
ALTER TABLE `ussd_history`
  ADD CONSTRAINT `ussd_history_ibfk_1` FOREIGN KEY (`module_id`) REFERENCES `modules` (`id`) ON DELETE CASCADE;
COMMIT;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
