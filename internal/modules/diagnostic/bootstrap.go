package diagnostic

import (
	"github.com/gin-gonic/gin"
	"github.com/tcb-odds/matching-engine/internal/shared/middlewares"
)

func Bootstrap(engine *gin.Engine) {
	engine.GET("/version", GetDiagnostic)
	engine.GET("/stats", GetStats)
	engine.DELETE("/erase", middlewares.AdminAuth(), EraseOrders)
}
