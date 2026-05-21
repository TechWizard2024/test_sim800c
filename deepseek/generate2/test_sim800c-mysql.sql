-- Se connecter en tant que root
mysql -u root -p

-- Créer la base de données
CREATE DATABASE sim800c_manager_deepseekv1 CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Créer l'utilisateur
CREATE USER 'sim800c_user'@'localhost' IDENTIFIED BY 'SIM800c@2026!';

-- Accorder les droits
GRANT ALL PRIVILEGES ON sim800c_manager_deepseekv1.* TO 'sim800c_user'@'localhost';
FLUSH PRIVILEGES;