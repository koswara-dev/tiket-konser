package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/smtp"
	"time"

	"user-service/config"
	"user-service/model"
	"user-service/redis"
	"user-service/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	cfg      *config.Config
	userRepo repository.UserRepository
}

func NewUserService(cfg *config.Config, userRepo repository.UserRepository) *UserService {
	return &UserService{cfg: cfg, userRepo: userRepo}
}

func (s *UserService) GenerateOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

func (s *UserService) SendOTPEmail(email, otp string) error {
	subject := "Subject: Concert Ticket Registration OTP\r\n"
	body := fmt.Sprintf("Hi,\r\n\r\nYour registration OTP code is: %s.\r\nIt will expire in 5 minutes.\r\n", otp)
	msg := []byte(subject + "\r\n" + body)

	addr := fmt.Sprintf("%s:%d", s.cfg.SmtpHost, s.cfg.SmtpPort)
	auth := smtp.PlainAuth("", s.cfg.SmtpUser, s.cfg.SmtpPassword, s.cfg.SmtpHost)

	log.Printf("[SMTP MOCK/LOG] Sending OTP email to %s (OTP: %s)", email, otp)

	if s.cfg.SmtpUser != "" && s.cfg.SmtpHost != "localhost" {
		err := smtp.SendMail(addr, auth, s.cfg.SmtpSender, []string{email}, msg)
		if err != nil {
			log.Printf("Warning: Failed to send actual email via SMTP: %v. Proceeding as mock.", err)
			return nil
		}
		log.Printf("Email successfully sent via SMTP to %s", email)
	} else {
		log.Printf("SMTP configuration is blank or points to localhost. Simulated email delivery.")
	}
	return nil
}

func (s *UserService) Register(name, email, password, role string) (*model.User, error) {
	_, err := s.userRepo.FindByEmail(email)
	if err == nil {
		return nil, errors.New("email is already registered")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	if role == "" {
		role = "user"
	}

	user := model.User{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
		Role:     role,
		IsActive: false,
	}

	otpCode := s.GenerateOTP()
	otp := model.OTP{
		Email:     email,
		Code:      otpCode,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := s.userRepo.CreateUserWithOTP(&user, &otp); err != nil {
		return nil, err
	}

	go func() {
		_ = s.SendOTPEmail(email, otpCode)
	}()

	return &user, nil
}

func (s *UserService) VerifyOTP(email, code string) error {
	return s.userRepo.VerifyOTPAndActivateUser(email, code)
}

func (s *UserService) Login(email, password string) (string, *model.User, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return "", nil, errors.New("invalid email or password")
	}

	if !user.IsActive {
		return "", nil, errors.New("account is not verified yet. please verify with OTP")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", nil, errors.New("invalid email or password")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.cfg.JwtSecret))
	if err != nil {
		return "", nil, err
	}

	return tokenString, user, nil
}

func (s *UserService) Logout(tokenString string, exp int64) error {
	timeToExpire := time.Until(time.Unix(exp, 0))
	if timeToExpire <= 0 {
		return nil
	}
	return redis.BlacklistToken(tokenString, timeToExpire)
}

func (s *UserService) ForceLogout(userID uint) error {
	return redis.SetUserForceLogout(userID)
}

func (s *UserService) IsTokenValid(userID uint, tokenString string, iat int64) (bool, error) {
	blacklisted, err := redis.IsTokenBlacklisted(tokenString)
	if err != nil || blacklisted {
		return false, err
	}

	forceLogoutTime, err := redis.GetUserForceLogoutTime(userID)
	if err != nil {
		return false, err
	}

	if forceLogoutTime > 0 && iat < forceLogoutTime {
		return false, nil
	}

	return true, nil
}

func (s *UserService) GetUserByID(id uint) (*model.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *UserService) ListUsers() ([]model.User, error) {
	return s.userRepo.FindAll()
}

func (s *UserService) UpdateUser(user *model.User) error {
	return s.userRepo.UpdateUser(user)
}
