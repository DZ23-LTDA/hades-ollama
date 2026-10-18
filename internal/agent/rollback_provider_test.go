package agent

// Testes do `RollbackProvider` (E-1268) -- e com ele fecham os DEZ providers do adendo 28.1.
//
// O que estes testes travam:
//
//  1. um directorio com instaladores produz um plano com candidatos, ordenados do mais recente;
//  2. o plano **nunca** tem `Executado: true` -- restaurar exige aprovacao humana;
//  3. cada candidato tem SHA-256 medido (um candidato sem checksum nao e' verificavel);
//  4. um directorio vazio devolve `ErrSemCandidatos`;
//  5. canais sem credencial (`GOOGLE_PLAY`, `APP_STORE`, `EAS_UPDATE`, `OCI`) devolvem
//     `ErrRollbackNaoSuportado` com o motivo;
//  6. um tipo desconhecido nao e' declarado suportado nem recusado por omissao;
//  7. ficheiros que nao parecem instaladores sao ignorados.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// criarDistDeTeste monta um `dist/` com instaladores de versoes diferentes.
func criarDistDeTeste(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	// Tres instaladores com conteudos e datas diferentes.
	for indice, nome := range []string{"HadesSetup-aaa.exe", "HadesSetup-bbb.exe", "HadesSetup-ccc.exe"} {
		caminho := filepath.Join(raiz, nome)
		if err := os.WriteFile(caminho, []byte("instalador "+nome), 0o600); err != nil {
			t.Fatalf("escrever %s: %v", nome, err)
		}
		// Datas distintas: `aaa` mais antigo, `ccc` mais recente.
		instante := time.Now().Add(time.Duration(indice-3) * time.Hour)
		if err := os.Chtimes(caminho, instante, instante); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
	}
	// Um ficheiro que NAO e' instalador: tem de ser ignorado.
	if err := os.WriteFile(filepath.Join(raiz, "leiame.txt"), []byte("texto"), 0o600); err != nil {
		t.Fatalf("escrever leiame: %v", err)
	}
	return raiz
}

func TestRollbackProviderPlaneiaComCandidatos(t *testing.T) {
	raiz := criarDistDeTeste(t)
	provider := ProviderDeRollback{}
	plano, err := provider.Planear(context.Background(), RollbackWindowsInstaller, raiz)
	if err != nil {
		t.Fatalf("planear: %v", err)
	}
	if !plano.Suportado {
		t.Fatal("suportado = false para WINDOWS_INSTALLER")
	}
	if plano.TotalDeCandidatos != 3 {
		t.Fatalf("candidatos = %d; esperado 3 (o leiame.txt nao conta)",
			plano.TotalDeCandidatos)
	}
	if plano.Alvo == nil {
		t.Fatal("alvo = nil com 3 candidatos")
	}
	// O mais recente e' o `ccc`.
	if plano.Alvo.Nome != "HadesSetup-ccc.exe" {
		t.Fatalf("alvo = %q; esperado o mais recente HadesSetup-ccc.exe", plano.Alvo.Nome)
	}
	// Ordem: mais recente primeiro.
	if plano.Candidatos[0].ModificadoEm.Before(plano.Candidatos[1].ModificadoEm) {
		t.Fatal("candidatos nao estao ordenados do mais recente para o mais antigo")
	}
}

// O PLANO NUNCA EXECUTA. Restaurar exige aprovacao humana no momento da accao.
func TestRollbackProviderNuncaExecuta(t *testing.T) {
	raiz := criarDistDeTeste(t)
	provider := ProviderDeRollback{}
	plano, err := provider.Planear(context.Background(), RollbackWindowsInstaller, raiz)
	if err != nil {
		t.Fatalf("planear: %v", err)
	}
	if plano.Executado {
		t.Fatal("executado = true: o provider nao pode restaurar sem aprovacao")
	}
	if plano.RequerAprovacao == "" {
		t.Fatal("requer_aprovacao vazio: quem le' nao sabe o que falta")
	}
	// E o aviso tem de dizer que nada foi movido.
	if !strings.Contains(plano.Aviso, "NAO executou") {
		t.Fatalf("aviso nao declara que nada foi executado: %q", plano.Aviso)
	}
	if !strings.Contains(plano.Aviso, "nenhum ficheiro foi movido") {
		t.Fatalf("aviso nao garante que nenhum ficheiro mudou: %q", plano.Aviso)
	}
}

// Cada candidato tem SHA-256 medido.
func TestRollbackProviderMedeChecksumDosCandidatos(t *testing.T) {
	raiz := criarDistDeTeste(t)
	provider := ProviderDeRollback{}
	plano, err := provider.Planear(context.Background(), RollbackWindowsInstaller, raiz)
	if err != nil {
		t.Fatalf("planear: %v", err)
	}
	for _, candidato := range plano.Candidatos {
		if len(candidato.SHA256) != 64 {
			t.Fatalf("%s: sha256 com %d caracteres", candidato.Nome, len(candidato.SHA256))
		}
		// E o checksum tem de ser o do CONTEUDO, nao inventado.
		conteudo, err := os.ReadFile(candidato.Caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", candidato.Nome, err)
		}
		soma := sha256.Sum256(conteudo)
		if hex.EncodeToString(soma[:]) != candidato.SHA256 {
			t.Fatalf("%s: sha256 nao corresponde ao conteudo", candidato.Nome)
		}
	}
	if !plano.Integro {
		t.Fatal("integro = false com todos os candidatos medidos")
	}
}

func TestRollbackProviderRecusaDirectorioVazio(t *testing.T) {
	provider := ProviderDeRollback{}
	_, err := provider.Planear(context.Background(), RollbackWindowsInstaller, t.TempDir())
	if !errors.Is(err, ErrSemCandidatos) {
		t.Fatalf("err = %v; esperado ErrSemCandidatos", err)
	}
}

// Canais sem credencial sao declarados, nao fingidos.
func TestRollbackProviderDeclaraCanaisNaoSuportados(t *testing.T) {
	provider := ProviderDeRollback{}
	raiz := criarDistDeTeste(t)
	for _, canal := range []TipoDeRollback{
		RollbackGooglePlay, RollbackAppStore, RollbackEASUpdate, RollbackOCI,
	} {
		plano, err := provider.Planear(context.Background(), canal, raiz)
		if !errors.Is(err, ErrRollbackNaoSuportado) {
			t.Fatalf("%s: err = %v; esperado ErrRollbackNaoSuportado", canal, err)
		}
		if plano.Suportado {
			t.Fatalf("%s: suportado = true sem credencial/host", canal)
		}
		if plano.Explicacao == "" {
			t.Fatalf("%s: explicacao vazia", canal)
		}
		// E o canal nao pode ter produzido candidatos: nao ha' o que planear.
		if plano.TotalDeCandidatos != 0 {
			t.Fatalf("%s: %d candidatos num canal nao suportado", canal, plano.TotalDeCandidatos)
		}
	}
}

func TestRollbackProviderTipoDesconhecido(t *testing.T) {
	provider := ProviderDeRollback{}
	plano, err := provider.Planear(context.Background(), TipoDeRollback("INVENTADO"), t.TempDir())
	if !errors.Is(err, ErrRollbackNaoSuportado) {
		t.Fatalf("err = %v; esperado ErrRollbackNaoSuportado", err)
	}
	if plano.Suportado {
		t.Fatal("tipo desconhecido declarado suportado")
	}
	// E nao afirma que nao serve -- diz que nao sabe.
	if !strings.Contains(plano.Explicacao, "nao afirma que serve nem que nao serve") {
		t.Fatalf("explicacao nao declara a ignorancia: %q", plano.Explicacao)
	}
}

// Ficheiros que nao parecem instaladores sao ignorados.
func TestRollbackProviderIgnoraNaoInstaladores(t *testing.T) {
	raiz := t.TempDir()
	for _, nome := range []string{"leiame.txt", "manifest.json", "script.ps1"} {
		if err := os.WriteFile(filepath.Join(raiz, nome), []byte("x"), 0o600); err != nil {
			t.Fatalf("escrever %s: %v", nome, err)
		}
	}
	provider := ProviderDeRollback{}
	_, err := provider.Planear(context.Background(), RollbackWindowsInstaller, raiz)
	// O directorio existe mas nao tem instaladores: e' "sem candidatos", nao um plano vazio aprovado.
	if !errors.Is(err, ErrSemCandidatos) {
		t.Fatalf("err = %v; esperado ErrSemCandidatos", err)
	}
}
