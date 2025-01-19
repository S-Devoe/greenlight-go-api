package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database Configuration
	DBName     string `env:"DB_NAME"`
	DBUser     string `env:"DB_USER"`
	DBPassword string `env:"DB_PASSWORD"`
	DbSource   string `env:"DB_SOURCE"`
	Port       int    `env:"PORT"`
	Env        string `env:"ENV"`

	// Rate Limiting Configuration
	LimiterRPS     int  `env:"LIMITER_RPS"`
	LimiterBurst   int  `env:"LIMITER_BURST"`
	LimiterEnabled bool `env:"LIMITER_ENABLED"`

	// SMTP Settings
	SMTPHost     string `env:"SMTP_HOST"`
	SMTPPort     int    `env:"SMTP_PORT"`
	SMTPUsername string `env:"SMTP_USERNAME"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	SMTPSender   string `env:"SMTP_SENDER"`
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	// Load the environment variable value
	val := getEnv(key, fmt.Sprintf("%d", fallback))

	// Try to convert it to an integer
	intVal, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("Warning: Invalid value for %s, using fallback %d", key, fallback)
		return fallback
	}

	return intVal
}

func InitConfig() Config {
	godotenv.Load()

	return Config{
		DBName:     getEnv("DB_NAME", "greenlight"),
		Port:       getEnvInt("PORT", 4000),
		DBUser:     getEnv("DB_USER", "greenlight"),
		DBPassword: getEnv("DB_PASSWORD", "greenlight1234"),
		DbSource:   getEnv("DB_SOURCE", "postgresql://greenlight_user:pa55word@localhost:5432/greenlight?sslmode=disable"),

		LimiterRPS:     getEnvInt("LIMITER_RPS", 2),
		LimiterBurst:   getEnvInt("LIMITER_BURST", 4),
		LimiterEnabled: getEnvBool("LIMITER_ENABLED", true),
		SMTPHost:       getEnv("SMTP_HOST", "smtp.mailtrap.io"),
		SMTPPort:       getEnvInt("SMTP_PORT", 2525),
		SMTPUsername:   getEnv("SMTP_USERNAME", ""),
		SMTPPassword:   getEnv("SMTP_PASSWORD", ""),
		SMTPSender:     getEnv("SMTP_SENDER", ""),
	}
}

var Envs = InitConfig()

func getEnvBool(key string, fallback bool) bool {
	// Load the environment variable value
	val := getEnv(key, fmt.Sprintf("%v", fallback))

	// Try to convert it to a boolean
	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		log.Printf("Warning: Invalid value for %s, using fallback %v", key, fallback)
		return fallback
	}

	return boolVal
}
