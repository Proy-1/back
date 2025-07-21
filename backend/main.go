package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Models
type Product struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Price       float64            `json:"price" bson:"price"`
	Description string             `json:"description" bson:"description"`
	Category    string             `json:"category" bson:"category"`
	Image       string             `json:"image" bson:"image"`
	Stock       int                `json:"stock" bson:"stock"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

type Admin struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Username  string             `json:"username" bson:"username"`
	Email     string             `json:"email" bson:"email"`
	Password  string             `json:"password" bson:"password"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type Stats struct {
	TotalProducts int64   `json:"total_products"`
	TotalAdmins   int64   `json:"total_admins"`
	TotalValue    float64 `json:"total_value"`
}

// Global variables
var (
	client      *mongo.Client
	db          *mongo.Database
	products    *mongo.Collection
	admins      *mongo.Collection
	port        string
	uploadDir   string
	maxFileSize int64
	mongoMode   string // "atlas" atau "local"
)

func init() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default values")
	}

	// Set default values
	port = getEnv("PORT", "5000")
	uploadDir = getEnv("UPLOAD_DIR", "static/uploads")
	maxFileSizeStr := getEnv("MAX_FILE_SIZE", "10485760") // 10MB default

	var err error
	maxFileSize, err = strconv.ParseInt(maxFileSizeStr, 10, 64)
	if err != nil {
		maxFileSize = 10485760 // 10MB default
	}

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("Error creating upload directory: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func connectMongoLocal() {
	mongoURI := getEnv("MONGO_URI_LOCAL", "mongodb://localhost:27017/pitipaw")
	fmt.Println("🏠 Using Local MongoDB")
	connectMongo(mongoURI)
}

func connectMongoAtlas() {
	mongoURI := getEnv("MONGO_URI_ATLAS", "")
	if mongoURI == "" {
		// fallback ke MONGO_URI jika tidak ada
		mongoURI = getEnv("MONGO_URI", "")
	}
	if mongoURI == "" {
		log.Fatal("MONGO_URI_ATLAS atau MONGO_URI belum di-set untuk Atlas!")
	}
	fmt.Println("🌐 Using MongoDB Atlas (Cloud Database)")
	connectMongo(mongoURI)
}

func connectMongo(mongoURI string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Error connecting to MongoDB:", err)
	}

	// Test connection
	if err = client.Ping(ctx, nil); err != nil {
		log.Fatal("Error pinging MongoDB:", err)
	}

	fmt.Println("✅ Connected to MongoDB successfully!")

	// Initialize database and collections
	db = client.Database("pitipaw")
	products = db.Collection("products")
	admins = db.Collection("admins")
}

func connectMongoDB() {
	mongoMode = getEnv("MONGO_MODE", "local")
	if mongoMode == "atlas" {
		connectMongoAtlas()
	} else {
		connectMongoLocal()
	}
}

func setupRoutes() *gin.Engine {
	// Set Gin mode based on environment
	if getEnv("ENVIRONMENT", "development") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// CORS configuration (umum untuk semua route)
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://localhost:8000", "http://127.0.0.1:8000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	// CORS khusus Atlas (jika ingin, misal untuk endpoint monitoring Atlas)
	r.OPTIONS("/atlas/*any", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Status(http.StatusOK)
	})

	// Serve static files
	r.Static("/static", "./static")

	// API routes
	api := r.Group("/api")
	{
		// Health check
		api.GET("/health", healthCheck)

		// Products
		api.GET("/products", getProducts)
		api.POST("/products", createProduct)
		api.GET("/products/:id", getProduct)
		api.PUT("/products/:id", updateProduct)
		api.DELETE("/products/:id", deleteProduct)

		// Admins
		api.GET("/admins", getAdmins)
		api.POST("/admins", createAdmin)
		api.DELETE("/admins/:id", deleteAdmin)

		// Authentication
		api.POST("/login", login)
		api.POST("/register", register)

		// File upload
		api.POST("/upload", uploadFile)

		// Statistics
		api.GET("/stats", getStats)
	}

	return r
}

// Handlers
func healthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test database connection
	err := client.Ping(ctx, nil)
	status := "ok"
	dbStatus := "connected"

	if err != nil {
		status = "error"
		dbStatus = "disconnected"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    status,
		"database":  dbStatus,
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

func getProducts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := products.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var productList []Product
	if err = cursor.All(ctx, &productList); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"products": productList})
}

func createProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	result, err := products.InsertOne(ctx, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	product.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, gin.H{"product": product})
}

func getProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product Product
	err = products.FindOne(ctx, bson.M{"_id": objectID}).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

func updateProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var updateData Product
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateData.UpdatedAt = time.Now()
	update := bson.M{"$set": updateData}

	result, err := products.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product updated successfully"})
}

func deleteProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	result, err := products.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

func getAdmins(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := admins.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var adminList []Admin
	if err = cursor.All(ctx, &adminList); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Remove passwords from response
	for i := range adminList {
		adminList[i].Password = ""
	}

	c.JSON(http.StatusOK, gin.H{"admins": adminList})
}

func createAdmin(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var admin Admin
	if err := c.ShouldBindJSON(&admin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin.CreatedAt = time.Now()

	result, err := admins.InsertOne(ctx, admin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	admin.ID = result.InsertedID.(primitive.ObjectID)
	admin.Password = "" // Don't return password
	c.JSON(http.StatusCreated, gin.H{"admin": admin})
}

func deleteAdmin(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid admin ID"})
		return
	}

	result, err := admins.DeleteOne(ctx, bson.M{"_id": objectID})
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

func login(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var admin Admin
	err := admins.FindOne(ctx, bson.M{"username": req.Username, "password": req.Password}).Decode(&admin)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	admin.Password = "" // Don't return password
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"admin":   admin,
		"token":   "dummy_token", // In production, use JWT
	})
}

func register(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if username already exists
	var existingAdmin Admin
	err := admins.FindOne(ctx, bson.M{"username": req.Username}).Decode(&existingAdmin)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	admin := Admin{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password, // In production, hash the password
		CreatedAt: time.Now(),
	}

	result, err := admins.InsertOne(ctx, admin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	admin.ID = result.InsertedID.(primitive.ObjectID)
	admin.Password = "" // Don't return password
	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful",
		"admin":   admin,
	})
}

func uploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Check file size
	if file.Size > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large"})
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	filepath := filepath.Join(uploadDir, filename)

	// Save file
	if err := c.SaveUploadedFile(file, filepath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return file URL
	fileURL := fmt.Sprintf("/static/uploads/%s", filename)
	c.JSON(http.StatusOK, gin.H{
		"message":  "File uploaded successfully",
		"filename": filename,
		"url":      fileURL,
	})
}

func getStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get total products
	totalProducts, err := products.CountDocuments(ctx, bson.M{})
	if err != nil {
		totalProducts = 0
	}

	// Get total admins
	totalAdmins, err := admins.CountDocuments(ctx, bson.M{})
	if err != nil {
		totalAdmins = 0
	}

	// Calculate total value
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":   nil,
				"total": bson.M{"$sum": bson.M{"$multiply": []string{"$price", "$stock"}}},
			},
		},
	}

	cursor, err := products.Aggregate(ctx, pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var result []bson.M
	var totalValue float64 = 0
	if err = cursor.All(ctx, &result); err == nil && len(result) > 0 {
		if val, ok := result[0]["total"].(float64); ok {
			totalValue = val
		}
	}

	stats := Stats{
		TotalProducts: totalProducts,
		TotalAdmins:   totalAdmins,
		TotalValue:    totalValue,
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func main() {
	// Connect to MongoDB
	connectMongoDB()
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}()

	// Setup routes
	r := setupRoutes()

	// Start server
	fmt.Printf("🚀 Server starting on port %s\n", port)
	fmt.Printf("💡 API available at http://localhost:%s/api\n", port)
	fmt.Printf("📊 Health check: http://localhost:%s/api/health\n", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Error starting server:", err)
	}
}
