package agent

// Testes da ligacao construir -> VERIFICAR (E-1261).
//
// O que estes testes travam:
//
//  1. um artefacto integro passa;
//  2. um artefacto com UM byte alterado reprova a construcao INTEIRA;
//  3. um build vazio NAO e' uma aprovacao -- devolve `ErrSemArtefactosParaVerificar`;
//  4. a explicacao conta os factos e nao os arredonda;
//  5. um erro de infraestrutura (ficheiro apagado) ABORTA em vez de virar "adulterado";
//  6. o artefacto REAL da E-1251, quando presente, e' verificado ponta a ponta.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// artefactoDeTeste escreve um ficheiro e devolve o `Artefato` que o build teria produzido.
func artefactoDeTeste(t *testing.T, nome, conteudo string) Artefato {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), nome)
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("escrever %q: %v", caminho, err)
	}
	soma := sha256.Sum256([]byte(conteudo))
	return Artefato{
		Caminho:      caminho,
		Nome:         nome,
		TamanhoBytes: int64(len(conteudo)),
		SHA256:       hex.EncodeToString(soma[:]),
		Alvo:         alvoDoHost(),
	}
}

func TestVerificarConstrucaoAprovaIntegros(t *testing.T) {
	a := artefactoDeTeste(t, "hades.exe", "conteudo do binario")
	b := artefactoDeTeste(t, "outro.exe", "outro conteudo")

	v, err := VerificarConstrucao(
		context.Background(),
		ResultadoDeBuild{Artefatos: []Artefato{a, b}},
		VerificadorDeArtefacto{},
	)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !v.Aprovada {
		t.Fatalf("aprovada = false com 2 artefactos integros: %+v", v)
	}
	if v.Total != 2 || v.Integros != 2 || v.Adulterados != 0 {
		t.Fatalf("contagens erradas: total=%d integros=%d adulterados=%d", v.Total, v.Integros, v.Adulterados)
	}
	// Sem adaptador de assinatura, os dois ficam `NAO_VERIFICADA` -- e isso e' DITO com o nome certo.
	//
	// *"Nao consegui olhar" e "olhei e nao ha'" sao factos diferentes.* Conta-los juntos produzia a
	// frase "2 com assinatura presente" para artefactos que ninguem inspeccionou.
	if v.NaoVerificadas != 2 {
		t.Fatalf("nao_verificadas = %d; esperado 2", v.NaoVerificadas)
	}
	if v.SemAssinatura != 0 {
		t.Fatalf("sem_assinatura = %d; esperado 0 (nao houve adaptador para declarar AUSENTE)", v.SemAssinatura)
	}
	if !strings.Contains(v.Explicacao, "NAO verificada") {
		t.Fatalf("explicacao nao declara a ausencia de instrumento: %q", v.Explicacao)
	}
	// E o mais importante: nao pode afirmar presenca de assinatura.
	if strings.Contains(v.Explicacao, "com assinatura presente") {
		t.Fatalf("explicacao AFIRMA assinatura presente sem ter olhado: %q", v.Explicacao)
	}
}

// O TESTE QUE IMPORTA: um byte diferente reprova a construcao INTEIRA.
func TestVerificarConstrucaoReprovaUmAdulterado(t *testing.T) {
	bom := artefactoDeTeste(t, "bom.exe", "intacto")
	mau := artefactoDeTeste(t, "mau.exe", "original")

	// Adultera o segundo DEPOIS de o build ter medido o SHA.
	conteudo, err := os.ReadFile(mau.Caminho)
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	conteudo[0] ^= 0xFF
	if err := os.WriteFile(mau.Caminho, conteudo, 0o600); err != nil {
		t.Fatalf("adulterar: %v", err)
	}

	v, err := VerificarConstrucao(
		context.Background(),
		ResultadoDeBuild{Artefatos: []Artefato{bom, mau}},
		VerificadorDeArtefacto{},
	)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if v.Aprovada {
		t.Fatal("aprovada = true com um artefacto adulterado")
	}
	if v.Adulterados != 1 {
		t.Fatalf("adulterados = %d; esperado 1", v.Adulterados)
	}
	if v.Integros != 1 {
		t.Fatalf("integros = %d; esperado 1", v.Integros)
	}
	if !strings.Contains(v.Explicacao, "ADULTERADOS") {
		t.Fatalf("explicacao nao declara a adulteracao: %q", v.Explicacao)
	}
}

// *Um plano vazio apresentado como sucesso e' a mesma mentira que um card clicavel que termina em
// tela vazia.*
func TestVerificarConstrucaoVaziaNaoAprova(t *testing.T) {
	v, err := VerificarConstrucao(context.Background(), ResultadoDeBuild{}, VerificadorDeArtefacto{})
	if !errors.Is(err, ErrSemArtefactosParaVerificar) {
		t.Fatalf("err = %v; esperado ErrSemArtefactosParaVerificar", err)
	}
	if v.Aprovada {
		t.Fatal("aprovada = true num build sem artefactos")
	}
	if v.Total != 0 {
		t.Fatalf("total = %d num build vazio", v.Total)
	}
	if v.Explicacao == "" {
		t.Fatal("explicacao vazia: quem le' nao sabe porque nao foi aprovada")
	}
}

// *Um erro que nao aborta nao e' um erro que nao aconteceu.*
func TestVerificarConstrucaoAbortaComFicheiroApagado(t *testing.T) {
	a := artefactoDeTeste(t, "apagado.exe", "vai desaparecer")
	if err := os.Remove(a.Caminho); err != nil {
		t.Fatalf("remover: %v", err)
	}

	_, err := VerificarConstrucao(
		context.Background(),
		ResultadoDeBuild{Artefatos: []Artefato{a}},
		VerificadorDeArtefacto{},
	)
	if err == nil {
		t.Fatal("erro = nil com o ficheiro apagado; devia propagar ErrFicheiroInexistente")
	}
	if !errors.Is(err, ErrFicheiroInexistente) {
		t.Fatalf("err = %v; esperado embrulhar ErrFicheiroInexistente", err)
	}
	// E nao pode ter convertido o desaparecimento num veredito de adulteracao.
	if errors.Is(err, ErrSemReferencia) {
		t.Fatal("erro de infraestrutura convertido em erro de referencia")
	}
}

// O artefacto REAL da E-1251, quando presente, e' verificado ponta a ponta.
func TestVerificarConstrucaoComArtefactoReal(t *testing.T) {
	caminho := filepath.Join("..", "..", "_tmp", "c1-artefacto", "hades-windows-amd64.exe")
	if _, err := os.Stat(caminho); err != nil {
		t.Skipf("artefacto real da E-1251 ausente neste checkout: %v", err)
	}
	absoluto, err := filepath.Abs(caminho)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	medido, tamanho, err := MedirArtefacto(context.Background(), absoluto)
	if err != nil {
		t.Fatalf("medir: %v", err)
	}
	t.Logf("artefacto real: %d bytes, sha256=%s", tamanho, medido)

	artefato := Artefato{Caminho: absoluto, Nome: "hades-windows-amd64.exe", TamanhoBytes: tamanho, SHA256: medido, Alvo: "windows"}
	v, err := VerificarConstrucao(
		context.Background(),
		ResultadoDeBuild{Artefatos: []Artefato{artefato}},
		VerificadorDeArtefacto{},
	)
	if err != nil {
		t.Fatalf("verificar: %v", err)
	}
	if !v.Aprovada {
		t.Fatalf("o artefacto real nao passou: %+v", v)
	}
	// A referencia errada tem de reprovar o MESMO ficheiro.
	//
	// NOTA: `artefato` (sem "c") e' a variavel local. `artefacto` (com "c") e' uma FUNCAO deste
	// pacote -- escrever o nome errado resolve silenciosamente para ela e o compilador acusa
	// "has no field or method SHA256". *Um identificador local que coincide com um do pacote nao
	// sombreia: se nao existir, o do pacote e' que responde.*
	comReferenciaErrada := artefato
	comReferenciaErrada.SHA256 = strings.Repeat("0", 64)
	v2, err := VerificarConstrucao(
		context.Background(),
		ResultadoDeBuild{Artefatos: []Artefato{comReferenciaErrada}},
		VerificadorDeArtefacto{},
	)
	if err != nil {
		t.Fatalf("verificar com referencia errada: %v", err)
	}
	if v2.Aprovada {
		t.Fatal("aprovada = true com uma referencia errada")
	}
}
