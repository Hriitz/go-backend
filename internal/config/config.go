package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Auth     AuthConfig
	CORS     CORSConfig
	Email    EmailConfig
	SMS      SMSConfig
}

// AppConfig holds application-level configuration
type AppConfig struct {
	Name    string
	Version string
	Debug   bool
	Port    string
	Host    string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL string
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	SecretKey          string
	TokenExpiryMinutes int
	Algorithm          string
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int
}

// EmailConfig holds email service configuration
type EmailConfig struct {
	Enabled  bool
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	FromEmail string
	FromName string
}

// SMSConfig holds SMS service configuration
type SMSConfig struct {
	Enabled    bool
	Provider   string // "twilio", "aws", "console" (for development)
	TwilioSID  string
	TwilioAuth string
	TwilioFrom string
}

var globalConfig *Config

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file (ignore error if it doesn't exist)
	_ = godotenv.Load()

	config := &Config{
		App: AppConfig{
			Name:    getEnv("APP_NAME", "Spring Street API"),
			Version: getEnv("APP_VERSION", "1.0.0"),
			Debug:   getEnvAsBool("DEBUG", false), // Default to false for security (no SQL query logging)
			Port:    getEnv("PORT", "8000"),
			Host:    getEnv("HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "sqlite:///./spring_street.db"),
		},
		Auth: AuthConfig{
			SecretKey:          getEnv("SECRET_KEY", "your-secret-key-change-in-production"),
			TokenExpiryMinutes: getEnvAsInt("ACCESS_TOKEN_EXPIRE_MINUTES", 30),
			Algorithm:          getEnv("ALGORITHM", "HS256"),
		},
		CORS: CORSConfig{
			AllowedOrigins: getEnvAsSlice("ALLOWED_HOSTS", []string{"*"}),
			AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
			AllowedHeaders: []string{"*"},
			MaxAge:         86400,
		},
		Email: EmailConfig{
			Enabled:  getEnvAsBool("EMAIL_ENABLED", false),
			SMTPHost: getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort: getEnvAsInt("SMTP_PORT", 587),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			FromEmail: getEnv("EMAIL_FROM", "noreply@springstreet.com"),
			FromName:  getEnv("EMAIL_FROM_NAME", "Spring Street"),
		},
		SMS: SMSConfig{
			Enabled:    getEnvAsBool("SMS_ENABLED", false),
			Provider:   getEnv("SMS_PROVIDER", "console"), // console for development
			TwilioSID:  getEnv("TWILIO_ACCOUNT_SID", ""),
			TwilioAuth: getEnv("TWILIO_AUTH_TOKEN", ""),
			TwilioFrom: getEnv("TWILIO_PHONE_NUMBER", ""),
		},
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	globalConfig = config
	return config, nil
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	if cfg.App.Port == "" {
		return fmt.Errorf("PORT must be set")
	}
	if cfg.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL must be set")
	}
	if cfg.Auth.SecretKey == "" {
		return fmt.Errorf("SECRET_KEY must be set")
	}
	if cfg.Auth.TokenExpiryMinutes <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_EXPIRE_MINUTES must be greater than 0")
	}
	return nil
}

// Get returns the global configuration
func Get() *Config {
	if globalConfig == nil {
		// Load default config if not loaded
		config, _ := Load()
		return config
	}
	return globalConfig
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}

// IsPostgres checks if the database URL is for PostgreSQL
func (c *DatabaseConfig) IsPostgres() bool {
	url := c.URL
	return len(url) > 10 && (url[:10] == "postgresql" || (len(url) > 8 && url[:8] == "postgres"))
}

// GetPostgresDSN converts database URL to PostgreSQL DSN format
// Converts: postgresql://user:pass@host:port/db?sslmode=disable
// To: host=host port=port user=user password=pass dbname=db sslmode=disable
func (c *DatabaseConfig) GetPostgresDSN() string {
	url := c.URL

	// If already in DSN format (contains spaces or =), return as is
	if strings.Contains(url, " ") || strings.Contains(url, "=") {
		return url
	}

	// Parse postgresql:// or postgres:// URL
	var prefix string
	if len(url) > 10 && url[:10] == "postgresql" {
		prefix = "postgresql://"
	} else if len(url) > 8 && url[:8] == "postgres" {
		prefix = "postgres://"
	} else {
		return url
	}

	// Remove prefix
	url = url[len(prefix):]

	// Split into parts: user:pass@host:port/db?params
	parts := strings.Split(url, "@")
	if len(parts) != 2 {
		return url // Return as-is if format is unexpected
	}

	// Parse credentials
	credentials := parts[0]
	rest := parts[1]

	var user, password string
	if strings.Contains(credentials, ":") {
		creds := strings.Split(credentials, ":")
		user = creds[0]
		password = strings.Join(creds[1:], ":") // Handle passwords with : in them
	} else {
		user = credentials
		password = ""
	}

	// Parse host:port/db?params
	var host, port, dbname, sslmode string
	host = "localhost"
	port = "5432"
	sslmode = "disable"

	if strings.Contains(rest, "/") {
		hostPort := strings.Split(rest, "/")[0]
		dbAndParams := strings.Split(rest, "/")[1]

		// Parse host:port
		if strings.Contains(hostPort, ":") {
			hp := strings.Split(hostPort, ":")
			host = hp[0]
			port = hp[1]
		} else {
			host = hostPort
		}

		// Parse dbname?params
		if strings.Contains(dbAndParams, "?") {
			dbParts := strings.Split(dbAndParams, "?")
			dbname = dbParts[0]
			params := dbParts[1]

			// Parse sslmode from params
			if strings.Contains(params, "sslmode=") {
				for _, param := range strings.Split(params, "&") {
					if strings.HasPrefix(param, "sslmode=") {
						sslmode = strings.TrimPrefix(param, "sslmode=")
					}
				}
			}
		} else {
			dbname = dbAndParams
		}
	} else {
		// No database specified
		if strings.Contains(rest, ":") {
			hp := strings.Split(rest, ":")
			host = hp[0]
			port = hp[1]
		} else {
			host = rest
		}
		dbname = "postgres"
	}

	// Build DSN string
	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s", host, port, user, dbname, sslmode)
	if password != "" {
		dsn += " password=" + password
	}

	return dsn
}

// GetSQLitePath extracts SQLite database path from URL
func (c *DatabaseConfig) GetSQLitePath() string {
	url := c.URL
	if len(url) > 10 && url[:10] == "sqlite:///" {
		return url[10:]
	}
	return url
}
