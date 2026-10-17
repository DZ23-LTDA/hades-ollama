package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/api"
)

// RECUPERAÇÃO PELO CAMINHO REAL (E-582) — junta as duas peças que ainda não tinham sido
// exercitadas juntas: `CaminhoDoHistorico` (E-581, com recusa de travessia) e
// `FluxoDaCompactacao` (E-575, com a ordem bruto-antes-do-resumo).
//
// PORQUE ISTO VALE UM TESTE PRÓPRIO: o E-575 provou a recuperação com um caminho de
// `t.TempDir()`, e o E-581 provou a recusa com uma raiz descartável. Nenhum dos dois
// provou que o caminho REALMENTE derivado de organização e projeto — o que o servidor vai
// usar — serve para gravar, reabrir e recuperar. Duas peças correctas isoladas não fazem
// uma junta correcta; é o defeito do "irmão esquecido" outra vez.

func TestRecuperacaoPeloCaminhoDerivado(t *testing.T) {
	raiz := t.TempDir()
	caminho, err := CaminhoDoHistorico(raiz, "org-a", "proj-1")
	if err != nil {
		t.Fatalf("derivar o caminho: %v", err)
	}

	historico, err := AbrirHistoricoBruto(caminho)
	if err != nil {
		t.Fatalf("abrir no caminho derivado: %v", err)
	}
	originais := conversaLonga(10)
	res, mensagens, err := FluxoDaCompactacao(originais, 4, TetoDeDigestPadrao, historico, "proj-1", "missao-9")
	if err != nil {
		t.Fatalf("fluxo: %v", err)
	}
	if !res.Compactado {
		t.Fatalf("não compactou: %+v", res)
	}
	if err := historico.Fechar(); err != nil {
		t.Fatal(err)
	}

	// O CAMINHO DERIVADO TEM DE SER ONDE O MATERIAL FICOU — não uma pasta irmã por
	// engano. Isto é o que uma junta mal feita esconderia: um teste com caminho fixo
	// passaria mesmo que a derivação escrevesse noutro sítio.
	if _, err := os.Stat(caminho); err != nil {
		t.Fatalf("nada foi gravado no caminho derivado %q: %v", caminho, err)
	}
	if _, err := os.Stat(filepath.Join(raiz, "historico", "org-a")); err != nil {
		t.Fatalf("a pasta da organização não existe onde devia: %v", err)
	}

	// E RECUPERA-SE pelo mesmo caminho derivado, mensagem a mensagem.
	reaberto, err := AbrirHistoricoBruto(caminho)
	if err != nil {
		t.Fatalf("reabrir pelo caminho derivado: %v", err)
	}
	defer reaberto.Fechar()
	registos, err := reaberto.LerTudo()
	if err != nil {
		t.Fatal(err)
	}
	if len(registos) != 6 {
		t.Fatalf("recuperados %d registos, a faixa coberta era de 6", len(registos))
	}
	if err := VerificarCadeia(registos); err != nil {
		t.Fatalf("cadeia quebrada: %v", err)
	}
	for i, reg := range registos {
		if reg.Conteudo != originais[i].Content {
			t.Fatalf("registo %d não recupera o material coberto", i)
		}
	}
	// O resumo referencia o topo da cadeia, e o topo é o do ficheiro recuperado.
	if !strings.Contains(mensagens[0].Content, res.UltimoHash) {
		t.Fatal("o resumo não referencia o bruto")
	}
	if res.UltimoHash != registos[len(registos)-1].Hash {
		t.Fatal("o hash devolvido não é o topo da cadeia recuperada")
	}
}

// DOIS PROJETOS NÃO PARTILHAM FICHEIRO: o caminho deriva do PAR organização+projeto, e
// uma colisão entre tenants seria vazamento de contexto, não apenas um ficheiro trocado.
func TestCaminhosDeProjetosDiferentesNaoColidem(t *testing.T) {
	raiz := t.TempDir()
	a, err := CaminhoDoHistorico(raiz, "org-a", "proj-1")
	if err != nil {
		t.Fatal(err)
	}
	b, err := CaminhoDoHistorico(raiz, "org-b", "proj-1")
	if err != nil {
		t.Fatal(err)
	}
	c, err := CaminhoDoHistorico(raiz, "org-a", "proj-2")
	if err != nil {
		t.Fatal(err)
	}
	if a == b || a == c || b == c {
		t.Fatalf("caminhos colidiram: %q %q %q", a, b, c)
	}
	// Gravar em A não pode aparecer em B.
	ha, err := AbrirHistoricoBruto(a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ha.Anexar(RegistroDeHistorico{Papel: "user", Conteudo: "segredo da org-a"}); err != nil {
		t.Fatal(err)
	}
	ha.Fechar()
	hb, err := AbrirHistoricoBruto(b)
	if err != nil {
		t.Fatal(err)
	}
	defer hb.Fechar()
	registos, err := hb.LerTudo()
	if err != nil {
		t.Fatal(err)
	}
	if len(registos) != 0 {
		t.Fatalf("o histórico da org-b tem %d registos que não são dela", len(registos))
	}
	_ = api.Message{} // mantém o import explícito do pacote de mensagens
}
