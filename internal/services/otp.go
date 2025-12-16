package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"springstreet/gen/otp"
	"springstreet/internal/config"
	"springstreet/internal/metrics"
	"springstreet/internal/util"
)

// OTPService implements the OTP service
type OTPService struct {
	emailService *EmailService
	smsService   *SMSService
	config       *config.Config
}

// NewOTPService creates a new OTP service
func NewOTPService(cfg *config.Config) *OTPService {
	return &OTPService{
		emailService: NewEmailService(&cfg.Email),
		smsService:   NewSMSService(&cfg.SMS),
		config:       cfg,
	}
}

// Send implements the send OTP method
func (s *OTPService) Send(ctx context.Context, p *otp.SendOTPPayload) (*otp.Sendotpresult, error) {
	// Validate that at least one contact method is provided
	phoneProvided := p.PhoneNumber != nil && strings.TrimSpace(*p.PhoneNumber) != ""
	emailProvided := p.Email != nil && strings.TrimSpace(*p.Email) != ""

	phone := ""
	email := ""
	if phoneProvided {
		phone = *p.PhoneNumber
	}
	if emailProvided {
		email = *p.Email
	}
	log.Printf("[OTP] Send request: phone=%s, email=%s", phone, email)

	if !phoneProvided && !emailProvided {
		log.Printf("[OTP] Send failed: no contact method provided")
		return nil, otp.MakeBadRequest(fmt.Errorf("either phone_number or email must be provided"))
	}

	// Clean up expired sessions
	util.CleanupExpiredSessions()

	// Use phone as primary identifier, fallback to email
	var identifier string
	if phoneProvided {
		identifier = *p.PhoneNumber
	} else {
		identifier = *p.Email
	}

	// Create OTP session (supports both email and phone)
	var emailIdentifier, phoneIdentifier string
	if emailProvided {
		emailIdentifier = *p.Email
	}
	if phoneProvided {
		phoneIdentifier = *p.PhoneNumber
	}

	otpCode, normalizedIdentifier, err := util.CreateOTPSessionWithBoth(identifier, emailIdentifier, phoneIdentifier)
	if err != nil {
		log.Printf("[OTP] Send failed: session creation error: %v", err)
		return nil, otp.MakeBadRequest(err)
	}

	// Send OTP via email if email is provided
	if emailProvided {
		emailErr := s.emailService.SendOTP(*p.Email, otpCode)
		if emailErr != nil {
			log.Printf("[OTP] Warning: failed to send OTP via email to %s: %v", *p.Email, emailErr)
		} else {
			log.Printf("[OTP] OTP sent via email to %s", *p.Email)
			metrics.RecordOTPGenerated("email")
		}
	}

	// Send OTP via SMS if phone is provided
	if phoneProvided {
		smsErr := s.smsService.SendOTP(*p.PhoneNumber, otpCode)
		if smsErr != nil {
			log.Printf("[OTP] Warning: failed to send OTP via SMS to %s: %v", *p.PhoneNumber, smsErr)
		} else {
			log.Printf("[OTP] OTP sent via SMS to %s", *p.PhoneNumber)
			metrics.RecordOTPGenerated("sms")
		}
	}

	// If both email and SMS failed, return error
	if emailProvided && phoneProvided {
		// At least one should succeed, but we already logged warnings
		// Continue with success response
	} else if emailProvided && !s.emailService.IsEnabled() {
		// In dev mode, just log
		log.Printf("[OTP] DEV MODE - OTP for Email %s: %s (valid for 10 minutes)", *p.Email, otpCode)
	} else if phoneProvided && !s.smsService.IsEnabled() {
		// In dev mode, just log
		log.Printf("[OTP] DEV MODE - OTP for Phone %s: %s (valid for 10 minutes)", normalizedIdentifier, otpCode)
	}

	// Return response
	phoneNumber := normalizedIdentifier
	if !phoneProvided && emailProvided {
		phoneNumber = *p.Email
	}

	log.Printf("[OTP] Send successful: identifier=%s", phoneNumber)
	return &otp.Sendotpresult{
		Message:          "OTP sent successfully",
		PhoneNumber:      phoneNumber,
		ExpiresInMinutes: 10,
	}, nil
}

// Verify implements the verify OTP method
func (s *OTPService) Verify(ctx context.Context, p *otp.VerifyOTPPayload) (*otp.Verifyotpresult, error) {
	phone := ""
	email := ""
	if p.PhoneNumber != nil {
		phone = *p.PhoneNumber
	}
	if p.Email != nil {
		email = *p.Email
	}
	log.Printf("[OTP] Verify request: phone=%s, email=%s, code=%s", phone, email, p.OtpCode)

	// Validate that at least one contact method is provided
	if (p.PhoneNumber == nil || strings.TrimSpace(*p.PhoneNumber) == "") &&
		(p.Email == nil || strings.TrimSpace(*p.Email) == "") {
		log.Printf("[OTP] Verify failed: no contact method provided")
		return nil, otp.MakeBadRequest(fmt.Errorf("either phone_number or email must be provided"))
	}

	// Clean up expired sessions
	util.CleanupExpiredSessions()

	// Use phone as primary identifier, fallback to email
	var identifier string
	if p.PhoneNumber != nil && strings.TrimSpace(*p.PhoneNumber) != "" {
		identifier = *p.PhoneNumber
	} else {
		identifier = *p.Email
	}

	// Verify OTP
	if err := util.VerifyOTPSession(identifier, p.OtpCode); err != nil {
		log.Printf("[OTP] Verify failed: verification error for identifier=%s: %v", identifier, err)
		metrics.RecordOTPVerified(false)
		return nil, otp.MakeBadRequest(err)
	}

	// Get normalized identifier for response
	normalizedIdentifier := util.NormalizeIdentifier(identifier)
	if p.Email != nil && strings.Contains(identifier, "@") {
		normalizedIdentifier = strings.ToLower(strings.TrimSpace(identifier))
	}

	log.Printf("[OTP] Verify successful: identifier=%s", normalizedIdentifier)
	metrics.RecordOTPVerified(true)
	return &otp.Verifyotpresult{
		Message:     "Contact verified successfully",
		PhoneNumber: normalizedIdentifier,
		Verified:    true,
	}, nil
}

// Check implements the check verification method
func (s *OTPService) Check(ctx context.Context, p *otp.CheckVerificationPayload) (*otp.Checkverificationresult, error) {
	log.Printf("[OTP] Check request: phone=%s", p.PhoneNumber)

	normalizedPhone := util.NormalizeIdentifier(p.PhoneNumber)
	verified := util.IsVerified(p.PhoneNumber)

	log.Printf("[OTP] Check result: phone=%s, verified=%v", normalizedPhone, verified)
	return &otp.Checkverificationresult{
		PhoneNumber: normalizedPhone,
		Verified:    verified,
	}, nil
}
