package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"springstreet/gen/auth"
	"springstreet/internal/domain"
	"springstreet/internal/metrics"
	"springstreet/internal/util"

	"goa.design/goa/v3/security"
	"gorm.io/gorm"
)

// Helper function to convert string to *string
func stringPtr(s string) *string {
	return &s
}

// AuthService implements the auth service
type AuthService struct {
	db *gorm.DB
}

// JWTAuth implements the authorization logic for the JWT security scheme
func (s *AuthService) JWTAuth(ctx context.Context, token string, schema *security.JWTScheme) (context.Context, error) {
	// Validate JWT token and extract claims
	claims, err := util.ValidateToken(token)
	if err != nil {
		return nil, auth.MakeUnauthorized(fmt.Errorf("invalid or expired token"))
	}

	// Get user from database
	var user domain.User
	if err := s.db.Where("username = ?", claims.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, auth.MakeUnauthorized(fmt.Errorf("user not found"))
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, auth.MakeUnauthorized(fmt.Errorf("user account is inactive"))
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
			return nil, auth.MakeUnauthorized(fmt.Errorf("insufficient permissions"))
		}
	}

	// Add user to context
	ctx = context.WithValue(ctx, "user", &user)
	return ctx, nil
}

// NewAuthService creates a new auth service
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

// Login implements the login method
func (s *AuthService) Login(ctx context.Context, p *auth.LoginPayload) (*auth.Loginresult, error) {
	// Trim whitespace from credentials
	username := strings.TrimSpace(p.Username)
	password := strings.TrimSpace(p.Password)

	log.Printf("[AUTH] Login attempt for user: %s", username)

	var user domain.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[AUTH] Login failed: user '%s' not found", username)
			metrics.RecordAuthAttempt(false)
			return nil, auth.MakeUnauthorized(fmt.Errorf("incorrect username or password"))
		}
		log.Printf("[AUTH] Login failed: database error for user '%s': %v", username, err)
		metrics.RecordAuthAttempt(false)
		return nil, err
	}

	if !util.CheckPasswordHash(password, user.HashedPassword) {
		log.Printf("[AUTH] Login failed: invalid password for user '%s'", username)
		metrics.RecordAuthAttempt(false)
		return nil, auth.MakeUnauthorized(fmt.Errorf("incorrect username or password"))
	}

	if !user.IsActive {
		log.Printf("[AUTH] Login failed: user '%s' is inactive", username)
		metrics.RecordAuthAttempt(false)
		return nil, auth.MakeUnauthorized(fmt.Errorf("user account is inactive"))
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	s.db.Save(&user)

	// Generate token
	token, err := util.GenerateToken(&user)
	if err != nil {
		log.Printf("[AUTH] Login failed: token generation error for user '%s': %v", username, err)
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	log.Printf("[AUTH] Login successful for user '%s' (id=%d, admin=%v, staff=%v)", username, user.ID, user.IsAdmin, user.IsStaff)
	metrics.RecordAuthAttempt(true)

	return &auth.Loginresult{
		AccessToken: token,
		TokenType:   "bearer",
	}, nil
}

// Logout implements the logout method
func (s *AuthService) Logout(ctx context.Context, p *auth.LogoutPayload) (*auth.Logoutresult, error) {
	user := ctx.Value("user").(*domain.User)
	log.Printf("[AUTH] Logout for user: %s (id=%d)", user.Username, user.ID)
	return &auth.Logoutresult{
		Message: stringPtr("Successfully logged out"),
	}, nil
}

// Me implements the me method
func (s *AuthService) Me(ctx context.Context, p *auth.MePayload) (*auth.Userresult, error) {
	user := ctx.Value("user").(*domain.User)
	log.Printf("[AUTH] Me request for user: %s (id=%d)", user.Username, user.ID)
	return convertUserToResult(user), nil
}

// CreateUser implements the create user method
func (s *AuthService) CreateUser(ctx context.Context, p *auth.CreateUserPayload) (*auth.Userresult, error) {
	// Trim and normalize inputs
	username := strings.TrimSpace(p.Username)
	email := strings.ToLower(strings.TrimSpace(p.Email))
	password := strings.TrimSpace(p.Password)

	log.Printf("[AUTH] CreateUser request: username=%s, email=%s", username, email)

	// Check if username exists
	var existingUser domain.User
	if err := s.db.Where("username = ?", username).First(&existingUser).Error; err == nil {
		log.Printf("[AUTH] CreateUser failed: username '%s' already exists", username)
		return nil, auth.MakeBadRequest(fmt.Errorf("username already registered"))
	}

	// Check if email exists
	if err := s.db.Where("email = ?", email).First(&existingUser).Error; err == nil {
		log.Printf("[AUTH] CreateUser failed: email '%s' already exists", email)
		return nil, auth.MakeBadRequest(fmt.Errorf("email already registered"))
	}

	// Hash password
	hashedPassword, err := util.HashPassword(password)
	if err != nil {
		log.Printf("[AUTH] CreateUser failed: password hashing error: %v", err)
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := domain.User{
		Username:       username,
		Email:          email,
		HashedPassword: hashedPassword,
		IsActive:       p.IsActive,
		IsAdmin:        p.IsAdmin,
		IsStaff:        p.IsStaff,
	}
	if p.FullName != nil {
		fullName := strings.TrimSpace(*p.FullName)
		user.FullName = &fullName
	}

	if err := s.db.Create(&user).Error; err != nil {
		log.Printf("[AUTH] CreateUser failed: database error: %v", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	log.Printf("[AUTH] CreateUser successful: username=%s, id=%d", username, user.ID)
	return convertUserToResult(&user), nil
}

// ListUsers implements the list users method
func (s *AuthService) ListUsers(ctx context.Context, p *auth.ListUsersPayload) ([]*auth.Userresult, error) {
	log.Printf("[AUTH] ListUsers request: skip=%d, limit=%d", p.Skip, p.Limit)

	var users []domain.User
	query := s.db.Order("created_at DESC")

	if p.Skip > 0 {
		query = query.Offset(p.Skip)
	}
	if p.Limit > 0 {
		query = query.Limit(p.Limit)
	} else {
		query = query.Limit(100)
	}

	if err := query.Find(&users).Error; err != nil {
		log.Printf("[AUTH] ListUsers failed: database error: %v", err)
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	results := make([]*auth.Userresult, len(users))
	for i, user := range users {
		results[i] = convertUserToResult(&user)
	}

	log.Printf("[AUTH] ListUsers successful: returned %d users", len(results))
	return results, nil
}

// GetUser implements the get user method
func (s *AuthService) GetUser(ctx context.Context, p *auth.GetUserPayload) (*auth.Userresult, error) {
	log.Printf("[AUTH] GetUser request: id=%d", p.ID)

	var user domain.User
	if err := s.db.First(&user, p.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[AUTH] GetUser failed: user id=%d not found", p.ID)
			return nil, auth.MakeNotFound(fmt.Errorf("user not found"))
		}
		log.Printf("[AUTH] GetUser failed: database error: %v", err)
		return nil, err
	}

	log.Printf("[AUTH] GetUser successful: id=%d, username=%s", user.ID, user.Username)
	return convertUserToResult(&user), nil
}

// UpdateUser implements the update user method
func (s *AuthService) UpdateUser(ctx context.Context, p *auth.UpdateUserPayload) (*auth.Userresult, error) {
	log.Printf("[AUTH] UpdateUser request: id=%d", p.ID)

	var user domain.User
	if err := s.db.First(&user, p.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[AUTH] UpdateUser failed: user id=%d not found", p.ID)
			return nil, auth.MakeNotFound(fmt.Errorf("user not found"))
		}
		log.Printf("[AUTH] UpdateUser failed: database error: %v", err)
		return nil, err
	}

	// Update fields (with input sanitization)
	if p.Username != nil {
		username := strings.TrimSpace(*p.Username)
		// Check if username is taken by another user
		var existingUser domain.User
		if err := s.db.Where("username = ? AND id != ?", username, p.ID).First(&existingUser).Error; err == nil {
			log.Printf("[AUTH] UpdateUser failed: username '%s' already taken", username)
			return nil, auth.MakeBadRequest(fmt.Errorf("username already taken"))
		}
		user.Username = username
	}
	if p.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*p.Email))
		// Check if email is taken by another user
		var existingUser domain.User
		if err := s.db.Where("email = ? AND id != ?", email, p.ID).First(&existingUser).Error; err == nil {
			log.Printf("[AUTH] UpdateUser failed: email '%s' already taken", email)
			return nil, auth.MakeBadRequest(fmt.Errorf("email already taken"))
		}
		user.Email = email
	}
	if p.FullName != nil {
		fullName := strings.TrimSpace(*p.FullName)
		user.FullName = &fullName
	}
	if p.IsActive != nil {
		user.IsActive = *p.IsActive
	}
	if p.IsAdmin != nil {
		user.IsAdmin = *p.IsAdmin
	}
	if p.IsStaff != nil {
		user.IsStaff = *p.IsStaff
	}
	if p.Password != nil {
		password := strings.TrimSpace(*p.Password)
		hashedPassword, err := util.HashPassword(password)
		if err != nil {
			log.Printf("[AUTH] UpdateUser failed: password hashing error: %v", err)
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.HashedPassword = hashedPassword
	}

	if err := s.db.Save(&user).Error; err != nil {
		log.Printf("[AUTH] UpdateUser failed: database error: %v", err)
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	log.Printf("[AUTH] UpdateUser successful: id=%d, username=%s", user.ID, user.Username)
	return convertUserToResult(&user), nil
}

// DeleteUser implements the delete user method
func (s *AuthService) DeleteUser(ctx context.Context, p *auth.DeleteUserPayload) error {
	currentUser := ctx.Value("user").(*domain.User)
	log.Printf("[AUTH] DeleteUser request: id=%d by user=%s", p.ID, currentUser.Username)

	var user domain.User
	if err := s.db.First(&user, p.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[AUTH] DeleteUser failed: user id=%d not found", p.ID)
			return auth.MakeNotFound(fmt.Errorf("user not found"))
		}
		log.Printf("[AUTH] DeleteUser failed: database error: %v", err)
		return err
	}

	// Prevent self-deletion
	if user.ID == currentUser.ID {
		log.Printf("[AUTH] DeleteUser failed: user '%s' attempted self-deletion", currentUser.Username)
		return auth.MakeBadRequest(fmt.Errorf("cannot delete your own account"))
	}

	if err := s.db.Delete(&user).Error; err != nil {
		log.Printf("[AUTH] DeleteUser failed: database error: %v", err)
		return fmt.Errorf("failed to delete user: %w", err)
	}

	log.Printf("[AUTH] DeleteUser successful: deleted user id=%d, username=%s", user.ID, user.Username)
	return nil
}

// Helper function to convert User model to UserResult
func convertUserToResult(user *domain.User) *auth.Userresult {
	result := &auth.Userresult{
		ID:        int(user.ID),
		Username:  user.Username,
		Email:     user.Email,
		IsActive:  user.IsActive,
		IsAdmin:   user.IsAdmin,
		IsStaff:   user.IsStaff,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}

	if user.FullName != nil {
		result.FullName = user.FullName
	}
	if user.UpdatedAt.After(user.CreatedAt) {
		result.UpdatedAt = &[]string{user.UpdatedAt.Format(time.RFC3339)}[0]
	}
	if user.LastLogin != nil {
		result.LastLogin = &[]string{user.LastLogin.Format(time.RFC3339)}[0]
	}

	return result
}
