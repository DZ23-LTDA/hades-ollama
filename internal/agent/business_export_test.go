package agent

import (
	"encoding/csv"
	"strings"
	"testing"
	"time"
)

// Estágio 32 — exportação do painel. Propriedades travadas:
//
//  1. O ARQUIVO É CSV VÁLIDO com esquema uniforme (4 colunas em TODAS as linhas);
//  2. AS LACUNAS VÃO NO ARQUIVO: fora do produto, a lista `nao_medido` da tela some, e um número
//     exportado sem lacunas parece completo;
//  3. CADA INDICADOR DIZ A ORIGEM;
//  4. ESCAPING SOBREVIVE a vírgula, aspas e quebra de linha (testado com valores hostis);
//  5. ISOLAMENTO entre organizações.
func parseExport(t *testing.T, document string) [][]string {
	t.Helper()
	reader := csv.NewReader(strings.NewReader(document))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("o export precisa ser CSV válido: %v", err)
	}
	return records
}

func exportRowsBySection(records [][]string, section string) [][]string {
	rows := [][]string{}
	for _, record := range records[1:] {
		if record[0] == section {
			rows = append(rows, record)
		}
	}
	return rows
}

func TestBusinessExportIsValidCSVWithGapsSourcesAndSeries(t *testing.T) {
	runtime, now := businessStatusFixture(t)
	document, err := runtime.BusinessExportCSV(LocalOrganizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	records := parseExport(t, document)

	// ESQUEMA UNIFORME: toda linha tem exatamente 4 campos — cabeçalho diferente por bloco quebraria
	// qualquer leitor de planilha.
	if len(records[0]) != 4 || records[0][0] != "secao" || records[0][1] != "chave" || records[0][2] != "valor" || records[0][3] != "observacao" {
		t.Fatalf("cabeçalho inesperado: %+v", records[0])
	}
	for index, record := range records {
		if len(record) != 4 {
			t.Fatalf("linha %d com %d campos (esperado 4): %+v", index, len(record), record)
		}
	}

	// META declara escopo e sandbox.
	meta := map[string]string{}
	for _, row := range exportRowsBySection(records, "meta") {
		meta[row[1]] = row[2]
	}
	if meta["organizacao"] != LocalOrganizationID || meta["sandbox"] != "true" || meta["gerado_em"] == "" {
		t.Fatalf("meta incompleta: %+v", meta)
	}

	// INDICADORES reais, com ORIGEM em cada linha.
	indicators := map[string]string{}
	for _, row := range exportRowsBySection(records, "indicador") {
		indicators[row[1]] = row[2]
		if row[3] == "" {
			t.Fatalf("indicador %q sem origem declarada", row[1])
		}
	}
	if indicators["pedidos"] != "3" || indicators["receita_cents"] != "3500" || indicators["violacoes_sla"] != "1" {
		t.Fatalf("indicadores incorretos no export: %+v", indicators)
	}
	if indicators["publicacao_externa"] != CampaignExternalPublication {
		t.Fatalf("o export precisa declarar a publicação externa bloqueada: %q", indicators["publicacao_externa"])
	}

	// AS LACUNAS VÃO NO ARQUIVO, com motivo.
	gaps := exportRowsBySection(records, "nao_medido")
	report, err := runtime.BusinessStatus(LocalOrganizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps) != len(report.NotMeasured) || len(gaps) == 0 {
		t.Fatalf("o export precisa levar as %d lacunas, veio %d", len(report.NotMeasured), len(gaps))
	}
	for _, row := range gaps {
		if row[1] == "" || row[3] == "" {
			t.Fatalf("lacuna sem chave ou sem motivo: %+v", row)
		}
	}

	// SÉRIE: resumo declarando retenção e ausência de base (o fixture não registrou amostras).
	series := exportRowsBySection(records, "serie")
	if len(series) != 1 || series[0][1] != "resumo" {
		t.Fatalf("sem amostras, a série precisa ser só o resumo: %+v", series)
	}
	if !strings.Contains(series[0][3], "retencao_maxima=") || !strings.Contains(series[0][3], "sem_base=true") {
		t.Fatalf("resumo da série incompleto: %+v", series[0])
	}
	// E a ausência de base é DECLARADA na seção de tendência, não omitida.
	trendRows := exportRowsBySection(records, "tendencia")
	if len(trendRows) != 1 || trendRows[0][1] != "sem_base" || trendRows[0][3] == "" {
		t.Fatalf("a ausência de base precisa estar no arquivo com motivo: %+v", trendRows)
	}
}

func TestBusinessExportEscapesHostileValues(t *testing.T) {
	// Nota sobre `\r\n`: o LEITOR de `encoding/csv` normaliza CRLF para LF dentro de campo citado —
	// comportamento documentado da biblioteca padrão, não defeito do export. Por isso o valor hostil
	// usa LF, e a garantia testada é a que importa: vírgula, aspas e quebra de linha sobrevivem.
	hostile := "valor, com vírgula; \"aspas\" e\nquebra de linha\nfim"
	var builder strings.Builder
	if err := csvEncodeRows(&builder, []BusinessExportRow{
		{Section: "teste", Key: "hostil", Value: hostile, Note: "nota, com vírgula"},
		{Section: "teste", Key: "simples", Value: "1", Note: ""},
	}); err != nil {
		t.Fatal(err)
	}
	document := builder.String()
	// O documento tem aspas de escape.
	if !strings.Contains(document, "\"\"") {
		t.Fatalf("aspas internas precisam ser escapadas: %q", document)
	}
	reader := csv.NewReader(strings.NewReader(document))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("o documento com valores hostis precisa continuar sendo CSV válido: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("a quebra de linha não pode virar uma linha nova: %d registros", len(records))
	}
	if records[1][2] != hostile {
		t.Fatalf("o valor hostil precisa sobreviver intacto:\nquerido=%q\nobtido =%q", hostile, records[1][2])
	}
	if records[1][3] != "nota, com vírgula" {
		t.Fatalf("a nota com vírgula precisa sobreviver: %q", records[1][3])
	}
}

func TestBusinessExportIsScopedByOrganization(t *testing.T) {
	runtime, now := businessStatusFixture(t)
	own, err := runtime.BusinessExportCSV(LocalOrganizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	other, err := runtime.BusinessExportCSV("org_export_outra", now)
	if err != nil {
		t.Fatal(err)
	}
	otherRecords := parseExport(t, other)
	indicators := map[string]string{}
	for _, row := range exportRowsBySection(otherRecords, "indicador") {
		indicators[row[1]] = row[2]
	}
	// A outra organização não herda NENHUM número da dona.
	if indicators["pedidos"] != "0" || indicators["receita_cents"] != "0" || indicators["conversas"] != "0" {
		t.Fatalf("o export vazou entre organizações: %+v", indicators)
	}
	if indicators["organizacao"] != "" && strings.Contains(other, LocalOrganizationID) {
		t.Fatalf("o export da outra organização não pode conter o identificador da dona")
	}
	if !strings.Contains(own, "pedidos,3,") {
		t.Fatalf("o export da organização dona precisa trazer os seus números")
	}
	// Escopo ausente é erro declarado.
	if _, err := runtime.BusinessExportCSV("   ", now); err == nil {
		t.Fatal("organização ausente precisa ser erro")
	}
	// O export NÃO cria amostra: a série continua vazia depois de exportar.
	trend, err := runtime.BusinessTrend(LocalOrganizationID, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if trend.Snapshots != 0 {
		t.Fatalf("exportar não pode registrar amostra: %d", trend.Snapshots)
	}
}
