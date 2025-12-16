package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"goa.design/goa/v3/security"
	"gorm.io/gorm"

	"springstreet/gen/contact"
	"springstreet/internal/domain"
	"springstreet/internal/metrics"
	"springstreet/internal/util"
)

// ContactService implements the contact service
type ContactService struct {
	db           *gorm.DB
	emailService *EmailService
}

// NewContactService creates a new contact service
func NewContactService(db *gorm.DB, emailService *EmailService) *ContactService {
	return &ContactService{
		db:           db,
		emailService: emailService,
	}
}

// JWTAuth implements the authorization logic for the JWT security scheme
func (s *ContactService) JWTAuth(ctx context.Context, token string, schema *security.JWTScheme) (context.Context, error) {
	// Validate JWT token and extract claims
	claims, err := util.ValidateToken(token)
	if err != nil {
		return nil, contact.MakeUnauthorized(fmt.Errorf("invalid or expired token"))
	}

	// Get user from database
	var user domain.User
	if err := s.db.Where("username = ?", claims.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, contact.MakeUnauthorized(fmt.Errorf("user not found"))
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, contact.MakeUnauthorized(fmt.Errorf("user account is inactive"))
	}

	// Check scopes if required
	if schema != nil && len(schema.RequiredScopes) > 0 {
		hasScope := false
		for _, requiredScope := range schema.RequiredScopes {
			if requiredScope == "admin" && user.IsAdmin {
				hasScope = true
				break
			}
			if requiredScope == "staff" && (user.IsStaff || user.IsAdmin) {
				hasScope = true
				break
			}
		}
		if !hasScope {
			return nil, contact.MakeUnauthorized(fmt.Errorf("insufficient permissions"))
		}
	}

	// Add user to context
	ctx = context.WithValue(ctx, "user", &user)
	return ctx, nil
}

// Submit implements the submit contact form method
func (s *ContactService) Submit(ctx context.Context, p *contact.ContactSubmitPayload) (*contact.Contactsubmitresult, error) {
	log.Printf("[CONTACT] Submit request: name=%s, email=%s", strings.TrimSpace(p.Name), strings.TrimSpace(p.Email))

	// Validate input
	if err := s.validateContactForm(p); err != nil {
		log.Printf("[CONTACT] Submit failed: validation error: %v", err)
		return nil, contact.MakeBadRequest(err)
	}

	// Create contact inquiry
	inquiry := &domain.ContactInquiry{
		Name:    strings.TrimSpace(p.Name),
		Email:   strings.ToLower(strings.TrimSpace(p.Email)),
		Message: strings.TrimSpace(p.Message),
		Status:  "new",
	}

	// Add phone if provided
	if p.Phone != nil && strings.TrimSpace(*p.Phone) != "" {
		phone := strings.TrimSpace(*p.Phone)
		inquiry.Phone = &phone
	}

	// Save to database
	if err := s.db.Create(inquiry).Error; err != nil {
		log.Printf("[CONTACT] Submit failed: database error: %v", err)
		return nil, fmt.Errorf("failed to save contact inquiry: %w", err)
	}

	log.Printf("[CONTACT] Submit successful: id=%d, name=%s, email=%s", inquiry.ID, inquiry.Name, inquiry.Email)
	metrics.RecordContactSubmission()

	// Send email notification to admin (async, don't fail if email fails)
	go func() {
		if err := s.sendContactNotification(inquiry); err != nil {
			log.Printf("[CONTACT] Warning: failed to send notification email: %v", err)
		} else {
			log.Printf("[CONTACT] Notification email sent for inquiry id=%d", inquiry.ID)
		}
	}()

	return &contact.Contactsubmitresult{
		ID:      int(inquiry.ID),
		Message: "Thank you for contacting us! We'll get back to you soon.",
	}, nil
}

// List returns all contact inquiries (Staff/Admin only)
func (s *ContactService) List(ctx context.Context, p *contact.ListContactInquiriesPayload) ([]*contact.Contactinquiryresult, error) {
	log.Printf("[CONTACT] List request: skip=%d, limit=%d", p.Skip, p.Limit)

	var inquiries []domain.ContactInquiry

	// Use provided values (Goa provides defaults)
	skip := p.Skip
	limit := p.Limit

	// Query database
	if err := s.db.Order("created_at DESC").Offset(skip).Limit(limit).Find(&inquiries).Error; err != nil {
		log.Printf("[CONTACT] List failed: database error: %v", err)
		return nil, fmt.Errorf("failed to fetch contact inquiries: %w", err)
	}

	// Convert to result type
	results := make([]*contact.Contactinquiryresult, len(inquiries))
	for i, inq := range inquiries {
		createdAt := inq.CreatedAt.Format("2006-01-02T15:04:05Z")
		var updatedAt *string
		if inq.UpdatedAt != nil {
			ua := inq.UpdatedAt.Format("2006-01-02T15:04:05Z")
			updatedAt = &ua
		}

		results[i] = &contact.Contactinquiryresult{
			ID:        int(inq.ID),
			Name:      inq.Name,
			Email:     inq.Email,
			Phone:     inq.Phone,
			Message:   inq.Message,
			Status:    inq.Status,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
	}

	log.Printf("[CONTACT] List successful: returned %d inquiries", len(results))
	return results, nil
}

// validateContactForm validates the contact form input
func (s *ContactService) validateContactForm(p *contact.ContactSubmitPayload) error {
	// Validate name
	name := strings.TrimSpace(p.Name)
	if len(name) < 2 || len(name) > 100 {
		return fmt.Errorf("name must be between 2 and 100 characters")
	}

	// Validate email
	email := strings.TrimSpace(p.Email)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email address")
	}

	// Validate message
	message := strings.TrimSpace(p.Message)
	if len(message) < 1 {
		return fmt.Errorf("message is required")
	}
	if len(message) > 5000 {
		return fmt.Errorf("message must not exceed 5000 characters")
	}

	// Validate phone if provided
	if p.Phone != nil && strings.TrimSpace(*p.Phone) != "" {
		phone := strings.TrimSpace(*p.Phone)
		// Basic phone validation (allows international format)
		phoneRegex := regexp.MustCompile(`^[\d\s\+\-\(\)]+$`)
		if !phoneRegex.MatchString(phone) || len(phone) < 10 || len(phone) > 20 {
			return fmt.Errorf("invalid phone number format")
		}
	}

	return nil
}

// sendContactNotification sends an email notification to admin about new contact inquiry
func (s *ContactService) sendContactNotification(inquiry *domain.ContactInquiry) error {
	if !s.emailService.IsEnabled() {
		fmt.Printf("[CONTACT] New contact inquiry from %s (%s)\n", inquiry.Name, inquiry.Email)
		return nil
	}

	// Admin email (should be configured in environment)
	adminEmail := "nishant@springstreet.in" // TODO: Move to config

	subject := fmt.Sprintf("New Contact Form Submission from %s", inquiry.Name)

	// Build email body
	phoneInfo := "Not provided"
	if inquiry.Phone != nil && *inquiry.Phone != "" {
		phoneInfo = *inquiry.Phone
	}

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>New Contact Form Submission</title>
</head>
<body style="font-family: 'Barlow', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #334155;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #1C5D99;">New Contact Form Submission</h2>
        
        <div style="background: #F8FAFC; padding: 20px; border-radius: 8px; margin: 20px 0;">
            <p><strong>Name:</strong> %s</p>
            <p><strong>Email:</strong> <a href="mailto:%s">%s</a></p>
            <p><strong>Phone:</strong> %s</p>
            <p><strong>Submitted:</strong> %s</p>
        </div>
        
        <div style="background: #FFFFFF; padding: 20px; border-left: 4px solid #1C5D99; border-radius: 4px; margin: 20px 0;">
            <h3 style="color: #0D1A2D; margin-top: 0;">Message:</h3>
            <p style="white-space: pre-wrap;">%s</p>
        </div>
        
        <p style="color: #64748B; font-size: 14px;">
            Contact Inquiry ID: #%d
        </p>
    </div>
</body>
</html>`, inquiry.Name, inquiry.Email, inquiry.Email, phoneInfo, inquiry.CreatedAt.Format("January 2, 2006 at 3:04 PM"), inquiry.Message, inquiry.ID)

	textBody := fmt.Sprintf(`New Contact Form Submission

Name: %s
Email: %s
Phone: %s
Submitted: %s

Message:
%s

Contact Inquiry ID: #%d`, inquiry.Name, inquiry.Email, phoneInfo, inquiry.CreatedAt.Format("January 2, 2006 at 3:04 PM"), inquiry.Message, inquiry.ID)

	return s.emailService.SendHTMLEmail(adminEmail, subject, htmlBody, textBody)
}

