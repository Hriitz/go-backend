package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	goahttp "goa.design/goa/v3/http"
	"goa.design/goa/v3/http/middleware"

	auth "springstreet/gen/auth"
	contact "springstreet/gen/contact"
	health "springstreet/gen/health"
	authsvr "springstreet/gen/http/auth/server"
	contactsvr "springstreet/gen/http/contact/server"
	healthsvr "springstreet/gen/http/health/server"
	investmentsvr "springstreet/gen/http/investment/server"
	otpsvr "springstreet/gen/http/otp/server"
	investment "springstreet/gen/investment"
	otp "springstreet/gen/otp"

	"springstreet/internal/config"
	"springstreet/internal/database"
	"springstreet/internal/metrics"
	"springstreet/internal/services"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	shutdownTimeout = 30 * time.Second
	readTimeout     = 15 * time.Second
	writeTimeout    = 15 * time.Second
	idleTimeout     = 60 * time.Second
)

func main() {
	// Initialize structured logging
	log.SetPrefix("[API] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate critical configuration
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	log.Printf("Starting %s v%s", cfg.App.Name, cfg.App.Version)
	log.Printf("Environment: debug=%v, port=%s, host=%s", cfg.App.Debug, cfg.App.Port, cfg.App.Host)

	// Initialize database
	log.Println("Initializing database connection...")
	if err := database.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		log.Println("Closing database connections...")
		if sqlDB, err := database.GetDB().DB(); err == nil {
			if closeErr := sqlDB.Close(); closeErr != nil {
				log.Printf("Error closing database: %v", closeErr)
			}
		}
	}()

	// Create service instances
	log.Println("Initializing services...")
	healthSvc := services.NewHealthService()
	authSvc := services.NewAuthService(database.GetDB())
	investmentSvc := services.NewInvestmentService(database.GetDB())
	otpSvc := services.NewOTPService(cfg)
	emailSvc := services.NewEmailService(&cfg.Email)
	contactSvc := services.NewContactService(database.GetDB(), emailSvc)

	// Create service endpoints
	healthEndpoints := health.NewEndpoints(healthSvc)
	authEndpoints := auth.NewEndpoints(authSvc)
	investmentEndpoints := investment.NewEndpoints(investmentSvc)
	otpEndpoints := otp.NewEndpoints(otpSvc)
	contactEndpoints := contact.NewEndpoints(contactSvc)

	// Create HTTP mux
	mux := goahttp.NewMuxer()

	// Create error handler that logs errors
	errorHandler := func(ctx context.Context, w http.ResponseWriter, err error) {
		log.Printf("[ERROR] %v", err)
	}

	// Mount HTTP handlers with middleware and error handler
	log.Println("Mounting HTTP handlers...")
	healthServer := healthsvr.New(healthEndpoints, mux, goahttp.RequestDecoder, goahttp.ResponseEncoder, errorHandler, nil)
	healthServer.Use(middleware.RequestID())
	healthServer.Use(middleware.PopulateRequestContext())
	healthServer.Mount(mux)

	authServer := authsvr.New(authEndpoints, mux, goahttp.RequestDecoder, goahttp.ResponseEncoder, errorHandler, nil)
	authServer.Use(middleware.RequestID())
	authServer.Use(middleware.PopulateRequestContext())
	authServer.Mount(mux)

	investmentServer := investmentsvr.New(investmentEndpoints, mux, goahttp.RequestDecoder, goahttp.ResponseEncoder, errorHandler, nil)
	investmentServer.Use(middleware.RequestID())
	investmentServer.Use(middleware.PopulateRequestContext())
	investmentServer.Mount(mux)

	otpServer := otpsvr.New(otpEndpoints, mux, goahttp.RequestDecoder, goahttp.ResponseEncoder, errorHandler, nil)
	otpServer.Use(middleware.RequestID())
	otpServer.Use(middleware.PopulateRequestContext())
	otpServer.Mount(mux)

	contactServer := contactsvr.New(contactEndpoints, mux, goahttp.RequestDecoder, goahttp.ResponseEncoder, errorHandler, nil)
	contactServer.Use(middleware.RequestID())
	contactServer.Use(middleware.PopulateRequestContext())
	contactServer.Mount(mux)

	// Create a wrapper handler that routes /metrics to Prometheus and everything else to Goa mux
	rootHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			promhttp.Handler().ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})

	// Setup middleware chain: Prometheus -> Security -> CORS -> Logging -> Handler
	handler := setupSecurityHeaders(setupCORS(requestLogging(metrics.PrometheusMiddleware(rootHandler)), cfg), cfg)

	// Create HTTP server with timeouts
	addr := fmt.Sprintf("%s:%s", cfg.App.Host, cfg.App.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		ErrorLog:     log.New(os.Stderr, "[HTTP] ", log.LstdFlags),
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Server listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for interrupt signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("Server failed to start: %v", err)
	case sig := <-shutdown:
		log.Printf("Received signal: %v. Starting graceful shutdown...", sig)
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Error during graceful shutdown: %v", err)
		if err == context.DeadlineExceeded {
			log.Println("Shutdown timeout exceeded, forcing close...")
			httpServer.Close()
		}
	}

	log.Println("Server shutdown complete")
}

// validateConfig validates critical configuration values
func validateConfig(cfg *config.Config) error {
	if cfg.Auth.SecretKey == "" || cfg.Auth.SecretKey == "your-secret-key-change-in-production" {
		return fmt.Errorf("SECRET_KEY must be set and changed from default value")
	}
	if len(cfg.Auth.SecretKey) < 32 {
		return fmt.Errorf("SECRET_KEY must be at least 32 characters for security")
	}
	if cfg.App.Port == "" {
		return fmt.Errorf("PORT must be set")
	}
	return nil
}

// setupSecurityHeaders adds security headers to responses
func setupSecurityHeaders(handler http.Handler, cfg *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Remove server identification
		w.Header().Set("Server", "")

		// HSTS (only in production with HTTPS)
		if !cfg.App.Debug && r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		handler.ServeHTTP(w, r)
	})
}

// setupCORS configures CORS based on environment
func setupCORS(handler http.Handler, cfg *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// In production, validate against allowed origins
		if !cfg.App.Debug && len(cfg.CORS.AllowedOrigins) > 0 && cfg.CORS.AllowedOrigins[0] != "*" {
			allowed := false
			for _, allowedOrigin := range cfg.CORS.AllowedOrigins {
				if origin == allowedOrigin {
					allowed = true
					break
				}
			}
			if !allowed && origin != "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}

		// Set CORS headers
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if cfg.App.Debug {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.CORS.AllowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.CORS.AllowedHeaders, ", "))
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.CORS.MaxAge))
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// requestLogging logs all incoming requests and their responses
func requestLogging(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Skip logging for health checks to reduce noise
		if r.URL.Path == "/health" {
			handler.ServeHTTP(w, r)
			return
		}

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Log request start
		log.Printf("[REQUEST] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Handle request
		handler.ServeHTTP(wrapped, r)

		// Log request completion
		duration := time.Since(start)
		statusText := "OK"
		if wrapped.statusCode >= 400 {
			statusText = "ERROR"
		}
		log.Printf("[RESPONSE] %s %s -> %d %s (%v)", r.Method, r.URL.Path, wrapped.statusCode, statusText, duration)
	})
}
