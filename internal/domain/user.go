package domain

import (
	"time"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	Username       string     `gorm:"uniqueIndex;not null" json:"username"`
	Email          string     `gorm:"uniqueIndex;not null" json:"email"`
	HashedPassword string     `gorm:"not null" json:"-"`
	FullName       *string    `json:"full_name"`
	IsActive       bool       `gorm:"default:true" json:"is_active"`
	IsAdmin        bool       `gorm:"default:false" json:"is_admin"`
	IsStaff        bool       `gorm:"default:false" json:"is_staff"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	LastLogin      *time.Time `json:"last_login"`
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}

// BeforeCreate hook
func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate hook
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}


