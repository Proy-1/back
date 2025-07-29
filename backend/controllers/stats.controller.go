package controllers

import (
	"context"
	"net/http"
	"pitipaw-backend/models"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

// HealthCheck memeriksa status koneksi database.
func (ctrl *Controller) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ctrl.DB.Client().Ping(ctx, nil)
	dbStatus := "connected"
	if err != nil {
		dbStatus = "disconnected"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"database":  dbStatus,
		"timestamp": time.Now().Unix(),
	})
}

// GetStats mengambil data statistik dari aplikasi.
func (ctrl *Controller) GetStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	productsCollection := ctrl.DB.Collection("products")
	adminsCollection := ctrl.DB.Collection("admins")

	totalProducts, _ := productsCollection.CountDocuments(ctx, bson.M{})
	totalAdmins, _ := adminsCollection.CountDocuments(ctx, bson.M{})

	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": bson.M{"$multiply": []string{"$price", "$stock"}}},
		}},
	}
	cursor, err := productsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var result []bson.M
	var totalValue float64
	if cursor.Next(ctx) {
		if err := cursor.All(ctx, &result); err == nil && len(result) > 0 {
			if val, ok := result[0]["total"].(float64); ok {
				totalValue = val
			}
		}
	}

	stats := models.Stats{
		TotalProducts: totalProducts,
		TotalAdmins:   totalAdmins,
		TotalValue:    totalValue,
	}
	c.JSON(http.StatusOK, gin.H{"stats": stats})
}