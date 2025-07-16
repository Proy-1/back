package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// Database
var db *mongo.Database

// Models
type Product struct {
	ID          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Price       float64            `json:"price" bson:"price"`
	Description string             `json:"description" bson:"description"`
	ImageURL    string             `json:"image_url" bson:"image_url"`
}

type Admin struct {
	ID       primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Username string             `json:"username" bson:"username"`
	Password string             `json:"password,omitempty" bson:"password"`
}

type ProductInput struct {
	Name        string  `json:"name" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
	Description string  `json:"description"`
	ImageURL    string  `json:"image_url"`
}

type AdminInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Constants
const (
	MaxFileSize = 10 * 1024 * 1024 // 10MB
	UploadDir   = "static/uploads"
)

var allowedExtensions = map[string]bool{
	"png":  true,
	"jpg":  true,
	"jpeg": true,
	"gif":  true,
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file")
	}

	// Initialize MongoDB
	initMongoDB()

	// Create upload directory
	os.MkdirAll(UploadDir, 0755)

	// Initialize Gin router
	r := gin.Default()

	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:3000",
		"http://localhost:8000",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:8000",
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// Static files with CORS
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
		api.POST("/register", registerAdmin)
		api.GET("/login", loginInfo)
		api.POST("/login", loginAdmin)

		// Upload
		api.POST("/upload", uploadImage)

		// Stats
		api.GET("/stats", getStats)
	}

	// Handler static dashboard (semua selain /api dan /static)
	dashboardPath := "../dashboard"
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") || strings.HasPrefix(c.Request.URL.Path, "/static") {
			c.Status(http.StatusNotFound)
			return
		}
		file := c.Request.URL.Path
		if file == "/" || file == "" {
			file = "/index.html"
		}
		fullPath := dashboardPath + file
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// Fallback ke index.html jika file tidak ada (SPA mode)
			c.File(dashboardPath + "/index.html")
			return
		}
		c.File(fullPath)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	fmt.Println("🚀 Go Backend Starting...")
	fmt.Printf("📊 Database: %s\n", os.Getenv("MONGO_URI"))
	fmt.Printf("📁 Upload folder: %s\n", UploadDir)
	fmt.Println("🌐 CORS enabled for frontend")
	fmt.Println("📋 Available endpoints:")
	fmt.Println("   GET  /api/health")
	fmt.Println("   GET  /api/products")
	fmt.Println("   POST /api/products")
	fmt.Println("   GET  /api/products/<id>")
	fmt.Println("   PUT  /api/products/<id>")
	fmt.Println("   DELETE /api/products/<id>")
	fmt.Println("   GET  /api/admins")
	fmt.Println("   POST /api/admins")
	fmt.Println("   DELETE /api/admins/<id>")
	fmt.Println("   GET  /api/login")
	fmt.Println("   POST /api/login")
	fmt.Println("   POST /api/register")
	fmt.Println("   POST /api/upload")
	fmt.Println("   GET  /api/stats")

	log.Fatal(r.Run(":" + port))
}

func initMongoDB() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017/pitipaw"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	// Test connection
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	db = client.Database("pitipaw")
	log.Println("✅ Connected to MongoDB")
}

// Health Check
func healthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test database connection
	err := db.RunCommand(ctx, bson.D{{"ping", 1}}).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Database connection failed",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"message":  "Backend is running",
		"database": "connected",
	})
}

// Product handlers
func getProducts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.Collection("products")
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching products: " + err.Error()})
		return
	}

	var products []Product
	if err = cursor.All(ctx, &products); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing products: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}

func createProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var input ProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama dan harga produk wajib diisi"})
		return
	}

	product := Product{
		Name:        input.Name,
		Price:       input.Price,
		Description: input.Description,
		ImageURL:    input.ImageURL,
	}

	collection := db.Collection("products")
	result, err := collection.InsertOne(ctx, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating product: " + err.Error()})
		return
	}

	product.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, product)
}

func getProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	collection := db.Collection("products")
	var product Product
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching product: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, product)
}

func updateProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak boleh kosong"})
		return
	}

	if len(input) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada data untuk diupdate"})
		return
	}

	collection := db.Collection("products")
	result, err := collection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": input})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating product: " + err.Error()})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
		return
	}

	var product Product
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching updated product: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, product)
}

func deleteProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	collection := db.Collection("products")
	result, err := collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting product: " + err.Error()})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Produk berhasil dihapus"})
}

// Admin handlers
func getAdmins(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.Collection("admins")
	cursor, err := collection.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"password": 0}))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching admins: " + err.Error()})
		return
	}

	var admins []Admin
	if err = cursor.All(ctx, &admins); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing admins: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, admins)
}

func createAdmin(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var input AdminInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username dan password wajib diisi"})
		return
	}

	collection := db.Collection("admins")

	// Check if username already exists
	var existingAdmin Admin
	err := collection.FindOne(ctx, bson.M{"username": input.Username}).Decode(&existingAdmin)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username sudah ada"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
		return
	}

	admin := Admin{
		Username: input.Username,
		Password: string(hashedPassword),
	}

	result, err := collection.InsertOne(ctx, admin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating admin: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"_id":      result.InsertedID,
		"username": admin.Username,
		"message":  "Admin created successfully",
	})
}

func deleteAdmin(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid admin ID"})
		return
	}

	collection := db.Collection("admins")
	result, err := collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting admin: " + err.Error()})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Admin berhasil dihapus"})
}

func registerAdmin(c *gin.Context) {
	createAdmin(c)
}

func loginInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Login endpoint ready",
		"methods": []string{"POST"},
	})
}

func loginAdmin(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username dan password wajib diisi"})
		return
	}

	collection := db.Collection("admins")
	var admin Admin
	err := collection.FindOne(ctx, bson.M{"username": input.Username}).Decode(&admin)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username/password salah"})
		return
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(input.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username/password salah"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login berhasil",
		"admin":   gin.H{"username": admin.Username},
	})
}

func uploadImage(c *gin.Context) {
	// Check if file exists
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file part"})
		return
	}

	// Check file size
	if file.Size > MaxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("File terlalu besar. Maksimal 10MB (ukuran file: %.1fMB)", float64(file.Size)/(1024*1024)),
		})
		return
	}

	// Check file extension
	filename := file.Filename
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File not allowed"})
		return
	}

	ext := strings.ToLower(parts[len(parts)-1])
	if !allowedExtensions[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File not allowed"})
		return
	}

	// Save file
	filepath := fmt.Sprintf("%s/%s", UploadDir, filename)
	if err := c.SaveUploadedFile(file, filepath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving file: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"image_url": fmt.Sprintf("/static/uploads/%s", filename),
		"file_size": fmt.Sprintf("%.1fMB", float64(file.Size)/(1024*1024)),
	})
}

func getStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	productsCount, err := db.Collection("products").CountDocuments(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stats error: " + err.Error()})
		return
	}

	adminsCount, err := db.Collection("admins").CountDocuments(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stats error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_products": productsCount,
		"total_admins":   adminsCount,
		"status":         "ok",
	})
}
