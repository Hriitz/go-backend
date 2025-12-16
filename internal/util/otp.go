package util

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	OTPValidityMinutes      = 10
	OTPLength               = 6
	MaxVerificationAttempts = 3
	RateLimitMinutes        = 1
	MaxRequestsPerMinute    = 5 // Maximum OTP requests allowed per minute
)

// OTPSession represents an OTP session
type OTPSession struct {
	OTP            string
	CreatedAt      time.Time
	ExpiresAt      time.Time
	Attempts       int
	Verified       bool
	Email          string // Email associated with this session
	PhoneNumber    string // Phone number associated with this session
}

var (
	otpStorage      = make(map[string]*OTPSession)
	rateLimitStore  = make(map[string][]time.Time) // Track request timestamps for rate limiting
	mu              sync.RWMutex
)

// GenerateOTP generates a random 6-digit OTP
func GenerateOTP() (string, error) {
	bytes := make([]byte, OTPLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	otp := ""
	for i := 0; i < OTPLength; i++ {
		otp += fmt.Sprintf("%d", bytes[i]%10)
	}
	return otp, nil
}

// NormalizeIdentifier normalizes phone number or email
func NormalizeIdentifier(identifier string) string {
	if strings.Contains(identifier, "@") {
		// Email - lowercase and trim
		return strings.ToLower(strings.TrimSpace(identifier))
	}
	// Phone - extract digits only
	re := regexp.MustCompile(`\d+`)
	digits := re.FindAllString(identifier, -1)
	return strings.Join(digits, "")
}

// checkRateLimit checks if the identifier has exceeded the rate limit
// Returns true if rate limit is exceeded, false otherwise
func checkRateLimit(normalized string) error {
	now := time.Now()
	oneMinuteAgo := now.Add(-RateLimitMinutes * time.Minute)

	// Get existing request timestamps
	requests, exists := rateLimitStore[normalized]
	if !exists {
		// First request, initialize
		rateLimitStore[normalized] = []time.Time{now}
		return nil
	}

	// Remove requests older than 1 minute
	validRequests := []time.Time{}
	for _, reqTime := range requests {
		if reqTime.After(oneMinuteAgo) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if we've exceeded the limit
	if len(validRequests) >= MaxRequestsPerMinute {
		oldestRequest := validRequests[0]
		timeUntilNextAllowed := oldestRequest.Add(RateLimitMinutes * time.Minute).Sub(now)
		if timeUntilNextAllowed > 0 {
			return fmt.Errorf("rate limit exceeded: maximum %d OTP requests per minute. Please wait %v before requesting again", MaxRequestsPerMinute, timeUntilNextAllowed.Round(time.Second))
		}
	}

	// Add current request
	validRequests = append(validRequests, now)
	rateLimitStore[normalized] = validRequests
	return nil
}

// CreateOTPSession creates a new OTP session
func CreateOTPSession(identifier string) (string, string, error) {
	normalized := NormalizeIdentifier(identifier)

	mu.Lock()
	defer mu.Unlock()

	// Check rate limiting
	if err := checkRateLimit(normalized); err != nil {
		return "", "", err
	}

	// Generate OTP
	otp, err := GenerateOTP()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Create session
	now := time.Now()
	otpStorage[normalized] = &OTPSession{
		OTP:       otp,
		CreatedAt: now,
		ExpiresAt: now.Add(OTPValidityMinutes * time.Minute),
		Attempts:  0,
		Verified:  false,
	}

	return otp, normalized, nil
}

// CreateOTPSessionWithBoth creates a new OTP session with both email and phone
// The primary identifier is used as the key, but both email and phone are stored
func CreateOTPSessionWithBoth(primaryIdentifier, email, phone string) (string, string, error) {
	normalized := NormalizeIdentifier(primaryIdentifier)

	mu.Lock()
	defer mu.Unlock()

	// Check rate limiting (check all identifiers to prevent bypass)
	identifiersToCheck := []string{normalized}
	if email != "" {
		normalizedEmail := NormalizeIdentifier(email)
		if normalizedEmail != normalized {
			identifiersToCheck = append(identifiersToCheck, normalizedEmail)
		}
	}
	if phone != "" {
		normalizedPhone := NormalizeIdentifier(phone)
		if normalizedPhone != normalized {
			identifiersToCheck = append(identifiersToCheck, normalizedPhone)
		}
	}

	// Check rate limit for all identifiers (use the most restrictive)
	for _, id := range identifiersToCheck {
		if err := checkRateLimit(id); err != nil {
			return "", "", err
		}
	}

	// Generate OTP
	otp, err := GenerateOTP()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Normalize email and phone
	normalizedEmail := ""
	normalizedPhone := ""
	if email != "" {
		normalizedEmail = NormalizeIdentifier(email)
	}
	if phone != "" {
		normalizedPhone = NormalizeIdentifier(phone)
	}

	// Create session with both email and phone
	now := time.Now()
	session := &OTPSession{
		OTP:         otp,
		CreatedAt:   now,
		ExpiresAt:   now.Add(OTPValidityMinutes * time.Minute),
		Attempts:     0,
		Verified:    false,
		Email:       normalizedEmail,
		PhoneNumber: normalizedPhone,
	}

	otpStorage[normalized] = session

	// Also store the session by email and phone if they're different from primary
	if normalizedEmail != "" && normalizedEmail != normalized {
		otpStorage[normalizedEmail] = session
	}
	if normalizedPhone != "" && normalizedPhone != normalized && normalizedPhone != normalizedEmail {
		otpStorage[normalizedPhone] = session
	}

	return otp, normalized, nil
}

// VerifyOTPSession verifies an OTP code
func VerifyOTPSession(identifier, otpCode string) error {
	normalized := NormalizeIdentifier(identifier)

	mu.Lock()
	defer mu.Unlock()

	session, exists := otpStorage[normalized]
	if !exists {
		return fmt.Errorf("OTP session not found. Please request a new OTP")
	}

	if session.Verified {
		return fmt.Errorf("this contact has already been verified")
	}

	if time.Now().After(session.ExpiresAt) {
		delete(otpStorage, normalized)
		return fmt.Errorf("OTP has expired. Please request a new OTP")
	}

	if session.Attempts >= MaxVerificationAttempts {
		delete(otpStorage, normalized)
		return fmt.Errorf("maximum verification attempts exceeded. Please request a new OTP")
	}

	session.Attempts++

	if session.OTP != otpCode {
		remaining := MaxVerificationAttempts - session.Attempts
		if remaining > 0 {
			return fmt.Errorf("invalid OTP. %d attempt(s) remaining", remaining)
		}
		delete(otpStorage, normalized)
		return fmt.Errorf("invalid OTP. Maximum attempts exceeded. Please request a new OTP")
	}

	session.Verified = true
	return nil
}

// IsVerified checks if an identifier is verified
func IsVerified(identifier string) bool {
	normalized := NormalizeIdentifier(identifier)

	mu.RLock()
	defer mu.RUnlock()

	session, exists := otpStorage[normalized]
	return exists && session.Verified
}

// ClearOTPSession clears an OTP session
func ClearOTPSession(identifier string) {
	normalized := NormalizeIdentifier(identifier)

	mu.Lock()
	defer mu.Unlock()

	delete(otpStorage, normalized)
}

// CleanupExpiredSessions removes expired sessions
func CleanupExpiredSessions() {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	oneMinuteAgo := now.Add(-RateLimitMinutes * time.Minute)

	// Clean up expired OTP sessions
	for key, session := range otpStorage {
		if now.After(session.ExpiresAt) {
			delete(otpStorage, key)
		}
	}

	// Clean up old rate limit entries (older than 1 minute)
	for key, requests := range rateLimitStore {
		validRequests := []time.Time{}
		for _, reqTime := range requests {
			if reqTime.After(oneMinuteAgo) {
				validRequests = append(validRequests, reqTime)
			}
		}
		if len(validRequests) == 0 {
			delete(rateLimitStore, key)
		} else {
			rateLimitStore[key] = validRequests
		}
	}
}


