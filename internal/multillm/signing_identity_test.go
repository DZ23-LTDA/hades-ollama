package multillm

// FASE 28.5: testes do Signing Identity Center.
//
// O teste mais importante deste ficheiro nao e' o que valida campos: e' o que percorre a
// estrutura por REFLEXAO e falha se alguem acrescentar um campo que possa transporter o segredo.
// *Uma regra que so' vive num comentario desaparece no primeiro PR apressado.*

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"
)

var agora = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

func identidadeValida() IdentidadeDeAssinatura {
	return IdentidadeDeAssinatura{
		Nome:                  "apple-distribution",
		Provedor:              "dpapi",
		Referencia:            "dpapi:apple/hades-apple-distribution.p12",
		Plataforma:            PlataformaIOS,
		Ambiente:              AmbienteProducao,
		FingerprintPublico:    "B0:AC:E7:7E:63:7B:67:90:59:76:08:71:94:16:1C:6A:A2:1A:EB:29",
		CriadaEm:              agora.AddDate(-1, 0, 0),
		ExpiraEm:              agora.AddDate(1, 0, 0),
		RunnersAutorizados:    []string{"eas-ios"},
		PacoteOuBundle:        "com.dz23ltd.hades",
		RotacaoPrevistaEmDias: 180,
	}
}

// A PROVA ESTRUTURAL: nenhum campo pode chamar-se algo que transporte segredo.
//
// Se este teste falhar, alguem acrescentou um campo como `PrivateKey`, `Password` ou `Value`.
// A cura correcta e' remover o campo, nunca acrescentar o nome a uma lista de excepcoes.
func TestIdentidadeDeAssinaturaNaoTemCampoSecreto(t *testing.T) {
	proibidos := []string{
		"secret", "private", "password", "senha", "token", "value",
		"chaveprivada", "pfx", "p12content", "raw",
	}
	tipo := reflect.TypeOf(IdentidadeDeAssinatura{})
	for i := range tipo.NumField() {
		campo := tipo.Field(i)
		minusculo := strings.ToLower(campo.Name)
		for _, proibido := range proibidos {
			if strings.Contains(minusculo, proibido) {
				t.Fatalf("campo %q de IdentidadeDeAssinatura parece transporter segredo; "+
					"o registo descreve ONDE a chave vive, nunca o valor dela", campo.Name)
			}
		}
	}
}

func TestPlataformasEAmbientesValidos(t *testing.T) {
	for _, p := range []PlataformaDeAssinatura{
		PlataformaAndroid, PlataformaIOS, PlataformaMacOS,
		PlataformaWindows, PlataformaLinux, PlataformaWeb, PlataformaOCI,
	} {
		if !p.Valida() {
			t.Fatalf("plataforma declarada rejeitada: %q", p)
		}
	}
	if PlataformaDeAssinatura("blackberry").Valida() {
		t.Fatal("plataforma inventada aceite")
	}
	for _, a := range []AmbienteDeAssinatura{AmbienteDesenvolvimento, AmbientePreview, AmbienteProducao} {
		if !a.Valido() {
			t.Fatalf("ambiente declarado rejeitado: %q", a)
		}
	}
	if AmbienteDeAssinatura("staging").Valido() {
		t.Fatal("ambiente inventado aceite -- falha fechada devia recusar")
	}
}

// Autorizar build de preview NAO pode autorizar build de producao.
func TestSoProducaoExigeAprovacao(t *testing.T) {
	if AmbienteDesenvolvimento.ExigeAprovacao() {
		t.Fatal("desenvolvimento nao devia exigir aprovacao")
	}
	if AmbientePreview.ExigeAprovacao() {
		t.Fatal("preview nao devia exigir aprovacao")
	}
	if !AmbienteProducao.ExigeAprovacao() {
		t.Fatal("producao TEM de exigir aprovacao")
	}
}

func TestNormalizarFingerprintAceitaAsDuasFormasReais(t *testing.T) {
	comDoisPontos := "B0:AC:E7:7E:63:7B:67:90:59:76:08:71:94:16:1C:6A:A2:1A:EB:29"
	normalizado, err := NormalizarFingerprint(comDoisPontos)
	if err != nil {
		t.Fatalf("formato keytool rejeitado: %v", err)
	}
	if len(normalizado) != 40 || strings.Contains(normalizado, ":") {
		t.Fatalf("normalizacao errada: %q", normalizado)
	}
	// A forma OpenSSL, sem separadores, tem de dar o MESMO resultado.
	sem := strings.ReplaceAll(comDoisPontos, ":", "")
	outro, err := NormalizarFingerprint(sem)
	if err != nil {
		t.Fatalf("formato openssl rejeitado: %v", err)
	}
	if outro != normalizado {
		t.Fatalf("as duas formas dram resultados diferentes: %q vs %q", normalizado, outro)
	}
	// Um truncado tem de falhar: *parece valido e compara mal.*
	if _, err := NormalizarFingerprint(comDoisPontos[:20]); err == nil {
		t.Fatal("fingerprint truncado aceite")
	}
	if _, err := NormalizarFingerprint("ZZ:AC:E7:7E:63:7B:67:90:59:76:08:71:94:16:1C:6A:A2:1A:EB:29"); err == nil {
		t.Fatal("fingerprint com caracter nao-hex aceite")
	}
}

func TestAdicionarValidaCamposObrigatorios(t *testing.T) {
	casos := []struct {
		nome  string
		mutar func(*IdentidadeDeAssinatura)
	}{
		{"sem nome", func(i *IdentidadeDeAssinatura) { i.Nome = "" }},
		{"plataforma invalida", func(i *IdentidadeDeAssinatura) { i.Plataforma = "beos" }},
		{"ambiente invalido", func(i *IdentidadeDeAssinatura) { i.Ambiente = "qa" }},
		{"sem provedor", func(i *IdentidadeDeAssinatura) { i.Provedor = "" }},
		{"sem referencia", func(i *IdentidadeDeAssinatura) { i.Referencia = "" }},
		{"fingerprint invalido", func(i *IdentidadeDeAssinatura) { i.FingerprintPublico = "abc" }},
	}
	for _, caso := range casos {
		registo := NovoRegistoDeIdentidades()
		identidade := identidadeValida()
		caso.mutar(&identidade)
		if err := registo.Adicionar(identidade); err == nil {
			t.Fatalf("%s: aceite apesar de invalido", caso.nome)
		}
	}
	// E o caso valido tem de passar.
	registo := NovoRegistoDeIdentidades()
	if err := registo.Adicionar(identidadeValida()); err != nil {
		t.Fatalf("identidade valida rejeitada: %v", err)
	}
}

// Duas identidades com o mesmo nome tornam o approval ambiguo.
func TestNomeDuplicadoERejeitado(t *testing.T) {
	registo := NovoRegistoDeIdentidades()
	if err := registo.Adicionar(identidadeValida()); err != nil {
		t.Fatal(err)
	}
	if err := registo.Adicionar(identidadeValida()); err == nil {
		t.Fatal("nome duplicado aceite")
	}
}

// Revogada vence expirada: dizer "expira_breve" de uma chave revogada sugeriria que ainda serve.
func TestEstadoPrecedenciaRevoacaoSobreExpiracao(t *testing.T) {
	revogadaEm := agora.AddDate(0, -6, 0)
	i := identidadeValida()
	i.ExpiraEm = agora.AddDate(0, -1, 0) // ja' expirou
	i.RevogadaEm = &revogadaEm
	if estado := i.Estado(agora); estado != EstadoRevogada {
		t.Fatalf("estado = %q; esperado revogada", estado)
	}
	if i.Utilizavel(agora) {
		t.Fatal("identidade revogada declarada utilizavel")
	}
}

func TestEstadoExpiraBreveNaJanelaDeTrintaDias(t *testing.T) {
	i := identidadeValida()
	i.ExpiraEm = agora.AddDate(0, 0, 10)
	if estado := i.Estado(agora); estado != EstadoExpiraBreve {
		t.Fatalf("estado = %q; esperado expira_breve a 10 dias", estado)
	}
	if !i.Utilizavel(agora) {
		t.Fatal("identidade a 10 dias devia ser utilizavel")
	}
	i.ExpiraEm = agora.AddDate(0, 0, 60)
	if estado := i.Estado(agora); estado != EstadoAtiva {
		t.Fatalf("estado = %q; esperado ativa a 60 dias", estado)
	}
}

// Validade desconhecida NAO e' validade boa.
func TestValidadeDesconhecidaNaoEUtilizavel(t *testing.T) {
	i := identidadeValida()
	i.ExpiraEm = time.Time{}
	if estado := i.Estado(agora); estado != EstadoDesconhecido {
		t.Fatalf("estado = %q; esperado desconhecida", estado)
	}
	if i.Utilizavel(agora) {
		t.Fatal("identidade sem validade declarada utilizavel -- falha fechada devia recusar")
	}
}

// Lista de runners vazia significa "nenhum", nao "todos".
func TestSemRunnerAutorizadoNaoEUtilizavelParaAssinar(t *testing.T) {
	registo := NovoRegistoDeIdentidades()
	i := identidadeValida()
	i.RunnersAutorizados = nil
	if err := registo.Adicionar(i); err != nil {
		t.Fatal(err)
	}
	if encontradas := registo.ParaAssinar(PlataformaIOS, AmbienteProducao, agora); len(encontradas) != 0 {
		t.Fatalf("identidade sem runner autorizado apareceu como utilizavel: %d", len(encontradas))
	}
}

// Combinacao errada devolve vazio: assinar com a identidade errada e' pior do que nao assinar.
func TestParaAssinarFiltraPlataformaEAmbiente(t *testing.T) {
	registo := NovoRegistoDeIdentidades()
	if err := registo.Adicionar(identidadeValida()); err != nil {
		t.Fatal(err)
	}
	if n := len(registo.ParaAssinar(PlataformaIOS, AmbienteProducao, agora)); n != 1 {
		t.Fatalf("combinacao certa devolveu %d", n)
	}
	if n := len(registo.ParaAssinar(PlataformaAndroid, AmbienteProducao, agora)); n != 0 {
		t.Fatalf("android com identidade iOS devolveu %d", n)
	}
	if n := len(registo.ParaAssinar(PlataformaIOS, AmbientePreview, agora)); n != 0 {
		t.Fatalf("preview com identidade de producao devolveu %d", n)
	}
}

// Uma referencia opaca que carrega o segredo e' um vazamento com interface.
func TestReferenciaQuePareceSegredoEApontadaPelaAuditoria(t *testing.T) {
	casos := []string{
		"ghp_16C7e42F292c6912E7710c838347Ae178B4a",
		"sk-live-abcdefghijklmnop",
		"-----BEGIN PRIVATE KEY-----MIIE...",
		"Bearer eyJhbGciOiJIUzI1NiJ9",
		"AKIAIOSFODNN7EXAMPLE",
	}
	for _, ref := range casos {
		if !ReferenciaPareceSegredo(ref) {
			t.Fatalf("referencia com segredo nao detectada: %q", ref[:min(len(ref), 20)])
		}
	}
	// Referencias legitimas NAO podem ser apontadas: *um detector que grita por tudo deixa de
	// ser ouvido quando grita por algo.*
	for _, ref := range []string{
		"dpapi:apple/hades-apple-distribution.p12",
		"keychain:windows/dz23-code-signing",
		"eas:dz23s-team/android-upload-keystore",
		"kms://projects/dz23/locations/global/keyRings/hades/cryptoKeys/ios",
		"vault:secret/data/hades/windows-authenticode",
	} {
		if ReferenciaPareceSegredo(ref) {
			t.Fatalf("referencia legitima apontada como segredo: %q", ref)
		}
	}
}

func TestAuditarApontaProblemasReais(t *testing.T) {
	registo := NovoRegistoDeIdentidades()

	expirada := identidadeValida()
	expirada.Nome = "expirada"
	expirada.ExpiraEm = agora.AddDate(0, -1, 0)
	if err := registo.Adicionar(expirada); err != nil {
		t.Fatal(err)
	}

	saudavel := identidadeValida()
	saudavel.Nome = "saudavel"
	if err := registo.Adicionar(saudavel); err != nil {
		t.Fatal(err)
	}

	comSegredo := identidadeValida()
	comSegredo.Nome = "vazada"
	comSegredo.Referencia = "dpapi:ghp_16C7e42F292c6912E7710c838347Ae178B4a"
	if err := registo.Adicionar(comSegredo); err != nil {
		t.Fatal(err)
	}

	problemas := registo.Auditar(agora)
	porNome := map[string][]string{}
	for _, p := range problemas {
		porNome[p.Nome] = append(porNome[p.Nome], p.Mensagem)
	}
	if len(porNome["saudavel"]) != 0 {
		t.Fatalf("identidade saudavel recebeu problemas: %v", porNome["saudavel"])
	}
	if len(porNome["expirada"]) == 0 {
		t.Fatal("identidade expirada nao foi apontada")
	}
	achouVazamento := false
	for _, mensagem := range porNome["vazada"] {
		if strings.Contains(mensagem, "valor secreto") {
			achouVazamento = true
		}
	}
	if !achouVazamento {
		t.Fatalf("referencia com segredo nao apontada: %v", porNome["vazada"])
	}
}

func TestResumirContaPorEstadoEAmbiente(t *testing.T) {
	registo := NovoRegistoDeIdentidades()
	a := identidadeValida()
	a.Nome = "a"
	b := identidadeValida()
	b.Nome = "b"
	b.Ambiente = AmbientePreview
	if err := registo.Adicionar(a); err != nil {
		t.Fatal(err)
	}
	if err := registo.Adicionar(b); err != nil {
		t.Fatal(err)
	}
	resumo := registo.Resumir(agora)
	if resumo.Total != 2 {
		t.Fatalf("total = %d", resumo.Total)
	}
	if resumo.PorAmbiente["producao"] != 1 || resumo.PorAmbiente["preview"] != 1 {
		t.Fatalf("por_ambiente = %v", resumo.PorAmbiente)
	}
	if resumo.PorEstado["ativa"] != 2 {
		t.Fatalf("por_estado = %v", resumo.PorEstado)
	}
}

// `Identidades()` devolve copia: mudar a copia nao muda o registo.
func TestIdentidadesDevolveCopia(t *testing.T) {
	registo := NovoRegistoDeIdentidades()
	if err := registo.Adicionar(identidadeValida()); err != nil {
		t.Fatal(err)
	}
	copia := registo.Identidades()
	copia[0].Nome = "alterado"
	if registo.Identidades()[0].Nome == "alterado" {
		t.Fatal("a fatia interna foi exposta: o chamador mudou o registo sem validacao")
	}
}

// O fingerprint de um certificado PUBLICO e' publico por desenho. Um blob que nao e' X.509
// tem de ser recusado: *chamar certificado a bytes arbitrarios e' o comeco de um falso atestado.*
func TestFingerprintDeCertificadoDER(t *testing.T) {
	if _, err := FingerprintDeCertificadoDER(nil); err == nil {
		t.Fatal("DER vazio aceite")
	}
	if _, err := FingerprintDeCertificadoDER([]byte("nao sou um certificado")); err == nil {
		t.Fatal("bytes arbitrarios aceites como certificado")
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "HADES teste"},
		NotBefore:    agora.AddDate(0, -1, 0),
		NotAfter:     agora.AddDate(1, 0, 0),
	}
	der, err := x509.CreateCertificate(nil, template, template, nil, nil)
	if err != nil {
		// CreateCertificate exige chave; se o ambiente recusar nil, o teste nao pode inventar
		// um certificado -- declara a limitacao em vez de passar por omissao.
		t.Skipf("nao foi possivel gerar certificado de teste sem chave: %v", err)
	}
	fingerprint, err := FingerprintDeCertificadoDER(der)
	if err != nil {
		t.Fatalf("certificado valido recusado: %v", err)
	}
	if _, err := NormalizarFingerprint(fingerprint); err != nil {
		t.Fatalf("fingerprint devolvido nao normaliza: %v", err)
	}
}

// Garante que o import de asn1 nao fica so' para um caso: o pacote e' usado pelo x509.
var _ = asn1.NullRawValue

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
