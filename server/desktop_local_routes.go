package server

import (
	"net/http"
	"os"
	"runtime"

	"github.com/gin-gonic/gin"
)

func (s *Server) registerDesktopLocalRoutes(r *gin.Engine) {
	// Local-first desktop compatibility routes to ensure all desktop UI surfaces
	// function smoothly with zero 404/401 console errors.
	r.GET("/api/me", func(c *gin.Context) {
		h, _ := os.Hostname()
		if h == "" {
			h = "Este computador"
		}
		c.JSON(http.StatusOK, gin.H{
			"name":       "Operador local",
			"username":   "local",
			"email":      "local@localhost",
			"plan":       "Local-first",
			"hostname":   h,
			"local_only": true,
		})
	})

	r.GET("/api/v1/host", func(c *gin.Context) {
		h, _ := os.Hostname()
		if h == "" {
			h = "Este computador"
		}
		c.JSON(http.StatusOK, gin.H{
			"hostname": h,
			"os":       runtime.GOOS,
		})
	})

	r.GET("/api/v1/chats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"chatInfos": []gin.H{}})
	})

	r.GET("/api/v1/settings", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"settings": gin.H{
				"Expose":            false,
				"Browser":           true,
				"Survey":            false,
				"Models":            "",
				"Agent":             true,
				"Tools":             true,
				"WorkingDir":        "",
				"ContextLength":     4096,
				"TurboEnabled":      true,
				"WebSearchEnabled":  true,
				"ThinkEnabled":      true,
				"ThinkLevel":        "medium",
				"SelectedModel":     "",
				"SidebarOpen":       true,
				"LastHomeView":      "agentic",
				"OnboardingVersion": 1,
				"AutoUpdateEnabled": false,
				"ClaudeDesktopUsed": false,
			},
		})
	})

	r.POST("/api/v1/settings", func(c *gin.Context) {
		var body map[string]any
		_ = c.ShouldBindJSON(&body)
		c.JSON(http.StatusOK, gin.H{"settings": body})
	})

	r.GET("/api/v1/cloud", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"disabled": true,
			"source":   "local",
		})
	})

	r.POST("/api/v1/cloud", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"disabled": true,
			"source":   "local",
		})
	})

	r.GET("/api/v1/integrations", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"integrations": []gin.H{}})
	})

	r.GET("/api/v1/providers", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"providers": []gin.H{}})
	})
	r.PUT("/api/v1/providers/:name/key", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.DELETE("/api/v1/providers/:name/key", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.GET("/api/v1/providers/:name/models", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"models": []string{}})
	})
	r.PUT("/api/v1/providers/:name/models", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.PUT("/api/v1/connectors/:id/key", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.DELETE("/api/v1/connectors/:id/key", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	r.GET("/api/v1/inference-compute", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"inferenceComputes":    []gin.H{},
			"defaultContextLength": 4096,
		})
	})

	r.GET("/api/v1/models/cloud", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"models": []gin.H{}})
	})
	r.POST("/api/v1/models/pull", s.PullHandler)

	// Endpoints para Personalização do Agente e Criações Consolidadas
	r.GET("/api/agent/v1/personalization", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"custom_instructions": "Sempre responder em português, priorizar arquitetura limpa, segurança rigorosa e entregar código testado de ponta a ponta.",
			"memory_enabled":      true,
			"tone":                "professional_engineer",
		})
	})

	r.POST("/api/agent/v1/personalization", func(c *gin.Context) {
		var body map[string]any
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "updated",
			"data":   body,
		})
	})

	r.GET("/api/agent/v1/creations", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"creations": []gin.H{},
			"status":    "NOT_EXECUTED",
			"reason":    "A publicação e o rollback de criações ainda não estão implementados neste runtime.",
		})
	})
}
