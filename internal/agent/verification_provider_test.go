package agent

// FASE 28.1 item 5: testes do `VerificationProvider`.
//
// O que estes testes travam:
//
//  1. sem checksum de referencia NAO se verifica -- devolve `ErrSemReferencia`;
//  2. referencia curta (prefixo) nao serve -- um prefixo "passaria" com outro ficheiro;
//  3. ficheiro inexistente e' recusado com erro tipado;
//  4. directorio e' recusado;
//  5. ficheiro de 0 bytes e' recusado;
//  6. checksum correcto -> Integro, Aprovado;
//  7. UM BYTE diferente -> Adulterado, NAO aprovado (o teste que importa);
//  8. sem adaptador de assinatura -> NAO_VERIFICADA, nunca AUSENTE;
//  9. adaptador com erro -> NAO_VERIFICADA, nunca AUSENTE;
// 10. adaptador que diz AUSENTE -> Aprovado (a RC e' nao assinada de proposito);
// 11. adaptador que diz INVALIDA -> NAO aprovado;
// 12. o mesmo ficheiro verificado duas vezes da' o mesmo checksum.

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

// ficheiroComConteudo escreve `conteudo` e devolve o caminho e o SHA-256.
func ficheiroComConteudo(t *testing.T, conteudo string) (string, string) {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "artefacto.bin")
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	soma := sha256.Sum256([]byte(conteudo))
	return caminho, hex.EncodeToString(soma[:])
}

func TestVerificadorRecusaSemReferencia(t *testing.T) {
	caminho, _ := ficheiroComConteudo(t, "conteudo")
	for _, vazio := range []string{"", "   ", "\t\n"} {
		_, err := VerificadorDeArtefacto{}.Verificar(context.Background(), caminho, vazio)
		if err == nil {
			t.Fatalf("checksum %q devia ser recusado", vazio)
		}
		if !errors.Is(err, ErrSemReferencia) {
			t.Fatalf("erro = %v; esperado ErrSemReferencia", err)
		}
	}
}

// Um PREFIXO de checksum nao e' uma referencia.
func TestVerificadorRecusaReferenciaCurta(t *testing.T) {
	caminho, soma := ficheiroComConteudo(t, "conteudo")
	// Os primeiros 16 caracteres do checksum REAL: o tentador e' aceitar como "quase certo".
	_, err := VerificadorDeArtefacto{}.Verificar(context.Background(), caminho, soma[:16])
	if err == nil {
		t.Fatal("prefixo de 16 caracteres devia ser recusado")
	}
	if !errors.Is(err, ErrSemReferencia) {
		t.Fatalf("erro = %v; esperado ErrSemReferencia", err)
	}
	// E uma string de 64 caracteres que nao e' hexadecimal tambem nao serve.
	naoHex := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	if _, err := (VerificadorDeArtefacto{}).Verificar(context.Background(), caminho, naoHex); err == nil {
		t.Fatal("referencia nao-hexadecimal devia ser recusada")
	}
}

func TestVerificadorRecusaFicheiroInexistente(t *testing.T) {
	_, soma := ficheiroComConteudo(t, "x")
	_, err := VerificadorDeArtefacto{}.Verificar(context.Background(),
		filepath.Join(t.TempDir(), "nao-existe.bin"), soma)
	if !errors.Is(err, ErrFicheiroInexistente) {
		t.Fatalf("erro = %v; esperado ErrFicheiroInexistente", err)
	}
}

func TestVerificadorRecusaDirectorio(t *testing.T) {
	_, soma := ficheiroComConteudo(t, "x")
	_, err := VerificadorDeArtefacto{}.Verificar(context.Background(), t.TempDir(), soma)
	if err == nil {
		t.Fatal("directorio devia ser recusado")
	}
}

// *Um ficheiro de zero bytes tem um SHA-256 valido e nao serve para nada.*
func TestVerificadorRecusaFicheiroVazio(t *testing.T) {
	caminho, soma := ficheiroComConteudo(t, "")
	_, err := VerificadorDeArtefacto{}.Verificar(context.Background(), caminho, soma)
	if !errors.Is(err, ErrFicheiroInexistente) {
		t.Fatalf("erro = %v; ficheiro vazio devia ser recusado", err)
	}
}

func TestVerificadorAprovaIntegro(t *testing.T) {
	caminho, soma := ficheiroComConteudo(t, "artefacto-real")
	r, err := VerificadorDeArtefacto{}.Verificar(context.Background(), caminho, soma)
	if err != nil {
		t.Fatalf("verificar: %v", err)
	}
	if !r.Integro || r.Adulterado {
		t.Fatalf("Integro=%v Adulterado=%v", r.Integro, r.Adulterado)
	}
	if r.SHA256Medido != soma || r.SHA256Esperado != soma {
		t.Fatalf("checksums: medido=%s esperado=%s", r.SHA256Medido, r.SHA256Esperado)
	}
	if !r.Aprovado {
		t.Fatal("integro com assinatura NAO_VERIFICADA devia ser aprovado")
	}
	if r.Assinatura != AssinaturaNaoVerificada {
		t.Fatalf("assinatura = %q; sem adaptador tem de ser NAO_VERIFICADA", r.Assinatura)
	}
	if r.TamanhoBytes != int64(len("artefacto-real")) {
		t.Fatalf("tamanho = %d", r.TamanhoBytes)
	}
}

// O TESTE QUE IMPORTA: um byte diferente reprova.
//
//	*Uma verificacao que so' e' exercitada com o ficheiro intacto prova que somar duas vezes o mesmo
//	ficheiro da' o mesmo numero.*
func TestVerificadorDetectaAdulteracao(t *testing.T) {
	caminho, somaOriginal := ficheiroComConteudo(t, "artefacto-original")

	// Verifica que o original passa.
	if r, err := (VerificadorDeArtefacto{}).Verificar(context.Background(), caminho, somaOriginal); err != nil {
		t.Fatalf("original: %v", err)
	} else if !r.Aprovado {
		t.Fatal("original devia ser aprovado")
	}

	// Adultera UM byte.
	conteudo, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	conteudo[0] ^= 0xFF
	if err := os.WriteFile(caminho, conteudo, 0o600); err != nil {
		t.Fatalf("adulterar: %v", err)
	}

	r, err := VerificadorDeArtefacto{}.Verificar(context.Background(), caminho, somaOriginal)
	if err != nil {
		t.Fatalf("verificar adulterado: %v", err)
	}
	if !r.Adulterado {
		t.Fatal("ADULTERACAO NAO DETECTADA: o checksum devia diferir")
	}
	if r.Integro {
		t.Fatal("Integro=true num ficheiro adulterado")
	}
	if r.Aprovado {
		t.Fatal("Aprovado=true num ficheiro adulterado")
	}
	if r.Explicacao == "" || !strings.Contains(r.Explicacao, "ADULTERADO") {
		t.Fatalf("explicacao = %q", r.Explicacao)
	}
}

// Os TRES estados de assinatura, provados com adaptadores injectados.
func TestVerificadorEstadosDeAssinatura(t *testing.T) {
	casos := []struct {
		nome         string
		estado       EstadoDeAssinatura
		erro         error
		querAprovado bool
	}{
		{"ausente", AssinaturaAusente, nil, true},   // a RC e' nao assinada de proposito
		{"presente", AssinaturaPresente, nil, true}, // assinado e valido
		{"invalida", AssinaturaInvalida, nil, false},
		// O adaptador FALHOU: a integridade foi verificada, a assinatura nao foi olhada.
		//
		//	*Aprovado significa "passou os gates que consegui aplicar", nao "todos os gates
		//	existiram". Reprovar aqui tornaria o verificador inutil fora do Windows -- onde nao ha'
		//	Authenticode nenhum -- e reprovar por ausencia de instrumento e' punir o auditor pelo
		//	instrumento que falta.*
		//
		// O chamador que EXIGE assinatura le' `Assinatura` e decide; o campo diz
		// `NAO_VERIFICADA`, nao `AUSENTE`.
		{"adaptador falhou", "", errors.New("powershell ausente"), true},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			caminho, soma := ficheiroComConteudo(t, "artefacto-"+c.nome)
			adaptador := func(context.Context, string) (EstadoDeAssinatura, string, error) {
				return c.estado, "detalhe do teste", c.erro
			}
			r, err := VerificadorDeArtefacto{Assinatura: adaptador}.Verificar(
				context.Background(), caminho, soma)
			if err != nil {
				t.Fatalf("verificar: %v", err)
			}
			if !r.Integro {
				t.Fatal("o ficheiro nao foi tocado: Integro tinha de ser true")
			}
			// O estado esperado: um adaptador com erro NUNCA produz AUSENTE.
			if c.erro != nil {
				if r.Assinatura != AssinaturaNaoVerificada {
					t.Fatalf("adaptador com erro deu %q; tinha de ser NAO_VERIFICADA", r.Assinatura)
				}
			}
			if r.Aprovado != c.querAprovado {
				t.Fatalf("Aprovado=%v; esperado %v (estado %q)", r.Aprovado, c.querAprovado, r.Assinatura)
			}
		})
	}
}

// Um estado que o adaptador nao reconheca NAO se torna AUSENTE nem PRESENTE.
func TestVerificadorAdaptadorNaoConfigured(t *testing.T) {
	caminho, soma := ficheiroComConteudo(t, "sem-adaptador")
	r, err := VerificadorDeArtefacto{}.Verificar(context.Background(), caminho, soma)
	if err != nil {
		t.Fatalf("verificar: %v", err)
	}
	if r.Assinatura != AssinaturaNaoVerificada {
		t.Fatalf("assinatura = %q", r.Assinatura)
	}
	if r.DetalheDaAssinatura == "" {
		t.Fatal("sem detalhe: quem le' nao sabe porque nao foi verificado")
	}
}

// Duas verificacoes do mesmo ficheiro dao' o mesmo checksum e o mesmo veredito.
func TestVerificadorEDeterministico(t *testing.T) {
	caminho, soma := ficheiroComConteudo(t, "deterministico")
	var primeiro ResultadoDeVerificacao
	for i := range 3 {
		r, err := VerificadorDeArtefacto{}.Verificar(context.Background(), caminho, soma)
		if err != nil {
			t.Fatalf("verificar %d: %v", i, err)
		}
		if i == 0 {
			primeiro = r
			continue
		}
		if r.SHA256Medido != primeiro.SHA256Medido {
			t.Fatalf("checksum mudou entre chamadas: %s vs %s", primeiro.SHA256Medido, r.SHA256Medido)
		}
		if r.Aprovado != primeiro.Aprovado {
			t.Fatal("veredito mudou entre chamadas")
		}
	}
}

// contemTexto NAO se define aqui: ja' existe em
// `internal/agent/service_agent_audit_diag_helpers_test.go:67`, e a primeira versao deste ficheiro
// declarava-o outra vez -- o pacote nao compilava.
//
//	*Dois testes que declaram o mesmo helper nao sao dois testes: sao um pacote que nao compila.*
//
// Usa-se `strings.Contains` directamente.
