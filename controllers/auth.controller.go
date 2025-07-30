package controllers

import (
	"context"
	"net/http"
	"pitipaw-backend/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/o1egl/paseto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)


// Login menangani proses login admin.
func (ctrl *Controller) Login(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var admin models.Admin
	collection := ctrl.DB.Collection("admins")
	err := collection.FindOne(ctx, bson.M{"username": req.Username}).Decode(&admin)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	now := time.Now()
	exp := now.Add(24 * time.Hour)
	jsonToken := paseto.JSONToken{
		Subject:    admin.ID.Hex(),
		IssuedAt:   now,
		Expiration: exp,
	}
	token, err := paseto.NewV2().Encrypt(ctrl.PasetoSecretKey, jsonToken, "pitipaw-admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	admin.Password = ""
	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "admin": admin, "token": token})
}

// Register menangani registrasi admin baru.
func (ctrl *Controller) Register(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	collection := ctrl.DB.Collection("admins")
	var existingAdmin models.Admin
	if err := collection.FindOne(ctx, bson.M{"username": req.Username}).Decode(&existingAdmin); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	admin := models.Admin{
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
	}

	result, err := collection.InsertOne(ctx, admin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	admin.ID = result.InsertedID.(primitive.ObjectID)
	admin.Password = ""
	c.JSON(http.StatusCreated, gin.H{"message": "Registration successful", "admin": admin})
}

// GetAdmins menangani pengambilan semua data admin.
func (ctrl *Controller) GetAdmins(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := ctrl.DB.Collection("admins")
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var adminList []models.Admin
	if err = cursor.All(ctx, &adminList); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range adminList {
		adminList[i].Password = ""
	}
	c.JSON(http.StatusOK, gin.H{"admins": adminList})
}

// DeleteAdmin menangani penghapusan admin.
func (ctrl *Controller) DeleteAdmin(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid admin ID"})
		return
	}

	collection := ctrl.DB.Collection("admins")
	result, err := collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Admin deleted successfully"})
}

// CreateAdmin menangani pembuatan admin baru.
// Fungsi ini mungkin tidak diperlukan jika Anda menggunakan endpoint /register.
func (ctrl *Controller) CreateAdmin(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var admin models.Admin
	if err := c.ShouldBindJSON(&admin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Anda harus menambahkan hashing password di sini seperti pada fungsi Register
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(admin.Password), bcrypt.DefaultCost)
	admin.Password = string(hashedPassword)
	admin.CreatedAt = time.Now()

	collection := ctrl.DB.Collection("admins")
	result, err := collection.InsertOne(ctx, admin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	admin.ID = result.InsertedID.(primitive.ObjectID)
	admin.Password = ""
	c.JSON(http.StatusCreated, gin.H{"admin": admin})
}