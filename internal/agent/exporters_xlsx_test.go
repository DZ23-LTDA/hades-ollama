package agent

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// XLSX REAL NO EXPORT PROFISSIONAL — TESTES (E-366).
//
// **MEDIDO antes**: `ExportProfessional` aceitava pdf/docx/pptx e **recusava** qualquer outro formato — o requisito de
// **planilha** estava desatendido: não havia caminho nenhum para gerar uma planilha.
//
// O teste **não** se contenta com "o arquivo abriu": verifica as **partes OOXML obrigatórias** e o **conteúdo** dentro
// da worksheet. *Arquivo que abre por acidente não é arquivo válido.* A compatibilidade real foi medida à parte, com
// **openpyxl** (biblioteca de terceiro) — prova de terceiro, não auto-atestado.

func projetoComConteudo(t *testing.T, corpo string) (*BuilderService, BuilderProject) {
	t.Helper()
	servico, err := NewBuilderService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	projeto, err := servico.Create(context.Background(), BuilderSpec{
		Name: "Relatório de Missão", Kind: BuilderGame,
		Files: map[string]string{"index.html": corpo},
	})
	if err != nil {
		t.Fatal(err)
	}
	return servico, projeto
}

func TestExportProfessionalGeraXLSXComAsPartesObrigatorias(t *testing.T) {
	servico, projeto := projetoComConteudo(t, "# Relatório\n\n| SKU | Quantidade |\n|---|---|\n| CAM-001 | 12 |\n| CAN-002 | 7 |\n")
	_, caminho, err := servico.ExportProfessional(context.Background(), projeto.ID, "xlsx")
	if err != nil {
		t.Fatalf("exportar planilha: %v", err)
	}
	if filepath.Ext(caminho) != ".xlsx" {
		t.Fatalf("caminho do artefato: %s", caminho)
	}
	leitor, err := zip.OpenReader(caminho)
	if err != nil {
		t.Fatalf("o artefato precisa ser um ZIP/OOXML legível: %v", err)
	}
	defer leitor.Close()

	partes := map[string]string{}
	for _, arquivo := range leitor.File {
		conteudo, err := arquivo.Open()
		if err != nil {
			t.Fatal(err)
		}
		dados, err := io.ReadAll(conteudo)
		conteudo.Close()
		if err != nil {
			t.Fatal(err)
		}
		partes[arquivo.Name] = string(dados)
	}
	// ARMADILHA: as cinco partes mínimas de um XLSX que um leitor de verdade aceita.
	for _, obrigatoria := range []string{
		"[Content_Types].xml", "_rels/.rels", "xl/workbook.xml",
		"xl/_rels/workbook.xml.rels", "xl/worksheets/sheet1.xml",
	} {
		if _, existe := partes[obrigatoria]; !existe {
			t.Fatalf("ARMADILHA: falta a parte OOXML obrigatória %q", obrigatoria)
		}
	}
	planilha := partes["xl/worksheets/sheet1.xml"]
	for _, esperado := range []string{"Relatório", "SKU", "Quantidade", "CAM-001", "CAN-002"} {
		if !strings.Contains(planilha, esperado) {
			t.Fatalf("ARMADILHA: o conteúdo %q precisa estar na planilha:\n%s", esperado, planilha)
		}
	}
	// A linha separadora do markdown (`|---|---|`) não é dado e não pode virar célula.
	if strings.Contains(planilha, "---") {
		t.Fatalf("a linha separadora de tabela virou dado:\n%s", planilha)
	}
	// E o tipo da worksheet precisa estar declarado no pacote.
	if !strings.Contains(partes["[Content_Types].xml"], "spreadsheetml.worksheet") {
		t.Fatal("o tipo da worksheet precisa estar declarado em [Content_Types].xml")
	}
}

// ARMADILHA MEDIDA POR TERCEIRO (E-366): o `#` do markdown chegou à célula quando o arquivo foi aberto com openpyxl.
func TestXLSXNaoLevaMarcadorDeTituloMarkdown(t *testing.T) {
	servico, projeto := projetoComConteudo(t, "# Relatório de Missão\n\n| SKU | Qtd |\n| A | 1 |\n")
	_, caminho, err := servico.ExportProfessional(context.Background(), projeto.ID, "xlsx")
	if err != nil {
		t.Fatal(err)
	}
	leitor, err := zip.OpenReader(caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer leitor.Close()
	var planilha string
	for _, arquivo := range leitor.File {
		if arquivo.Name == "xl/worksheets/sheet1.xml" {
			conteudo, _ := arquivo.Open()
			dados, _ := io.ReadAll(conteudo)
			conteudo.Close()
			planilha = string(dados)
		}
	}
	if strings.Contains(planilha, "# Relatório") {
		t.Fatalf("ARMADILHA: o marcador de título do markdown virou dado na planilha:\n%s", planilha)
	}
	if !strings.Contains(planilha, "Relatório de Missão") {
		t.Fatalf("o título precisa permanecer, sem o marcador:\n%s", planilha)
	}
}

func TestXLSXEscapaConteudoHostil(t *testing.T) {
	servico, projeto := projetoComConteudo(t, "| Produto | Observação |\n| A <b>&</b> | \"aspas\" e 'apóstrofo' |\n")
	_, caminho, err := servico.ExportProfessional(context.Background(), projeto.ID, "xlsx")
	if err != nil {
		t.Fatal(err)
	}
	leitor, err := zip.OpenReader(caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer leitor.Close()
	var planilha string
	for _, arquivo := range leitor.File {
		if arquivo.Name == "xl/worksheets/sheet1.xml" {
			conteudo, _ := arquivo.Open()
			dados, _ := io.ReadAll(conteudo)
			conteudo.Close()
			planilha = string(dados)
		}
	}
	// ARMADILHA: XML quebrado por conteúdo do usuário é o defeito clássico de exportador.
	if strings.Contains(planilha, "<b>") {
		t.Fatalf("conteúdo hostil NÃO foi escapado — o XML está quebrado:\n%s", planilha)
	}
	if !strings.Contains(planilha, "&lt;b&gt;") {
		t.Fatalf("o escape precisa aparecer como entidade:\n%s", planilha)
	}
}

func TestLinhasDaPlanilhaNaoInventaDados(t *testing.T) {
	linhas := linhasDaPlanilha("Titulo\n\n| a | b |\n|---|---|\n| 1 | 2 |\n")
	if len(linhas) != 3 {
		t.Fatalf("esperava 3 linhas de dado (título e duas da tabela), veio %d: %+v", len(linhas), linhas)
	}
	if len(linhas[1]) != 2 || linhas[1][0] != "a" || linhas[1][1] != "b" {
		t.Fatalf("a linha de tabela precisa virar duas células: %+v", linhas[1])
	}
	// Texto vazio não gera planilha vazia nem dado inventado.
	if vazio := linhasDaPlanilha("   \n\n"); len(vazio) != 1 || vazio[0][0] != "" {
		t.Fatalf("texto vazio precisa virar uma célula vazia: %+v", vazio)
	}
}

func TestExportProfessionalSegueRecusandoFormatoDesconhecido(t *testing.T) {
	servico, projeto := projetoComConteudo(t, "conteudo")
	if _, _, err := servico.ExportProfessional(context.Background(), projeto.ID, "exe"); err == nil {
		t.Fatal("ARMADILHA: formato desconhecido foi aceito")
	}
	if _, _, err := servico.ExportProfessional(context.Background(), projeto.ID, ".xlsx"); err != nil {
		t.Fatalf("a extensão com ponto precisa continuar aceita para xlsx: %v", err)
	}
	// E o arquivo do formato aceito existe de verdade.
	caminho := filepath.Join(servico.root, projeto.ID+".xlsx")
	if _, err := os.Stat(caminho); err != nil {
		t.Fatalf("o artefato precisa existir no disco: %v", err)
	}
}
