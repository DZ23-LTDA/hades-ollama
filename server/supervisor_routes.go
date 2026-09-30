package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func (a *agentAPI) supervisorStatus(c *gin.Context) {
	if a.runtime == nil || a.runtime.Supervisor() == nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"running": false,
			"error":   "supervisor not initialized in runtime",
		})
		return
	}
	st := a.runtime.Supervisor().Status()
	c.JSON(http.StatusOK, st)
}

func (a *agentAPI) supervisorConfig(c *gin.Context) {
	if a.runtime == nil || a.runtime.Supervisor() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "supervisor not initialized in runtime"})
		return
	}

	var req agent.SupervisorConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supervisor config: " + err.Error()})
		return
	}

	a.runtime.Supervisor().SetConfig(req)
	st := a.runtime.Supervisor().Status()
	c.JSON(http.StatusOK, st)
}

func (a *agentAPI) supervisorTick(c *gin.Context) {
	if a.runtime == nil || a.runtime.Supervisor() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "supervisor not initialized in runtime"})
		return
	}

	now := time.Now().UTC()
	res, err := a.runtime.Supervisor().Tick(c.Request.Context(), now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "result": res})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result": res,
		"status": a.runtime.Supervisor().Status(),
	})
}
