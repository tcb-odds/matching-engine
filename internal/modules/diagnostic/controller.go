package diagnostic

import (
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
