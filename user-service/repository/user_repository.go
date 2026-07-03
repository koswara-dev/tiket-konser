package repository

import (
	"errors"
	"time"

	"user-service/model"

	"gorm.io/gorm"
)

type UserRepository interface {
	FindByEmail(email string) (*model.User, error)
	CreateUserWithOTP(user *model.User, otp *model.OTP) error
	VerifyOTPAndActivateUser(email, code string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByEmail(email string) (*model.User, error) {
	if r.db == nil {
		return nil, errors.New("database connection is unavailable")
	}
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) CreateUserWithOTP(user *model.User, otp *model.OTP) error {
	if r.db == nil {
		return errors.New("database connection is unavailable")
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if err := tx.Create(otp).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *userRepository) VerifyOTPAndActivateUser(email, code string) error {
	if r.db == nil {
		return errors.New("database connection is unavailable")
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		var otp model.OTP
		err := tx.Where("email = ? AND code = ? AND expires_at > ?", email, code, time.Now()).
			Order("created_at desc").First(&otp).Error
		if err != nil {
			return errors.New("invalid or expired OTP code")
		}

		err = tx.Model(&model.User{}).Where("email = ?", email).Update("is_active", true).Error
		if err != nil {
			return err
		}

		return tx.Delete(&otp).Error
	})
}
