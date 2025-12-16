package services

import (
	"fmt"
	"net/smtp"
	"time"

	"springstreet/internal/config"
)

// EmailService handles sending emails
type EmailService struct {
	cfg *config.EmailConfig
}

// NewEmailService creates a new email service
func NewEmailService(cfg *config.EmailConfig) *EmailService {
	return &EmailService{cfg: cfg}
}

// SendOTP sends an OTP code via email
func (s *EmailService) SendOTP(to, otpCode string) error {
	if !s.cfg.Enabled {
		// In development mode, just log
		fmt.Printf("[EMAIL] OTP would be sent to %s: %s\n", to, otpCode)
		return nil
	}

	subject := "Your Spring Street Verification Code"
	htmlBody := s.generateOTPEmailHTML(otpCode)
	textBody := fmt.Sprintf(`
Hello,

Your verification code for Spring Street is: %s

This code will expire in 10 minutes.

If you did not request this code, please ignore this email.

Best regards,
Spring Street Team
`, otpCode)

	return s.SendHTMLEmail(to, subject, htmlBody, textBody)
}

// generateOTPEmailHTML generates a professional HTML email template for OTP
func (s *EmailService) generateOTPEmailHTML(otpCode string) string {
	// Split OTP into individual digits for display
	otpDigits := ""
	spacer := `<span style="display: inline-block; width: 10px;"></span>`
	digitStyle := `style="display: inline-block; width: 52px; height: 64px; line-height: 64px; background: linear-gradient(135deg, #F8FAFC 0%%, #FFFFFF 100%%); border: 2px solid #1C5D99; border-radius: 10px; text-align: center; font-size: 32px; font-weight: 700; color: #1C5D99; font-family: 'Barlow', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; box-shadow: 0 2px 4px rgba(28, 93, 153, 0.1);"`
	for i, digit := range otpCode {
		if i > 0 {
			otpDigits += spacer
		}
		otpDigits += fmt.Sprintf(`<span %s>%c</span>`, digitStyle, digit)
	}

	logoURL := "https://springstreet.in/logo-new.png"
	currentYear := time.Now().Format("2006")

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <title>Spring Street Verification Code</title>
</head>
<body style="margin: 0; padding: 0; background: linear-gradient(135deg, #F8FAFC 0%%, #EEF2F7 100%%); font-family: 'Barlow', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;">
    <table role="presentation" cellspacing="0" cellpadding="0" border="0" width="100%%" style="background: linear-gradient(135deg, #F8FAFC 0%%, #EEF2F7 100%%);">
        <tr>
            <td style="padding: 48px 20px;">
                <table role="presentation" cellspacing="0" cellpadding="0" border="0" width="600" style="margin: 0 auto; background-color: #FFFFFF; border-radius: 16px; box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08); overflow: hidden;">
                    <!-- Header with Logo -->
                    <tr>
                        <td style="padding: 0; background: linear-gradient(135deg, #1C5D99 0%%, #0D4A7A 100%%);">
                            <table role="presentation" cellspacing="0" cellpadding="0" border="0" width="100%%">
                                <tr>
                                    <td style="padding: 40px 40px 32px; text-align: center;">
                                        <img src="%s" alt="Spring Street" width="180" height="auto" style="max-width: 180px; height: auto; display: block; margin: 0 auto;" />
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                    
                    <!-- Content -->
                    <tr>
                        <td style="padding: 48px 40px 40px;">
                            <h2 style="margin: 0 0 12px; font-size: 28px; font-weight: 700; color: #0D1A2D; line-height: 1.3; letter-spacing: -0.5px;">Verify Your Account</h2>
                            <p style="margin: 0 0 40px; font-size: 16px; line-height: 1.6; color: #64748B;">We've sent you a verification code to complete your registration. Enter this code in the verification form:</p>
                            
                            <!-- OTP Code Display -->
                            <table role="presentation" cellspacing="0" cellpadding="0" border="0" width="100%%" style="margin: 0 0 40px;">
                                <tr>
                                    <td style="text-align: center; padding: 24px; background: linear-gradient(135deg, #F8FAFC 0%%, #FFFFFF 100%%); border-radius: 12px; border: 1px solid #E2E8F0;">
                                        %s
                                    </td>
                                </tr>
                            </table>
                            
                            <!-- Info Box -->
                            <table role="presentation" cellspacing="0" cellpadding="0" border="0" width="100%%" style="margin: 0 0 32px;">
                                <tr>
                                    <td style="padding: 20px; background: linear-gradient(135deg, #F1F5F9 0%%, #FFFFFF 100%%); border-left: 4px solid #1C5D99; border-radius: 8px; box-shadow: 0 2px 8px rgba(28, 93, 153, 0.08);">
                                        <table role="presentation" cellspacing="0" cellpadding="0" border="0" width="100%%">
                                            <tr>
                                                <td style="padding-right: 12px; vertical-align: top;">
                                                    <div style="width: 24px; height: 24px; background-color: #1C5D99; border-radius: 50%%; display: inline-block; text-align: center; line-height: 24px;">
                                                        <span style="color: #FFFFFF; font-size: 14px; font-weight: 700;">!</span>
                                                    </div>
                                                </td>
                                                <td>
                                                    <p style="margin: 0; font-size: 14px; line-height: 1.6; color: #334155;">
                                                        <strong style="color: #1C5D99;">Important:</strong> This code will expire in <strong style="color: #0D1A2D;">10 minutes</strong>. If you didn't request this code, please ignore this email.
                                                    </p>
                                                </td>
                                            </tr>
                                        </table>
                                    </td>
                                </tr>
                            </table>
                            
                            <p style="margin: 0; font-size: 15px; line-height: 1.6; color: #64748B;">If you have any questions, feel free to contact our support team.</p>
                        </td>
                    </tr>
                    
                    <!-- Divider -->
                    <tr>
                        <td style="padding: 0 40px;">
                            <table role="presentation" cellspacing="0" cellpadding="0" border="0" width="100%%">
                                <tr>
                                    <td style="height: 1px; background: linear-gradient(90deg, transparent 0%%, #E2E8F0 50%%, transparent 100%%);"></td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                    
                    <!-- Footer -->
                    <tr>
                        <td style="padding: 32px 40px; background-color: #F8FAFC;">
                            <table role="presentation" cellspacing="0" cellpadding="0" border="0" width="100%%">
                                <tr>
                                    <td>
                                        <p style="margin: 0 0 8px; font-size: 15px; font-weight: 600; color: #334155;">Best regards,</p>
                                        <p style="margin: 0 0 24px; font-size: 15px; color: #64748B;">The Spring Street Team</p>
                                        
                                        <table role="presentation" cellspacing="0" cellpadding="0" border="0">
                                            <tr>
                                                <td style="padding-right: 16px;">
                                                    <a href="https://springstreet.in" style="color: #1C5D99; text-decoration: none; font-size: 14px; font-weight: 500;">Visit Website</a>
                                                </td>
                                                <td style="padding-right: 16px;">
                                                    <span style="color: #CBD5E1;">|</span>
                                                </td>
                                                <td>
                                                    <a href="https://springstreet.in/contact" style="color: #1C5D99; text-decoration: none; font-size: 14px; font-weight: 500;">Contact Support</a>
                                                </td>
                                            </tr>
                                        </table>
                                        
                                        <p style="margin: 24px 0 0; font-size: 12px; color: #94A3B8; line-height: 1.6;">
                                            This is an automated message. Please do not reply to this email.<br>
                                            © %s Spring Street. All rights reserved.
                                        </p>
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`, logoURL, otpDigits, currentYear)
}

// SendEmail sends a generic email (plain text)
func (s *EmailService) SendEmail(to, subject, body string) error {
	return s.SendHTMLEmail(to, subject, "", body)
}

// SendHTMLEmail sends an HTML email with plain text fallback
func (s *EmailService) SendHTMLEmail(to, subject, htmlBody, textBody string) error {
	if !s.cfg.Enabled {
		fmt.Printf("[EMAIL] Would send to %s: %s\n", to, subject)
		return nil
	}

	// Validate configuration
	if s.cfg.SMTPHost == "" || s.cfg.Username == "" || s.cfg.Password == "" {
		return fmt.Errorf("email service not properly configured")
	}

	// Set up authentication
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.SMTPHost)

	// Create email message
	from := s.cfg.FromEmail
	if s.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.FromEmail)
	}

	// Build multipart message
	boundary := "----=_NextPart_1234567890"

	headers := fmt.Sprintf("From: %s\r\n", from) +
		fmt.Sprintf("To: %s\r\n", to) +
		fmt.Sprintf("Subject: %s\r\n", subject) +
		"MIME-Version: 1.0\r\n" +
		fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary) +
		"\r\n"

	// Plain text part
	message := headers +
		fmt.Sprintf("--%s\r\n", boundary) +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		textBody + "\r\n"

	// HTML part (if provided)
	if htmlBody != "" {
		message += fmt.Sprintf("--%s\r\n", boundary) +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			htmlBody + "\r\n"
	}

	message += fmt.Sprintf("--%s--\r\n", boundary)

	// Send email
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	err := smtp.SendMail(addr, auth, s.cfg.FromEmail, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// IsEnabled returns whether email service is enabled
func (s *EmailService) IsEnabled() bool {
	return s.cfg.Enabled
}
