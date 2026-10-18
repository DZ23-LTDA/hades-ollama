package server

// FASE 28.7 do adendo V3: `sbom.generate`.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// A E-1266 implementou o `ComplianceProvider` e provou-o com 8 testes -- **mas so' os testes o
// invocavam**:
//
//	*Um provider que existe no catalogo e nunca correu e' uma promessa com nome de modulo.*
//
// `sbom.generate` e' o nome que o adendo 28.7 da'.
//
// A RAIZ VEM DO AMBIENTE, NAO DO PEDIDO
//
// A rota le' `HADES_RAIZ_SBOM` -- uma variavel propria, porque o SBOM mede o PROJETO, nao a pasta de
// artefactos:
//
//	*Duas rotas que leem o mesmo cofre nao podem ter duas fronteiras diferentes.*
//
// `HADES_ARTEFACTOS` e' a fronteira dos binarios; `HADES_RAIZ_SBOM` e' a fronteira do codigo. Sao
// cofres diferentes, logo variaveis diferentes -- e nenhuma das duas e' escolhida pelo cliente.
//
// E DIZ O QUE NAO PRODUZ
//
// O inventario traz `nao_produzido` por extenso. `sbom.generate` **nao** e' um SBOM completo:
//
//	*Um SBOM que inclui o que o programador supoe e' pior do que nenhum: quem le' acredita.*

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// variavelDaRaizDoSBOM e' a fronteira do codigo para o SBOM.
const variavelDaRaizDoSBOM = "HADES_RAIZ_SBOM"

// timeoutDoInventario limita a varredura.
const timeoutDoInventario = 2 * time.Minute

// respostaDeSBOM e' a resposta de `sbom.generate`.
type respostaDeSBOM struct {
	// Gerado diz que o inventario foi produzido. E' sempre `true` quando a rota responde 200.
	Gerado bool `json:"gerado"`
	// Assinado e' sempre `false`: o inventario NAO leva assinatura criptografica.
	Assinado bool `json:"assinado"`
	// Provenance e' sempre `false`: nenhuma atestacao e' emitida aqui.
	Provenance bool `json:"provenance"`
	// Inventario e' o SBOM minimo medido.
	Inventario agent.InventarioDeSoftware `json:"inventario"`
	// AssinaturaDoInventario sao as linhas que tornam o resultado verificavel sem o ler todo.
	AssinaturaDoInventario []string `json:"assinatura_do_inventario"`
	// AvisoDeFronteira diz, no corpo, o que este SBOM NAO e'.
	AvisoDeFronteira string `json:"aviso_de_fronteira"`
}

// avisoDeFronteiraDoSBOM diz o que o inventario nao cobre.
func avisoDeFronteiraDoSBOM() string {
	return "este e' um SBOM MINIMO: cobre `go.mod` e `package.json` ate' " +
		"`maxFicheirosDeDependencia` ficheiros. NAO inclui dependencias transitivas, " +
		"THIRD_PARTY_NOTICES, provenance, assinatura criptografica nem attestation. " +
		"`assinado` e `provenance` sao sempre false. A raiz vem da variavel de ambiente `" +
		variavelDaRaizDoSBOM + "`, nunca do pedido."
}

// gerarSBOM responde a `sbom.generate`.
func (a *agentAPI) gerarSBOM(c *gin.Context) {
	raiz := strings.TrimSpace(os.Getenv(variavelDaRaizDoSBOM))
	if raiz == "" {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "a raiz do projecto nao esta' configurada neste servidor; defina " +
				variavelDaRaizDoSBOM + " para o directorio do projecto a inventariar",
		})
		return
	}
	// A raiz tem de ser um directorio. Um ficheiro nao e' um projecto.
	informacao, err := os.Stat(raiz)
	if err != nil || !informacao.IsDir() {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "a raiz configurada em `" + variavelDaRaizDoSBOM + "` nao e' um directorio legivel",
		})
		return
	}

	ctx, cancelar := context.WithTimeout(c.Request.Context(), timeoutDoInventario)
	defer cancelar()

	provider := agent.ProviderDeConformidade{}
	inventario, err := provider.Inventariar(ctx, raiz)
	if err != nil {
		// 409, nao 500: o servidor esta' bem, o projecto e' que nao da' para inventariar.
		//
		//	*Um 500 aqui confundiria "o servidor falhou" com "o projecto nao tem fontes".*
		c.JSON(http.StatusConflict, gin.H{
			"erro":               err.Error(),
			"gerado":             false,
			"assinado":           false,
			"provenance":         false,
			"aviso_de_fronteira": avisoDeFronteiraDoSBOM(),
		})
		return
	}

	c.JSON(http.StatusOK, respostaDeSBOM{
		Gerado:                 true,
		Assinado:               false,
		Provenance:             false,
		Inventario:             inventario,
		AssinaturaDoInventario: agent.AssinaturaDoInventario(inventario),
		AvisoDeFronteira:       avisoDeFronteiraDoSBOM(),
	})
}
