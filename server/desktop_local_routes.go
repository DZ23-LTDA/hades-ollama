package server

import (
	"net/http"
	"os"
	"runtime"

	"github.com/gin-gonic/gin"
)

func (s *Server) registerDesktopLocalRoutes(r *gin.Engine, agentAPI *agentAPI) {
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

	// servedByDesktopApp responde honestamente para rotas de escrita cujo
	// handler real vive na webview do app desktop (app/ui/ui.go). O servidor
	// local (porta 11434) não grava credenciais nem provedores, então em vez de
	// fingir sucesso (204) e perder os dados em silêncio, devolvemos 501 com uma
	// mensagem clara. O corpo {"error": ...} é o formato que a UI já interpreta.
	servedByDesktopApp := func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "Este endpoint é atendido pelo aplicativo desktop do Hades; o servidor local (porta 11434) não grava credenciais nem provedores. Use a janela do aplicativo para concluir esta ação.",
		})
	}

	r.GET("/api/v1/providers", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"providers": []gin.H{}})
	})
	r.PUT("/api/v1/providers/:name/key", servedByDesktopApp)
	r.DELETE("/api/v1/providers/:name/key", servedByDesktopApp)
	r.GET("/api/v1/providers/:name/models", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"models": []string{}})
	})
	r.PUT("/api/v1/providers/:name/models", servedByDesktopApp)
	r.PUT("/api/v1/connectors/:id/key", servedByDesktopApp)
	r.DELETE("/api/v1/connectors/:id/key", servedByDesktopApp)

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

	// Endpoint de personalização do desktop.
	//
	// Este caminho NÃO implementa persistência e antes mentia: o POST respondia
	// {"status":"updated"} sem gravar nada, e o GET devolvia instruções, tom e
	// memória FABRICADOS no código. Um cliente recebia sucesso e conteúdo
	// inventado — exatamente o "controle que finge funcionar" que a missão
	// proíbe. As instruções personalizadas de verdade vivem no cliente
	// (Configurações → localStorage) e são aplicadas a cada turno pelo servidor
	// do chat via `composeSystemPrompt`.
	//
	// Sem o middleware de autenticação/origem não existe forma segura de expor
	// nada sob /api/agent/v1, então a rota simplesmente não é registrada.
	if agentAPI == nil {
		return
	}
	personalization := r.Group("/api/agent/v1")
	personalization.Use(agentAPI.authMiddleware)
	personalization.GET("/personalization", notImplementedPersonalization)
	personalization.POST("/personalization", notImplementedPersonalization)
}

// notImplementedPersonalization responde a verdade: este servidor não guarda
// personalização. Sem dado inventado e sem falso sucesso.
func notImplementedPersonalization(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "personalization endpoint is not implemented on this server",
		"detail": "as instruções personalizadas são enviadas pelo chat a cada turno " +
			"(campo custom_instructions, aplicado no system prompt) e configuradas em Configurações na interface",
	})
}
