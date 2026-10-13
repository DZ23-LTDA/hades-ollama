package agent

import (
	"fmt"
	"html"
	"strings"
)

// XLSX REAL NO EXPORT PROFISSIONAL (E-366).
//
// **MEDIDO**: `ExportProfessional` aceitava **pdf, docx e pptx** e recusava o resto — o requisito de **planilha** do
// estágio de paridade (T1) estava **desatendido**: não havia caminho nenhum para gerar uma planilha.
//
// **O QUE É "REAL" AQUI**: um XLSX é um ZIP com as partes OOXML exigidas. Não basta zipar um arquivo com extensão
// `.xlsx` — *arquivo que abre por acidente não é arquivo válido*. Estas são as partes mínimas que um leitor de verdade
// (Excel, LibreOffice, openpyxl) exige:
//
//	[Content_Types].xml                — declara os tipos das partes
//	_rels/.rels                        — relaciona o pacote ao workbook
//	xl/workbook.xml                    — define a planilha e o seu nome
//	xl/_rels/workbook.xml.rels         — relaciona o workbook à worksheet
//	xl/worksheets/sheet1.xml           — os dados (aqui, inline strings: sem tabela de strings compartilhada)
//
// A VALIDAÇÃO NÃO É "ABRIU": o teste verifica a **estrutura** e o **conteúdo**, e a compatibilidade real foi medida
// abrindo o arquivo com **openpyxl** (biblioteca de terceiro) — *prova de terceiro, não auto-atestado*.

// linhasDaPlanilha transforma o texto do projeto em linhas de células.
//
// Uma linha com `|` vira **células** (tabela); sem `|`, a linha inteira é uma célula. Linhas vazias são descartadas.
// **Nada é inventado**: o que não está no texto não aparece na planilha.
func linhasDaPlanilha(texto string) [][]string {
	linhas := make([][]string, 0, 16)
	for _, bruta := range strings.Split(texto, "\n") {
		linha := strings.TrimSpace(bruta)
		if linha == "" {
			continue
		}
		// O marcador de título do markdown não é dado: MEDIDO com openpyxl (E-366), `# Relatório` chegava à célula
		// como texto. ***Validação de terceiro existe para achar o que o teste do autor não vê.***
		linha = strings.TrimSpace(strings.TrimLeft(linha, "#"))
		if linha == "" {
			continue
		}
		if strings.Contains(linha, "|") {
			celulas := make([]string, 0, 8)
			for _, celula := range strings.Split(linha, "|") {
				celulas = append(celulas, strings.TrimSpace(celula))
			}
			// As BORDAS de uma tabela markdown (`| a | b |`) produzem células vazias nas pontas: elas não são dado.
			// MEDIDO no teste: sem isto, `| a | b |` virava 4 células (["", "a", "b", ""]) em vez de 2.
			for len(celulas) > 0 && celulas[0] == "" {
				celulas = celulas[1:]
			}
			for len(celulas) > 0 && celulas[len(celulas)-1] == "" {
				celulas = celulas[:len(celulas)-1]
			}
			if len(celulas) == 0 {
				continue
			}
			// Linha separadora de tabela markdown (`|---|---|`) não carrega dado.
			semDado := true
			for _, celula := range celulas {
				if strings.Trim(celula, "-: ") != "" {
					semDado = false
					break
				}
			}
			if semDado {
				continue
			}
			linhas = append(linhas, celulas)
			continue
		}
		linhas = append(linhas, []string{linha})
	}
	if len(linhas) == 0 {
		linhas = append(linhas, []string{""})
	}
	return linhas
}

// colunaDeXLSX devolve a referência de coluna (A, B, …, Z, AA, …).
func colunaDeXLSX(indice int) string {
	referencia := ""
	for indice >= 0 {
		referencia = string(rune('A'+indice%26)) + referencia
		indice = indice/26 - 1
	}
	return referencia
}

// writeMinimalXLSX grava uma planilha de uma aba, com strings inline.
func writeMinimalXLSX(path, title string, linhas [][]string) error {
	titulo := html.EscapeString(strings.TrimSpace(title))
	if titulo == "" {
		titulo = "Planilha"
	}
	// O nome da aba tem limites do formato: até 31 caracteres e sem alguns caracteres reservados.
	nomeAba := titulo
	if len([]rune(nomeAba)) > 31 {
		nomeAba = string([]rune(nomeAba)[:31])
	}
	nomeAba = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`[]:*?/\`, r) {
			return '-'
		}
		return r
	}, nomeAba)

	var corpo strings.Builder
	corpo.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	corpo.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for indiceLinha, linha := range linhas {
		fmt.Fprintf(&corpo, `<row r="%d">`, indiceLinha+1)
		for indiceCelula, celula := range linha {
			if celula == "" {
				continue
			}
			fmt.Fprintf(&corpo, `<c r="%s%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`,
				colunaDeXLSX(indiceCelula), indiceLinha+1, html.EscapeString(celula))
		}
		corpo.WriteString(`</row>`)
	}
	corpo.WriteString(`</sheetData></worksheet>`)

	return writeZip(path, map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
			`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
			`</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
			`</Relationships>`,
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
			`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
			`<sheets><sheet name="` + nomeAba + `" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
			`</Relationships>`,
		"xl/worksheets/sheet1.xml": corpo.String(),
	})
}
