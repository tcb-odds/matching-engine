package diagnostic

import "github.com/gin-gonic/gin"

func Bootstrap(engine *gin.Engine) {
	engine.GET("/version", GetDiagnostic)
	engine.GET("/stats", GetStats)
}
