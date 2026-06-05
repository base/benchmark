package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/base/base-bench/server/internal/services"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gin-gonic/gin"
)

// StaticEmulationHandler serves metrics files using the same URL structure as static files
// This allows the API to emulate the static file structure: /output/<outputDir>/metrics-<nodeType>.json
func StaticEmulationHandler(s3Service services.BackendStorage, l log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		outputDir := c.Param("outputDir")
		filename := c.Param("filename")

		// Extract nodeType from filename (e.g., "metrics-sequencer.json" -> "sequencer")
		if !strings.HasPrefix(filename, "metrics-") || !strings.HasSuffix(filename, ".json") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid metrics filename format. Expected: metrics-<nodeType>.json",
			})
			return
		}

		nodeType := strings.TrimPrefix(filename, "metrics-")
		nodeType = strings.TrimSuffix(nodeType, ".json")

		if outputDir == "" || nodeType == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "outputDir and nodeType are required",
			})
			return
		}

		data, err := s3Service.GetObject(fmt.Sprintf("%s/metrics-%s.json", outputDir, nodeType))
		if err != nil {
			l.Error("Failed to get metrics", "error", err, "outputDir", outputDir, "nodeType", nodeType)

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
