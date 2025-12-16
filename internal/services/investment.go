package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"springstreet/gen/investment"
	"springstreet/internal/domain"
	"springstreet/internal/metrics"
	"springstreet/internal/util"

	"goa.design/goa/v3/security"
	"gorm.io/gorm"
)

// InvestmentService implements the investment service
type InvestmentService struct {
	db *gorm.DB
}

// JWTAuth implements the authorization logic for the JWT security scheme
func (s *InvestmentService) JWTAuth(ctx context.Context, token string, schema *security.JWTScheme) (context.Context, error) {
	// Validate JWT token and extract claims
	claims, err := util.ValidateToken(token)
	if err != nil {
		return nil, investment.MakeUnauthorized(fmt.Errorf("invalid or expired token"))
	}

	// Get user from database
	var user domain.User
	if err := s.db.Where("username = ?", claims.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, investment.MakeUnauthorized(fmt.Errorf("user not found"))
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, investment.MakeUnauthorized(fmt.Errorf("user account is inactive"))
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
			return nil, investment.MakeUnauthorized(fmt.Errorf("insufficient permissions"))
		}
	}

	// Add user to context
	ctx = context.WithValue(ctx, "user", &user)
	return ctx, nil
}

// NewInvestmentService creates a new investment service
func NewInvestmentService(db *gorm.DB) *InvestmentService {
	return &InvestmentService{db: db}
}

// Create implements the create investment inquiry method
func (s *InvestmentService) Create(ctx context.Context, p *investment.InvestmentInquiryCreatePayload) (*investment.Investmentinquiryresult, error) {
	email := ""
	if p.Email != nil {
		email = *p.Email
	}
	phone := ""
	if p.Phone != nil {
		phone = *p.Phone
	}
	log.Printf("[INVESTMENT] Create request: email=%s, phone=%s", email, phone)

	// Normalize phone - convert empty string to nil
	var phoneValue *string
	if p.Phone != nil && strings.TrimSpace(*p.Phone) != "" {
		trimmed := strings.TrimSpace(*p.Phone)
		phoneValue = &trimmed
	}

	// Normalize current_exposure - handle comma-separated values
	var currentExposureValue *string
	if p.CurrentExposure != nil && strings.TrimSpace(*p.CurrentExposure) != "" {
		normalized := normalizeCurrentExposure(*p.CurrentExposure)
		currentExposureValue = &normalized
	}

	// Create inquiry
	inquiry := domain.InvestmentInquiry{
		Phone:           phoneValue,
		Email:           p.Email,
		InvestmentSize:  p.InvestmentSize,
		CurrentExposure: currentExposureValue,
		Verified:        false,
	}

	if p.FirstName != nil {
		inquiry.FirstName = p.FirstName
	}
	if p.LastName != nil {
		inquiry.LastName = p.LastName
	}
	if p.ExitType != "" {
		inquiry.ExitType = &p.ExitType
	} else {
		defaultExitType := "abandoned"
		inquiry.ExitType = &defaultExitType
	}

	if err := s.db.Create(&inquiry).Error; err != nil {
		log.Printf("[INVESTMENT] Create failed: database error: %v", err)
		return nil, fmt.Errorf("failed to create inquiry: %w", err)
	}

	log.Printf("[INVESTMENT] Create successful: id=%d, email=%s, phone=%s", inquiry.ID, email, phone)
	metrics.RecordInvestmentInquiry()
	return convertInquiryToResult(&inquiry), nil
}

// UpdateByPhone implements the update by phone method
func (s *InvestmentService) UpdateByPhone(ctx context.Context, p *investment.UpdateInquiryByPhonePayload) (*investment.Investmentinquiryresult, error) {
	log.Printf("[INVESTMENT] UpdateByPhone request: phone=%s", p.Phone)

	// Normalize phone number
	normalizedPhone := normalizePhone(p.Phone)

	// Find most recent inquiry by phone
	var inquiry domain.InvestmentInquiry
	query := s.db.Where("phone LIKE ?", "%"+normalizedPhone[len(normalizedPhone)-10:]+"%").
		Order("created_at DESC").
		First(&inquiry)

	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		log.Printf("[INVESTMENT] UpdateByPhone failed: inquiry not found for phone=%s", p.Phone)
		return nil, investment.MakeNotFound(fmt.Errorf("investment inquiry not found for this phone number"))
	}
	if query.Error != nil {
		log.Printf("[INVESTMENT] UpdateByPhone failed: database error: %v", query.Error)
		return nil, fmt.Errorf("failed to find inquiry: %w", query.Error)
	}

	// Update fields
	if p.FirstName != nil {
		inquiry.FirstName = p.FirstName
	}
	if p.LastName != nil {
		inquiry.LastName = p.LastName
	}
	if p.Email != nil {
		inquiry.Email = p.Email
	}
	if p.InvestmentSize != nil {
		inquiry.InvestmentSize = p.InvestmentSize
	}
	if p.CurrentExposure != nil && strings.TrimSpace(*p.CurrentExposure) != "" {
		normalized := normalizeCurrentExposure(*p.CurrentExposure)
		inquiry.CurrentExposure = &normalized
	}

	if err := s.db.Save(&inquiry).Error; err != nil {
		log.Printf("[INVESTMENT] UpdateByPhone failed: save error: %v", err)
		return nil, fmt.Errorf("failed to update inquiry: %w", err)
	}

	log.Printf("[INVESTMENT] UpdateByPhone successful: id=%d, phone=%s", inquiry.ID, p.Phone)
	return convertInquiryToResult(&inquiry), nil
}

// Verify implements the verify inquiry method
func (s *InvestmentService) Verify(ctx context.Context, p *investment.VerifyInquiryPayload) (*investment.Investmentinquiryresult, error) {
	identifier := p.Identifier
	isEmail := strings.Contains(identifier, "@")
	log.Printf("[INVESTMENT] Verify request: identifier=%s, isEmail=%v", identifier, isEmail)

	var inquiry domain.InvestmentInquiry
	var query *gorm.DB

	if isEmail {
		query = s.db.Where("email = ?", strings.ToLower(strings.TrimSpace(identifier))).
			Order("created_at DESC").
			First(&inquiry)
	} else {
		normalizedPhone := normalizePhone(identifier)
		query = s.db.Where("phone LIKE ?", "%"+normalizedPhone[len(normalizedPhone)-10:]+"%").
			Order("created_at DESC").
			First(&inquiry)
	}

	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		log.Printf("[INVESTMENT] Verify failed: inquiry not found for identifier=%s", identifier)
		return nil, investment.MakeNotFound(fmt.Errorf("investment inquiry not found for this contact"))
	}
	if query.Error != nil {
		log.Printf("[INVESTMENT] Verify failed: database error: %v", query.Error)
		return nil, fmt.Errorf("failed to find inquiry: %w", query.Error)
	}

	// Mark as verified
	inquiry.Verified = true
	exitType := "verified"
	inquiry.ExitType = &exitType

	if err := s.db.Save(&inquiry).Error; err != nil {
		log.Printf("[INVESTMENT] Verify failed: save error: %v", err)
		return nil, fmt.Errorf("failed to verify inquiry: %w", err)
	}

	log.Printf("[INVESTMENT] Verify successful: id=%d, identifier=%s", inquiry.ID, identifier)
	return convertInquiryToResult(&inquiry), nil
}

// GetByPhone implements the get by phone method
func (s *InvestmentService) GetByPhone(ctx context.Context, p *investment.GetInquiryByPhonePayload) (*investment.Investmentinquiryresult, error) {
	log.Printf("[INVESTMENT] GetByPhone request: phone=%s", p.Phone)
	normalizedPhone := normalizePhone(p.Phone)

	var inquiry domain.InvestmentInquiry
	query := s.db.Where("phone LIKE ?", "%"+normalizedPhone[len(normalizedPhone)-10:]+"%").
		Order("created_at DESC").
		First(&inquiry)

	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		log.Printf("[INVESTMENT] GetByPhone: inquiry not found for phone=%s", p.Phone)
		return nil, investment.MakeNotFound(fmt.Errorf("investment inquiry not found"))
	}
	if query.Error != nil {
		log.Printf("[INVESTMENT] GetByPhone failed: database error: %v", query.Error)
		return nil, fmt.Errorf("failed to find inquiry: %w", query.Error)
	}

	log.Printf("[INVESTMENT] GetByPhone successful: id=%d, phone=%s", inquiry.ID, p.Phone)
	return convertInquiryToResult(&inquiry), nil
}

// List implements the list inquiries method
func (s *InvestmentService) List(ctx context.Context, p *investment.ListInquiriesPayload) ([]*investment.Investmentinquiryresult, error) {
	log.Printf("[INVESTMENT] List request: skip=%d, limit=%d", p.Skip, p.Limit)

	var inquiries []domain.InvestmentInquiry
	query := s.db.Order("created_at DESC")

	if p.Skip > 0 {
		query = query.Offset(p.Skip)
	}
	if p.Limit > 0 {
		limit := p.Limit
		if limit > 500 {
			limit = 500
		}
		query = query.Limit(limit)
	} else {
		query = query.Limit(100)
	}

	if err := query.Find(&inquiries).Error; err != nil {
		log.Printf("[INVESTMENT] List failed: database error: %v", err)
		return nil, fmt.Errorf("failed to list inquiries: %w", err)
	}

	results := make([]*investment.Investmentinquiryresult, len(inquiries))
	for i, inquiry := range inquiries {
		results[i] = convertInquiryToResult(&inquiry)
	}

	log.Printf("[INVESTMENT] List successful: returned %d inquiries", len(results))
	return results, nil
}

// Get implements the get inquiry method
func (s *InvestmentService) Get(ctx context.Context, p *investment.GetInquiryPayload) (*investment.Investmentinquiryresult, error) {
	log.Printf("[INVESTMENT] Get request: id=%d", p.ID)

	var inquiry domain.InvestmentInquiry
	if err := s.db.First(&inquiry, p.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[INVESTMENT] Get failed: inquiry id=%d not found", p.ID)
			return nil, investment.MakeNotFound(fmt.Errorf("investment inquiry not found"))
		}
		log.Printf("[INVESTMENT] Get failed: database error: %v", err)
		return nil, err
	}

	log.Printf("[INVESTMENT] Get successful: id=%d", inquiry.ID)
	return convertInquiryToResult(&inquiry), nil
}

// Helper functions
func normalizePhone(phone string) string {
	re := regexp.MustCompile(`\d+`)
	digits := re.FindAllString(phone, -1)
	return strings.Join(digits, "")
}

// normalizeCurrentExposure normalizes comma-separated current exposure values
// Removes duplicates, trims whitespace, and sorts for consistency
func normalizeCurrentExposure(exposure string) string {
	if exposure == "" {
		return ""
	}

	// Split by comma and process each value
	parts := strings.Split(exposure, ",")
	seen := make(map[string]bool)
	var normalized []string

	validValues := map[string]bool{
		"direct-stocks": true,
		"mutual-funds":  true,
		"sip":           true,
		"none":          true,
		"pms-aif":       true,
	}

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" && !seen[trimmed] {
			// Only include valid values (or allow any if you want to be flexible)
			if validValues[trimmed] || len(validValues) == 0 {
				normalized = append(normalized, trimmed)
				seen[trimmed] = true
			}
		}
	}

	return strings.Join(normalized, ",")
}

func convertInquiryToResult(inquiry *domain.InvestmentInquiry) *investment.Investmentinquiryresult {
	result := &investment.Investmentinquiryresult{
		ID:        int(inquiry.ID),
		Verified:  inquiry.Verified,
		CreatedAt: inquiry.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if inquiry.FirstName != nil {
		result.FirstName = inquiry.FirstName
	}
	if inquiry.LastName != nil {
		result.LastName = inquiry.LastName
	}
	if inquiry.Phone != nil {
		result.Phone = inquiry.Phone
	}
	if inquiry.Email != nil {
		result.Email = inquiry.Email
	}
	if inquiry.InvestmentSize != nil {
		result.InvestmentSize = inquiry.InvestmentSize
	}
	if inquiry.CurrentExposure != nil {
		result.CurrentExposure = inquiry.CurrentExposure
	}
	if inquiry.ExitType != nil {
		result.ExitType = inquiry.ExitType
	}
	if inquiry.UpdatedAt != nil {
		updatedAt := inquiry.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
		result.UpdatedAt = &updatedAt
	}

	return result
}
