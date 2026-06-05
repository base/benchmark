package handlers

import (
	"net/http"

	"github.com/base/base-bench/server/internal/services"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gin-gonic/gin"
)

// LoadTestListHandler returns the list of available load test results for a network.
func LoadTestListHandler(s3Service services.BackendStorage, l log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		network := c.Param("network")
		if network == "" {
			network = "sepolia"
		}

		entries, err := s3Service.ListLoadTests(network)
		if err != nil {
			l.Error("Failed to list load test results", "error", err, "network", network)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list load test results"})
			return
		}

		c.Header("Cache-Control", "public, max-age=300")
		c.JSON(http.StatusOK, entries)
	}
}

// LoadTestResultHandler returns a single load test result JSON.
func LoadTestResultHandler(s3Service services.BackendStorage, l log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		network := c.Param("network")
		timestamp := c.Param("timestamp")

		data, err := s3Service.GetLoadTest(network, timestamp)
		if err != nil {
			l.Error("Failed to get load test result", "error", err, "network", network, "timestamp", timestamp)
			c.JSON(http.StatusNotFound, gin.H{"error": "Load test result not found"})
			return
		}

		c.Header("Cache-Control", "public, max-age=43200")
		c.Data(http.StatusOK, "application/json", data)
	}
}
