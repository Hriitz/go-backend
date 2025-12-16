package main

import (
	"fmt"
	"log"

	"springstreet/internal/util"
	"springstreet/internal/database"
	"springstreet/internal/domain"
	"springstreet/internal/config"
)

func main() {
	// Load configuration
	_, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	if err := database.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	db := database.GetDB()

	// Check if admin already exists
	var existingUser domain.User
	if err := db.Where("username = ?", "admin").First(&existingUser).Error; err == nil {
		fmt.Println("Admin user already exists!")
		return
	}

	// Create admin user
	hashedPassword, err := util.HashPassword("admin")
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	fullName := "System Administrator"
	adminUser := domain.User{
		Username:       "admin",
		Email:          "admin@springstreet.com",
		HashedPassword: hashedPassword,
		FullName:       &fullName,
		IsActive:       true,
		IsAdmin:        true,
		IsStaff:        true,
	}

	if err := db.Create(&adminUser).Error; err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	fmt.Println("Admin user created successfully!")
	fmt.Println("Username: admin")
	fmt.Println("Password: admin")
	fmt.Println("Please change the password after first login!")
}


