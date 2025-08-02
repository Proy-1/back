// File: controllers/product.controller.go
package controllers

import (
	"context"
	"log"
	"net/http"
	"pitipaw-backend/models"
	"time"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetProducts menangani pengambilan semua produk.
func (ctrl *Controller) GetProducts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := ctrl.DB.Collection("products")
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var productList []models.Product
	if err = cursor.All(ctx, &productList); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": productList})
}

// CreateProduct menangani pembuatan produk baru.
func (ctrl *Controller) CreateProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if product.ImageBase64 != "" && ctrl.Cld != nil {
		uploadResult, err := ctrl.Cld.Upload.Upload(
			context.Background(),
			product.ImageBase64,
			uploader.UploadParams{Folder: "pitipaw/products"},
		)
		if err != nil {
			log.Println("Cloudinary upload error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
			return
		}
		product.ImageURL = uploadResult.SecureURL
		product.Image = uploadResult.PublicID
	}

	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()
	product.ImageBase64 = ""

	collection := ctrl.DB.Collection("products")
	result, err := collection.InsertOne(ctx, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	product.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, gin.H{"product": product})
}

// GetProduct menangani pengambilan satu produk berdasarkan ID.
func (ctrl *Controller) GetProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	collection := ctrl.DB.Collection("products")
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&product)
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

// UpdateProduct menangani pembaruan data produk.
func (ctrl *Controller) UpdateProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var updateData models.Product
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Jika ada gambar baru (imageBase64), upload ke Cloudinary
	if updateData.ImageBase64 != "" && ctrl.Cld != nil {
		uploadResult, err := ctrl.Cld.Upload.Upload(
			context.Background(),
			updateData.ImageBase64,
			uploader.UploadParams{Folder: "pitipaw/products"},
		)
		if err != nil {
			log.Println("Cloudinary upload error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
			return
		}
		updateData.ImageURL = uploadResult.SecureURL
		updateData.Image = uploadResult.PublicID
	}

	updateData.UpdatedAt = time.Now()
	updateData.ImageBase64 = ""
	update := bson.M{"$set": updateData}

	collection := ctrl.DB.Collection("products")
	result, err := collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
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

// DeleteProduct menangani penghapusan produk.
func (ctrl *Controller) DeleteProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	collection := ctrl.DB.Collection("products")
	result, err := collection.DeleteOne(ctx, bson.M{"_id": objectID})
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
