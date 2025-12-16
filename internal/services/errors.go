package services

import (
	"errors"
	"fmt"

	goa "goa.design/goa/v3/pkg"
	"springstreet/gen/auth"
	"springstreet/gen/contact"
	"springstreet/gen/investment"
	"springstreet/gen/otp"
)

// ErrorType represents the type of error
type ErrorType int

const (
	ErrTypeBadRequest ErrorType = iota
	ErrTypeUnauthorized
	ErrTypeNotFound
	ErrTypeInternal
)

// ServiceError is a standardized error interface for all services
type ServiceError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// NewBadRequestError creates a new bad request error
func NewBadRequestError(message string) *ServiceError {
	return &ServiceError{
		Type:    ErrTypeBadRequest,
		Message: message,
	}
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string) *ServiceError {
	return &ServiceError{
		Type:    ErrTypeUnauthorized,
		Message: message,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string) *ServiceError {
	return &ServiceError{
		Type:    ErrTypeNotFound,
		Message: message,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(message string, err error) *ServiceError {
	return &ServiceError{
		Type:    ErrTypeInternal,
		Message: message,
		Err:     err,
	}
}

// ============================================================
// Auth Service Error Helpers
// ============================================================

// AuthBadRequest creates a properly formatted bad request error for auth service
func AuthBadRequest(message string) *goa.ServiceError {
	return auth.MakeBadRequest(errors.New(message))
}

// AuthUnauthorized creates a properly formatted unauthorized error for auth service
func AuthUnauthorized(message string) *goa.ServiceError {
	return auth.MakeUnauthorized(errors.New(message))
}

// AuthNotFound creates a properly formatted not found error for auth service
func AuthNotFound(message string) *goa.ServiceError {
	return auth.MakeNotFound(errors.New(message))
}

// ============================================================
// Contact Service Error Helpers
// ============================================================

// ContactBadRequest creates a properly formatted bad request error for contact service
func ContactBadRequest(message string) *goa.ServiceError {
	return contact.MakeBadRequest(errors.New(message))
}

// ContactUnauthorized creates a properly formatted unauthorized error for contact service
func ContactUnauthorized(message string) *goa.ServiceError {
	return contact.MakeUnauthorized(errors.New(message))
}

// ============================================================
// Investment Service Error Helpers
// ============================================================

// InvestmentUnauthorized creates a properly formatted unauthorized error for investment service
func InvestmentUnauthorized(message string) *goa.ServiceError {
	return investment.MakeUnauthorized(errors.New(message))
}

// InvestmentNotFound creates a properly formatted not found error for investment service
func InvestmentNotFound(message string) *goa.ServiceError {
	return investment.MakeNotFound(errors.New(message))
}

// ============================================================
// OTP Service Error Helpers
// ============================================================

// OTPBadRequest creates a properly formatted bad request error for OTP service
func OTPBadRequest(message string) *goa.ServiceError {
	return otp.MakeBadRequest(errors.New(message))
}

