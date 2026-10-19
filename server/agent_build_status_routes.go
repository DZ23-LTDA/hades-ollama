package server

// FASE 28.7: as rotas de estado do build.
//
// O DEFEITO QUE ESTAS ROTAS FECHAM
//
// O doc-comment do `build.verify` dizia, com honestidade, *"nao existe registo de builds"*. O adendo
// pede `build.status`, `build.logs`, `build.cancel` e `build.artifacts`. Nenhuma delas pode existir
// sem um sitio onde o build viva depois de a rota `build.start` responder.
//
//	*Uma rota de estado que reconstroi o estado a partir do pedido nao consulta o servidor: consulta
//	o cliente.*
//
// A FRONTEIRA QUE ESTAS ROTAS RESPEITAM
//
//  1. NENHUMA escreve no repositorio. `cancel` mata um processo; nao apaga artefactos, nao reverte
//     ficheiros e nao "desfaz" um build pela metade.
//  2. `cancel` sobre um build terminado devolve 409, nao 200. Um botao que diz "cancelei" sobre um
//     build que ja' acabou e' um botao que mente.
//  3. `artifacts` so' mostra o que o provider MEDIU. Nao ha' lista de artefactos "esperados".
//  4. `logs` devolve a saida GUARDADA, com o truncamento declarado. Nao inventa o que nao viu.
//
// O QUE ESTAS ROTAS NAO FAZEM (e dizem-no)
//
// Nao assinam, nao publicam, nao instalam e nao prometem durabilidade: um build vive enquanto o
// processo vive. A persistencia em disco e' diagnostico, nao garantia.

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// storeDeBuilds devolve o store, criando-o na primeira chamada.
//
// O store e' por instancia de `agentAPI`, nao global: um singleton de package seria estado
// partilhado entre testes e esconderia o facto de dois servidores nao partilharem builds.
func (a *agentAPI) storeDeBuilds() *agent.BuildStore {
	a.buildsOnce.Do(func() {
		if a.builds != nil {
			return
		}
		// Os builds persistem sob o DataRoot do runtime, ao lado do resto do estado do agente.
		//
		//	*Um store que existe e nao persiste e' um store que perde o trabalho quando o processo
		//	reinicia -- e um reinicio nao e' um caso raro: e' a proxima vez que o utilizador abrir o
		//	produto.*
		//
		// Sem runtime (testes de rota com `&agentAPI{}`), fica so' em memoria -- e diz-se.
		if a.runtime != nil {
			if raiz := strings.TrimSpace(a.runtime.DataRoot()); raiz != "" {
				a.builds = agent.NovoBuildStoreComDirectorio(filepath.Join(raiz, "builds"))
				if _, interrompidos := a.builds.CarregarDoDisco(); interrompidos > 0 {
					// Um build que estava a correr quando o processo morreu esta' interrompido, nao a
					// correr: `CarregarDoDisco` ja' o marcou FALHOU com motivo.
					_ = interrompidos
				}
				return
			}
		}
		a.builds = agent.NovoBuildStore()
	})
	return a.builds
}

// avisoDaFronteiraDoEstadoDeBuild repete a fronteira em todas as respostas de estado. Um cliente que
// leia so' um campo tem de encontrar a fronteira ao lado dele, nao num documento.
func avisoDaFronteiraDoEstadoDeBuild() string {
	return "estado de build: nada e' assinado, nada e' publicado e nada e' instalado por estas rotas; " +
		"o registo vive enquanto o processo vive"
}

// estadoDeBuild GET /api/agent/v1/build/status/:id
func (a *agentAPI) estadoDeBuild(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "`id` obrigatorio"})
		return
	}
	r, err := a.storeDeBuilds().Obter(id)
	if err != nil {
		if errors.Is(err, agent.ErrBuildDesconhecido) {
			c.JSON(http.StatusNotFound, gin.H{
				"erro":    "build desconhecido",
				"id":      id,
				"detalhe": "o registo nao existe neste processo; um build nao sobrevive a um reinicio",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"build":              r,
		"nada_assinado":      true,
		"nada_publicado":     true,
		"nada_instalado":     true,
		"aviso_de_fronteira": avisoDaFronteiraDoEstadoDeBuild(),
	})
}

// registosDeBuild GET /api/agent/v1/build/registos
//
// Uma lista vazia e' uma lista vazia -- nao e' um erro e nao e' "desconhecido". Distinguir os dois
// e' o que permite a um painel mostrar "ainda nao construiste nada" em vez de "falhou".
func (a *agentAPI) registosDeBuild(c *gin.Context) {
	registos := a.storeDeBuilds().Listar()
	c.JSON(http.StatusOK, gin.H{
		"total":              len(registos),
		"builds":             registos,
		"nada_assinado":      true,
		"nada_publicado":     true,
		"aviso_de_fronteira": avisoDaFronteiraDoEstadoDeBuild(),
	})
}

// saidaDoBuild GET /api/agent/v1/build/logs/:id
func (a *agentAPI) saidaDoBuild(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "`id` obrigatorio"})
		return
	}
	r, err := a.storeDeBuilds().Obter(id)
	if err != nil {
		if errors.Is(err, agent.ErrBuildDesconhecido) {
			c.JSON(http.StatusNotFound, gin.H{"erro": "build desconhecido", "id": id})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":       r.ID,
		"estado":   r.Estado,
		"saida":    r.SaidaDoComando,
		"erro":     r.Erro,
		"truncada": len(r.SaidaDoComando) >= 256*1024,
		"aviso":    "a saida guardada e' limitada; um build muito ruidoso aparece truncado e diz isso",
	})
}

// artefactosDoBuild GET /api/agent/v1/build/artifacts/:id
//
// Devolve APENAS o que o provider mediu. Se o build falhou antes de produzir artefactos, a lista vem
// vazia e `produziu` e' falso -- nao ha' aqui uma lista de artefactos "esperados".
func (a *agentAPI) artefactosDoBuild(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "`id` obrigatorio"})
		return
	}
	r, err := a.storeDeBuilds().Obter(id)
	if err != nil {
		if errors.Is(err, agent.ErrBuildDesconhecido) {
			c.JSON(http.StatusNotFound, gin.H{"erro": "build desconhecido", "id": id})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	artefactos := []agent.Artefato{}
	explicacao := ""
	if r.Resultado != nil {
		artefactos = r.Resultado.Artefatos
		explicacao = r.Resultado.Explicacao
	}
	c.JSON(http.StatusOK, gin.H{
		"id":            r.ID,
		"alvo":          r.Alvo,
		"estado":        r.Estado,
		"artefactos":    artefactos,
		"produziu":      len(artefactos) > 0,
		"explicacao":    explicacao,
		"nada_assinado": true,
		"aviso":         "estes artefactos foram medidos pelo provider; nenhum foi assinado nem publicado",
	})
}

// cancelarBuild POST /api/agent/v1/build/cancel/:id
//
// 409 sobre um build terminado: o pedido nao "funciona" sobre o que ja' acabou. 200 sobre um build a
// correr, e o contexto e' cancelado -- nao basta mudar o estado no store e deixar o processo vivo.
func (a *agentAPI) cancelarBuild(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "`id` obrigatorio"})
		return
	}
	err := a.storeDeBuilds().Cancelar(id)
	switch {
	case err == nil:
		r, _ := a.storeDeBuilds().Obter(id)
		c.JSON(http.StatusOK, gin.H{
			"id":                 r.ID,
			"estado":             r.Estado,
			"cancelado":          true,
			"aviso_de_fronteira": "cancelar pede o fim do processo; nao apaga artefactos ja' produzidos nem reverte ficheiros",
		})
	case errors.Is(err, agent.ErrBuildDesconhecido):
		c.JSON(http.StatusNotFound, gin.H{"erro": "build desconhecido", "id": id})
	case errors.Is(err, agent.ErrBuildJaTerminou):
		r, _ := a.storeDeBuilds().Obter(id)
		c.JSON(http.StatusConflict, gin.H{
			"erro":      "build ja' terminou",
			"id":        id,
			"estado":    r.Estado,
			"detalhe":   "cancelar um build terminado nao tem efeito; o estado final ja' esta' decidido",
			"cancelado": false,
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
	}
}
