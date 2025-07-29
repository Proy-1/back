// File: controllers/controller.go
package controllers

import (
	"github.com/cloudinary/cloudinary-go/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

// Controller menampung dependensi yang akan digunakan oleh semua handler.
// Pastikan field diawali huruf besar agar bisa diakses dari package lain.
type Controller struct {
	DB              *mongo.Database
	Cld             *cloudinary.Cloudinary
	PasetoSecretKey []byte
}