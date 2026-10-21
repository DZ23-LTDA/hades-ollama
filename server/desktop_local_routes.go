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

	// POST /api/v1/settings NAO e' atendido por este servidor local: o handler real
	// (com persistencia) vive na webview do app desktop (`app/ui/ui.go`: `s.settings`).
	//
	// Ate' 2026-10-15 este handler devolvia 200 e ECOAVA o corpo recebido sem gravar
	// nada. Medido com dois pedidos (ronda 107):
	//     POST {"SelectedModel":"llama3.2:3b"} -> 200 {"settings":{"SelectedModel":"llama3.2:3b"}}
	//     GET                                  -> "SelectedModel":""            <- nao persistiu
	// Efeito no produto: a UI acreditava que tinha gravado, o GET seguinte devolvia vazio,
	// `selectedModel` ficava null e o envio abortava com "No model selected" — ou seja,
	// *** o seletor de modelo do Chat nunca funcionava contra este servidor ***, e a falha
	// era silenciosa. E' exactamente o que o `servedByDesktopApp` abaixo evita; este
	// handler nao aplicava a mesma regra.
	//
	// Devolver 501 com mensagem clara e' o minimo honesto enquanto a persistencia real
	// nao for ligada aqui (ver `app/store`: `SetSettings`/`Settings.SelectedModel` ja'
	// existem e ja' tem teste de round-trip em `app/store/store_test.go`).
	r.POST("/api/v1/settings", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "Este endpoint e' atendido pelo aplicativo desktop do Hades; o servidor local nao persiste configuracoes. Use a janela do aplicativo para escolher o modelo.",
		})
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

	// E-784: a familia `/api/v1/chat/...` **nao** estava registada aqui, e o SPA chama-a:
	// `app/ui/app/src/api.ts:176,423,530,544` montam `/api/v1/chat/${chatId}`, e na criacao
	// esse `chatId` e' `new`. Medido no E2E de motor real (`E-783`): `POST /api/v1/chat/new`
	// devolvia **404**, imediatamente antes de a resposta gerada deixar de aparecer no DOM.
	//
	// O cabecalho deste ficheiro (`:12-13`) declara o proposito - "ensure all desktop UI
	// surfaces function smoothly with zero 404/401 console errors" - e esta familia
	// contradizia-o. *Um 404 e' o pior dos mundos: nao diz ao utilizador o que falta.*
	//
	// A persistencia real do chat vive na webview (`app/ui/ui.go` + `app/store`) e **nao** se
	// duplica aqui (proibido sem ADR). O honesto e' o mesmo 501 que ja' se usa acima.
	r.POST("/api/v1/chat/new", servedByDesktopApp)
	r.POST("/api/v1/chat/:id", servedByDesktopApp)
	r.POST("/api/v1/chat/:id/rename", servedByDesktopApp)

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

	// E-794: `POST /api/v1/model/upstream` - a UI chama esta rota em
	// `app/ui/app/src/api.ts:558` (`getModelUpstreamInfo`) para saber se um modelo
	// ficou desactualizado face ao registry. Ate' aqui respondia **404** (`E-784`).
	//
	// O cliente **ja' trata `!response.ok`**: avisa na consola e devolve
	// `{stale:false, exists:false}` - logo **nada muda** no produto. O que muda e' o
	// **significado**: *404 diz «nao existe esta rota», 501 diz «este servidor nao a serve» -*
	// e a segunda e' a verdade. O handler real vive na webview (`app/ui/ui.go`) e a
	// verificacao consulta o registry; **nao** se duplica aqui.
	r.POST("/api/v1/model/upstream", notImplementedModelUpstream)

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
// notImplementedModelUpstream responde a verdade: este servidor nao consulta o registry
// upstream do Ollama. Exercido pelo aplicativo desktop; sem resposta inventada.
func notImplementedModelUpstream(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "a verificacao de modelo desactualizado (upstream) e' feita pelo aplicativo desktop do Hades; este servidor local nao consulta o registry.",
	})
}

func notImplementedPersonalization(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "personalization endpoint is not implemented on this server",
		"detail": "as instruções personalizadas são enviadas pelo chat a cada turno " +
			"(campo custom_instructions, aplicado no system prompt) e configuradas em Configurações na interface",
	})
}
