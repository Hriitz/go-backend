package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"springstreet/internal/config"
)

// SMSService handles sending SMS messages
type SMSService struct {
	cfg *config.SMSConfig
}

// NewSMSService creates a new SMS service
func NewSMSService(cfg *config.SMSConfig) *SMSService {
	return &SMSService{cfg: cfg}
}

// SendOTP sends an OTP code via SMS
func (s *SMSService) SendOTP(phoneNumber, otpCode string) error {
	if !s.cfg.Enabled {
		// In development mode, just log
		fmt.Printf("[SMS] OTP would be sent to %s: %s\n", phoneNumber, otpCode)
		return nil
	}

	message := fmt.Sprintf("Your Spring Street verification code is: %s. Valid for 10 minutes.", otpCode)

	switch strings.ToLower(s.cfg.Provider) {
	case "twilio":
		return s.sendViaTwilio(phoneNumber, message)
	case "aws":
		// AWS SNS implementation can be added here
		return fmt.Errorf("AWS SMS provider not yet implemented")
	case "console", "dev", "development":
		// Development mode - just log
		fmt.Printf("[SMS] OTP would be sent to %s: %s\n", phoneNumber, otpCode)
		return nil
	default:
		return fmt.Errorf("unsupported SMS provider: %s", s.cfg.Provider)
	}
}

// sendViaTwilio sends SMS via Twilio API
func (s *SMSService) sendViaTwilio(phoneNumber, message string) error {
	if s.cfg.TwilioSID == "" || s.cfg.TwilioAuth == "" || s.cfg.TwilioFrom == "" {
		return fmt.Errorf("Twilio not properly configured")
	}

	// Normalize phone number (ensure it starts with +)
	normalizedPhone := phoneNumber
	if !strings.HasPrefix(normalizedPhone, "+") {
		// Assume US number if no country code
		if strings.HasPrefix(normalizedPhone, "1") {
			normalizedPhone = "+" + normalizedPhone
		} else {
			normalizedPhone = "+1" + normalizedPhone
		}
	}

	// Twilio API endpoint
	url := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", s.cfg.TwilioSID)

	// Prepare request data
	data := map[string]string{
		"From": s.cfg.TwilioFrom,
		"To":   normalizedPhone,
		"Body": message,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal request data: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.SetBasicAuth(s.cfg.TwilioSID, s.cfg.TwilioAuth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return fmt.Errorf("Twilio API error (status %d): %v", resp.StatusCode, errorResp)
	}

	return nil
}

// IsEnabled returns whether SMS service is enabled
func (s *SMSService) IsEnabled() bool {
	return s.cfg.Enabled
}













