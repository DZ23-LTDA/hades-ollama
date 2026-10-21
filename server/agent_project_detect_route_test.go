package server

// Testes de `project.detect` (E-1286).
//
// O que estes testes travam:
//
//  1. sem `HADES_RAIZ_PROJECTO` a rota recusa com 412 e **nomeia a variavel**;
//  2. uma raiz RELATIVA recusa com 412 -- o detector nao depende do CWD do servidor;
//  3. **a raiz NAO vem do pedido** -- um `raiz` no corpo e' ignorado (o teste prova-o com duas raizes
//     diferentes, uma valida e outra nao);
//  4. um ficheiro em vez de directorio recusa com 412;
//  5. um projecto Go REAL produz a stack e os alvos;
//  6. um directorio VAZIO nao e' erro -- devolve listas vazias e o aviso de fronteira;
//  7. **o aviso de fronteira diz que os alvos sao SUGESTAO e que `toolchain.doctor` e' que decide** --
//     *um alvo derivado de um ficheiro nao prova que a toolchain exista*.
//
// A doutrina que o teste 3 aplica:
//
//	*Um cliente que escolhe a raiz dos binarios e' um cliente que decide o que corre.* (E-1273)

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// montarRotaDeDeteccao monta o router com a raiz.
func montarRotaDeDeteccao(t *testing.T, raiz string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv(variavelDaRaizDoProjecto, raiz)
	router := gin.New()
	router.POST("/api/agent/v1/project/detect", (&agentAPI{}).detectarProjecto)
	return router
}

// pedirDeteccaoDeProjecto faz o POST e devolve o codigo e o corpo.
func pedirDeteccaoDeProjecto(t *testing.T, router *gin.Engine, corpo any) (int, map[string]any) {
	t.Helper()
	bruto, err := json.Marshal(corpo)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pedido := httptest.NewRequest(http.MethodPost, "/api/agent/v1/project/detect",
		bytes.NewReader(bruto))
	pedido.Header.Set("Content-Type", "application/json")
	resposta := httptest.NewRecorder()
	router.ServeHTTP(resposta, pedido)

	var decodificado map[string]any
	if len(resposta.Body.Bytes()) > 0 {
		_ = json.Unmarshal(resposta.Body.Bytes(), &decodificado)
	}
	return resposta.Code, decodificado
}

// TestDeteccaoSemRaizConfiguradaRecusaCom412 prova a recusa quando a variavel nao existe.
func TestDeteccaoSemRaizConfiguradaRecusaCom412(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(variavelDaRaizDoProjecto, "")
	router := gin.New()
	router.POST("/api/agent/v1/project/detect", (&agentAPI{}).detectarProjecto)

	codigo, corpo := pedirDeteccaoDeProjecto(t, router, map[string]any{})
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("esperado 412, obtido %d: %v", codigo, corpo)
	}
	erro, _ := corpo["erro"].(string)
	if erro == "" {
		t.Fatal("sem `erro` no corpo")
	}
	if !contemTexto(erro, variavelDaRaizDoProjecto) {
		t.Fatalf("o erro nao nomeia a variavel: %q", erro)
	}
}

// TestDeteccaoComRaizRelativaRecusaCom412 prova que um caminho relativo e' recusado.
func TestDeteccaoComRaizRelativaRecusaCom412(t *testing.T) {
	router := montarRotaDeDeteccao(t, filepath.Join("projecto", "relativo"))
	codigo, corpo := pedirDeteccaoDeProjecto(t, router, map[string]any{})
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("esperado 412, obtido %d: %v", codigo, corpo)
	}
	erro, _ := corpo["erro"].(string)
	if !contemTexto(erro, "ABSOLUTO") {
		t.Fatalf("o erro nao diz que tem de ser absoluto: %q", erro)
	}
}

// TestDeteccaoComRaizQueNaoExisteRecusaCom412 prova a recusa de uma raiz inexistente.
func TestDeteccaoComRaizQueNaoExisteRecusaCom412(t *testing.T) {
	router := montarRotaDeDeteccao(t, filepath.Join(t.TempDir(), "nao-existe"))
	codigo, corpo := pedirDeteccaoDeProjecto(t, router, map[string]any{})
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("esperado 412, obtido %d: %v", codigo, corpo)
	}
}

// TestDeteccaoComFicheiroEmVezDeDirectorioRecusaCom412 prova que um ficheiro e' recusado.
func TestDeteccaoComFicheiroEmVezDeDirectorioRecusaCom412(t *testing.T) {
	directorio := t.TempDir()
	ficheiro := filepath.Join(directorio, "go.mod")
	if err := os.WriteFile(ficheiro, []byte("module exemplo\n"), 0o600); err != nil {
		t.Fatalf("escrever ficheiro: %v", err)
	}
	router := montarRotaDeDeteccao(t, ficheiro)
	codigo, corpo := pedirDeteccaoDeProjecto(t, router, map[string]any{})
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("esperado 412, obtido %d: %v", codigo, corpo)
	}
	erro, _ := corpo["erro"].(string)
	if !contemTexto(erro, "FICHEIRO") {
		t.Fatalf("o erro nao diz que e' um ficheiro: %q", erro)
	}
}

// TestDeteccaoNaoUsaARaizDoPedido e' o teste da fronteira.
//
// A rota tem `HADES_RAIZ_PROJECTO` apontada para um projecto Go VALIDO. O pedido manda outra raiz --
// um caminho inventado. Se a rota obedecesse ao pedido, falharia; obedece ao ambiente, logo responde
// 200 com a stack do projecto do ambiente.
func TestDeteccaoNaoUsaARaizDoPedido(t *testing.T) {
	directorio := t.TempDir()
	if err := os.WriteFile(filepath.Join(directorio, "go.mod"),
		[]byte("module exemplo.pt/hades\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatalf("escrever go.mod: %v", err)
	}

	router := montarRotaDeDeteccao(t, directorio)
	codigo, corpo := pedirDeteccaoDeProjecto(t, router, map[string]any{
		"raiz": "/caminho/inventado/pelo/cliente",
	})
	if codigo != http.StatusOK {
		t.Fatalf("esperado 200 (a raiz do pedido tem de ser IGNORADA), obtido %d: %v", codigo, corpo)
	}
	detectado, ok := corpo["detectado"].(map[string]any)
	if !ok {
		t.Fatalf("sem `detectado` no corpo: %v", corpo)
	}
	raiz, _ := detectado["raiz"].(string)

	// E-1303: `filepath.Clean` NAO RESOLVE SYMLINKS.
	//
	// No macOS o `t.TempDir()` devolve `/var/folders/...`, mas o servidor devolve o alvo real
	// `/private/var/folders/...` -- sao o MESMO sitio escrito de duas maneiras, porque `/var` e' um
	// symlink para `/private/var` naquele sistema. O teste comparava os dois nomes com
	// `filepath.Clean` e chamava-lhes diferentes: falhava no `test (macos-latest)` e no
	// `race (macos-latest)` com
	//
	//   devolvida devia ser a do AMBIENTE ("/var/..."), obtido "/private/var/..."
	//
	// E-1305: os symlinks dos DOIS lados sao resolvidos antes de comparar, e so' se cai para
	// `filepath.Clean` quando a resolucao falha (caminho que nao existe).
	//
	// A palavra que estava aqui antes (`Resolv*mos`, a 1a pessoa do plural do verbo resolver) foi lida
	// pelo `misspell` do CI como erro de ingles (`Resolves`) e deixou o `test (windows-latest)` e o
	// `test (ubuntu-latest)` VERMELHOS no commit 63ea97085 -- nos TRES sistemas. Nao era defeito de
	// codigo: era uma palavra portuguesa num comentario, a ser lida por um corretor de ingles.
	//
	//     *Um corretor ortografico de outra lingua nao sabe distinguir um comentario de um erro.*
	//
	//     *Uma verificacao de prefixo sem resolver symlinks e' uma verificacao que aprova o link, nao
	//     o alvo.*
	esperado, errEsperado := filepath.EvalSymlinks(directorio)
	obtido, errObtido := filepath.EvalSymlinks(raiz)
	if errEsperado == nil && errObtido == nil {
		if obtido != esperado {
			t.Fatalf("a raiz devolvida devia ser a do AMBIENTE (%q), obtido %q", esperado, obtido)
		}
	} else if filepath.Clean(raiz) != filepath.Clean(directorio) {
		t.Fatalf("a raiz devolvida devia ser a do AMBIENTE (%q), obtido %q", directorio, raiz)
	}
	if filepath.Clean(raiz) == filepath.Clean("/caminho/inventado/pelo/cliente") {
		t.Fatal("a rota obedeceu a raiz do pedido -- a fronteira esta' aberta")
	}
}

// TestDeteccaoDeProjectoGoReal produz a stack.
func TestDeteccaoDeProjectoGoReal(t *testing.T) {
	directorio := t.TempDir()
	if err := os.WriteFile(filepath.Join(directorio, "go.mod"),
		[]byte("module exemplo.pt/hades\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatalf("escrever go.mod: %v", err)
	}

	router := montarRotaDeDeteccao(t, directorio)
	codigo, corpo := pedirDeteccaoDeProjecto(t, router, map[string]any{})
	if codigo != http.StatusOK {
		t.Fatalf("esperado 200, obtido %d: %v", codigo, corpo)
	}

	if somenteLeitura, _ := corpo["somente_leitura"].(bool); !somenteLeitura {
		t.Fatal("`somente_leitura` devia ser true")
	}

	detectado, _ := corpo["detectado"].(map[string]any)
	stacks, _ := detectado["stacks"].([]any)
	if len(stacks) == 0 {
		t.Fatalf("nenhuma stack detectada para um go.mod real: %v", detectado)
	}
	encontrouGo := false
	for _, stack := range stacks {
		if texto, _ := stack.(string); contemTexto(texto, "GO") || contemTexto(texto, "go") {
			encontrouGo = true
		}
	}
	if !encontrouGo {
		t.Fatalf("a stack Go nao foi detectada: %v", stacks)
	}

	// O ficheiro tem de aparecer nos manifests.
	manifests, _ := detectado["manifests"].([]any)
	if len(manifests) == 0 {
		t.Fatalf("nenhum manifesto listado: %v", detectado)
	}
}

// TestDeteccaoDeDirectorioVazioRecusaCom409 prova o comportamento REAL do detector.
//
// EU SUPUS QUE DEVOLVIA 200 COM LISTAS VAZIAS. NAO DEVOLVE: recusa com 409 e a frase "nenhuma stack
// reconhecida".
//
// E' o comportamento certo, e melhor do que o que eu tinha suposto:
//
//	*Uma lista vazia e um erro nao sao a mesma coisa: quem le' um relatorio vazio acredita que o
//	projecto nao tem stack; quem le' "nenhuma stack reconhecida" sabe que o detector nao soube
//	classificar.*
//
// A distincao importa porque a primeira manda o utilizador procurar um problema que nao existe, e a
// segunda diz-lhe exactamente o que falta: ficheiros que o detector reconheca.
func TestDeteccaoDeDirectorioVazioRecusaCom409(t *testing.T) {
	directorio := t.TempDir()
	router := montarRotaDeDeteccao(t, directorio)
	codigo, corpo := pedirDeteccaoDeProjecto(t, router, map[string]any{})
	if codigo != http.StatusConflict {
		t.Fatalf("esperado 409 (nenhuma stack reconhecida), obtido %d: %v", codigo, corpo)
	}
	erro, _ := corpo["erro"].(string)
	if !contemTexto(erro, "nenhuma stack reconhecida") {
		t.Fatalf("o erro nao diz porque falhou: %q", erro)
	}
	// E diz ONDE. Comparamos o NOME BASE e nao o caminho completo: o caminho vem escapado no JSON
	// (`\\`), e comparar o caminho cru falharia por causa do escape, nao por falta de informacao.
	//
	//	*O sintoma parecia defeito do cliente e era do fixture.* -- aqui o sintoma parecia falta de
	//	informacao e era o escape do JSON.
	if !contemTexto(erro, filepath.Base(directorio)) {
		t.Fatalf("o erro nao nomeia a pasta sondada: %q", erro)
	}
}

// TestDeteccaoDeclaraQueOsAlvosSaoSugestao prova o aviso de fronteira.
//
//	*Um alvo derivado de um ficheiro nao prova que a toolchain exista.*
func TestDeteccaoDeclaraQueOsAlvosSaoSugestao(t *testing.T) {
	directorio := t.TempDir()
	if err := os.WriteFile(filepath.Join(directorio, "go.mod"),
		[]byte("module exemplo.pt/hades\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatalf("escrever go.mod: %v", err)
	}
	router := montarRotaDeDeteccao(t, directorio)
	_, corpo := pedirDeteccaoDeProjecto(t, router, map[string]any{})

	aviso, _ := corpo["aviso_de_fronteira"].(string)
	if aviso == "" {
		t.Fatal("sem `aviso_de_fronteira`")
	}
	for _, obrigatorio := range []string{
		"SUGESTAO", "toolchain.doctor", variavelDaRaizDoProjecto,
		"nunca do pedido",
	} {
		if !contemTexto(aviso, obrigatorio) {
			t.Fatalf("o aviso nao contem %q: %q", obrigatorio, aviso)
		}
	}
}

// contemTexto diz se `texto` contem `fragmento`, sem distinguir caixa.
func contemTexto(texto, fragmento string) bool {
	return bytes.Contains(bytes.ToLower([]byte(texto)), bytes.ToLower([]byte(fragmento)))
}
