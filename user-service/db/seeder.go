package db

import (
	"log"
	"user-service/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedUsers(database *gorm.DB) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Indonesia"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Warning: Failed to hash seed passwords: %v", err)
		return
	}

	usersToSeed := []model.User{
		{
			Name:     "Admin Concert",
			Email:    "admin@concert.id",
			Password: string(hashedPassword),
			Role:     "admin",
			IsActive: true,
		},
		{
			Name:     "Staff Concert",
			Email:    "staff@concert.id",
			Password: string(hashedPassword),
			Role:     "staff",
			IsActive: true,
		},
		{
			Name:     "User Concert",
			Email:    "user@concert.id",
			Password: string(hashedPassword),
			Role:     "user",
			IsActive: true,
		},
	}

	for _, user := range usersToSeed {
		var count int64
		err := database.Model(&model.User{}).Where("email = ?", user.Email).Count(&count).Error
		if err != nil {
			log.Printf("Warning: Failed to check if user %s exists: %v", user.Email, err)
			continue
		}

		if count > 0 {
			log.Printf("User %s already exists. Skipping seeder.", user.Email)
			continue
		}

		if err := database.Create(&user).Error; err != nil {
			log.Printf("Warning: Failed to seed user %s: %v", user.Email, err)
			continue
		}

		log.Printf("User successfully seeded: %s (%s)", user.Name, user.Email)
	}
}
