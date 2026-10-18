package server

// FASE 28.7 do adendo V3: as rotas `artifact.checksum` e `artifact.verify`.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// A E-1256 implementou o `VerificationProvider` -- item 5 dos dez do adendo 28.1 -- e medi-o contra
// o artefacto real da E-1251. Mas ele so' era invocavel por Go: **o produto nao tinha como pedir uma
// verificacao**. E' o mesmo defeito que as rondas 65-68 fecharam para o plano de build e que o
// proprio adendo nomeia:
//
//	*Uma comparacao que so' o programador consegue invocar nao e' uma funcionalidade: e' uma funcao.*
//
// Duas rotas, e a razao de serem DUAS:
//
//	GET /api/agent/v1/build/artifact/checksum?caminho=<path>
//	  -> mede o SHA-256 do ficheiro AGORA. Nao compara com nada.
//
//	GET /api/agent/v1/build/artifact/verify?caminho=<path>&sha256=<hex>
//	  -> compara com a referencia e declara integridade e assinatura.
//
//	*A medicao e a verificacao sao operacoes diferentes. Junta-las obrigaria quem so' quer o hash a
//	inventar uma referencia.*
//
// A FRONTEIRA, DITA NO CORPO DA RESPOSTA
//
// Nenhuma das rotas escreve, assina ou instala. `somente_leitura: true` e' um campo, nao uma nota de
// rodape': *um painel que mostra "pronto" sem poder construir e' um painel que promete o que nao
// faz -- a menos que diga que so' le'.*
//
// O QUE ESTAS ROTAS NAO FAZEM
//
// Nao procuram o artefacto por ID, nao leem do registo, nao aceitam caminho relativo a um projecto.
// Aceitam um caminho ABSOLUTO e medem-no. Procurar por ID exigiria um registo de artefactos, que
// **nao existe** e nao se inventa por simetria com o adendo.
//
//	*Um ID que resolve para um ficheiro que ninguem registou e' um apelido, nao uma identidade.*

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// timeoutDaVerificacao limita a medicao de um artefacto.
//
//	*Ler um ficheiro de 4 GB para calcular um hash e' uma operacao que o cliente pode pedir ao
//	servidor e que o servidor nao pode recusar depois de comecar -- por isso tem tecto.*
const timeoutDaVerificacao = 90 * time.Second

// respostaDeChecksum e' o corpo de `artifact.checksum`.
type respostaDeChecksum struct {
	SomenteLeitura   bool   `json:"somente_leitura"`
	NadaAssinado     bool   `json:"nada_assinado"`
	Caminho          string `json:"caminho"`
	TamanhoBytes     int64  `json:"tamanho_bytes"`
	SHA256           string `json:"sha256"`
	AvisoDeFronteira string `json:"aviso_de_fronteira"`
}

// respostaDeVerificacao e' o corpo de `artifact.verify`.
type respostaDeVerificacao struct {
	SomenteLeitura   bool                         `json:"somente_leitura"`
	NadaAssinado     bool                         `json:"nada_assinado"`
	Resultado        agent.ResultadoDeVerificacao `json:"resultado"`
	AvisoDeFronteira string                       `json:"aviso_de_fronteira"`
}

// avisoDeFronteiraDoArtefacto descreve, no corpo, o que estas rotas NAO fazem.
func avisoDeFronteiraDoArtefacto() string {
	return "estas rotas apenas MEDEM e COMPARAM o ficheiro indicado; nao constroem, nao assinam, " +
		"nao instalam e nao o procuram por ID. O caminho tem de ser absoluto e apontar para um " +
		"ficheiro que ja' exista."
}

// checksumDeArtefacto responde a `artifact.checksum`.
func (a *agentAPI) checksumDeArtefacto(c *gin.Context) {
	caminho := strings.TrimSpace(c.Query("caminho"))
	if caminho == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "o parametro `caminho` e' obrigatorio"})
		return
	}
	if !caminhoAbsoluto(caminho) {
		// *Um caminho relativo e' relativo ao directorio de quem pede, e o servidor nao sabe qual
		// e'.* Recusar e' mais honesto do que adivinhar.
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "o parametro `caminho` tem de ser ABSOLUTO: um caminho relativo e' relativo ao " +
				"directorio de quem pede, e o servidor nao sabe qual e'",
		})
		return
	}

	ctx, cancelar := context.WithTimeout(c.Request.Context(), timeoutDaVerificacao)
	defer cancelar()

	// `Verificar` exige referencia; para MEDIR sem referenciar usa-se o mesmo medidor com um
	// checksum calculado do proprio ficheiro? Nao: isso seria comparar o ficheiro consigo mesmo.
	// A medicao usa o caminho directo do pacote `agent`, que e' a mesma funcao que o verificador
	// usa para medir.
	medido, tamanho, err := agent.MedirArtefacto(ctx, caminho)
	if err != nil {
		c.JSON(codigoDeErroDeArtefacto(err), gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, respostaDeChecksum{
		SomenteLeitura:   true,
		NadaAssinado:     true,
		Caminho:          caminho,
		TamanhoBytes:     tamanho,
		SHA256:           medido,
		AvisoDeFronteira: avisoDeFronteiraDoArtefacto(),
	})
}

// verificarArtefacto responde a `artifact.verify`.
func (a *agentAPI) verificarArtefacto(c *gin.Context) {
	caminho := strings.TrimSpace(c.Query("caminho"))
	esperado := strings.TrimSpace(c.Query("sha256"))
	if caminho == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "o parametro `caminho` e' obrigatorio"})
		return
	}
	if esperado == "" {
		// Sem referencia NAO se verifica: ver `ErrSemReferencia` no pacote `agent`.
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "o parametro `sha256` e' obrigatorio: sem referencia isto seria uma medicao, " +
				"nao uma verificacao (use /artifact/checksum para medir)",
		})
		return
	}
	if !caminhoAbsoluto(caminho) {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "o parametro `caminho` tem de ser ABSOLUTO",
		})
		return
	}

	ctx, cancelar := context.WithTimeout(c.Request.Context(), timeoutDaVerificacao)
	defer cancelar()

	// O adaptador de assinatura e' o de Authenticode: no Windows le' o estado real; fora do Windows
	// declara `NAO_VERIFICADA` em vez de fingir `AUSENTE`.
	verificador := agent.VerificadorDeArtefacto{Assinatura: agent.AdaptadorAuthenticode()}
	resultado, err := verificador.Verificar(ctx, caminho, esperado)
	if err != nil {
		c.JSON(codigoDeErroDeArtefacto(err), gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, respostaDeVerificacao{
		SomenteLeitura:   true,
		NadaAssinado:     false, // esta rota LE' uma assinatura; nao assina nada
		Resultado:        resultado,
		AvisoDeFronteira: avisoDeFronteiraDoArtefacto(),
	})
}

// caminhoAbsoluto diz se o caminho e' absoluto no sistema onde o servidor corre.
func caminhoAbsoluto(caminho string) bool {
	if len(caminho) >= 3 && caminho[1] == ':' && (caminho[2] == '\\' || caminho[2] == '/') {
		return true // Windows: C:\... ou C:/...
	}
	return strings.HasPrefix(caminho, "/") || strings.HasPrefix(caminho, `\\`)
}

// codigoDeErroDeArtefacto traduz o erro do verificador num codigo HTTP, por SIGNIFICADO.
//
//	*Um erro que diz "nao sei o que isto e'" nao e' o mesmo que "isto nao existe" -- e o codigo HTTP
//	e' a primeira coisa que um cliente le'.*
func codigoDeErroDeArtefacto(err error) int {
	switch {
	case errors.Is(err, agent.ErrFicheiroInexistente):
		return http.StatusNotFound
	case errors.Is(err, agent.ErrSemReferencia):
		return http.StatusBadRequest
	case errors.Is(err, agent.ErrCertificadoIlegivel):
		// O pedido aponta para um ficheiro que nao serve: e' do PEDIDO, nao do servidor. Sem este
		// ramo, `sign.plan` devolvia **500** para um certificado ilegivel -- e um 500 diz "o
		// servidor falhou" quando o que falhou foi o ficheiro que o cliente indicou.
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
