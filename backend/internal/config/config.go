package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Email    EmailConfig
}

type ServerConfig struct {
	Port                   string
	Environment            string
	ReadTimeoutSeconds     int
	WriteTimeoutSeconds    int
	IdleTimeoutSeconds     int
	ShutdownTimeoutSeconds int
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	Secret string
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

func New() *Config {
	return &Config{
		Server: ServerConfig{
			Port:                   getEnv("SERVER_PORT", "8080"),
			Environment:            getEnv("ENVIRONMENT", "development"),
			ReadTimeoutSeconds:     getEnvAsInt("SERVER_READ_TIMEOUT", 10),
			WriteTimeoutSeconds:    getEnvAsInt("SERVER_WRITE_TIMEOUT", 10),
			IdleTimeoutSeconds:     getEnvAsInt("SERVER_IDLE_TIMEOUT", 120),
			ShutdownTimeoutSeconds: getEnvAsInt("SERVER_SHUTDOWN_TIMEOUT", 10),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "iskonnect"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "your-secret-key"),
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:     getEnv("SMTP_PORT", "587"),
			SMTPUser:     getEnv("SMTP_USER", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("FROM_EMAIL", "no-reply@iskonnect.com"),
			FromName:     getEnv("FROM_NAME", "ISKOnnect"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, err := strconv.Atoi(getEnv(key, "")); err == nil {
		return value
	}
	return defaultValue
}
