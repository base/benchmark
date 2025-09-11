package handlers

import (
	"net/http"

	"benchmark-report-api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/ethereum/go-ethereum/log"
)

// MetadataHandler returns a handler function for serving benchmark metadata
func MetadataHandler(s3Service *services.S3Service, l log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		metadata, err := s3Service.GetMetadata()
		if err != nil {
			l.Error("Failed to get metadata", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve metadata",
			})
			return
		}

		c.Header("Cache-Control", "public, max-age=43200") // 12 hours
		c.JSON(http.StatusOK, metadata)
	}
}
