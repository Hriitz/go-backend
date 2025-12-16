package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status_code"},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000},
		},
		[]string{"method", "endpoint"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000},
		},
		[]string{"method", "endpoint"},
	)

	// Database metrics
	dbConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)

	dbConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	dbQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "status"},
	)

	dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"operation"},
	)

	// Business metrics
	authAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"status"}, // success, failure
	)

	investmentInquiriesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "investment_inquiries_total",
			Help: "Total number of investment inquiries",
		},
	)

	contactSubmissionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "contact_submissions_total",
			Help: "Total number of contact form submissions",
		},
	)

	otpGeneratedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "otp_generated_total",
			Help: "Total number of OTP codes generated",
		},
		[]string{"method"}, // email, sms
	)

	otpVerifiedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "otp_verified_total",
			Help: "Total number of OTP verifications",
		},
		[]string{"status"}, // success, failure
	)
)

// PrometheusMiddleware creates a middleware that records Prometheus metrics
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Skip metrics endpoint itself
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		// Wrap response writer to capture status code and size
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Record request size
		if r.ContentLength > 0 {
			httpRequestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
		}

		// Handle request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(wrapped.statusCode)

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusCode).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path, statusCode).Observe(duration)
		httpResponseSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(wrapped.size))
	})
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// RecordAuthAttempt records an authentication attempt
func RecordAuthAttempt(success bool) {
	status := "failure"
	if success {
		status = "success"
	}
	authAttemptsTotal.WithLabelValues(status).Inc()
}

// RecordInvestmentInquiry records a new investment inquiry
func RecordInvestmentInquiry() {
	investmentInquiriesTotal.Inc()
}

// RecordContactSubmission records a new contact form submission
func RecordContactSubmission() {
	contactSubmissionsTotal.Inc()
}

// RecordOTPGenerated records OTP generation
func RecordOTPGenerated(method string) {
	otpGeneratedTotal.WithLabelValues(method).Inc()
}

// RecordOTPVerified records OTP verification
func RecordOTPVerified(success bool) {
	status := "failure"
	if success {
		status = "success"
	}
	otpVerifiedTotal.WithLabelValues(status).Inc()
}

// RecordDBQuery records a database query
func RecordDBQuery(operation string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	dbQueriesTotal.WithLabelValues(operation, status).Inc()
	dbQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// UpdateDBConnections updates database connection metrics
func UpdateDBConnections(active, idle int) {
	dbConnectionsActive.Set(float64(active))
	dbConnectionsIdle.Set(float64(idle))
}

