package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Product mendefinisikan struktur untuk produk.
type Product struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Price       float64            `json:"price" bson:"price"`
	Description string             `json:"description" bson:"description"`
	Category    string             `json:"category" bson:"category"`
	Image       string             `json:"image" bson:"image"`
	ImageURL    string             `json:"image_url,omitempty" bson:"image_url,omitempty"`
	Stock       int                `json:"stock" bson:"stock"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
	ImageBase64 string             `json:"image_base64,omitempty" bson:"-"`
}

// Stats mendefinisikan struktur untuk statistik aplikasi.
type Stats struct {
	TotalProducts int64   `json:"total_products"`
	TotalAdmins   int64   `json:"total_admins"`
	TotalValue    float64 `json:"total_value"`
}