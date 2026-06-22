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
		DBHost: getEnv("DB_HOST", "10.6.11.157"),
		DBPort: getEnv("DB_PORT", "1521"),
		DBUser: getEnv("DB_USER", "SCONE_B2B"),
		DBPass: getEnv("DB_PASS", ""),
		DBService: getEnv("DB_SERVICE", "SCONE_B2B"),
		SSHHost: getEnv("SSH_HOST", "10.62.169.91"),
		SSHPort: getEnv("SSH_PORT", "22"),
		SSHUser: getEnv("SSH_USER", "neuron_indratristia"),
		SSHPass: getEnv("SSH_PASS", ""),
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
