package multillm

// FASE 28.5 (continuacao): testes do carregamento do ficheiro de metadados.
//
// O teste central e' `TestFicheiroComCampoDesconhecidoERecusado`: se alguem escrever
// `"private_key": "..."` no ficheiro, o carregador TEM de recusar o ficheiro inteiro. Um
// carregador tolerante descartaria o campo em silencio -- deixaria o segredo no disco e daria a
// impressao de o ter recusado.
//
// *Um ficheiro de configuracao que aceita campos que o tipo nao conhece e' um canal para
// contrabandear o segredo pela porta de servico.*

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func escreverRegisto(t *testing.T, conteudo string) string {
	t.Helper()
	dir := t.TempDir()
	caminho := filepath.Join(dir, NomeDoFicheiroDeIdentidades)
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("escrever registo: %v", err)
	}
	return caminho
}

func documentoValido(t *testing.T) string {
	t.Helper()
	documento := documentoDeIdentidades{
		SchemaVersion: esquemaSuportado,
		Identidades:   []IdentidadeDeAssinatura{identidadeValida()},
	}
	bruto, err := json.Marshal(documento)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(bruto)
}

func TestCarregarRegistoValido(t *testing.T) {
	caminho := escreverRegisto(t, documentoValido(t))
	registo, err := CarregarIdentidades(caminho)
	if err != nil {
		t.Fatalf("registo valido recusado: %v", err)
	}
	if len(registo.Identidades()) != 1 {
		t.Fatalf("identidades = %d", len(registo.Identidades()))
	}
	if registo.Identidades()[0].Nome != "apple-distribution" {
		t.Fatalf("nome = %q", registo.Identidades()[0].Nome)
	}
}

// A DEFESA CENTRAL: campo desconhecido recusa o ficheiro INTEIRO.
//
// Nome do teste explicito de proposito -- se falhar, o leitor sabe exactamente o que se perdeu.
func TestFicheiroComCampoDesconhecidoERecusado(t *testing.T) {
	// O cenario real: alguem escreve o segredo no ficheiro de metadados.
	//
	// O VALOR e' deliberadamente inofensivo, e nao um blob com entropia de chave. O que este
	// teste prova e' que o CAMPO desconhecido e' recusado -- o `DisallowUnknownFields` --, e nao
	// que um valor especifico e' detectado. *Um fixture que parece uma chave real faz o secret
	// scan disparar sobre o proprio teste que prova que o segredo e' recusado, e o gate passa a
	// gritar contra quem o defende.*
	conteudo := `{
	  "schema_version": 1,
	  "identidades": [{
	    "nome": "tentativa-de-contrabando",
	    "provedor": "dpapi",
	    "referencia": "dpapi:apple/hades.p12",
	    "plataforma": "ios",
	    "ambiente": "producao",
	    "fingerprint_publico": "B0:AC:E7:7E:63:7B:67:90:59:76:08:71:94:16:1C:6A:A2:1A:EB:29",
	    "expira_em": "2027-10-17T00:00:00Z",
	    "runners_autorizados": ["eas-ios"],
	    "private_key": "valor-inofensivo-de-teste"
	  }]
	}`
	caminho := escreverRegisto(t, conteudo)
	registo, err := CarregarIdentidades(caminho)
	if err == nil {
		t.Fatal("ficheiro com campo `private_key` foi ACEITE -- o segredo passaria pela porta de servico")
	}
	if !strings.Contains(err.Error(), "desconhecidos") {
		t.Fatalf("erro nao identifica campos desconhecidos: %v", err)
	}
	// E o registo devolvido tem de estar VAZIO, nunca parcialmente preenchido.
	if n := len(registo.Identidades()); n != 0 {
		t.Fatalf("registo parcialmente carregado (%d identidades) apesar do erro: "+
			"*um registo parcial parece completo*", n)
	}
}

func TestSchemaVersionDesconhecidoFalhaFechado(t *testing.T) {
	conteudo := `{"schema_version": 99, "identidades": []}`
	if _, err := CarregarIdentidades(escreverRegisto(t, conteudo)); err == nil {
		t.Fatal("schema_version 99 aceite")
	}
}

func TestFicheiroAusenteNaoEVazio(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, "nao-existe.json")
	registo, err := CarregarIdentidades(caminho)
	if err == nil {
		t.Fatal("ausencia de ficheiro devolveu sucesso")
	}
	if len(registo.Identidades()) != 0 {
		t.Fatal("registo nao vazio")
	}
	if !strings.Contains(err.Error(), "nao configurado") {
		t.Fatalf("mensagem nao diz que nao esta' configurado: %v", err)
	}
}

func TestCaminhoVazioERro(t *testing.T) {
	if _, err := CarregarIdentidades(""); err == nil {
		t.Fatal("caminho vazio aceite")
	}
}

func TestDiretorioNoLugarDoFicheiroERecusado(t *testing.T) {
	dir := t.TempDir()
	if _, err := CarregarIdentidades(dir); err == nil {
		t.Fatal("diretorio aceite como ficheiro")
	}
}

func TestJSONInvalidoERecusado(t *testing.T) {
	if _, err := CarregarIdentidades(escreverRegisto(t, "{nao sou json")); err == nil {
		t.Fatal("JSON invalido aceite")
	}
}

// Uma identidade invalida nao pode deixar as outras carregadas: um registo parcial parece completo.
func TestIdentidadeInvalidaInvalidaORegistoInteiro(t *testing.T) {
	boa := identidadeValida()
	boa.Nome = "boa"
	mau := identidadeValida()
	mau.Nome = "mau"
	mau.Plataforma = "beos" // invalida

	documento := documentoDeIdentidades{SchemaVersion: esquemaSuportado, Identidades: []IdentidadeDeAssinatura{boa, mau}}
	bruto, err := json.Marshal(documento)
	if err != nil {
		t.Fatal(err)
	}
	registo, err := CarregarIdentidades(escreverRegisto(t, string(bruto)))
	if err == nil {
		t.Fatal("registo com identidade invalida aceite")
	}
	if n := len(registo.Identidades()); n != 0 {
		t.Fatalf("registo parcial com %d identidades; devia vir vazio", n)
	}
}

func TestFicheiroDemasiadoGrandeERecusado(t *testing.T) {
	// Enchimento acima do limite. O JSON e' valido na forma, mas o tamanho e' a primeira barreira.
	enorme := `{"schema_version":1,"identidades":[]}` + strings.Repeat(" ", maxIdentidadesBytes+1024)
	if _, err := CarregarIdentidades(escreverRegisto(t, enorme)); err == nil {
		t.Fatal("ficheiro acima do limite aceite")
	}
}

func TestDescreverEstadoDistingueOsTresCasos(t *testing.T) {
	// 1. nao configurado
	_, estado := DescreverEstadoDoRegisto(filepath.Join(t.TempDir(), "ausente.json"))
	if estado.Configurado || estado.Carregado {
		t.Fatalf("ausente: Configurado=%v Carregado=%v", estado.Configurado, estado.Carregado)
	}
	if estado.Mensagem == "" {
		t.Fatal("ausente sem mensagem")
	}

	// 2. carregado e vazio
	vazio := `{"schema_version":1,"identidades":[]}`
	_, estado = DescreverEstadoDoRegisto(escreverRegisto(t, vazio))
	if !estado.Configurado || !estado.Carregado {
		t.Fatalf("vazio: Configurado=%v Carregado=%v", estado.Configurado, estado.Carregado)
	}
	if !strings.Contains(estado.Mensagem, "vazio") {
		t.Fatalf("mensagem de registo vazio confusa: %q", estado.Mensagem)
	}

	// 3. carregado com uma identidade
	_, estado = DescreverEstadoDoRegisto(escreverRegisto(t, documentoValido(t)))
	if !estado.Carregado || estado.Total != 1 {
		t.Fatalf("carregado: Carregado=%v Total=%d", estado.Carregado, estado.Total)
	}
}

// O fingerprint na projecao e' truncado: a UI precisa de distinguir, nao de reconstruir a cadeia.
func TestProjetarIdentidadesTruncaOFingerprint(t *testing.T) {
	registo := NovoRegistoDeIdentidades()
	if err := registo.Adicionar(identidadeValida()); err != nil {
		t.Fatal(err)
	}
	projetadas := ProjetarIdentidades(registo, agora)
	if len(projetadas) != 1 {
		t.Fatalf("projetadas = %d", len(projetadas))
	}
	projetada := projetadas[0]
	if len(projetada.FingerprintFim) != 8 {
		t.Fatalf("fingerprint truncado = %q (len %d); esperado 8", projetada.FingerprintFim, len(projetada.FingerprintFim))
	}
	// O fim do fingerprint real (sha1 B0ACE77E...1AEB29) termina em 1AEB29; os ultimos 4 bytes
	// sao A21AEB29. O truncado mostra 8 caracteres hex = 4 bytes.
	if !strings.HasSuffix(projetada.FingerprintFim, "1AEB29") {
		t.Fatalf("truncado nao corresponde ao fim do fingerprint: %q", projetada.FingerprintFim)
	}
	// Producao exige aprovacao; a projecao tem de dizer isso.
	if !projetada.ExigeAprovacao {
		t.Fatal("identidade de producao projetada sem ExigeAprovacao")
	}
	if !projetada.UtilizavelAgora {
		t.Fatal("identidade valida com runner nao e' utilizavel")
	}
}

// Uma identidade expirada nao pode aparecer como utilizavel na projecao.
func TestProjetarIdentidadeExpiradaNaoEUtilizavel(t *testing.T) {
	registo := NovoRegistoDeIdentidades()
	i := identidadeValida()
	i.ExpiraEm = agora.AddDate(0, -1, 0)
	if err := registo.Adicionar(i); err != nil {
		t.Fatal(err)
	}
	projetada := ProjetarIdentidades(registo, agora)[0]
	if projetada.UtilizavelAgora {
		t.Fatal("expirada projetada como utilizavel")
	}
	if projetada.Estado != EstadoExpirada {
		t.Fatalf("estado = %q", projetada.Estado)
	}
}

// O estado e' calculado na leitura, nao gravado: uma chave que expirou ontem nao pode continuar
// a dizer "ativa" so' porque o ficheiro foi escrito antes.
func TestEstadoCalculadoNaLeituraNaoGravado(t *testing.T) {
	registo := NovoRegistoDeIdentidades()
	i := identidadeValida()
	i.ExpiraEm = agora.AddDate(0, 0, 5) // expira_breve neste instante
	if err := registo.Adicionar(i); err != nil {
		t.Fatal(err)
	}
	agoraProjetado := ProjetarIdentidades(registo, agora)[0]
	if agoraProjetado.Estado != EstadoExpiraBreve {
		t.Fatalf("estado = %q; esperado expira_breve", agoraProjetado.Estado)
	}
	depois := ProjetarIdentidades(registo, agora.AddDate(0, 0, 10))[0]
	if depois.Estado != EstadoExpirada {
		t.Fatalf("estado depois = %q; esperado expirada "+
			"(*um estado gravado envelhece em silencio*)", depois.Estado)
	}
}

var _ = time.Now
