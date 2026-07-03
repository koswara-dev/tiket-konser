package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	AppEnv        string
	DbHost        string
	DbPort        string
	DbUser        string
	DbPassword    string
	DbName        string
	DbSSLMode     string
	RedisHost     string
	RedisPort     string
	RedisPassword string
	JwtSecret     string
	SmtpHost      string
	SmtpPort      int
	SmtpUser      string
	SmtpPassword  string
	SmtpSender    string
}

func LoadConfig() *Config {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	var envFile string
	if env == "production" {
		envFile = ".env.production"
	} else {
		envFile = ".env.development"
	}

	log.Printf("Loading environment configuration from: %s", envFile)
	// Try loading the env file from current dir or parent dir if needed
	_ = godotenv.Load(envFile)

	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))

	return &Config{
		Port:          getEnv("PORT", "8081"),
		AppEnv:        getEnv("APP_ENV", "development"),
		DbHost:        getEnv("DB_HOST", "localhost"),
		DbPort:        getEnv("DB_PORT", "5432"),
		DbUser:        getEnv("DB_USER", "postgres"),
		DbPassword:    getEnv("DB_PASSWORD", "password123"),
		DbName:        getEnv("DB_NAME", "concert_db"),
		DbSSLMode:     getEnv("DB_SSLMODE", "disable"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		JwtSecret:     getEnv("JWT_SECRET", "super_secret_jwt_key"),
		SmtpHost:      getEnv("SMTP_HOST", "localhost"),
		SmtpPort:      smtpPort,
		SmtpUser:      getEnv("SMTP_USER", ""),
		SmtpPassword:  getEnv("SMTP_PASSWORD", ""),
		SmtpSender:    getEnv("SMTP_SENDER", "noreply@concert.com"),
	}
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
