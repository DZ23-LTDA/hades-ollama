package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestAppFullstackGeradoSobeComLoginCRUDEpersistencia cobre a lacuna central do estagio 12: o app publicado era
// ESTATICO (heading/table/button), sem backend, sem login e sem dados.
//
// Aqui o Builder gera um app de gestao REAL e o teste NAO confia em nenhuma afirmacao: COMPILA o app gerado, sobe
// o processo, fala HTTP com ele, reinicia e verifica que os dados sobreviveram. Sem credencial padrao: a aplicacao
// precisa RECUSAR subir sem APP_ADMIN_PASSWORD.
func TestAppFullstackGeradoSobeComLoginCRUDEpersistencia(t *testing.T) {
	servico, err := NewBuilderService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Files nil: o conteudo vem do template do kind.
	projeto, err := servico.Create(context.Background(), BuilderSpec{Name: "Gestao de itens", Kind: BuilderFullstack})
	if err != nil {
		t.Fatal(err)
	}
	_, publicado, err := servico.PublishLocal(context.Background(), projeto.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, nome := range []string{"main.go", "index.html", "README.md"} {
		if _, err := os.Stat(filepath.Join(publicado, nome)); err != nil {
			t.Fatalf("app gerado sem %s: %v", nome, err)
		}
	}

	// No Windows o binario termina em .exe: nomear explicitamente evita exec de caminho inexistente.
	nomeBinario := "appfullstack"
	if runtime.GOOS == "windows" {
		nomeBinario += ".exe"
	}
	binario := filepath.Join(t.TempDir(), nomeBinario)
	compilar := exec.Command("go", "build", "-o", binario, ".")
	compilar.Dir = publicado
	if saida, err := compilar.CombinedOutput(); err != nil {
		t.Fatalf("app gerado NAO compila: %v\n%s", err, saida)
	}
	// No Windows o `go build -o nome` grava "nome.exe": sem isto o exec falha por caminho inexistente.
	if _, err := os.Stat(binario); err != nil {
		if _, errExe := os.Stat(binario + ".exe"); errExe == nil {
			binario += ".exe"
		} else {
			t.Fatalf("binario do app gerado nao existe em %s", binario)
		}
	}

	// SEM senha: precisa recusar subir (nenhuma credencial padrao embutida).
	semSenha := exec.Command(binario)
	semSenha.Dir = publicado
	if saida, err := semSenha.CombinedOutput(); err == nil {
		t.Fatalf("app subiu SEM APP_ADMIN_PASSWORD — credencial padrao nao pode existir: %s", saida)
	}

	dados := t.TempDir()
	senha := "senha-de-teste-forte"
	processo, porta := subirAppGerado(t, binario, publicado, dados, senha)
	listar := func(token string) []map[string]any {
		t.Helper()
		requisicao, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+porta+"/api/items", nil)
		if token != "" {
			requisicao.Header.Set("Authorization", "Bearer "+token)
		}
		resposta, err := http.DefaultClient.Do(requisicao)
		if err != nil {
			t.Fatalf("GET /api/items: %v", err)
		}
		defer resposta.Body.Close()
		if resposta.StatusCode == http.StatusUnauthorized {
			return nil
		}
		var corpo struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.NewDecoder(resposta.Body).Decode(&corpo); err != nil {
			t.Fatalf("resposta de listagem invalida: %v", err)
		}
		return corpo.Items
	}

	// SEM token: 401. Com senha errada: 401.
	if listar("") != nil {
		t.Fatal("listagem sem token precisa ser recusada com 401")
	}
	if _, status := chamarApp(t, porta, http.MethodPost, "/api/login", "", map[string]any{"password": "errada"}); status != http.StatusUnauthorized {
		t.Fatalf("senha errada deveria dar 401, veio %d", status)
	}

	_, status := chamarApp(t, porta, http.MethodPost, "/api/login", "", map[string]any{"password": senha})
	if status != http.StatusOK {
		t.Fatalf("login com a senha correta deveria dar 200, veio %d", status)
	}
	token := tokenDoLogin(t, porta, senha)
	if token == "" {
		t.Fatal("login nao devolveu token")
	}

	// CRUD real.
	for _, nome := range []string{"Parafuso", "Porca"} {
		if _, status := chamarApp(t, porta, http.MethodPost, "/api/items", token, map[string]any{"name": nome, "quantity": 3}); status != http.StatusCreated {
			t.Fatalf("criar %s deveria dar 201, veio %d", nome, status)
		}
	}
	if itens := listar(token); len(itens) != 2 {
		t.Fatalf("esperava 2 itens apos criar, veio %d", len(itens))
	}
	// Entrada invalida e recusada (validacao real, nao decorativa).
	if _, status := chamarApp(t, porta, http.MethodPost, "/api/items", token, map[string]any{"name": "  ", "quantity": 1}); status != http.StatusBadRequest {
		t.Fatalf("nome vazio deveria dar 400, veio %d", status)
	}

	// REINICIO: os dados precisam sobreviver.
	_ = processo.Process.Kill()
	_, _ = processo.Process.Wait()
	processo2, porta2 := subirAppGerado(t, binario, publicado, dados, senha)
	token2 := tokenDoLogin(t, porta2, senha)
	lista := func() []map[string]any {
		requisicao, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+porta2+"/api/items", nil)
		requisicao.Header.Set("Authorization", "Bearer "+token2)
		resposta, err := http.DefaultClient.Do(requisicao)
		if err != nil {
			t.Fatalf("GET /api/items apos reinicio: %v", err)
		}
		defer resposta.Body.Close()
		var corpo struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.NewDecoder(resposta.Body).Decode(&corpo); err != nil {
			t.Fatalf("listagem apos reinicio invalida: %v", err)
		}
		return corpo.Items
	}
	apos := lista()
	if len(apos) != 2 {
		t.Fatalf("os 2 itens deveriam PERSISTIR ao reinicio, veio %d", len(apos))
	}
	id, _ := apos[0]["id"].(string)
	if id == "" {
		t.Fatalf("item sem id: %+v", apos[0])
	}

	// DELETE real.
	if _, status := chamarApp(t, porta2, http.MethodDelete, "/api/items/"+id, token2, nil); status != http.StatusNoContent {
		t.Fatalf("remover item deveria dar 204, veio %d", status)
	}
	if len(lista()) != 1 {
		t.Fatalf("apos remover deveria sobrar 1 item")
	}
	_ = processo2.Process.Kill()
	_, _ = processo2.Process.Wait()

	t.Logf("APP FULL-STACK GERADO: compila · login=200 · sem_token=401 · senha_errada=401 · criados=2 · apos_reinicio=2 · apos_delete=1")
}

// subirAppGerado compila... na verdade executa o binario ja compilado e devolve a porta REAL anunciada na saida.
func subirAppGerado(t *testing.T, binario, diretorio, dados, senha string) (*exec.Cmd, string) {
	t.Helper()
	processo := exec.Command(binario)
	processo.Dir = diretorio
	processo.Env = append(os.Environ(), "APP_ADMIN_PASSWORD="+senha, "APP_DATA_DIR="+dados, "PORT=0")
	saida, err := processo.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	processo.Stderr = os.Stderr
	if err := processo.Start(); err != nil {
		t.Fatalf("subir app gerado: %v", err)
	}
	t.Cleanup(func() { _ = processo.Process.Kill() })
	leitor := bufio.NewReader(saida)
	linha, err := leitor.ReadString('\n')
	if err != nil {
		t.Fatalf("app gerado nao anunciou a porta: %v", err)
	}
	linha = strings.TrimSpace(linha)
	if !strings.HasPrefix(linha, "PORT=") {
		t.Fatalf("esperava PORT=<numero>, veio %q", linha)
	}
	porta := strings.TrimPrefix(linha, "PORT=")
	if porta == "" {
		t.Fatal("porta vazia")
	}
	return processo, porta
}

func chamarApp(t *testing.T, porta, metodo, caminho, token string, corpo any) (map[string]any, int) {
	t.Helper()
	var leitor io.Reader
	if corpo != nil {
		dados, err := json.Marshal(corpo)
		if err != nil {
			t.Fatal(err)
		}
		leitor = bytes.NewReader(dados)
	}
	requisicao, err := http.NewRequest(metodo, "http://127.0.0.1:"+porta+caminho, leitor)
	if err != nil {
		t.Fatal(err)
	}
	requisicao.Header.Set("Content-Type", "application/json")
	if token != "" {
		requisicao.Header.Set("Authorization", "Bearer "+token)
	}
	resposta, err := http.DefaultClient.Do(requisicao)
	if err != nil {
		t.Fatalf("%s %s: %v", metodo, caminho, err)
	}
	defer resposta.Body.Close()
	var decodificado map[string]any
	_ = json.NewDecoder(resposta.Body).Decode(&decodificado)
	return decodificado, resposta.StatusCode
}

func tokenDoLogin(t *testing.T, porta, senha string) string {
	t.Helper()
	corpo, status := chamarApp(t, porta, http.MethodPost, "/api/login", "", map[string]any{"password": senha})
	if status != http.StatusOK {
		t.Fatalf("login deveria dar 200, veio %d", status)
	}
	token, _ := corpo["token"].(string)
	return token
}

var _ = time.Second
