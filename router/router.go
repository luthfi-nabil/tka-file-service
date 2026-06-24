package router

import (
	"crypto/rsa"

	"github.com/gin-gonic/gin"

	"tka-learning-portal/file-service/handler"
	"tka-learning-portal/file-service/middleware"
)

func Setup(fileHandler *handler.FileHandler, publicKey *rsa.PublicKey) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")

	protected := v1.Group("", middleware.RequireAuth(publicKey))
	{
		files := protected.Group("/files")
		{
			files.POST("/upload", fileHandler.Upload)
			files.GET("/:id", fileHandler.GetMeta)
			files.GET("/:id/download", fileHandler.Download)
			files.DELETE("/:id", fileHandler.Delete)
		}
	}

	return r
}
