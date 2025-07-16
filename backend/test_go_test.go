package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

const baseURL = "http://localhost:5000"

func TestHealthCheck(t *testing.T) {
	resp, err := http.Get(baseURL + "/api/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	fmt.Println("✅ Health check passed")
}

func TestProductCRUD(t *testing.T) {
	// Create product
	product := map[string]interface{}{
		"name":        "Test Product",
		"price":       25000,
		"description": "Test description",
		"image_url":   "/static/uploads/test.jpg",
	}

	jsonData, _ := json.Marshal(product)
	resp, err := http.Post(baseURL+"/api/products", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create product: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var createdProduct map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&createdProduct)
	productID := createdProduct["_id"].(string)

	fmt.Printf("✅ Product created with ID: %s\n", productID)

	// Get products
	resp, err = http.Get(baseURL + "/api/products")
	if err != nil {
		t.Fatalf("Failed to get products: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	fmt.Println("✅ Products retrieved successfully")

	// Clean up: delete product
	req, _ := http.NewRequest("DELETE", baseURL+"/api/products/"+productID, nil)
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to delete product: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("✅ Product deleted successfully")
}

func TestAdminAuth(t *testing.T) {
	// Register admin
	admin := map[string]string{
		"username": "testadmin",
		"password": "testpass123",
	}

	jsonData, _ := json.Marshal(admin)
	resp, err := http.Post(baseURL+"/api/register", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to register admin: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var createdAdmin map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&createdAdmin)
	adminID := createdAdmin["_id"].(string)

	fmt.Printf("✅ Admin registered with ID: %s\n", adminID)

	// Login admin
	jsonData, _ = json.Marshal(admin)
	resp, err = http.Post(baseURL+"/api/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to login admin: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	fmt.Println("✅ Admin login successful")

	// Clean up: delete admin
	req, _ := http.NewRequest("DELETE", baseURL+"/api/admins/"+adminID, nil)
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to delete admin: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("✅ Admin deleted successfully")
}

func TestUpload(t *testing.T) {
	// Create a test file
	testFile := "test_image.jpg"
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.WriteString("fake image content")
	file.Close()
	defer os.Remove(testFile)

	// Upload file
	file, err = os.Open(testFile)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", filepath.Base(testFile))
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	io.Copy(part, file)
	writer.Close()

	req, err := http.NewRequest("POST", baseURL+"/api/upload", body)
	if err != nil {
		t.Fatalf("Failed to create upload request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	fmt.Println("✅ File upload successful")
}

func TestStats(t *testing.T) {
	resp, err := http.Get(baseURL + "/api/stats")
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var stats map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&stats)

	fmt.Printf("✅ Stats retrieved: %v\n", stats)
}
