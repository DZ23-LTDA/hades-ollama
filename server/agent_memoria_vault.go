package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// ErrCorpoComCampoDesconhecido recusa um corpo que traga campos que o contrato nao declara.
var ErrCorpoComCampoDesconhecido = errors.New("body contains unknown fields")

// IdentidadeDaSessao devolve o ID do utilizador autenticado, lido do contexto que o `authMiddleware`
// preenche. E' a UNICA fonte de identidade: nao ha' parametro, cabecalho ou campo de corpo que a
// possa fornecer. Sem sessao, devolve ("", false) e o chamador recusa.
func IdentidadeDaSessao(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	valor, existe := c.Get("agent.user")
	if !existe {
		return "", false
	}
	utilizador, ok := valor.(agent.User)
	if !ok || strings.TrimSpace(utilizador.ID) == "" {
		return "", false
	}
	return strings.TrimSpace(utilizador.ID), true
}

// OrganizacaoDaSessao devolve o ID da organizacao do contexto. Tambem nunca vem do corpo: o
// `authMiddleware` verifica o cabecalho `X-Ollama-Organization` CONTRA a organizacao do token antes
// de o publicar aqui.
func OrganizacaoDaSessao(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	valor, existe := c.Get("agent.organization")
	if !existe {
		return "", false
	}
	organizacao, ok := valor.(agent.Organization)
	if !ok || strings.TrimSpace(organizacao.ID) == "" {
		return "", false
	}
	return strings.TrimSpace(organizacao.ID), true
}

// DecodificarCorpoEstrito descodifica o corpo recusando campos que o contrato nao declara.
//
// PORQUE E' ESTRITO: se um corpo pode trazer `identidade`, `user_id` ou `tenant`, mais cedo ou mais
// tarde um deles sera' lido como se fosse sessao. Ignorar o campo em silencio deixa o pedido passar
// com uma intencao que o servidor nao honrou; recusar diz ao chamador que o contrato nao tem esse
// campo. *Um campo ignorado em silencio e' uma porta que parece fechada.*
func DecodificarCorpoEstrito(corpo io.Reader, destino any) error {
	if corpo == nil {
		return ErrCorpoComCampoDesconhecido
	}
	descodificador := json.NewDecoder(corpo)
	descodificador.DisallowUnknownFields()
	if err := descodificador.Decode(destino); err != nil {
		return err
	}
	// Um segundo valor JSON no mesmo corpo tambem e' recusado.
	if err := descodificador.Decode(&struct{}{}); err != io.EOF {
		return ErrCorpoComCampoDesconhecido
	}
	return nil
}

// AppendNoVaultGovernado trata POST /memory/vault/append.
//
// A identidade e a organizacao vem do contexto do `authMiddleware` -- nunca do corpo, que e'
// descodificado de forma estrita (`DecodificarCorpoEstrito`): um corpo com `identidade` ou `tenant`
// e' recusado, nao ignorado.
//
// A politica nasce VAZIA: sem concessao explicita, ninguem escreve. Nesta fatia nao ha' UI nem
// configuracao de concessoes; o endpoint fica, por construcao, a negar. E' preferivel uma porta que
// recusa a uma porta que aceita.
func (a *agentAPI) appendMemoriaNoVault(c *gin.Context) {
	raiz := strings.TrimSpace(os.Getenv("HADES_VAULT_ROOT"))
	if raiz == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vault_memory_disabled"})
		return
	}
	identidade, ok := IdentidadeDaSessao(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session identity required"})
		return
	}
	tenant, ok := OrganizacaoDaSessao(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session organization required"})
		return
	}

	var corpo struct {
		Relativo       string `json:"relativo"`
		VersaoEsperada int    `json:"versao_esperada"`
		Conteudo       string `json:"conteudo"`
		Fonte          string `json:"fonte"`
		Aprovador      string `json:"aprovador"`
		Rotulo         string `json:"rotulo"`
	}
	if err := DecodificarCorpoEstrito(c.Request.Body, &corpo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	politica, erroDePolitica := agent.ParsearPoliticaDeAcesso(os.Getenv("HADES_VAULT_ACL"))
	if erroDePolitica != nil {
		// Configuracao ilegivel NAO vira "sem concessoes": seria esconder o defeito atras de uma
		// negacao que parece politica. O endpoint recusa-se a funcionar e diz porque.
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vault_acl_invalid"})
		return
	}
	servico := agent.ServicoDeAppendGovernado{Raiz: raiz, Politica: politica}
	escreveu, err := servico.Anexar(agent.PedidoDeAppendGovernado{
		Identidade:     identidade,
		Tenant:         tenant,
		Relativo:       corpo.Relativo,
		VersaoEsperada: corpo.VersaoEsperada,
		Memoria: agent.Memoria{
			Rotulo:    agent.RotuloDeMemoria(strings.TrimSpace(corpo.Rotulo)),
			Conteudo:  corpo.Conteudo,
			Fonte:     corpo.Fonte,
			Aprovador: corpo.Aprovador,
		},
	})
	switch {
	case errors.Is(err, agent.ErrSegredoNaNota):
		// Recusa por DLP: 422 (o corpo e' compreendido e recusado por politica), nunca 500.
		// A resposta NAO ecoa o valor detetado -- ecoa-lo seria o segundo vazamento.
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "secret_detected", "detail": "content carries credential-shaped material and was not stored"})
	case errors.Is(err, agent.ErrVersaoDivergente):
		// Conflito de versao nao e' falha do servidor: e' o resultado normal de duas escritas
		// concorrentes. 500 esconderia a causa e faria o cliente repetir a escrita a' cegas.
		c.JSON(http.StatusConflict, gin.H{"error": "version diverged", "detail": "re-read the note and retry with the current version"})
	case errors.Is(err, agent.ErrConteudoAlterado):
		c.JSON(http.StatusConflict, gin.H{"error": "content was modified", "detail": "the note no longer matches its content_hash"})
	case errors.Is(err, agent.ErrConteudoAlterado):
		// A nota foi LIDA mas o corpo nao corresponde ao content_hash gravado: alguem a editou
		// fora do adaptador. 409 diz "o estado mudou", nao "nao tens acesso".
		c.JSON(http.StatusConflict, gin.H{"error": "content was modified", "detail": "the note no longer matches its content_hash"})
	case errors.Is(err, agent.ErrAcessoNegado):
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, agent.ErrForaDaRaiz):
		c.JSON(http.StatusBadRequest, gin.H{"error": "path outside the vault root"})
	case errors.Is(err, agent.ErrIdentidadeDeSessaoAusente), errors.Is(err, agent.ErrTenantDaSessaoAusente):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session scope required"})
	case errors.Is(err, agent.ErrRaizDoVaultAusente):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vault_memory_disabled"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "vault append failed"})
	default:
		c.JSON(http.StatusCreated, gin.H{"ok": true, "escreveu": escreveu})
	}
}

// LerDoVaultGovernado trata GET /memory/vault/read.
//
// Mesma regra do append: identidade e organizacao vem do contexto do middleware, nunca da query nem
// do corpo. Negacao de permissao, nota inexistente e nota de outro tenant devolvem a MESMA resposta
// (403) -- distinguir entre elas entregaria a existencia por omissao.
func (a *agentAPI) lerMemoriaDoVault(c *gin.Context) {
	raiz := strings.TrimSpace(os.Getenv("HADES_VAULT_ROOT"))
	if raiz == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vault_memory_disabled"})
		return
	}
	identidade, ok := IdentidadeDaSessao(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session identity required"})
		return
	}
	tenant, ok := OrganizacaoDaSessao(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session organization required"})
		return
	}
	politica, erroDePolitica := agent.ParsearPoliticaDeAcesso(os.Getenv("HADES_VAULT_ACL"))
	if erroDePolitica != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vault_acl_invalid"})
		return
	}
	relativo := strings.TrimSpace(c.Query("relativo"))
	if relativo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "relativo is required"})
		return
	}

	leitura := agent.LeituraGovernada{Raiz: raiz, Politica: politica}
	memoria, err := leitura.Ler(agent.PedidoDeLeitura{Identidade: identidade, Tenant: tenant, Relativo: relativo})
	switch {
	case errors.Is(err, agent.ErrSegredoNaNota):
		// Recusa por DLP: 422 (o corpo e' compreendido e recusado por politica), nunca 500.
		// A resposta NAO ecoa o valor detetado -- ecoa-lo seria o segundo vazamento.
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "secret_detected", "detail": "content carries credential-shaped material and was not stored"})
	case errors.Is(err, agent.ErrVersaoDivergente):
		// Conflito de versao nao e' falha do servidor: e' o resultado normal de duas escritas
		// concorrentes. 500 esconderia a causa e faria o cliente repetir a escrita a' cegas.
		c.JSON(http.StatusConflict, gin.H{"error": "version diverged", "detail": "re-read the note and retry with the current version"})
	case errors.Is(err, agent.ErrConteudoAlterado):
		c.JSON(http.StatusConflict, gin.H{"error": "content was modified", "detail": "the note no longer matches its content_hash"})
	case errors.Is(err, agent.ErrConteudoAlterado):
		// A nota foi LIDA mas o corpo nao corresponde ao content_hash gravado: alguem a editou
		// fora do adaptador. 409 diz "o estado mudou", nao "nao tens acesso".
		c.JSON(http.StatusConflict, gin.H{"error": "content was modified", "detail": "the note no longer matches its content_hash"})
	case errors.Is(err, agent.ErrAcessoNegado):
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, agent.ErrForaDaRaiz):
		c.JSON(http.StatusBadRequest, gin.H{"error": "path outside the vault root"})
	case errors.Is(err, agent.ErrIdentidadeDeSessaoAusente), errors.Is(err, agent.ErrTenantDaSessaoAusente):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session scope required"})
	case errors.Is(err, agent.ErrRaizDoVaultAusente):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vault_memory_disabled"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "vault read failed"})
	default:
		c.JSON(http.StatusOK, memoria)
	}
}

// ApagarDoVaultGovernado trata DELETE /memory/vault/note?relativo=...
//
// Mesma autoridade: identidade e organizacao do contexto do middleware; permissao de ESCRITA no
// escopo do tenant (apagar e' escrita). A remocao deixa uma linha no livro de auditoria do vault,
// sem guardar o conteudo apagado.
func (a *agentAPI) apagarDoVaultGovernado(c *gin.Context) {
	raiz := strings.TrimSpace(os.Getenv("HADES_VAULT_ROOT"))
	if raiz == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vault_memory_disabled"})
		return
	}
	identidade, ok := IdentidadeDaSessao(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session identity required"})
		return
	}
	tenant, ok := OrganizacaoDaSessao(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session organization required"})
		return
	}
	politica, erroDePolitica := agent.ParsearPoliticaDeAcesso(os.Getenv("HADES_VAULT_ACL"))
	if erroDePolitica != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vault_acl_invalid"})
		return
	}
	relativo := strings.TrimSpace(c.Query("relativo"))
	if relativo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "relativo is required"})
		return
	}

	remocao := agent.RemocaoGovernada{Raiz: raiz, Politica: politica}
	apagou, err := remocao.Apagar(agent.PedidoDeRemocao{Identidade: identidade, Tenant: tenant, Relativo: relativo})
	switch {
	case errors.Is(err, agent.ErrAcessoNegado):
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, agent.ErrForaDaRaiz):
		c.JSON(http.StatusBadRequest, gin.H{"error": "path outside the vault root"})
	case errors.Is(err, agent.ErrIdentidadeDeSessaoAusente), errors.Is(err, agent.ErrTenantDaSessaoAusente):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session scope required"})
	case errors.Is(err, agent.ErrRaizDoVaultAusente):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vault_memory_disabled"})
	case errors.Is(err, agent.ErrNotaNaoApagada):
		c.JSON(http.StatusInternalServerError, gin.H{"error": "note was not removed"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "vault delete failed"})
	default:
		c.JSON(http.StatusOK, gin.H{"ok": true, "apagou": apagou})
	}
}

// CompactarOVaultGovernado trata POST /memory/vault/compact.
//
// Compactacao NAO DESTRUTIVA: os originais sao movidos para `arquivo/<instante>/` com os mesmos
// bytes, e o resumo cita memory_id, caminho e hash de cada um. Se algo falhar a meio, o bruto ja'
// esta' no arquivo -- nenhuma nota se perde.
func (a *agentAPI) compactarOVaultGovernado(c *gin.Context) {
	raiz := strings.TrimSpace(os.Getenv("HADES_VAULT_ROOT"))
	if raiz == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vault_memory_disabled"})
		return
	}
	identidade, ok := IdentidadeDaSessao(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session identity required"})
		return
	}
	tenant, ok := OrganizacaoDaSessao(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session organization required"})
		return
	}
	politica, erroDePolitica := agent.ParsearPoliticaDeAcesso(os.Getenv("HADES_VAULT_ACL"))
	if erroDePolitica != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vault_acl_invalid"})
		return
	}
	var corpo struct {
		Pasta          string `json:"pasta"`
		ResumoRelativo string `json:"resumo_relativo"`
		TextoDoResumo  string `json:"texto_do_resumo"`
	}
	if err := DecodificarCorpoEstrito(c.Request.Body, &corpo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	compactacao := agent.CompactacaoGovernada{Raiz: raiz, Politica: politica}
	relatorio, err := compactacao.Compactar(agent.PedidoDeCompactacao{
		Identidade:     identidade,
		Tenant:         tenant,
		Pasta:          corpo.Pasta,
		ResumoRelativo: corpo.ResumoRelativo,
		TextoDoResumo:  corpo.TextoDoResumo,
	})
	switch {
	case errors.Is(err, agent.ErrAcessoNegado):
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, agent.ErrForaDaRaiz):
		c.JSON(http.StatusBadRequest, gin.H{"error": "path outside the vault root"})
	case errors.Is(err, agent.ErrNadaACompactar):
		c.JSON(http.StatusConflict, gin.H{"error": "nothing to compact"})
	case errors.Is(err, agent.ErrCompactacaoInsegura):
		c.JSON(http.StatusConflict, gin.H{"error": "compaction refused: destination collision"})
	case errors.Is(err, agent.ErrIdentidadeDeSessaoAusente), errors.Is(err, agent.ErrTenantDaSessaoAusente):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session scope required"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "vault compaction failed"})
	default:
		c.JSON(http.StatusOK, relatorio)
	}
}
