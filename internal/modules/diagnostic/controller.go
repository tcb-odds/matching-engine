package diagnostic

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tcb-odds/matching-engine/internal/shared/config"
	"github.com/tcb-odds/matching-engine/internal/utils"
)

var serviceStartTime = time.Now()

// StatsProvider interface for getting engine statistics
type StatsProvider interface {
	GetStats() interface{}
	EraseOrders(pair string) error
}

var statsProvider StatsProvider

// SetStatsProvider sets the stats provider for the diagnostic module
func SetStatsProvider(provider StatsProvider) {
	statsProvider = provider
}

func GetDiagnostic(c *gin.Context) {
	elapsedTime := time.Since(serviceStartTime)
	c.JSON(http.StatusOK, gin.H{
		"version":   config.AppVersion,
		"startTime": serviceStartTime.Format(time.RFC3339),
		"uptime":    utils.FormatDuration(elapsedTime),
	})
}

func GetStats(c *gin.Context) {
	if statsProvider == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Stats provider not initialized",
		})
		return
	}

	stats := statsProvider.GetStats()
	c.JSON(http.StatusOK, stats)
}

func EraseOrders(c *gin.Context) {
	if statsProvider == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Stats provider not initialized",
		})
		return
	}

	pair := c.Query("pair")

	err := statsProvider.EraseOrders(pair)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	message := "All orders cleared successfully"
	if pair != "" {
		message = "Orders cleared for pair: " + pair
	}

	fmt.Printf("EraseOrders: %s\n", message)

	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"pair":    pair,
	})
}
