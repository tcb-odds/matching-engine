package initializers

import (
	"github.com/gin-gonic/gin"
	"github.com/tcb-odds/matching-engine/internal/modules/diagnostic"
)

func InitModules(engine *gin.Engine) {
	diagnostic.Bootstrap(engine)
}
