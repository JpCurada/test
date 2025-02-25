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
	Storage  StorageConfig
}

type ServerConfig struct {
	Port                   string
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
	Secret           string
	AccessExpiryHrs  int
	RefreshExpiryHrs int
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

type StorageConfig struct {
	Type        string // local, s3, etc.
	Bucket      string
	Region      string
	AccessKey   string
	SecretKey   string
	LocalPath   string
}

func New() *Config {
	return &Config{
		Server: ServerConfig{
			Port:                   getEnv("SERVER_PORT", "8080"),
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
			Secret:           getEnv("JWT_SECRET", "your-secret-key"),
			AccessExpiryHrs:  getEnvAsInt("JWT_ACCESS_EXPIRY_HOURS", 24),
			RefreshExpiryHrs: getEnvAsInt("JWT_REFRESH_EXPIRY_HOURS", 168), // 7 days
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:     getEnv("SMTP_PORT", "587"),
			SMTPUser:     getEnv("SMTP_USER", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("FROM_EMAIL", "no-reply@iskonnect.com"),
			FromName:     getEnv("FROM_NAME", "ISKOnnect"),
		},
		Storage: StorageConfig{
			Type:      getEnv("STORAGE_TYPE", "local"),
			Bucket:    getEnv("STORAGE_BUCKET", "iskonnect"),
			Region:    getEnv("STORAGE_REGION", "us-west-2"),
			AccessKey: getEnv("STORAGE_ACCESS_KEY", ""),
			SecretKey: getEnv("STORAGE_SECRET_KEY", ""),
			LocalPath: getEnv("STORAGE_LOCAL_PATH", "./uploads"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}