package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type ENVConfig struct {
	DBHost string
	DBPort string
	DBUser string
	DBPass string
	DBService string
	SSHHost string
	SSHPort string
	SSHUser string
	SSHPass string
}

func LoadAllConfig() *ENVConfig{
		if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env not found, using system environment variables")
	}
	return &ENVConfig{
		DBHost: os.Getenv("DB_HOST"),
		DBPort: os.Getenv("DB_PORT"),
		DBUser: os.Getenv("DB_USER"),
		DBPass: os.Getenv("DB_PASS"),
		DBService: os.Getenv("DB_SERVICE"),
		SSHHost: os.Getenv("SSH_HOST"),
		SSHPort: os.Getenv("SSH_PORT"),
		SSHUser: os.Getenv("SSH_USER"),
		SSHPass: os.Getenv("SSH_PASS"),
	}
	

}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
