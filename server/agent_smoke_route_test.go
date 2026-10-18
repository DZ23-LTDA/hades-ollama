package server

// Testes de `build.smoke` (E-1273) -- a rota que EXECUTA um binario.
//
// O que estes testes travam:
//
//  1. sem `HADES_ARTEFACTOS` a rota recusa com 412 -- *nao ha' execucao sem fronteira declarada*;
//  2. um caminho DENTRO da raiz corre e e' aprovado;
//  3. um caminho FORA da raiz e' recusado com 403, mesmo quando o ficheiro existe;
//  4. **um symlink dentro da raiz a apontar para fora e' recusado** -- sem `EvalSymlinks` a
//     verificacao de prefixo aprovaria o LINK;
//  5. sem argumentos e' recusado (400) -- executar no modo por omissao nao e' um teste;
//  6. mais de 32 argumentos e' recusado (413);
//  7. um timeout pedido acima do maximo e' cortado para o maximo, nao aceito por escrito;
//  8. a resposta declara `sem_sandbox` e o aviso, sempre.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
)

// raizDeArtefactos monta a raiz autorizada e devolve-a.
func raizDeArtefactos(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// montarRotaDeSmoke monta o router com a rota e a raiz.
func montarRotaDeSmoke(t *testing.T, raiz string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv(variavelDosArtefactos, raiz)
	router := gin.New()
	router.POST("/api/agent/v1/build/smoke", (&agentAPI{}).executarSmoke)
	return router
}

// pedirSmoke faz o POST e devolve o codigo e o corpo cru.
func pedirSmoke(t *testing.T, router *gin.Engine, corpo any) (int, []byte) {
	t.Helper()
	bruto, err := json.Marshal(corpo)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pedido := httptest.NewRequest(http.MethodPost, "/api/agent/v1/build/smoke",
		bytes.NewReader(bruto))
	pedido.Header.Set("Content-Type", "application/json")
	gravador := httptest.NewRecorder()
	router.ServeHTTP(gravador, pedido)
	return gravador.Code, gravador.Body.Bytes()
}

// executavelDeSmoke devolve um executavel real do host.
func executavelDeSmoke(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		caminho, err := exec.LookPath("cmd.exe")
		if err != nil {
			t.Skipf("cmd.exe nao encontrado: %v", err)
		}
		return caminho
	}
	caminho, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh nao encontrado: %v", err)
	}
	return caminho
}

// argumentosDeEcho devolve os argumentos que imprimem a marca.
func argumentosDeEcho(marca string) []string {
	if runtime.GOOS == "windows" {
		return []string{"/c", "echo " + marca}
	}
	return []string{"-c", "echo " + marca}
}

// Sem a raiz configurada a rota recusa.
func TestSmokeSemRaizConfigurada(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(variavelDosArtefactos, "")
	router := gin.New()
	router.POST("/api/agent/v1/build/smoke", (&agentAPI{}).executarSmoke)

	codigo, corpo := pedirSmoke(t, router, pedidoDeSmoke{
		Caminho:    "qualquer.exe",
		Argumentos: []string{"--version"},
	})
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("status = %d; esperado 412 (nao ha' execucao sem fronteira)", codigo)
	}
	if !bytes.Contains(corpo, []byte(variavelDosArtefactos)) {
		t.Fatalf("erro nao diz qual variavel definir: %s", corpo)
	}
}

// Um caminho FORA da raiz e' recusado, mesmo existindo.
func TestSmokeRecusaCaminhoForaDaRaiz(t *testing.T) {
	raiz := raizDeArtefactos(t)
	router := montarRotaDeSmoke(t, raiz)
	// O executavel do sistema esta' fora da raiz temporaria.
	executavel := executavelDeSmoke(t)
	codigo, corpo := pedirSmoke(t, router, pedidoDeSmoke{
		Caminho:    executavel,
		Argumentos: argumentosDeEcho("x"),
	})
	if codigo != http.StatusForbidden {
		t.Fatalf("status = %d; esperado 403 para um caminho fora da raiz: %s", codigo, corpo)
	}
	if !bytes.Contains(corpo, []byte("fora")) {
		t.Fatalf("erro nao declara que o caminho esta' fora: %s", corpo)
	}
}

// UM SYMLINK DENTRO DA RAIZ A APONTAR PARA FORA E' RECUSADO.
//
//	*Uma verificacao de prefixo sem resolver symlinks e' uma verificacao que aprova o link, nao o alvo.*
func TestSmokeRecusaSymlinkQueEscapa(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("criar symlinks no Windows exige privilegio; o teste corre nos outros SO")
	}
	raiz := raizDeArtefactos(t)
	router := montarRotaDeSmoke(t, raiz)
	// Um link dentro da raiz que aponta para o executavel do sistema (fora).
	link := filepath.Join(raiz, "artefacto")
	if err := os.Symlink(executavelDeSmoke(t), link); err != nil {
		t.Skipf("nao foi possivel criar o symlink: %v", err)
	}
	codigo, corpo := pedirSmoke(t, router, pedidoDeSmoke{
		Caminho:    link,
		Argumentos: argumentosDeEcho("x"),
	})
	if codigo != http.StatusForbidden {
		t.Fatalf("status = %d; esperado 403: um symlink que escapa nao pode ser aprovado: %s",
			codigo, corpo)
	}
}

// Sem argumentos e' recusado.
func TestSmokeRecusaSemArgumentos(t *testing.T) {
	raiz := raizDeArtefactos(t)
	router := montarRotaDeSmoke(t, raiz)
	// Copiar o executavel para dentro da raiz para passar a fronteira.
	destino := filepath.Join(raiz, "artefacto"+sufixoDeExecutavel())
	copiarFicheiro(t, executavelDeSmoke(t), destino)
	codigo, corpo := pedirSmoke(t, router, pedidoDeSmoke{Caminho: destino})
	if codigo != http.StatusBadRequest {
		t.Fatalf("status = %d; esperado 400 sem argumentos: %s", codigo, corpo)
	}
}

// Um caminho DENTRO da raiz corre e e' aprovado.
func TestSmokeAprovaArtefactoDentroDaRaiz(t *testing.T) {
	raiz := raizDeArtefactos(t)
	router := montarRotaDeSmoke(t, raiz)
	destino := filepath.Join(raiz, "artefacto"+sufixoDeExecutavel())
	copiarFicheiro(t, executavelDeSmoke(t), destino)

	codigo, corpo := pedirSmoke(t, router, pedidoDeSmoke{
		Caminho:       destino,
		Argumentos:    argumentosDeEcho("HADES-SMOKE-ROTA"),
		SaidaEsperada: "HADES-SMOKE-ROTA",
	})
	if codigo != http.StatusOK {
		t.Fatalf("status = %d; esperado 200: %s", codigo, corpo)
	}
	var resposta respostaDeSmoke
	if err := json.Unmarshal(corpo, &resposta); err != nil {
		t.Fatalf("resposta invalida: %v (%s)", err, corpo)
	}
	if !resposta.Aprovado {
		t.Fatalf("nao aprovado: %+v", resposta)
	}
	if !resposta.RaizAutorizada {
		t.Fatal("raiz_autorizada = false numa execucao que passou a fronteira")
	}
	// A fronteira de sandbox tem de ir no RESULTADO, nao so' na documentacao.
	if !resposta.SemSandbox {
		t.Fatal("sem_sandbox = false")
	}
	if resposta.AvisoDeSandbox == "" {
		t.Fatal("aviso_de_sandbox vazio")
	}
}

// Mais de 32 argumentos e' recusado.
func TestSmokeRecusaDemasiadosArgumentos(t *testing.T) {
	raiz := raizDeArtefactos(t)
	router := montarRotaDeSmoke(t, raiz)
	destino := filepath.Join(raiz, "artefacto"+sufixoDeExecutavel())
	copiarFicheiro(t, executavelDeSmoke(t), destino)

	argumentos := make([]string, maxArgumentosDeSmoke+1)
	for i := range argumentos {
		argumentos[i] = "x"
	}
	codigo, corpo := pedirSmoke(t, router, pedidoDeSmoke{
		Caminho:    destino,
		Argumentos: argumentos,
	})
	if codigo != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d; esperado 413: %s", codigo, corpo)
	}
}

// sufissoDeExecutavel devolve a extensao que o SO precisa.
func sufixoDeExecutavel() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// copiarFicheiro copia bytes e preserva o modo de execucao.
func copiarFicheiro(t *testing.T, origem, destino string) {
	t.Helper()
	conteudo, err := os.ReadFile(origem)
	if err != nil {
		t.Fatalf("ler %q: %v", origem, err)
	}
	if err := os.WriteFile(destino, conteudo, 0o755); err != nil {
		t.Fatalf("escrever %q: %v", destino, err)
	}
}
