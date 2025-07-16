package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	ImageURL    string  `json:"image_url"`
}

type AdminInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Response struct {
	Status   string      `json:"status,omitempty"`
	Message  string      `json:"message,omitempty"`
	Error    string      `json:"error,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	Database string      `json:"database,omitempty"`
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
	// Initialize MongoDB
	initMongoDB()

	// Create upload directory
	os.MkdirAll(UploadDir, 0755)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// CORS middleware wrapper
	corsHandler := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			setCORSHeaders(w)
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			h(w, r)
		}
	}

	// API Routes
	mux.HandleFunc("/api/health", corsHandler(healthCheck))
	mux.HandleFunc("/api/products", corsHandler(productsHandler))
	mux.HandleFunc("/api/products/", corsHandler(productHandler))
	mux.HandleFunc("/api/admins", corsHandler(adminsHandler))
	mux.HandleFunc("/api/admins/", corsHandler(adminHandler))
	mux.HandleFunc("/api/register", corsHandler(registerHandler))
	mux.HandleFunc("/api/login", corsHandler(loginHandler))
	mux.HandleFunc("/api/upload", corsHandler(uploadHandler))
	mux.HandleFunc("/api/stats", corsHandler(statsHandler))

	// Static files
	mux.HandleFunc("/static/", corsHandler(staticHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	fmt.Println("🚀 Pure Go Backend Starting...")
	fmt.Printf("📊 Database: mongodb://localhost:27017/pitipaw\n")
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

	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, Response{Error: message})
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
func healthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.RunCommand(ctx, bson.D{{"ping", 1}}).Err()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Database connection failed",
			Error:   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Status:   "ok",
		Message:  "Backend is running",
		Database: "connected",
	})
}

// Products handler
func productsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getProducts(w, r)
	case "POST":
		createProduct(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func getProducts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.Collection("products")
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error fetching products: "+err.Error())
		return
	}

	var products []Product
	if err = cursor.All(ctx, &products); err != nil {
		writeError(w, http.StatusInternalServerError, "Error parsing products: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, products)
}

func createProduct(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var input ProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if input.Name == "" || input.Price <= 0 {
		writeError(w, http.StatusBadRequest, "Nama dan harga produk wajib diisi")
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
		writeError(w, http.StatusInternalServerError, "Error creating product: "+err.Error())
		return
	}

	product.ID = result.InsertedID.(primitive.ObjectID)
	writeJSON(w, http.StatusCreated, product)
}

// Product handler (with ID)
func productHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/products/")
	if path == "" {
		writeError(w, http.StatusBadRequest, "Product ID required")
		return
	}

	switch r.Method {
	case "GET":
		getProduct(w, r, path)
	case "PUT":
		updateProduct(w, r, path)
	case "DELETE":
		deleteProduct(w, r, path)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func getProduct(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid product ID")
		return
	}

	collection := db.Collection("products")
	var product Product
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			writeError(w, http.StatusNotFound, "Produk tidak ditemukan")
			return
		}
		writeError(w, http.StatusInternalServerError, "Error fetching product: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, product)
}

func updateProduct(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid product ID")
		return
	}

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if len(input) == 0 {
		writeError(w, http.StatusBadRequest, "Tidak ada data untuk diupdate")
		return
	}

	collection := db.Collection("products")
	result, err := collection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": input})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error updating product: "+err.Error())
		return
	}

	if result.MatchedCount == 0 {
		writeError(w, http.StatusNotFound, "Produk tidak ditemukan")
		return
	}

	var product Product
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error fetching updated product: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, product)
}

func deleteProduct(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid product ID")
		return
	}

	collection := db.Collection("products")
	result, err := collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error deleting product: "+err.Error())
		return
	}

	if result.DeletedCount == 0 {
		writeError(w, http.StatusNotFound, "Produk tidak ditemukan")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Produk berhasil dihapus"})
}

// Admins handler
func adminsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getAdmins(w, r)
	case "POST":
		createAdmin(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func getAdmins(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.Collection("admins")
	cursor, err := collection.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"password": 0}))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error fetching admins: "+err.Error())
		return
	}

	var admins []Admin
	if err = cursor.All(ctx, &admins); err != nil {
		writeError(w, http.StatusInternalServerError, "Error parsing admins: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, admins)
}

func createAdmin(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var input AdminInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if input.Username == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "Username dan password wajib diisi")
		return
	}

	collection := db.Collection("admins")

	// Check if username already exists
	var existingAdmin Admin
	err := collection.FindOne(ctx, bson.M{"username": input.Username}).Decode(&existingAdmin)
	if err == nil {
		writeError(w, http.StatusBadRequest, "Username sudah ada")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error hashing password")
		return
	}

	admin := Admin{
		Username: input.Username,
		Password: string(hashedPassword),
	}

	result, err := collection.InsertOne(ctx, admin)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error creating admin: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"_id":      result.InsertedID,
		"username": admin.Username,
		"message":  "Admin created successfully",
	})
}

// Admin handler (with ID)
func adminHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/admins/")
	if path == "" {
		writeError(w, http.StatusBadRequest, "Admin ID required")
		return
	}

	deleteAdmin(w, r, path)
}

func deleteAdmin(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid admin ID")
		return
	}

	collection := db.Collection("admins")
	result, err := collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error deleting admin: "+err.Error())
		return
	}

	if result.DeletedCount == 0 {
		writeError(w, http.StatusNotFound, "Admin tidak ditemukan")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Admin berhasil dihapus"})
}

// Register handler
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	createAdmin(w, r)
}

// Login handler
func loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Login endpoint ready",
			"methods": []string{"POST"},
		})
	case "POST":
		loginAdmin(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func loginAdmin(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var input LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if input.Username == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "Username dan password wajib diisi")
		return
	}

	collection := db.Collection("admins")
	var admin Admin
	err := collection.FindOne(ctx, bson.M{"username": input.Username}).Decode(&admin)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Username/password salah")
		return
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(input.Password))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Username/password salah")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Login berhasil",
		"admin":   map[string]string{"username": admin.Username},
	})
}

// Upload handler
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(MaxFileSize)
	if err != nil {
		writeError(w, http.StatusBadRequest, "File terlalu besar. Maksimal 10MB")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "No file part")
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > MaxFileSize {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("File terlalu besar. Maksimal 10MB (ukuran file: %.1fMB)", float64(header.Size)/(1024*1024)))
		return
	}

	// Check file extension
	filename := header.Filename
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		writeError(w, http.StatusBadRequest, "File not allowed")
		return
	}

	ext := strings.ToLower(parts[len(parts)-1])
	if !allowedExtensions[ext] {
		writeError(w, http.StatusBadRequest, "File not allowed")
		return
	}

	// Save file
	filepath := fmt.Sprintf("%s/%s", UploadDir, filename)
	dst, err := os.Create(filepath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error saving file: "+err.Error())
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error saving file: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"image_url": fmt.Sprintf("/static/uploads/%s", filename),
		"file_size": fmt.Sprintf("%.1fMB", float64(header.Size)/(1024*1024)),
	})
}

// Stats handler
func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	productsCount, err := db.Collection("products").CountDocuments(ctx, bson.M{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Stats error: "+err.Error())
		return
	}

	adminsCount, err := db.Collection("admins").CountDocuments(ctx, bson.M{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Stats error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_products": productsCount,
		"total_admins":   adminsCount,
		"status":         "ok",
	})
}

// Static files handler
func staticHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract filename from URL
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	if path == "" {
		writeError(w, http.StatusBadRequest, "File not found")
		return
	}

	// Security: prevent directory traversal
	if strings.Contains(path, "..") {
		writeError(w, http.StatusBadRequest, "Invalid file path")
		return
	}

	fullPath := filepath.Join("static", path)
	
	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "File not found")
		return
	}

	// Serve file
	http.ServeFile(w, r, fullPath)
}
