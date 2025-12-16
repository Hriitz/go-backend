package domain

import (
	"time"
	"gorm.io/gorm"
)

// InvestmentInquiry represents an investment inquiry
type InvestmentInquiry struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	FirstName       *string    `json:"first_name"`
	LastName        *string    `json:"last_name"`
	Phone           *string    `gorm:"index" json:"phone"`
	Email           *string    `gorm:"index" json:"email"`
	InvestmentSize  *string    `json:"investment_size"`
	CurrentExposure *string    `json:"current_exposure"`
	Verified        bool       `gorm:"default:false" json:"verified"`
	ExitType        *string    `gorm:"default:'abandoned'" json:"exit_type"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
}

// TableName specifies the table name for InvestmentInquiry
func (InvestmentInquiry) TableName() string {
	return "investment_inquiries"
}

// BeforeCreate hook
func (i *InvestmentInquiry) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	i.CreatedAt = now
	if i.ExitType == nil {
		defaultExitType := "abandoned"
		i.ExitType = &defaultExitType
	}
	return nil
}

// BeforeUpdate hook
func (i *InvestmentInquiry) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now()
	i.UpdatedAt = &now
	return nil
}


