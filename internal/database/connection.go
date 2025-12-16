package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"springstreet/internal/config"
	"springstreet/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

var (
	db *gorm.DB
)

const (
	maxOpenConns    = 25
	maxIdleConns    = 5
	connMaxLifetime = 5 * time.Minute
	connMaxIdleTime = 10 * time.Minute
	pingTimeout     = 5 * time.Second
)

// Init initializes the database connection with connection pooling
func Init() error {
	cfg := config.Get()
	var err error
	var dialector gorm.Dialector

	log.SetPrefix("[DB] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Determine database type
	if cfg.Database.IsPostgres() {
		log.Println("Connecting to PostgreSQL database...")
		dsn := cfg.Database.GetPostgresDSN()
		dialector = postgres.Open(dsn)
	} else {
		log.Println("Connecting to SQLite database...")
		dbPath := cfg.Database.GetSQLitePath()
		sqlDB, err := sql.Open("sqlite", dbPath)
		if err != nil {
			return fmt.Errorf("failed to open SQLite database: %w", err)
		}
		dialector = sqlite.Dialector{
			DriverName: "sqlite",
			DSN:        dbPath,
			Conn:       sqlDB,
		}
	}

	// Configure GORM logger - never log SQL queries for security
	// Use Silent mode to completely disable SQL query logging
	// Errors will still be returned and can be handled by application code
	var gormLogger logger.Interface
	// Always use Silent mode to prevent SQL queries from appearing in logs
	// This prevents exposing sensitive data (queries, parameters, etc.) in logs
	gormLogger = logger.Default.LogMode(logger.Silent)

	gormConfig := &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// Connect to database
	db, err = gorm.Open(dialector, gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool (PostgreSQL only)
	if cfg.Database.IsPostgres() {
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("failed to get underlying sql.DB: %w", err)
		}

		sqlDB.SetMaxOpenConns(maxOpenConns)
		sqlDB.SetMaxIdleConns(maxIdleConns)
		sqlDB.SetConnMaxLifetime(connMaxLifetime)
		sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

		log.Printf("Connection pool configured: maxOpen=%d, maxIdle=%d", maxOpenConns, maxIdleConns)
	}

	// Test connection
	if err := testConnection(); err != nil {
		return fmt.Errorf("database connection test failed: %w", err)
	}

	// Auto-migrate models
	log.Println("Running database migrations...")
	err = db.AutoMigrate(
		&domain.User{},
		&domain.InvestmentInquiry{},
		&domain.ContactInquiry{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database connected and migrated successfully")
	return nil
}

// testConnection tests the database connection
func testConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	if db == nil {
		log.Fatal("Database not initialized. Call database.Init() first.")
	}
	return db
}

// HealthCheck performs a database health check
func HealthCheck() error {
	return testConnection()
}

// GetStats returns database connection statistics
func GetStats() (*sql.DBStats, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	stats := sqlDB.Stats()
	return &stats, nil
}
