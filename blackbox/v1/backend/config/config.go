package config

import (
	"github.com/joho/godotenv"
	"os"
	"strconv"
)

type Config struct {
	Mode string

	App struct {
		FrontendOrigin string
		Port            int
		WSPath          string
	}

	MySQL struct {
		DSN string
	}

	Serial struct {
		BaudRate   int
		Ports      []string
		USSDTimeoutMs int
	}

	Security struct {
		JWTSecret string
		RateLimitPerMinute int
	}

	Excel struct {
		CodesPath string
		ScopeInValue string
	}
}

func Load() Config {
	_ = godotenv.Load()

	c := Config{}
	c.Mode = getenv("APP_MODE", "dev")

	c.App.Port = getenvInt("APP_PORT", 8080)
	c.App.WSPath = getenv("APP_WS_PATH", "/ws")
	c.App.FrontendOrigin = getenv("FRONTEND_ORIGIN", "http://test_sim800c.local")

	c.MySQL.DSN = getenv("MYSQL_DSN", "root:password@tcp(127.0.0.1:3306)/sim800?charset=utf8mb4&parseTime=True&loc=Local")

	c.Serial.BaudRate = getenvInt("SERIAL_BAUDRATE", 115200)
	c.Serial.Ports = getenvCSV("SERIAL_PORTS", "COM5,COM6,COM7")
	c.Serial.USSDTimeoutMs = getenvInt("USSD_TIMEOUT_MS", 45000)

	c.Security.JWTSecret = getenv("JWT_SECRET", "change-me")
	c.Security.RateLimitPerMinute = getenvInt("RATE_LIMIT_PER_MIN", 120)

	c.Excel.CodesPath = getenv("EXCEL_CODES_PATH", "./storage/excel/Codes_USSD_CI.xlsx")
	c.Excel.ScopeInValue = getenv("EXCEL_SCOPE_IN", "In")

	return c
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}
	return def
}

func getenvCSV(k, def string) []string {
	v := os.Getenv(k)
	if v == "" {
		v = def
	}
	// simple split by comma
	parts := []string{}
	cur := ""
	for _, r := range v {
		if r == ',' {
			parts = append(parts, trim(cur))
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		parts = append(parts, trim(cur))
	}
	return parts
}

func trim(s string) string {
	// minimal trim without strings import to keep file tiny
	// remove leading/trailing spaces and tabs
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

