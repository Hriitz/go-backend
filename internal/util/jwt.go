package util

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"springstreet/internal/config"
	"springstreet/internal/domain"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims represents JWT claims
type Claims struct {
	Username string `json:"sub"`
	IsAdmin  bool   `json:"is_admin"`
	IsStaff  bool   `json:"is_staff"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for a user
func GenerateToken(user *domain.User) (string, error) {
	cfg := config.Get()
	expirationTime := time.Now().Add(time.Duration(cfg.Auth.TokenExpiryMinutes) * time.Minute)
	
	claims := &Claims{
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		IsStaff:  user.IsStaff,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Auth.SecretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*Claims, error) {
	cfg := config.Get()
	claims := &Claims{}
	
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.Auth.SecretKey), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, ErrExpiredToken
	}

	return claims, nil
}

// GetUserFromToken gets user from token claims
func GetUserFromToken(db *gorm.DB, claims *Claims) (*domain.User, error) {
	var user domain.User
	if err := db.Where("username = ?", claims.Username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

// RequireAdmin checks if user is admin
func RequireAdmin(user *domain.User) error {
	if !user.IsAdmin {
		return errors.New("admin access required")
	}
	return nil
}

// RequireStaff checks if user is staff or admin
func RequireStaff(user *domain.User) error {
	if !user.IsStaff && !user.IsAdmin {
		return errors.New("staff or admin access required")
	}
	return nil
}


