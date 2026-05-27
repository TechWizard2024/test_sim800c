-- =============================================================================
-- reset_admin_password.sql
-- Réinitialise le mot de passe admin à 'admin123'
-- 
-- UTILISATION (Windows) :
--   C:\xampp\mysql\bin\mysql.exe -u root sim800c_manager_deepseekv1 < scripts\reset_admin_password.sql
--
-- IMPORTANT : Après exécution de ce script, REDÉMARREZ l'application (stop_app.bat puis start_app.bat)
-- Le démarrage de l'application détectera le placeholder et régénérera un hash bcrypt valide.
-- =============================================================================

USE sim800c_manager_deepseekv1;

-- Réinitialiser le hash avec un placeholder invalide que l'application remplacera au démarrage
UPDATE users 
SET password_hash = '$2a$12$PLACEHOLDER_INVALID_HASH_WILL_BE_RESET_BY_APP_ON_START'
WHERE username = 'admin';

-- Vérification
SELECT username, role, 
       LEFT(password_hash, 20) AS hash_debut,
       created_at 
FROM users 
WHERE username = 'admin';

SELECT 'DONE: Redémarrez l''application pour activer le nouveau mot de passe admin123' AS instructions;
