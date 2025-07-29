package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Admin mendefinisikan struktur untuk pengguna admin.
type Admin struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Username  string             `json:"username" bson:"username"`
	Email     string             `json:"email" bson:"email"`
	Password  string             `json:"password" bson:"password"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

// LoginRequest mendefinisikan struktur untuk permintaan login.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest mendefinisikan struktur untuk permintaan registrasi.
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}