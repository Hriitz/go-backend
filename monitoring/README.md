# Prometheus Monitoring Setup

This directory contains the configuration files for Prometheus monitoring integration.

## Overview

The Spring Street backend is instrumented with Prometheus metrics, which are collected by Prometheus for monitoring and alerting.

## Architecture

```
┌─────────────┐      ┌──────────────┐
│ Backend API │─────▶│  Prometheus  │
│  :8000      │      │    :9090     │
└─────────────┘      └──────────────┘
     │                      │
     └── /metrics ──────────┘
```

## Quick Start

1. **Start services** (including Prometheus):
   ```bash
   docker-compose up -d
   ```

2. **Access the services**:
   - **Prometheus**: http://localhost:9090
     - No authentication by default (consider adding authentication in production)
   - **Backend API**: http://localhost:8000
   - **Metrics Endpoint**: http://localhost:8000/metrics

3. **Query Metrics**:
   - Access Prometheus UI at http://localhost:9090
   - Use PromQL to query metrics (e.g., `rate(http_requests_total[5m])`)
   - View targets at http://localhost:9090/targets

## Metrics Collected

### HTTP Metrics (Automatic)
- Request rate, duration, size
- Response size, status codes
- Error rates

### Database Metrics (Manual instrumentation needed)
- Connection pool status
- Query duration and rate
- Query success/failure rates

### Business Metrics (Instrumented)
- Authentication attempts
- Investment inquiries
- Contact submissions
- OTP generation and verification

## Using Metrics in Code

### Recording HTTP Metrics
HTTP metrics are automatically recorded by the `PrometheusMiddleware`. No additional code needed.

### Recording Business Metrics

```go
import "springstreet/internal/metrics"

// Record authentication attempt
metrics.RecordAuthAttempt(true) // true for success, false for failure

// Record investment inquiry
metrics.RecordInvestmentInquiry()

// Record contact submission
metrics.RecordContactSubmission()

// Record OTP generation
metrics.RecordOTPGenerated("email") // or "sms"

// Record OTP verification
metrics.RecordOTPVerified(true) // true for success, false for failure

// Record database query
start := time.Now()
// ... perform database operation ...
duration := time.Since(start)
metrics.RecordDBQuery("SELECT", duration, err) // err is nil if successful

// Update database connection metrics
metrics.UpdateDBConnections(active, idle)
```

## Configuration Files

### Prometheus (`prometheus/prometheus.yml`)
- Scrape interval: 15 seconds
- Retention: 200 hours
- Scrapes metrics from `backend-go:8000/metrics`

## Customizing Prometheus

1. **Edit Configuration**:
   - Modify `monitoring/prometheus/prometheus.yml`
   - Reload Prometheus: `curl -X POST http://localhost:9090/-/reload` (if `--web.enable-lifecycle` is enabled)
   - Or restart: `docker-compose restart prometheus`

## Prometheus Queries Examples

### Request Rate
```promql
rate(http_requests_total[5m])
```

### Error Rate
```promql
sum(rate(http_requests_total{status_code=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100
```

### P95 Latency
```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### Database Query Rate
```promql
rate(db_queries_total[5m])
```

## Troubleshooting

### Metrics endpoint not accessible
- Check that the backend is running: `docker-compose ps`
- Verify metrics endpoint: `curl http://localhost:8000/metrics`
- Check backend logs: `docker-compose logs backend-go`

### Prometheus not scraping
- Check Prometheus targets: http://localhost:9090/targets
- Verify Prometheus config: `docker-compose exec prometheus cat /etc/prometheus/prometheus.yml`
- Check Prometheus logs: `docker-compose logs prometheus`

## Data Persistence

- **Prometheus data**: Stored in `prometheus_data` Docker volume

To reset monitoring data:
```bash
docker-compose down -v  # WARNING: This deletes all data
```

## Production Considerations

1. **Security**:
   - Prometheus has no authentication by default (consider adding basic auth or reverse proxy in production)
   - Restrict access to Prometheus ports (use firewall rules)
   - Consider authentication for metrics endpoint (`/metrics`) in production

2. **Performance**:
   - Adjust scrape intervals based on load
   - Configure appropriate retention policies
   - Monitor Prometheus memory usage

3. **High Availability**:
   - Consider Prometheus federation for multiple instances
   - Set up alerting rules in Prometheus

## Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [PromQL Query Language](https://prometheus.io/docs/prometheus/latest/querying/basics/)
