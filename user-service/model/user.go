package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name     string `json:"name" gorm:"type:varchar(100);not null"`
	Email    string `json:"email" gorm:"type:varchar(100);uniqueIndex;not null"`
	Password string `json:"-" gorm:"type:varchar(255);not null"`
	IsActive bool   `json:"is_active" gorm:"default:false"`
	Role     string `json:"role" gorm:"type:varchar(20);default:'user';not null"`
}

type OTP struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"type:varchar(100);index;not null"`
	Code      string    `json:"code" gorm:"type:varchar(6);not null"`
	Purpose   string    `json:"purpose" gorm:"type:varchar(50);default:'registration'"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time
}
