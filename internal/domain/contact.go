package domain

import (
	"time"
	"gorm.io/gorm"
)

// ContactInquiry represents a contact form submission
type ContactInquiry struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Name      string     `gorm:"not null" json:"name"`
	Email     string     `gorm:"not null;index" json:"email"`
	Phone     *string    `json:"phone"`
	Message   string     `gorm:"type:text;not null" json:"message"`
	Status    string     `gorm:"default:'new'" json:"status"` // new, read, replied
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// TableName specifies the table name for ContactInquiry
func (ContactInquiry) TableName() string {
	return "contact_inquiries"
}

// BeforeCreate hook
func (c *ContactInquiry) BeforeCreate(tx *gorm.DB) error {
	c.CreatedAt = time.Now()
	if c.Status == "" {
		c.Status = "new"
	}
	return nil
}

// BeforeUpdate hook
func (c *ContactInquiry) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now()
	c.UpdatedAt = &now
	return nil
}




