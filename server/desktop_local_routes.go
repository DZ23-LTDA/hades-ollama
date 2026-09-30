package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) registerDesktopLocalRoutes(r *gin.Engine) {
	// Local-first desktop compatibility routes to ensure all desktop UI surfaces
	// function smoothly with zero 404/401 console errors.
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

	r.GET("/api/v1/inference-compute", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"inferenceComputes":    []gin.H{},
			"defaultContextLength": 4096,
		})
	})

	r.GET("/api/v1/models/cloud", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"models": []gin.H{}})
	})
}
