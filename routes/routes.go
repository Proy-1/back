package routes

import (
	"net/http"
	"pitipaw-backend/controllers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Setup mengonfigurasi dan mengembalikan Gin engine.
func Setup(ctrl *controllers.Controller, env string) *gin.Engine {
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://localhost:8000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	api := r.Group("/api")
	{
		// Rute utilitas
		api.GET("/health", ctrl.HealthCheck)
		api.GET("/stats", ctrl.GetStats)

		// Rute otentikasi
		api.POST("/login", ctrl.Login)
		api.POST("/register", ctrl.Register)
		
		// Rute produk
		api.GET("/products", ctrl.GetProducts)
		api.POST("/products", ctrl.CreateProduct)
		api.GET("/products/:id", ctrl.GetProduct)
		api.PUT("/products/:id", ctrl.UpdateProduct)
		api.DELETE("/products/:id", ctrl.DeleteProduct)

		// Rute admin
		api.GET("/admins", ctrl.GetAdmins)
		api.POST("/admins", ctrl.CreateAdmin) // Mungkin tidak perlu jika ada /register
		api.DELETE("/admins/:id", ctrl.DeleteAdmin)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
	})
	return r
}