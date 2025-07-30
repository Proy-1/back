package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// AppConfig menampung semua variabel konfigurasi aplikasi.
type AppConfig struct {
	Port            string
	Env             string
	MongoMode       string
	MongoURI        string
	PasetoSecretKey []byte
	CloudinaryURL   string
}

// Load memuat konfigurasi dari file .env atau environment variables.
func Load() *AppConfig {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &AppConfig{
		Port:          getEnv("PORT", "5000"),
		Env:           getEnv("ENVIRONMENT", "development"),
		MongoMode:     getEnv("MONGO_MODE", "local"),
		CloudinaryURL: getEnv("CLOUDINARY_URL", ""),
	}

	// Atur URI MongoDB berdasarkan mode
	if cfg.MongoMode == "atlas" {
		cfg.MongoURI = getEnv("MONGO_URI_ATLAS", "")
		if cfg.MongoURI == "" {
			log.Fatal("MONGO_MODE 'atlas' but MONGO_URI_ATLAS is not set")
		}
	} else {
		cfg.MongoURI = getEnv("MONGO_URI_LOCAL", "mongodb://localhost:27017/pitipaw")
	}

	// Atur Kunci Paseto
	key := getEnv("PASETO_SECRET_KEY", "")
	if len(key) != 32 {
		log.Fatal("PASETO_SECRET_KEY must be 32 characters long!")
	}
	cfg.PasetoSecretKey = []byte(key)

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}