package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// E-285 — CONTRATO DO ARTEFATO GERADO: OFFLINE E COM LIMITES DECLARADOS.
//
// O app gerado usa persistência em ARQUIVO, não SGBD relacional — e isso é DECISÃO MEDIDA, não descuido:
// `mattn/go-sqlite3` exige CGO e esta máquina NÃO tem compilador C (E-274); `pgx` exige PostgreSQL rodando.
// Um artefato entregue ao usuário não pode assumir nenhum dos dois.
//
// As duas travas: (1) o artefato NÃO pode ter dependência externa (senão deixa de compilar offline);
// (2) os limites PRECISAM estar declarados (limite não declarado é o que engana). O caminho negativo prova que
// cada trava ACUSA — uma verificação que só confirma não verifica nada.
func TestArtefatoGeradoEOfflineEDeclaraSeusLimites(t *testing.T) {
	ctx := context.Background()
	raiz := t.TempDir()
	servico, err := NewBuilderService(filepath.Join(raiz, "builder"))
	if err != nil {
		t.Fatal(err)
	}
	projeto, err := servico.Create(ctx, BuilderSpec{Name: "Estoque Offline", Kind: BuilderFullstack})
	if err != nil {
		t.Fatal(err)
	}
	_, publicado, err := servico.PublishLocal(ctx, projeto.ID)
	if err != nil {
		t.Fatal(err)
	}

	// 1. O artefato publicado está conforme.
	if problemas := GeneratedArtifactIssues(publicado); len(problemas) != 0 {
		t.Fatalf("artefato publicado precisa estar conforme: %+v", problemas)
	}

	// 2. Caminho NEGATIVO: dependência externa é ACUSADA (deixaria de compilar offline).
	comDependencia := filepath.Join(raiz, "com-dep")
	copiarArtefato(t, publicado, comDependencia)
	goMod, err := os.ReadFile(filepath.Join(comDependencia, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comDependencia, "go.mod"), []byte(string(goMod)+"\nrequire modernc.org/sqlite v1.0.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	problemas := GeneratedArtifactIssues(comDependencia)
	if len(problemas) != 1 || !strings.Contains(problemas[0], "offline") {
		t.Fatalf("dependência externa precisa ser acusada com o motivo (offline): %+v", problemas)
	}

	// 3. Caminho NEGATIVO: README sem a declaração de limites é ACUSADO.
	semDeclaracao := filepath.Join(raiz, "sem-declaracao")
	copiarArtefato(t, publicado, semDeclaracao)
	if err := os.WriteFile(filepath.Join(semDeclaracao, "README.md"), []byte("# App\n\nRode com go run main.go\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	problemas = GeneratedArtifactIssues(semDeclaracao)
	if len(problemas) < 2 {
		t.Fatalf("README sem limites declarados precisa ser acusado (arquivo E memória): %+v", problemas)
	}

	// 4. Diretório vazio não passa: ausência de artefato não é artefato conforme.
	vazio := filepath.Join(raiz, "vazio")
	if err := os.MkdirAll(vazio, 0o700); err != nil {
		t.Fatal(err)
	}
	if problemas := GeneratedArtifactIssues(vazio); len(problemas) == 0 {
		t.Fatal("diretório sem go.mod/README não pode ser declarado conforme")
	}
}

// copiarArtefato duplica o artefato publicado para os cenários negativos.
func copiarArtefato(t *testing.T, origem, destino string) {
	t.Helper()
	if err := os.MkdirAll(destino, 0o700); err != nil {
		t.Fatal(err)
	}
	entradas, err := os.ReadDir(origem)
	if err != nil {
		t.Fatal(err)
	}
	for _, entrada := range entradas {
		if entrada.IsDir() {
			continue
		}
		dados, err := os.ReadFile(filepath.Join(origem, entrada.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destino, entrada.Name()), dados, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
