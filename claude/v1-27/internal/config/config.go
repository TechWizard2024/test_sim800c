package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Serial    SerialConfig    `yaml:"serial"`
	MySQL     MySQLConfig     `yaml:"mysql"`
	Excel     ExcelConfig     `yaml:"excel"`
	USSD      USSDConfig      `yaml:"ussd"`
	SMS       SMSConfig       `yaml:"sms"`
	Security  SecurityConfig  `yaml:"security"`
	Logging   LoggingConfig   `yaml:"logging"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
}

type ServerConfig struct {
	Port                int    `yaml:"port"`
	WebsocketPath       string `yaml:"websocket_path"`
	APIPath             string `yaml:"api_path"`
	ReadTimeoutSeconds  int    `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `yaml:"write_timeout_seconds"`
}

type SerialConfig struct {
	Ports               []string      `yaml:"ports"`
	BaudRate            int           `yaml:"baud_rate"`
	DataBits            int           `yaml:"data_bits"`
	StopBits            int           `yaml:"stop_bits"`
	Parity              string        `yaml:"parity"`
	TimeoutSeconds      int           `yaml:"timeout_seconds"`
	ReconnectDelaySeconds int         `yaml:"reconnect_delay_seconds"`
	MaxRetries          int           `yaml:"max_retries"`
	CommandQueueSize    int           `yaml:"command_queue_size"`
}

type MySQLConfig struct {
	Host                   string `yaml:"host"`
	Port                   int    `yaml:"port"`
	User                   string `yaml:"user"`
	Password               string `yaml:"password"`
	Database               string `yaml:"database"`
	MaxOpenConns           int    `yaml:"max_open_conns"`
	MaxIdleConns           int    `yaml:"max_idle_conns"`
	ConnMaxLifetimeMinutes int    `yaml:"conn_max_lifetime_minutes"`
}

type ExcelConfig struct {
	BasePath            string `yaml:"base_path"`
	FilenamePattern     string `yaml:"filename_pattern"`
	ReloadIntervalMinutes int  `yaml:"reload_interval_minutes"`
	BackupEnabled       bool   `yaml:"backup_enabled"`
	MaxVersions         int    `yaml:"max_versions"`
}

type USSDConfig struct {
	MaxMenuDepth              int  `yaml:"max_menu_depth"`
	SessionTimeoutSeconds     int  `yaml:"session_timeout_seconds"`
	DefaultChoiceTimeoutSeconds int `yaml:"default_choice_timeout_seconds"`
	ExploreDelayMs            int  `yaml:"explore_delay_ms"` // delay between auto-explore steps
	NavDelayMs                int  `yaml:"nav_delay_ms"`     // delay between manual navigation steps
	RetryOnError              bool `yaml:"retry_on_error"`
	MaxRetries                int  `yaml:"max_retries"`
}

type SMSConfig struct {
	AutoTrashKeyword      string `yaml:"auto_trash_keyword"`
	MaxSMSPerModule       int    `yaml:"max_sms_per_module"`
	CheckIntervalSeconds  int    `yaml:"check_interval_seconds"`
	StorageMode           string `yaml:"storage_mode"`
}

type SecurityConfig struct {
	JWTSecret          string `yaml:"jwt_secret"`
	JWTExpirationHours int    `yaml:"jwt_expiration_hours"`
	EncryptionKey      string `yaml:"encryption_key"`
	EnableAuth         bool   `yaml:"enable_auth"`
	BcryptCost         int    `yaml:"bcrypt_cost"`
}

type LoggingConfig struct {
	Level       string `yaml:"level"`
	OutputPath  string `yaml:"output_path"`
	MaxSizeMB   int    `yaml:"max_size_mb"`
	MaxBackups  int    `yaml:"max_backups"`
	MaxAgeDays  int    `yaml:"max_age_days"`
}

type MonitoringConfig struct {
	Enabled               bool `yaml:"enabled"`
	CheckIntervalSeconds  int  `yaml:"check_interval_seconds"`
	AlertOnDisconnect     bool `yaml:"alert_on_disconnect"`
	AlertOnErrorThreshold int  `yaml:"alert_on_error_threshold"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture fichier config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("erreur parsing YAML: %w", err)
	}

	// ── MICRO-BLOC D4 : variables d'environnement prioritaires ──────────────

	// JWT secret : SIM800C_JWT_SECRET remplace config.yaml
	if envJWT := os.Getenv("SIM800C_JWT_SECRET"); envJWT != "" {
		cfg.Security.JWTSecret = envJWT
	}

	// Base de données (priorité env > yaml)
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.MySQL.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.MySQL.Port)
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.MySQL.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.MySQL.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.MySQL.Database = v
	}

	// Chemin Excel : normaliser les chemins relatifs (./storage/excel → absolu)
	if v := os.Getenv("EXCEL_PATH"); v != "" {
		cfg.Excel.BasePath = v
	}
	if cfg.Excel.BasePath == "" || cfg.Excel.BasePath == "." {
		cfg.Excel.BasePath = "./storage/excel"
	}

	// ── Valeurs par défaut ───────────────────────────────────────────────────

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8082
	}
	if cfg.Serial.BaudRate == 0 {
		cfg.Serial.BaudRate = 9600
	}
	if cfg.USSD.MaxMenuDepth == 0 {
		cfg.USSD.MaxMenuDepth = 10
	}
	if cfg.USSD.ExploreDelayMs == 0 {
		cfg.USSD.ExploreDelayMs = 3000
	}
	if cfg.USSD.NavDelayMs == 0 {
		cfg.USSD.NavDelayMs = 500
	}
	if cfg.SMS.AutoTrashKeyword == "" {
		cfg.SMS.AutoTrashKeyword = "Test"
	}
	if cfg.Security.JWTSecret == "" {
		cfg.Security.JWTSecret = "SIM800c-Supervisor-Secret-Key-2026"
	}

	return &cfg, nil
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.MySQL.User, c.MySQL.Password, c.MySQL.Host, c.MySQL.Port, c.MySQL.Database)
}

func (c *Config) GetConnectionTimeout() time.Duration {
	return time.Duration(c.Serial.TimeoutSeconds) * time.Second
}