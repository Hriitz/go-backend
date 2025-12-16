package services

import (
	"context"
	health "springstreet/gen/health"
)

// HealthService implements the health service
type HealthService struct{}

// NewHealthService creates a new health service
func NewHealthService() *HealthService {
	return &HealthService{}
}

// Check implements the health check method
func (s *HealthService) Check(ctx context.Context) (*health.Healthresult, error) {
	status := "healthy"
	service := "Spring Street API"
	return &health.Healthresult{
		Status:  &status,
		Service: &service,
	}, nil
}


