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

	"strings"

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
	subject := "Subject: Concert Ticket Registration OTP\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	spacedOTP := ""
	for i, r := range otp {
		if i > 0 {
			spacedOTP += " "
		}
		spacedOTP += string(r)
	}

	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  body {
    margin: 0;
    padding: 0;
    background-color: #f7f9fb;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  }
</style>
</head>
<body style="margin: 0; padding: 40px 0; background-color: #f7f9fb; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">
  <table border="0" cellpadding="0" cellspacing="0" width="100%">
    <tr>
      <td align="center">
        <!-- Main Container -->
        <table border="0" cellpadding="0" cellspacing="0" width="600" style="background-color: #ffffff; border: 1px solid #e1e8ed; border-radius: 8px; overflow: hidden; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05);">
          <!-- Header -->
          <tr>
            <td style="padding: 20px 30px; border-bottom: 1px solid #e1e8ed;">
              <table border="0" cellpadding="0" cellspacing="0" width="100%">
                <tr>
                  <td>
                    <img src="https://upload.wikimedia.org/wikipedia/commons/thumb/5/5c/Logo_BCA.svg/512px-Logo_BCA.svg.png" alt="BCA" style="height: 30px; display: block;" />
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <!-- Body Content -->
          <tr>
            <td style="padding: 40px 40px 30px 40px; text-align: center;">
              <!-- Lock Icon -->
              <table border="0" cellpadding="0" cellspacing="0" width="100%">
                <tr>
                  <td align="center" style="padding-bottom: 15px;">
                    <img src="https://cdn-icons-png.flaticon.com/512/1000/1000966.png" alt="Lock" style="width: 48px; height: 48px; display: block;" />
                  </td>
                </tr>
                <tr>
                  <td align="center" style="padding-bottom: 15px;">
                    <h2 style="margin: 0; font-size: 22px; font-weight: 700; color: #1a1a1a; letter-spacing: -0.5px;">Kode OTP Anda</h2>
                  </td>
                </tr>
                <tr>
                  <td align="center" style="padding-bottom: 30px;">
                    <p style="margin: 0; font-size: 14px; line-height: 1.6; color: #4a5568; max-width: 480px;">
                      Gunakan kode berikut untuk memverifikasi transaksi Anda. Jangan bagikan kode ini kepada siapapun, termasuk petugas bank.
                    </p>
                  </td>
                </tr>
                <!-- OTP Box -->
                <tr>
                  <td align="center" style="padding-bottom: 30px;">
                    <table border="0" cellpadding="0" cellspacing="0" width="100%" style="background-color: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 24px 20px;">
                      <tr>
                        <td align="center" style="font-size: 11px; font-weight: 700; color: #94a3b8; text-transform: uppercase; letter-spacing: 1.5px; padding-bottom: 12px;">
                          Kode Rahasia
                        </td>
                      </tr>
                      <tr>
                        <td align="center" style="font-size: 32px; font-weight: 700; color: #005691; letter-spacing: 8px; padding-left: 8px;">
                          {{.OTP}}
                        </td>
                      </tr>
                    </table>
                  </td>
                </tr>
                <!-- Warning Box -->
                <tr>
                  <td align="center" style="padding-bottom: 20px;">
                    <table border="0" cellpadding="0" cellspacing="0" width="100%" style="background-color: #fef2f2; border: 1px solid #fee2e2; border-radius: 8px; padding: 16px;">
                      <tr>
                        <td valign="top" style="padding-right: 12px; width: 20px;">
                          <!-- Warning Icon -->
                          <span style="font-size: 18px; color: #b91c1c; display: block; line-height: 1;">⚠️</span>
                        </td>
                        <td align="left" style="font-size: 13px; line-height: 1.5; color: #991b1b;">
                          <strong style="color: #b91c1c; display: block; margin-bottom: 4px;">Peringatan Keamanan</strong>
                          Kode ini hanya berlaku selama <strong>5 menit</strong>. Bank BCA tidak pernah meminta kode OTP untuk alasan apapun. Jika Anda tidak meminta kode ini, segera hubungi Halo BCA di 1500888.
                        </td>
                      </tr>
                    </table>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <!-- Footer -->
          <tr>
            <td style="padding: 30px 40px; background-color: #f8fafc; border-top: 1px solid #e1e8ed; text-align: center;">
              <p style="margin: 0 0 12px 0; font-size: 12px; font-weight: 600; color: #4a5568;">PT Bank Central Asia Tbk.</p>
              <p style="margin: 0 0 16px 0; font-size: 12px; color: #718096;">
                <a href="#" style="color: #4a5568; text-decoration: none;">Privacy Policy</a> &nbsp;|&nbsp; 
                <a href="#" style="color: #4a5568; text-decoration: none;">Contact Us</a> &nbsp;|&nbsp; 
                <a href="#" style="color: #4a5568; text-decoration: none;">Terms of Service</a>
              </p>
              <p style="margin: 0; font-size: 11px; color: #a0aec0;">&copy; 2024 PT Bank Central Asia Tbk. All Rights Reserved.</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`

	htmlBody := strings.ReplaceAll(htmlTemplate, "{{.OTP}}", spacedOTP)
	msg := []byte(subject + mime + htmlBody)

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
