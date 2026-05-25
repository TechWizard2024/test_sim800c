package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"sim800c-supervisor/internal/config"

	_ "github.com/go-sql-driver/mysql"
)

type Module struct {
	ID          int       `json:"id"`
	COMPort     string    `json:"com_port"`
	IMEI        string    `json:"imei"`
	PhoneNumber string    `json:"phone_number"`
	Carrier     string    `json:"carrier"`
	Status      string    `json:"status"`
	LastSeen    time.Time `json:"last_seen"`
	CreatedAt   time.Time `json:"created_at"`
}

type USSDHistory struct {
	ID         int       `json:"id"`
	ModuleID   int       `json:"module_id"`
	USSDCode   string    `json:"ussd_code"`
	InputData  string    `json:"input_data"`
	OutputData string    `json:"output_data"`
	Status     string    `json:"status"`
	DurationMs int       `json:"duration_ms"`
	ExecutedBy string    `json:"executed_by"`
	ExecutedAt time.Time `json:"executed_at"`
}

type SMSMessage struct {
	ID             int       `json:"id"`
	ModuleID       int       `json:"module_id"`
	SenderNumber   string    `json:"sender_number"`
	ReceiverNumber string    `json:"receiver_number"`
	Message        string    `json:"message"`
	Direction      string    `json:"direction"`
	IsDeleted      bool      `json:"is_deleted"`
	IsTrash        bool      `json:"is_trash"`
	SMSIndex       int       `json:"sms_index"`
	ReceivedAt     time.Time `json:"received_at"`
}

type DB struct {
	*sql.DB
}

// Added 21052026-2002
// User structure
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// ExcelVersion structure
type ExcelVersion struct {
	ID            int       `json:"id"`
	Filename      string    `json:"filename"`
	VersionDate   time.Time `json:"version_date"`
	CreatedBy     string    `json:"created_by"`
	NewCodesCount int       `json:"new_codes_count"`
}

// AuditLog structure
type AuditLog struct {
	ID         int                    `json:"id"`
	UserID     string                 `json:"user_id"`
	Action     string                 `json:"action"`
	TargetType string                 `json:"target_type"`
	TargetID   int                    `json:"target_id"`
	Details    map[string]interface{} `json:"details"`
	IPAddress  string                 `json:"ip_address"`
	CreatedAt  time.Time              `json:"created_at"`
}

//
// Added 21052026-2002

func InitDB(cfg *config.Config) (*DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("erreur ouverture DB: %w", err)
	}

	db.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.MySQL.ConnMaxLifetimeMinutes) * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erreur ping DB: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("erreur création tables: %w", err)
	}

	return &DB{db}, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS modules (
			id INT AUTO_INCREMENT PRIMARY KEY,
			com_port VARCHAR(10) NOT NULL UNIQUE,
			imei VARCHAR(15),
			phone_number VARCHAR(20),
			carrier VARCHAR(50),
			status ENUM('connected', 'disconnected', 'error') DEFAULT 'disconnected',
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS ussd_history (
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
			INDEX idx_executed_at (executed_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS sms_messages (
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
			INDEX idx_received_at (received_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS audit_log (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id VARCHAR(50),
			action VARCHAR(100) NOT NULL,
			target_type VARCHAR(50),
			target_id INT,
			details JSON,
			ip_address VARCHAR(45),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_user (user_id),
			INDEX idx_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS excel_versions (
			id INT AUTO_INCREMENT PRIMARY KEY,
			filename VARCHAR(255) NOT NULL,
			version_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_by VARCHAR(50) DEFAULT 'system',
			new_codes_count INT DEFAULT 0
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("erreur exécution requête: %w\nRequête: %s", err, query)
		}
	}

	return nil
}

// GetModuleByCOMPort - Récupère un module par son port COM
func (db *DB) GetModuleByCOMPort(comPort string) (*Module, error) {
	query := `SELECT id, com_port, imei, phone_number, carrier, status, last_seen, created_at 
			  FROM modules WHERE com_port = ?`

	row := db.QueryRow(query, comPort)

	var module Module
	err := row.Scan(&module.ID, &module.COMPort, &module.IMEI, &module.PhoneNumber,
		&module.Carrier, &module.Status, &module.LastSeen, &module.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &module, nil
}

// SaveModule - Sauvegarde un module
func (db *DB) SaveModule(module *Module) error {
	query := `INSERT INTO modules (com_port, imei, phone_number, carrier, status, last_seen) 
			  VALUES (?, ?, ?, ?, ?, NOW())
			  ON DUPLICATE KEY UPDATE 
			  imei = VALUES(imei), phone_number = VALUES(phone_number), 
			  carrier = VALUES(carrier), status = VALUES(status), last_seen = NOW()`

	_, err := db.Exec(query, module.COMPort, module.IMEI, module.PhoneNumber, module.Carrier, module.Status)
	return err
}

// GetAllModules - Récupère tous les modules
func (db *DB) GetAllModules() ([]Module, error) {
	query := `SELECT id, com_port, imei, phone_number, carrier, status, last_seen, created_at 
			  FROM modules ORDER BY id`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []Module
	for rows.Next() {
		var module Module
		err := rows.Scan(&module.ID, &module.COMPort, &module.IMEI, &module.PhoneNumber,
			&module.Carrier, &module.Status, &module.LastSeen, &module.CreatedAt)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}

	return modules, nil
}

// SaveUSSDHistory - Sauvegarde l'historique USSD
func (db *DB) SaveUSSDHistory(history *USSDHistory) error {
	query := `INSERT INTO ussd_history (module_id, ussd_code, input_data, output_data, status, duration_ms, executed_by) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query, history.ModuleID, history.USSDCode, history.InputData,
		history.OutputData, history.Status, history.DurationMs, history.ExecutedBy)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	history.ID = int(id)
	return nil
}

// GetUSSDHistory - Récupère l'historique USSD
func (db *DB) GetUSSDHistory(moduleID int, limit int) ([]USSDHistory, error) {
	query := `SELECT id, module_id, ussd_code, input_data, output_data, status, duration_ms, executed_by, executed_at 
			  FROM ussd_history WHERE module_id = ? ORDER BY executed_at DESC LIMIT ?`

	rows, err := db.Query(query, moduleID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []USSDHistory
	for rows.Next() {
		var h USSDHistory
		err := rows.Scan(&h.ID, &h.ModuleID, &h.USSDCode, &h.InputData, &h.OutputData,
			&h.Status, &h.DurationMs, &h.ExecutedBy, &h.ExecutedAt)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, nil
}

// SaveSMS - Sauvegarde un SMS
func (db *DB) SaveSMS(sms *SMSMessage) error {
	query := `INSERT INTO sms_messages (module_id, sender_number, receiver_number, message, direction, is_deleted, is_trash, sms_index) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query, sms.ModuleID, sms.SenderNumber, sms.ReceiverNumber,
		sms.Message, sms.Direction, sms.IsDeleted, sms.IsTrash, sms.SMSIndex)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	sms.ID = int(id)
	return nil
}

// GetSMSByModule - Récupère les SMS d'un module
func (db *DB) GetSMSByModule(moduleID int, includeTrash bool) ([]SMSMessage, error) {
	var query string
	var rows *sql.Rows
	var err error

	if includeTrash {
		query = `SELECT id, module_id, sender_number, receiver_number, message, direction, is_deleted, is_trash, sms_index, received_at 
				 FROM sms_messages WHERE module_id = ? ORDER BY received_at DESC`
		rows, err = db.Query(query, moduleID)
	} else {
		query = `SELECT id, module_id, sender_number, receiver_number, message, direction, is_deleted, is_trash, sms_index, received_at 
				 FROM sms_messages WHERE module_id = ? AND is_trash = FALSE ORDER BY received_at DESC`
		rows, err = db.Query(query, moduleID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var smsList []SMSMessage
	for rows.Next() {
		var sms SMSMessage
		err := rows.Scan(&sms.ID, &sms.ModuleID, &sms.SenderNumber, &sms.ReceiverNumber,
			&sms.Message, &sms.Direction, &sms.IsDeleted, &sms.IsTrash, &sms.SMSIndex, &sms.ReceivedAt)
		if err != nil {
			return nil, err
		}
		smsList = append(smsList, sms)
	}

	return smsList, nil
}

// MarkSMSDeleted - Marque un SMS comme supprimé
func (db *DB) MarkSMSDeleted(moduleID int, smsIndex int) error {
	query := `UPDATE sms_messages SET is_deleted = TRUE WHERE module_id = ? AND sms_index = ?`
	_, err := db.Exec(query, moduleID, smsIndex)
	return err
}

// MoveSMSToTrash - Déplace un SMS vers la corbeille
func (db *DB) MoveSMSToTrash(smsID int) error {
	query := `UPDATE sms_messages SET is_trash = TRUE WHERE id = ?`
	_, err := db.Exec(query, smsID)
	return err
}

//
// Added 21052026-2002

// UserExists - Vérifie si un utilisateur existe
func (db *DB) UserExists(username string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM users WHERE username = ?"
	err := db.QueryRow(query, username).Scan(&count)
	return count > 0, err
}

// CreateUser - Crée un nouvel utilisateur
func (db *DB) CreateUser(user *User) error {
	user.ID = generateUUID()
	query := "INSERT INTO users (id, username, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)"
	_, err := db.Exec(query, user.ID, user.Username, user.PasswordHash, user.Role, user.CreatedAt)
	return err
}

// GetUserByUsername - Récupère un utilisateur par son nom
func (db *DB) GetUserByUsername(username string) (*User, error) {
	query := "SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?"
	row := db.QueryRow(query, username)

	var user User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID - Récupère un utilisateur par son ID
func (db *DB) GetUserByID(id string) (*User, error) {
	query := "SELECT id, username, password_hash, role, created_at FROM users WHERE id = ?"
	row := db.QueryRow(query, id)

	var user User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUserPassword - Met à jour le mot de passe d'un utilisateur
func (db *DB) UpdateUserPassword(userID, newHash string) error {
	query := "UPDATE users SET password_hash = ? WHERE id = ?"
	_, err := db.Exec(query, newHash, userID)
	return err
}

// SaveAuditLog - Sauvegarde un log d'audit
func (db *DB) SaveAuditLog(userID, action, targetType string, targetID int, details interface{}, ipAddress string) error {
	query := "INSERT INTO audit_log (user_id, action, target_type, target_id, details, ip_address) VALUES (?, ?, ?, ?, ?, ?)"
	_, err := db.Exec(query, userID, action, targetType, targetID, details, ipAddress)
	return err
}

// GetAuditLogs - Récupère les logs d'audit
func (db *DB) GetAuditLogs(limit int) ([]AuditLog, error) {
	query := "SELECT id, user_id, action, target_type, target_id, details, ip_address, created_at FROM audit_log ORDER BY created_at DESC LIMIT ?"
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		var detailsJSON []byte
		err := rows.Scan(&log.ID, &log.UserID, &log.Action, &log.TargetType, &log.TargetID, &detailsJSON, &log.IPAddress, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(detailsJSON, &log.Details)
		logs = append(logs, log)
	}
	return logs, nil
}

// SMSExists - Vérifie si un SMS existe déjà
func (db *DB) SMSExists(moduleID, smsIndex int) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM sms_messages WHERE module_id = ? AND sms_index = ?"
	err := db.QueryRow(query, moduleID, smsIndex).Scan(&count)
	return count > 0, err
}

// GetExcelVersions - Récupère les versions du fichier Excel
func (db *DB) GetExcelVersions() ([]ExcelVersion, error) {
	query := "SELECT id, filename, version_date, created_by, new_codes_count FROM excel_versions ORDER BY version_date DESC"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []ExcelVersion
	for rows.Next() {
		var v ExcelVersion
		err := rows.Scan(&v.ID, &v.Filename, &v.VersionDate, &v.CreatedBy, &v.NewCodesCount)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, nil
}

// SaveExcelVersion - Sauvegarde une nouvelle version Excel
func (db *DB) SaveExcelVersion(filename, createdBy string, newCodesCount int) error {
	query := "INSERT INTO excel_versions (filename, created_by, new_codes_count) VALUES (?, ?, ?)"
	_, err := db.Exec(query, filename, createdBy, newCodesCount)
	return err
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

//
// Added 21052026-2002
