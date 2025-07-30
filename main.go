package main

import (
	"context"
	"fmt"
	"log"
	"pitipaw-backend/config"
	"pitipaw-backend/controllers"
	"pitipaw-backend/routes"

	"github.com/cloudinary/cloudinary-go/v2"
)

func main() {
	// 1. Muat Konfigurasi
	cfg := config.Load()

	// 2. Hubungkan ke Database
	dbClient, err := config.ConnectDB(cfg.MongoURI, cfg.MongoMode)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	defer func() {
		if err := dbClient.Disconnect(context.Background()); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}()
	db := dbClient.Database("pitipaw")

	// 3. Inisialisasi Cloudinary
	var cld *cloudinary.Cloudinary
	if cfg.CloudinaryURL != "" {
		cld, err = cloudinary.NewFromURL(cfg.CloudinaryURL)
		if err != nil {
			log.Fatalf("Failed to initialize Cloudinary: %v", err)
		}
		fmt.Println("☁️  Successfully connected to Cloudinary")
	}

	// 4. Inisialisasi Controller dengan dependensi
	ctrl := &controllers.Controller{
		DB:              db,
		Cld:             cld,
		PasetoSecretKey: cfg.PasetoSecretKey,
	}

	// 5. Atur Rute
	r := routes.Setup(ctrl, cfg.Env)

	// 6. Jalankan Server
	fmt.Printf("🚀 Server starting on port %s\n", cfg.Port)
	fmt.Printf("💡 API available at http://localhost:%s/api\n", cfg.Port)
	
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal("Error starting server:", err)
	}
}