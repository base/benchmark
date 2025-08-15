package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Health provides a health check endpoint
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "benchmark-report-api",
	})
}
