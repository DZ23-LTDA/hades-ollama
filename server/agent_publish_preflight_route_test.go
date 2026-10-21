package server

// FASE 28.7 do adendo V3: testes da rota `publish.preflight`.
//
// O que estes testes travam:
//
//  1. canal DESCONHECIDO e' recusado com 400 **antes** de tocar no filesystem;
//  2. sem `HADES_ARTEFACTO_A_PUBLICAR` a rota recusa com 412 -- nao devolve um preflight vazio que
//     parece aprovacao;
//  3. `publicado` e' **sempre** false, inclusive no caminho felizes;
//  4. `LOCAL_ONLY` com artefacto integro fica `pronto` mas diz que NAO e' deploy publico;
//  5. canal publico **nunca** fica `pronto` nesta missao, e `requer_aprovacao` e' true;
//  6. a credencial e' medida por existencia e tamanho -- **o conteudo nao e' lido**;
//  7. os caminhos vem do AMBIENTE, nunca do pedido.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// montarRotaDoPreflight cria o router minimo com a rota real.
func montarRotaDoPreflight(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := &agentAPI{}
	router.POST("/publish/preflight/:canal", api.preflightDePublicacao)
	return router
}

// pedirPreflight faz o pedido e devolve o corpo descodificado.
func pedirPreflight(t *testing.T, router *gin.Engine, canal string) (int, respostaDoPreflightDePublicacao, map[string]any) {
	t.Helper()
	pedido := httptest.NewRequest(http.MethodPost, "/publish/preflight/"+canal, nil)
	gravador := httptest.NewRecorder()
	router.ServeHTTP(gravador, pedido)

	var corpo respostaDoPreflightDePublicacao
	var generico map[string]any
	if gravador.Code == http.StatusOK {
		if err := json.Unmarshal(gravador.Body.Bytes(), &corpo); err != nil {
			t.Fatalf("descodificar resposta 200: %v (corpo: %s)", err, gravador.Body.String())
		}
		return gravador.Code, corpo, nil
	}
	if err := json.Unmarshal(gravador.Body.Bytes(), &generico); err != nil {
		t.Fatalf("descodificar resposta %d: %v (corpo: %s)", gravador.Code, err, gravador.Body.String())
	}
	return gravador.Code, corpo, generico
}

// criarArtefactoDeTeste escreve um ficheiro com conteudo, para o preflight medir.
func criarArtefactoDeTeste(t *testing.T) string {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "HadesSetup-teste.exe")
	if err := os.WriteFile(caminho, []byte("conteudo do artefacto candidato"), 0o600); err != nil {
		t.Fatalf("criar artefacto: %v", err)
	}
	return caminho
}

// TestPreflightRecusaCanalDesconhecido: canal invalido nao deve provocar leitura de ficheiro.
func TestPreflightRecusaCanalDesconhecido(t *testing.T) {
	t.Setenv(variavelDoArtefactoAPublicar, criarArtefactoDeTeste(t))
	router := montarRotaDoPreflight(t)

	codigo, _, corpo := pedirPreflight(t, router, "CANAL_QUE_NAO_EXISTE")
	if codigo != http.StatusBadRequest {
		t.Fatalf("canal desconhecido: esperado 400, obtido %d (corpo: %v)", codigo, corpo)
	}
	erro, _ := corpo["erro"].(string)
	if !strings.Contains(erro, "canal desconhecido") {
		t.Fatalf("esperado erro a dizer 'canal desconhecido', obtido %q", erro)
	}
	// A mensagem deve listar os canais reais, para a accao ser possivel.
	canais, ok := corpo["canais"].([]any)
	if !ok || len(canais) == 0 {
		t.Fatalf("esperado a lista de canais na mensagem de erro, obtido %v", corpo["canais"])
	}
	for _, nome := range []string{"LOCAL_ONLY", "GOOGLE_PLAY", "APP_STORE"} {
		encontrado := false
		for _, candidato := range canais {
			if candidato == nome {
				encontrado = true
				break
			}
		}
		if !encontrado {
			t.Fatalf("canal %q ausente da lista %v", nome, canais)
		}
	}
}

// TestPreflightSemArtefactoRecusaCom412: sem artefacto nao ha' preflight.
func TestPreflightSemArtefactoRecusaCom412(t *testing.T) {
	t.Setenv(variavelDoArtefactoAPublicar, "")
	router := montarRotaDoPreflight(t)

	codigo, _, corpo := pedirPreflight(t, router, "LOCAL_ONLY")
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("sem artefacto: esperado 412, obtido %d (corpo: %v)", codigo, corpo)
	}
	erro, _ := corpo["erro"].(string)
	if !strings.Contains(erro, variavelDoArtefactoAPublicar) {
		t.Fatalf("o erro deve nomear a variavel %q, obtido %q", variavelDoArtefactoAPublicar, erro)
	}
}

// TestPreflightRecusaCaminhoRelativo: um caminho relativo depende do cwd do servidor.
func TestPreflightRecusaCaminhoRelativo(t *testing.T) {
	t.Setenv(variavelDoArtefactoAPublicar, "artefacto-relativo.exe")
	router := montarRotaDoPreflight(t)

	codigo, _, corpo := pedirPreflight(t, router, "LOCAL_ONLY")
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("caminho relativo: esperado 412, obtido %d (corpo: %v)", codigo, corpo)
	}
	erro, _ := corpo["erro"].(string)
	if !strings.Contains(erro, "caminho absoluto") {
		t.Fatalf("o erro deve dizer 'caminho absoluto', obtido %q", erro)
	}
}

// TestPreflightLocalOnlyComArtefactoIntegro: artefacto real -> pronto, mas NAO publicado.
func TestPreflightLocalOnlyComArtefactoIntegro(t *testing.T) {
	t.Setenv(variavelDoArtefactoAPublicar, criarArtefactoDeTeste(t))
	router := montarRotaDoPreflight(t)

	codigo, corpo, _ := pedirPreflight(t, router, "LOCAL_ONLY")
	if codigo != http.StatusOK {
		t.Fatalf("LOCAL_ONLY: esperado 200, obtido %d", codigo)
	}
	// AS DUAS GARANTIAS, EM CAMPOS SEPARADOS.
	if corpo.NadaPublicado != true {
		t.Fatal("`nada_publicado` deve ser SEMPRE true: esta rota nao publica")
	}
	if corpo.Preflight.Publicado {
		t.Fatal("`publicado` deve ser SEMPRE false: o provider nao tem submit")
	}
	if !corpo.SomenteLeitura {
		t.Fatal("`somente_leitura` deve ser true")
	}
	if !corpo.Preflight.Pronto {
		t.Fatalf("artefacto integro: esperado pronto, condicoes: %+v", corpo.Preflight.Condicoes)
	}
	if corpo.Preflight.Publico {
		t.Fatal("LOCAL_ONLY nao e' canal publico")
	}
	if corpo.Preflight.RequerAprovacao {
		t.Fatal("canal local nao exige aprovacao de publicacao")
	}
	// A explicacao tem de negar a equivalencia com deploy publico.
	if !strings.Contains(corpo.Preflight.Explicacao, "NAO") {
		t.Fatalf("a explicacao de LOCAL_ONLY deve negar o deploy publico, obtida %q", corpo.Preflight.Explicacao)
	}
	// O aviso de fronteira nomeia a proibicao.
	if !strings.Contains(corpo.AvisoDeFronteira, "NAO publica") {
		t.Fatalf("o aviso de fronteira deve dizer que nao publica, obtido %q", corpo.AvisoDeFronteira)
	}
}

// TestPreflightCanalPublicoNuncaFicaPronto: mesmo com artefacto E credencial, o canal publico nao
// esta' "disponivel" -- a autorizacao vigente proibe publicar.
func TestPreflightCanalPublicoNuncaFicaPronto(t *testing.T) {
	artefacto := criarArtefactoDeTeste(t)
	credencial := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(credencial, []byte(`{"tipo":"marcador-de-teste"}`), 0o600); err != nil {
		t.Fatalf("criar credencial: %v", err)
	}
	t.Setenv(variavelDoArtefactoAPublicar, artefacto)
	t.Setenv(variavelDaCredencialDePublicacao, credencial)

	router := montarRotaDoPreflight(t)
	for _, canal := range []string{"GOOGLE_PLAY", "APP_STORE", "GITHUB_RELEASE", "OCI_REGISTRY", "EAS_UPDATE"} {
		t.Run(canal, func(t *testing.T) {
			codigo, corpo, _ := pedirPreflight(t, router, canal)
			if codigo != http.StatusOK {
				t.Fatalf("%s: esperado 200, obtido %d", canal, codigo)
			}
			if !corpo.Preflight.Publico {
				t.Fatalf("%s deveria ser canal publico", canal)
			}
			if corpo.Preflight.Pronto {
				t.Fatalf("%s NAO pode ficar pronto: a autorizacao vigente proibe publicar", canal)
			}
			if !corpo.Preflight.RequerAprovacao {
				t.Fatalf("%s: canal publico exige aprovacao SEMPRE", canal)
			}
			if corpo.Preflight.Publicado {
				t.Fatalf("%s: `publicado` tem de ser false", canal)
			}
			// A condicao "canal disponivel" tem de estar explicitamente insatisfeita.
			var encontrouCanal bool
			for _, condicao := range corpo.Preflight.Condicoes {
				if condicao.Nome == "canal disponivel" {
					encontrouCanal = true
					if condicao.Satisfeita {
						t.Fatalf("%s: 'canal disponivel' nao pode estar satisfeita", canal)
					}
				}
			}
			if !encontrouCanal {
				t.Fatalf("%s: falta a condicao 'canal disponivel'", canal)
			}
		})
	}
}

// TestPreflightNaoLeConteudoDaCredencial: o detalhe reporta tamanho, nunca o valor.
func TestPreflightNaoLeConteudoDaCredencial(t *testing.T) {
	artefacto := criarArtefactoDeTeste(t)
	credencial := filepath.Join(t.TempDir(), "credencial.json")
	// Um canario que NAO pode aparecer em nenhuma resposta.
	const canario = "CANARIO-QUE-NAO-PODE-VAZAR-12345"
	if err := os.WriteFile(credencial, []byte(canario), 0o600); err != nil {
		t.Fatalf("criar credencial: %v", err)
	}
	t.Setenv(variavelDoArtefactoAPublicar, artefacto)
	t.Setenv(variavelDaCredencialDePublicacao, credencial)

	router := montarRotaDoPreflight(t)
	codigo, corpo, _ := pedirPreflight(t, router, "GOOGLE_PLAY")
	if codigo != http.StatusOK {
		t.Fatalf("esperado 200, obtido %d", codigo)
	}

	var encontrouCredencial bool
	for _, condicao := range corpo.Preflight.Condicoes {
		if condicao.Nome != "credencial presente" {
			continue
		}
		encontrouCredencial = true
		if !condicao.Satisfeita {
			t.Fatalf("a credencial existe; a condicao devia estar satisfeita: %q", condicao.Detalhe)
		}
		if !strings.Contains(condicao.Detalhe, "conteudo NAO lido") {
			t.Fatalf("o detalhe deve declarar que o conteudo nao foi lido, obtido %q", condicao.Detalhe)
		}
		if strings.Contains(condicao.Detalhe, canario) {
			t.Fatal("O CONTEUDO DA CREDENCIAL VAZOU para a resposta")
		}
	}
	if !encontrouCredencial {
		t.Fatal("falta a condicao 'credencial presente'")
	}
	// E o canario nao pode estar em NENHUM campo textual da resposta.
	serializado, err := json.Marshal(corpo)
	if err != nil {
		t.Fatalf("serializar: %v", err)
	}
	if strings.Contains(string(serializado), canario) {
		t.Fatal("O CONTEUDO DA CREDENCIAL VAZOU no corpo serializado")
	}
}

// TestPreflightSemCredencialExplicaORequisito: canal publico sem credencial diz o que falta.
func TestPreflightSemCredencialExplicaORequisito(t *testing.T) {
	t.Setenv(variavelDoArtefactoAPublicar, criarArtefactoDeTeste(t))
	t.Setenv(variavelDaCredencialDePublicacao, "")

	router := montarRotaDoPreflight(t)
	codigo, corpo, _ := pedirPreflight(t, router, "GOOGLE_PLAY")
	if codigo != http.StatusOK {
		t.Fatalf("esperado 200, obtido %d", codigo)
	}
	var detalhe string
	for _, condicao := range corpo.Preflight.Condicoes {
		if condicao.Nome == "credencial presente" {
			detalhe = condicao.Detalhe
		}
	}
	if detalhe == "" {
		t.Fatal("falta a condicao 'credencial presente'")
	}
	// O requisito concreto do canal: service account do Play Console.
	if !strings.Contains(detalhe, "service account") {
		t.Fatalf("o detalhe deve nomear o requisito do Play, obtido %q", detalhe)
	}
}

// TestPreflightArtefactoInexistenteNaoPassa: um caminho absoluto que nao existe.
func TestPreflightArtefactoInexistenteNaoPassa(t *testing.T) {
	inexistente := filepath.Join(t.TempDir(), "nao-existe.exe")
	t.Setenv(variavelDoArtefactoAPublicar, inexistente)

	router := montarRotaDoPreflight(t)
	codigo, corpo, _ := pedirPreflight(t, router, "LOCAL_ONLY")
	if codigo != http.StatusOK {
		t.Fatalf("esperado 200 (o preflight falha, nao a rota), obtido %d", codigo)
	}
	if corpo.Preflight.Pronto {
		t.Fatal("artefacto inexistente nao pode deixar o preflight pronto")
	}
	if !strings.Contains(corpo.Preflight.Explicacao, "NAO passou") {
		t.Fatalf("a explicacao deve dizer que nao passou, obtida %q", corpo.Preflight.Explicacao)
	}
}

// TestCanaisDePublicacaoTemSeisCanais garante que o catalogo e' o esperado -- se o provider ganhar
// um canal, este teste avisa em vez de a rota aceitar um canal que o provider nao conhece.
func TestCanaisDePublicacaoTemSeisCanais(t *testing.T) {
	catalogo := agent.CanaisDePublicacao()
	if len(catalogo) != 6 {
		t.Fatalf("esperado 6 canais, obtido %d: %v", len(catalogo), catalogo)
	}
	// `LOCAL_ONLY` e' o unico nao-publico.
	for canal, publico := range catalogo {
		if canal == agent.PublicacaoLocal {
			if publico {
				t.Fatal("LOCAL_ONLY nao pode ser declarado publico")
			}
			continue
		}
		if !publico {
			t.Fatalf("canal %q deveria ser publico", canal)
		}
	}
}
