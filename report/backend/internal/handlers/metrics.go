package handlers

import (
	"net/http"

	"benchmark-report-api/internal/services"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gin-gonic/gin"
)

// MetricsHandler returns a handler function for serving metrics data
func MetricsHandler(s3Service *services.S3Service, l log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		runID := c.Param("runId")
		outputDir := c.Param("outputDir")
		nodeType := c.Param("nodeType")

		if runID == "" || outputDir == "" || nodeType == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "runId, outputDir, and nodeType are required",
			})
			return
		}

		data, err := s3Service.GetMetrics(runID, outputDir, nodeType)
		if err != nil {
			l.Error("Failed to get metrics", "error", err, "runId", runID, "outputDir", outputDir, "nodeType", nodeType)

			c.JSON(http.StatusNotFound, gin.H{
				"error": "Metrics not found",
			})
			return
		}

		c.Header("Cache-Control", "public, max-age=43200") // 12 hours
		c.Header("Content-Type", "application/json")
		c.Data(http.StatusOK, "application/json", data)
	}
}
