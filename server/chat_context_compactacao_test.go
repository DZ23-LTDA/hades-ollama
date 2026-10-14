package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/api"
)

// Testes da compactação no funil do chat (E-583). A PROPRIEDADE CENTRAL: em qualquer
// recusa, o utilizador fica com a conversa ORIGINAL — uma compactação que falha não pode
// custar a conversa.

func mensagensDeTeste(n int) []api.Message {
	mensagens := make([]api.Message, 0, n)
	for i := 0; i < n; i++ {
		papel := "user"
		if i%2 == 1 {
			papel = "assistant"
		}
		mensagens = append(mensagens, api.Message{Role: papel, Content: "linha numero " + string(rune('a'+i))})
	}
	return mensagens
}

func decisaoQueMandaCompactar() DecisaoDeOrcamento {
	return DecisaoDeOrcamento{Compactar: true, Motivo: "uso acima do teto (teste)"}
}

func TestCompactacaoDoChatAplicaQuandoTudoEstaDeclarado(t *testing.T) {
	raiz := t.TempDir()
	originais := mensagensDeTeste(10)
	novas, res := CompactarMensagensDoChat(raiz, "org-a", "proj-1", originais, decisaoQueMandaCompactar())

	if !res.Aplicada {
		t.Fatalf("não aplicou com tudo declarado: %+v", res)
	}
	if len(novas) != 1+6 {
		t.Fatalf("mensagens = %d, esperado 1 resumo + 6 intactas", len(novas))
	}
	// O BRUTO TEM DE ESTAR NO DISCO: é o que torna a perda controlada em vez de perda.
	if _, err := os.Stat(res.Caminho); err != nil {
		t.Fatalf("o bruto não foi gravado em %q: %v", res.Caminho, err)
	}
	if res.ReferenciaDoBruto == "" {
		t.Fatal("não devolveu a referência do bruto")
	}
	// O resumo enviado referencia o bruto que cobre.
	if !strings.Contains(novas[0].Content, res.ReferenciaDoBruto) {
		t.Fatal("o resumo não referencia o bruto")
	}
	// E o caminho está DENTRO da raiz declarada.
	if rel, err := filepath.Rel(raiz, res.Caminho); err != nil || strings.HasPrefix(rel, "..") {
		t.Fatalf("o bruto ficou fora da raiz: %q", res.Caminho)
	}
	// As últimas mensagens seguem intactas, com o conteúdo original.
	for i := 0; i < 6; i++ {
		esperado := originais[len(originais)-6+i].Content
		if novas[1+i].Content != esperado {
			t.Fatalf("intacta %d alterada: %q != %q", i, novas[1+i].Content, esperado)
		}
	}
}

func TestCompactacaoDoChatNuncaPerdeAConversaNasRecusas(t *testing.T) {
	originais := mensagensDeTeste(10)

	mesmas := func(t *testing.T, nome string, novas []api.Message) {
		t.Helper()
		if len(novas) != len(originais) {
			t.Fatalf("%s: devolveu %d mensagens em vez das %d originais", nome, len(novas), len(originais))
		}
		for i := range originais {
			if novas[i].Content != originais[i].Content || novas[i].Role != originais[i].Role {
				t.Fatalf("%s: mensagem %d alterada numa recusa", nome, i)
			}
		}
	}

	// 1) Decisão não manda compactar: nada acontece, e o motivo diz a decisão original.
	novas, res := CompactarMensagensDoChat(t.TempDir(), "org-a", "proj-1", originais, DecisaoDeOrcamento{Compactar: false, Motivo: "dentro do orçamento"})
	mesmas(t, "decisão falsa", novas)
	if res.Aplicada || !strings.Contains(res.Motivo, "dentro do orçamento") {
		t.Fatalf("decisão falsa: %+v", res)
	}

	// 2) Sem raiz declarada: FALHA FECHADO. Compactar sem gravar o material perderia a
	// faixa antiga sem rasto.
	novas, res = CompactarMensagensDoChat("   ", "org-a", "proj-1", originais, decisaoQueMandaCompactar())
	mesmas(t, "sem raiz", novas)
	if res.Aplicada || !strings.Contains(res.Motivo, "raiz do histórico não declarada") {
		t.Fatalf("sem raiz: %+v", res)
	}

	// 3) Caminho recusado (travessia no projeto): mantém a conversa e DIZ porquê.
	novas, res = CompactarMensagensDoChat(t.TempDir(), "org-a", "../../etc/passwd", originais, decisaoQueMandaCompactar())
	mesmas(t, "caminho recusado", novas)
	if res.Aplicada || !strings.Contains(res.Motivo, "caminho do histórico recusado") {
		t.Fatalf("caminho recusado: %+v", res)
	}

	// 4) Histórico inabrível: `historico` é um FICHEIRO, logo não se pode criar o
	// directório e a gravação é impossível. Injeção no sistema de ficheiros, sem mock.
	raiz := t.TempDir()
	if err := os.WriteFile(filepath.Join(raiz, "historico"), []byte("bloqueio"), 0o600); err != nil {
		t.Fatal(err)
	}
	novas, res = CompactarMensagensDoChat(raiz, "org-a", "proj-1", originais, decisaoQueMandaCompactar())
	mesmas(t, "histórico inabrível", novas)
	if res.Aplicada || !strings.Contains(res.Motivo, "não consegui abrir o histórico") {
		t.Fatalf("histórico inabrível: %+v", res)
	}
}

func TestCompactacaoDoChatNaoCompactaConversaCurta(t *testing.T) {
	// Uma conversa que cabe na janela recente não é compactada: substituir mensagens
	// recentes por um resumo delas mesmas seria pior do que não fazer nada.
	raiz := t.TempDir()
	curtas := mensagensDeTeste(3)
	novas, res := CompactarMensagensDoChat(raiz, "org-a", "proj-1", curtas, decisaoQueMandaCompactar())
	if res.Aplicada {
		t.Fatalf("compactou uma conversa curta: %+v", res)
	}
	if len(novas) != 3 {
		t.Fatalf("mensagens = %d, esperado 3", len(novas))
	}
	// E não se criou histórico nenhum: não se guarda material para uma compactação que
	// não aconteceu.
	if _, err := os.Stat(filepath.Join(raiz, "historico")); err == nil {
		t.Fatal("criou histórico para uma compactação que não ocorreu")
	}
}
