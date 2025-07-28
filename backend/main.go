package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/o1egl/paseto"
	"golang.org/x/crypto/bcrypt"
)

// Health check handler
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

// Models
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
	ImageData   []byte             `bson:"image_data,omitempty" json:"image_data,omitempty"`
	ImageBase64 string             `json:"image_base64,omitempty" bson:"-"` // Only for response, not stored in DB
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
	client          *mongo.Client
	db              *mongo.Database
	products        *mongo.Collection
	admins          *mongo.Collection
	port            string
	uploadDir       string
	maxFileSize     int64
	mongoMode       string // "atlas" atau "local"
	pasetoSecretKey []byte // di-set di init()
	cloudinaryURL   string
	cld             *cloudinary.Cloudinary
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

	// Set Paseto secret key from environment variable
	key := getEnv("PASETO_SECRET_KEY", "")
	if len(key) != 32 {
		log.Fatal("PASETO_SECRET_KEY harus 32 karakter! Contoh: '12345678901234567890123456789012'")
	}
	pasetoSecretKey = []byte(key)

	// Set Cloudinary URL from environment variable
	cloudinaryURL = getEnv("CLOUDINARY_URL", "")
	if cloudinaryURL == "" {
		log.Println("CLOUDINARY_URL belum di-set di .env, upload gambar ke Cloudinary tidak aktif")
	} else {
		var err error
		cld, err = cloudinary.NewFromURL(cloudinaryURL)
		if err != nil {
			log.Fatalf("Gagal inisialisasi Cloudinary: %v", err)
		}
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
		log.Fatal("MONGO_URI_ATLAS belum di-set untuk Atlas!")
	}

	// Pastikan database name 'pitipaw' ada di connection string
	if !strings.Contains(mongoURI, "/pitipaw") && !strings.Contains(mongoURI, "mongodb.net/?") {
		// Jika connection string tidak memiliki database name, tambahkan
		mongoURI = strings.Replace(mongoURI, "mongodb.net/", "mongodb.net/pitipaw", 1)
	} else if strings.Contains(mongoURI, "mongodb.net/?") {
		// Jika ada /? di akhir, ganti dengan /pitipaw?
		mongoURI = strings.Replace(mongoURI, "mongodb.net/?", "mongodb.net/pitipaw?", 1)
	}

	fmt.Println("🌐 Using MongoDB Atlas (Cloud Database)")
	fmt.Printf("📡 Connecting to: %s\n", mongoURI)
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
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "cache-control"}
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
		// Endpoint untuk mengambil gambar dari database
		api.GET("/image/:id", getImageFromDB)

		// Statistics
		api.GET("/stats", getStats)
	}
	return r
}

func getProducts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := products.Find(ctx, bson.M{})
	if err != nil {
		log.Println("Error products.Find:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var productList []Product
	if err = cursor.All(ctx, &productList); err != nil {
		log.Println("Error cursor.All:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert image data to base64 for each product (for frontend display)
	for i := range productList {
		if len(productList[i].ImageData) > 0 {
			productList[i].ImageBase64 = "data:image/jpeg;base64," + encodeToBase64(productList[i].ImageData)
		}
		// Set image_url from database if available
		// If productList[i].ImageURL is empty, try to fill from Image field (for backward compatibility)
		if productList[i].ImageURL == "" && productList[i].Image != "" {
			// If Image is a path, use it as image_url
			productList[i].ImageURL = productList[i].Image
		}
		productList[i].ImageData = nil
	}

	log.Println("getProducts response count:", len(productList))
	c.JSON(http.StatusOK, gin.H{"products": productList})
}

// Helper to encode []byte to base64 string
func encodeToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func createProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Jika ada image_base64 dan Cloudinary aktif, upload ke Cloudinary
	if product.ImageBase64 != "" && cld != nil {
		base64Data := product.ImageBase64
		// Hilangkan prefix jika ada (misal: data:image/jpeg;base64,)
		if idx := len("data:image/jpeg;base64,"); len(base64Data) > idx && base64Data[:idx] == "data:image/jpeg;base64," {
			base64Data = base64Data[idx:]
		}
		// Upload ke Cloudinary
		uploadParams := uploader.UploadParams{
			Folder: "pitipaw/products",
		}
		uploadResult, err := cld.Upload.Upload(context.Background(), "data:image/jpeg;base64,"+base64Data, uploadParams)
		if err != nil {
			log.Println("Cloudinary upload error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal upload gambar ke Cloudinary"})
			return
		}
		product.ImageURL = uploadResult.SecureURL
		product.Image = uploadResult.PublicID
	}

	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	// Jangan simpan image_base64 di database
	product.ImageBase64 = ""
	product.ImageData = nil // Tidak simpan binary di database

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

	// Jika ada image_base64 dan Cloudinary aktif, upload ke Cloudinary
	if updateData.ImageBase64 != "" && cld != nil {
		base64Data := updateData.ImageBase64
		// Hilangkan prefix jika ada (misal: data:image/jpeg;base64,)
		if idx := len("data:image/jpeg;base64,"); len(base64Data) > idx && base64Data[:idx] == "data:image/jpeg;base64," {
			base64Data = base64Data[idx:]
		}
		// Upload ke Cloudinary
		uploadParams := uploader.UploadParams{
			Folder: "pitipaw/products",
		}
		uploadResult, err := cld.Upload.Upload(context.Background(), "data:image/jpeg;base64,"+base64Data, uploadParams)
		if err != nil {
			log.Println("Cloudinary upload error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal upload gambar ke Cloudinary"})
			return
		}
		updateData.ImageURL = uploadResult.SecureURL
		updateData.Image = uploadResult.PublicID
	}

	updateData.UpdatedAt = time.Now()
	// Jangan simpan image_base64 di database
	updateData.ImageBase64 = ""
	updateData.ImageData = nil

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
		log.Println("BindJSON error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var admin Admin
	err := admins.FindOne(ctx, bson.M{"username": req.Username}).Decode(&admin)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		log.Println("FindOne error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Compare password with bcrypt
	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)) != nil {
		log.Println("Password mismatch for user:", req.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate Paseto token
	now := time.Now()
	exp := now.Add(24 * time.Hour)
	jsonToken := paseto.JSONToken{
		Subject:    admin.ID.Hex(),
		IssuedAt:   now,
		Expiration: exp,
	}
	footer := "pitipaw-admin"
	token, err := paseto.NewV2().Encrypt(pasetoSecretKey, jsonToken, footer)
	if err != nil {
		log.Println("Paseto token generation error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	admin.Password = "" // Don't return password
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"admin":   admin,
		"token":   token,
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

	// Hash password with bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	admin := Admin{
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
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

	// Baca file sebagai []byte
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Simpan ke database sebagai dokumen baru di koleksi products (atau bisa juga update produk tertentu)
	product := Product{
		Name:      file.Filename,
		Image:     file.Filename,
		ImageData: fileBytes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	result, err := products.InsertOne(context.Background(), product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "File uploaded and saved to database successfully",
		"id":       result.InsertedID,
		"filename": file.Filename,
	})
}

// Handler untuk mengambil gambar dari database dan menampilkannya langsung
func getImageFromDB(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	var product Product
	err = products.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}
	if len(product.ImageData) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No image data found"})
		return
	}

	// Set Content-Type (bisa dideteksi dari nama file, di sini default ke image/jpeg)
	c.Header("Content-Type", "image/jpeg")
	c.Writer.Write(product.ImageData)
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
