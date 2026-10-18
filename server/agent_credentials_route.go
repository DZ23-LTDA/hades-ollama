package server

// FASE 28.7 do adendo V3: `credentials.status`.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// A E-1265 implementou o `CredentialProvider` e mediu-o contra o kit REAL da DZ23 LTDA: 29 ficheiros,
// 15 lidos, 14 NAO lidos, 6 certificados. Mas **so' os testes o invocavam**:
//
//	*Uma comparacao que so' o programador consegue invocar nao e' uma funcionalidade: e' uma funcao.*
//
// `credentials.status` e' o nome que o adendo 28.7 da' a esta operacao.
//
// O CAMINHO VEM DO AMBIENTE, NAO DO PEDIDO
//
// A rota NAO aceita um caminho como parametro. Le' `HADES_KIT_CREDENCIAIS` do ambiente do servidor.
//
//	*Uma rota que aceita qualquer caminho e' uma rota que le' qualquer caminho.*
//
// O adendo 28.13 diz que o kit esta' FORA do repositorio, com ACL restrita. Se a rota aceitasse
// `?raiz=/etc`, o inventario poderia ser usado para sondar o sistema de ficheiros -- e o
// `CredentialProvider` faz `os.Stat` em tudo o que encontra. O ambiente e' a fronteira.
//
// A REDACCAAO
//
// O inventario do provider tem caminhos ABSOLUTOS. A rota devolve apenas os caminhos RELATIVOS a'
// raiz e **nao devolve a raiz**. O nome completo de um ficheiro de credencial ja' e' informacao:
//
//	*A rota que devolve o mapa do cofre nao e' a rota que mostra o estado do cofre.*
//
// E os segredos continuam nao lidos: o provider so' faz `os.Stat` neles, e a rota nao tem como
// contornar isso.

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// variavelDoKitDeCredenciais e' o nome da variavel de ambiente que aponta para o kit.
const variavelDoKitDeCredenciais = "HADES_KIT_CREDENCIAIS"

// timeoutDoInventarioDeCredenciais limita a varredura.
const timeoutDoInventarioDeCredenciais = 60 * time.Second

// credencialPublica e' uma credencial sem caminho absoluto.
type credencialPublica struct {
	CaminhoRelativo string                 `json:"caminho_relativo"`
	Tipo            agent.TipoDeCredencial `json:"tipo"`
	TamanhoBytes    int64                  `json:"tamanho_bytes"`
	ModificadoEm    string                 `json:"modificado_em"`
	ConteudoLido    bool                   `json:"conteudo_lido"`
	Fingerprint     string                 `json:"fingerprint_sha256,omitempty"`
	Sujeito         string                 `json:"sujeito,omitempty"`
	Emissor         string                 `json:"emissor,omitempty"`
	ValidoAte       string                 `json:"valido_ate,omitempty"`
	Expirado        bool                   `json:"expirado"`
	Nota            string                 `json:"nota,omitempty"`
}

// respostaDeCredenciais e' a resposta de `credentials.status`.
type respostaDeCredenciais struct {
	SomenteLeitura bool                `json:"somente_leitura"`
	Configurado    bool                `json:"configurado"`
	Credenciais    []credencialPublica `json:"credenciais"`
	Total          int                 `json:"total"`
	Lidos          int                 `json:"lidos"`
	NaoLidos       int                 `json:"nao_lidos"`
	Certificados   int                 `json:"certificados"`
	Expirados      int                 `json:"expirados"`
	Aviso          string              `json:"aviso"`
	// AvisoDeFronteira diz o que a rota nao faz.
	AvisoDeFronteira string `json:"aviso_de_fronteira"`
}

// avisoDeFronteiraDasCredenciais diz, no corpo, que nenhum segredo foi aberto.
func avisoDeFronteiraDasCredenciais() string {
	return "esta rota apenas INVENTARIA o kit de credenciais: mede tamanho, data e, nos certificados " +
		"publicos, o fingerprint e o sujeito. Os ficheiros de segredo (`.p12`, `.jks`, `.p8`, " +
		"`.dpapi`, `.mobileprovision`) NAO sao abertos -- so' o `os.Stat` e' feito neles. " +
		"`conteudo_lido: false` significa exactamente isso. A rota nao valida cadeia criptografica, " +
		"nao verifica revogacao, nao assina e nao publica. O caminho vem da variavel de ambiente " +
		"`" + variavelDoKitDeCredenciais + "`, nunca do pedido."
}

// inventariarCredenciais responde a `credentials.status`.
func (a *agentAPI) inventariarCredenciais(c *gin.Context) {
	raiz := strings.TrimSpace(os.Getenv(variavelDoKitDeCredenciais))
	if raiz == "" {
		// Nao configurado NAO e' erro: o kit e' opcional e o adendo 28.13 diz que ele vive fora do
		// repositorio. Uma resposta 404 aqui faria um cliente concluir que o servidor esta' quebrado.
		c.JSON(http.StatusOK, respostaDeCredenciais{
			SomenteLeitura: true,
			Configurado:    false,
			Credenciais:    []credencialPublica{},
			Aviso: "o kit de credenciais nao esta' configurado neste servidor; defina " +
				variavelDoKitDeCredenciais + " para o directorio do kit (fora do repositorio)",
			AvisoDeFronteira: avisoDeFronteiraDasCredenciais(),
		})
		return
	}

	contexto, cancelar := context.WithTimeout(c.Request.Context(), timeoutDoInventarioDeCredenciais)
	defer cancelar()

	provider := agent.ProviderDeCredenciais{}
	inventario, err := provider.Inventariar(contexto, raiz)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "nao foi possivel inventariar o kit: " + err.Error(),
		})
		return
	}

	publicas := make([]credencialPublica, 0, len(inventario.Credenciais))
	for _, credencial := range inventario.Credenciais {
		publicas = append(publicas, paraCredencialPublica(credencial))
	}

	c.JSON(http.StatusOK, respostaDeCredenciais{
		SomenteLeitura:   true,
		Configurado:      true,
		Credenciais:      publicas,
		Total:            inventario.Total,
		Lidos:            inventario.Lidos,
		NaoLidos:         inventario.NaoLidos,
		Certificados:     inventario.Certificados,
		Expirados:        inventario.Expirados,
		Aviso:            inventario.Aviso,
		AvisoDeFronteira: avisoDeFronteiraDasCredenciais(),
	})
}

// paraCredencialPublica remove o caminho absoluto e formata as datas.
//
//	*A rota que devolve o mapa do cofre nao e' a rota que mostra o estado do cofre.*
func paraCredencialPublica(credencial agent.Credencial) credencialPublica {
	publica := credencialPublica{
		CaminhoRelativo: credencial.CaminhoRelativo,
		Tipo:            credencial.Tipo,
		TamanhoBytes:    credencial.TamanhoBytes,
		ConteudoLido:    credencial.ConteudoLido,
		Fingerprint:     credencial.FingerprintSHA256,
		Sujeito:         credencial.Sujeito,
		Emissor:         credencial.Emissor,
		Expirado:        credencial.Expirado,
		Nota:            credencial.Nota,
	}
	// O caminho relativo devolvido e' sempre relativo: se por alguma razao vier absoluto, corta-se.
	if filepath.IsAbs(publica.CaminhoRelativo) {
		publica.CaminhoRelativo = filepath.Base(publica.CaminhoRelativo)
	}
	if !credencial.ModificadoEm.IsZero() {
		publica.ModificadoEm = credencial.ModificadoEm.UTC().Format(time.RFC3339)
	}
	// `ValidoAte` e' uma string JA' formatada pelo provider: o certificado traz a data como texto e
	// nao se reformata o que nao foi medido aqui.
	return publica
}
