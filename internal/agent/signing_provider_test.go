package agent

// Testes do `SigningProvider` (E-1263).
//
// O que estes testes travam:
//
//  1. um certificado COM o EKU de Code Signing e' `APTA`;
//  2. um certificado SEM o EKU e' `SEM_EKU_DE_ASSINATURA` -- o caso do A1 da DZ23 LTDA;
//  3. a validade e' verificada ANTES do EKU (um expirado diz `EXPIRADA`, nao `SEM_EKU`);
//  4. `NotBefore` no futuro diz `AINDA_NAO_VALIDA`;
//  5. o motivo lista os EKUs que o certificado TEM, nao so' o que falta;
//  6. um formato sem regra de EKU e' `FORMATO_DESCONHECIDO`, nunca `APTA`;
//  7. um ficheiro que nao e' certificado devolve `ErrCertificadoIlegivel`;
//  8. uma identidade sem caminho e' recusada com `ErrSemReferencia`;
//  9. `Alvos` declara so' o que sabe avaliar.
//
// Os certificados sao GERADOS no teste, para o resultado nao depender do store desta maquina. *
// Um teste que passa na maquina onde foi escrito e falha nas outras tres nao testa o produto.*

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// certificadoDeTeste gera um certificado auto-assinado com os EKUs pedidos e a validade pedida.
func certificadoDeTeste(
	t *testing.T,
	ekus []x509.ExtKeyUsage,
	naoAntes, naoDepois time.Time,
) (*x509.Certificate, []byte) {
	t.Helper()
	chave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gerar chave: %v", err)
	}
	serie, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("serie: %v", err)
	}
	modelo := &x509.Certificate{
		SerialNumber:          serie,
		Subject:               pkix.Name{CommonName: "HADES Certificado de Teste", Organization: []string{"DZ23 LTDA"}},
		NotBefore:             naoAntes,
		NotAfter:              naoDepois,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           ekus,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, modelo, modelo, &chave.PublicKey, chave)
	if err != nil {
		t.Fatalf("criar certificado: %v", err)
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	return parsed, der
}

// escreverPEM escreve o certificado em PEM e devolve o caminho.
func escreverPEM(t *testing.T, der []byte) string {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "certificado.pem")
	bloco := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(caminho, bloco, 0o600); err != nil {
		t.Fatalf("escrever PEM: %v", err)
	}
	return caminho
}

// escreverDER escreve o certificado em DER e devolve o caminho.
func escreverDER(t *testing.T, der []byte) string {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "certificado.cer")
	if err := os.WriteFile(caminho, der, 0o600); err != nil {
		t.Fatalf("escrever DER: %v", err)
	}
	return caminho
}

func TestSigningProviderAprovaComEKU(t *testing.T) {
	agora := time.Now()
	certificado, _ := certificadoDeTeste(t, []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		agora.Add(-time.Hour), agora.Add(24*time.Hour))

	avaliacao := AvaliarCertificado(certificado, AssinaturaAuthenticode, agora)
	if !avaliacao.Apta {
		t.Fatalf("apta = false com EKU de Code Signing: %+v", avaliacao)
	}
	if avaliacao.Aptidao != IdentidadeApta {
		t.Fatalf("aptidao = %q; esperado APTA", avaliacao.Aptidao)
	}
	if !avaliacao.TemEKUNecessario {
		t.Fatal("tem_eku_necessario = false com EKU presente")
	}
	if !strings.Contains(avaliacao.Motivo, "a assinatura em si continua a exigir a chave privada") {
		t.Fatalf("motivo nao declara a fronteira: %q", avaliacao.Motivo)
	}
}

// O CASO DO A1 DA DZ23 LTDA: certificado valido, sem o EKU de Code Signing.
func TestSigningProviderRecusaSemEKU(t *testing.T) {
	agora := time.Now()
	// Um A1 ICP-Brasil costuma trazer EKU de email/protecao, nao de Code Signing.
	certificado, _ := certificadoDeTeste(t,
		[]x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection, x509.ExtKeyUsageClientAuth},
		agora.Add(-time.Hour), agora.Add(24*time.Hour))

	avaliacao := AvaliarCertificado(certificado, AssinaturaAuthenticode, agora)
	if avaliacao.Apta {
		t.Fatal("apta = true sem o EKU de Code Signing; e' o caso do A1 da DZ23 LTDA")
	}
	if avaliacao.Aptidao != IdentidadeSemEKU {
		t.Fatalf("aptidao = %q; esperado SEM_EKU_DE_ASSINATURA", avaliacao.Aptidao)
	}
	if avaliacao.TemEKUNecessario {
		t.Fatal("tem_eku_necessario = true sem o EKU")
	}
	// O motivo tem de dizer QUAL falta E listar os que existem.
	if !strings.Contains(avaliacao.Motivo, oidEKUCodeSigning) {
		t.Fatalf("motivo nao nomeia o EKU em falta: %q", avaliacao.Motivo)
	}
	if !strings.Contains(avaliacao.Motivo, "1.3.6.1.5.5.7.3.4") {
		t.Fatalf("motivo nao lista os EKUs presentes: %q", avaliacao.Motivo)
	}
	if !strings.Contains(avaliacao.Motivo, "NotValidForUsage") {
		t.Fatalf("motivo nao nomeia a falha da politica: %q", avaliacao.Motivo)
	}
}

// A validade vem ANTES do EKU: um expirado diz `EXPIRADA`, nao `SEM_EKU`.
//
// *Dizer "falta o EKU" a um certificado expirado manda quem le' corrigir a coisa errada.*
func TestSigningProviderReportaExpiracaoAntesDoEKU(t *testing.T) {
	agora := time.Now()
	// SEM EKU de Code Signing E expirado: qual dos dois motivos ganha?
	certificado, _ := certificadoDeTeste(t, []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection},
		agora.Add(-48*time.Hour), agora.Add(-24*time.Hour))

	avaliacao := AvaliarCertificado(certificado, AssinaturaAuthenticode, agora)
	if avaliacao.Aptidao != IdentidadeExpirada {
		t.Fatalf("aptidao = %q; esperado EXPIRADA (a validade vem primeiro)", avaliacao.Aptidao)
	}
	if strings.Contains(avaliacao.Motivo, "EKU") {
		t.Fatalf("motivo fala de EKU num certificado expirado: %q", avaliacao.Motivo)
	}
}

func TestSigningProviderReportaAindaNaoValido(t *testing.T) {
	agora := time.Now()
	certificado, _ := certificadoDeTeste(t, []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		agora.Add(24*time.Hour), agora.Add(48*time.Hour))

	avaliacao := AvaliarCertificado(certificado, AssinaturaAuthenticode, agora)
	if avaliacao.Aptidao != IdentidadeAindaNaoValida {
		t.Fatalf("aptidao = %q; esperado AINDA_NAO_VALIDA", avaliacao.Aptidao)
	}
}

// Um formato sem regra de EKU nao pode ser declarado APTA por omissao.
func TestSigningProviderNaoAprovaFormatoSemRegra(t *testing.T) {
	agora := time.Now()
	certificado, _ := certificadoDeTeste(t, []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		agora.Add(-time.Hour), agora.Add(24*time.Hour))

	for _, tipo := range []TipoDeAssinatura{AssinaturaAndroidAPK, AssinaturaOpenPGP} {
		avaliacao := AvaliarCertificado(certificado, tipo, agora)
		if avaliacao.Apta {
			t.Fatalf("%q declarado APTA sem regra de EKU conhecida", tipo)
		}
		if avaliacao.Aptidao != IdentidadeFormatoDesconhecido {
			t.Fatalf("%q -> %q; esperado FORMATO_DESCONHECIDO", tipo, avaliacao.Aptidao)
		}
		// E diz que nao sabe, em vez de inventar.
		if !strings.Contains(avaliacao.Motivo, "nao afirma que serve nem que nao serve") {
			t.Fatalf("motivo nao declara a ignorancia: %q", avaliacao.Motivo)
		}
	}
}

func TestSigningProviderCarregaPEMeDER(t *testing.T) {
	agora := time.Now()
	_, der := certificadoDeTeste(t, []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		agora.Add(-time.Hour), agora.Add(24*time.Hour))

	provider := ProviderDeAssinaturaX509{Agora: agora}
	for nome, caminho := range map[string]string{
		"PEM": escreverPEM(t, der),
		"DER": escreverDER(t, der),
	} {
		avaliacao, err := provider.Avaliar(context.Background(), AssinaturaAuthenticode,
			IdentidadeDeAssinatura{CaminhoDoCertificado: caminho, Rotulo: nome})
		if err != nil {
			t.Fatalf("%s: erro inesperado: %v", nome, err)
		}
		if !avaliacao.Apta {
			t.Fatalf("%s: nao aprovado: %+v", nome, avaliacao)
		}
	}
}

func TestSigningProviderRecusaFicheiroQueNaoECertificado(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "texto.pem")
	if err := os.WriteFile(caminho, []byte("isto nao e' um certificado"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	provider := ProviderDeAssinaturaX509{}
	avaliacao, err := provider.Avaliar(context.Background(), AssinaturaAuthenticode,
		IdentidadeDeAssinatura{CaminhoDoCertificado: caminho})
	if !errors.Is(err, ErrCertificadoIlegivel) {
		t.Fatalf("err = %v; esperado ErrCertificadoIlegivel", err)
	}
	if avaliacao.Apta {
		t.Fatal("apta = true para um ficheiro ilegivel")
	}
	if avaliacao.Aptidao != IdentidadeIlegivel {
		t.Fatalf("aptidao = %q; esperado ILEGIVEL", avaliacao.Aptidao)
	}
}

// Uma identidade sem certificado nao e' uma identidade: e' um rotulo.
func TestSigningProviderRecusaIdentidadeSemCaminho(t *testing.T) {
	provider := ProviderDeAssinaturaX509{}
	_, err := provider.Avaliar(context.Background(), AssinaturaAuthenticode,
		IdentidadeDeAssinatura{Rotulo: "A1 DZ23 LTDA"})
	if !errors.Is(err, ErrSemReferencia) {
		t.Fatalf("err = %v; esperado ErrSemReferencia", err)
	}
}

func TestSigningProviderAlvos(t *testing.T) {
	alvos := ProviderDeAssinaturaX509{}.Alvos()
	// *Um provider que existe no catalogo e nunca correu e' uma promessa com nome de modulo.*
	// `Alvos` declara exactamente o que `Avaliar` sabe decidir, e nada mais.
	if len(alvos) != 2 {
		t.Fatalf("alvos = %v; esperado 2", alvos)
	}
	conjunto := map[TipoDeAssinatura]bool{}
	for _, alvo := range alvos {
		conjunto[alvo] = true
	}
	if !conjunto[AssinaturaAuthenticode] || !conjunto[AssinaturaAppleDeveloperID] {
		t.Fatalf("alvos = %v; faltam os dois formatos X.509", alvos)
	}
}

// ---------------------------------------------------------------------------
// O CASO QUE UM CERTIFICADO REAL EXPOS
// ---------------------------------------------------------------------------

// certificadoComEmissor gera um certificado cujo EMISSOR e' o indicado.
func certificadoComEmissor(
	t *testing.T,
	emissorCN string,
	ekus []x509.ExtKeyUsage,
	naoAntes, naoDepois time.Time,
) *x509.Certificate {
	t.Helper()
	chave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gerar chave: %v", err)
	}
	serie, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("serie: %v", err)
	}
	modelo := &x509.Certificate{
		SerialNumber:          serie,
		Subject:               pkix.Name{CommonName: "HADES Teste"},
		NotBefore:             naoAntes,
		NotAfter:              naoDepois,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           ekus,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, modelo, modelo, &chave.PublicKey, chave)
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	// O emissor de um auto-assinado e' o proprio sujeito; reescrevemo-lo para o caso de teste.
	parsed.Issuer = pkix.Name{CommonName: emissorCN, Organization: []string{"CA de Teste"}}
	return parsed
}

// UMA CA DA APPLE NAO ASSINA WINDOWS.
//
// Este e' o defeito que o certificado Apple Distribution REAL do kit da DZ23 LTDA expos: ele tem o
// EKU `1.3.6.1.5.5.7.3.3` (Code Signing), tal como um certificado Authenticode. Sem a verificacao de
// emissor, o provider declarava-o APTO a assinar um `.exe`.
func TestSigningProviderRecusaCertificadoAppleParaAuthenticode(t *testing.T) {
	agora := time.Now()
	certificado := certificadoComEmissor(t,
		"Apple Worldwide Developer Relations Certification Authority",
		[]x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		agora.Add(-time.Hour), agora.Add(24*time.Hour))

	avaliacao := AvaliarCertificado(certificado, AssinaturaAuthenticode, agora)
	if avaliacao.Apta {
		t.Fatal("um certificado da Apple foi declarado apto a assinar Authenticode")
	}
	if avaliacao.Aptidao != IdentidadeDeOutraCA {
		t.Fatalf("aptidao = %q; esperado EMISSOR_INESPERADO", avaliacao.Aptidao)
	}
	if !strings.Contains(avaliacao.Motivo, "Apple") {
		t.Fatalf("motivo nao nomeia o emissor: %q", avaliacao.Motivo)
	}
	// E o oposto: para Apple Developer ID o emissor Apple E' o certo -- recusa-se pelo EKU, nao pelo
	// emissor. E' a prova de que a verificacao de emissor nao esta' a recusar tudo.
	avaliacaoApple := AvaliarCertificado(certificado, AssinaturaAppleDeveloperID, agora)
	if avaliacaoApple.Aptidao != IdentidadeSemEKU {
		t.Fatalf("Apple -> %q; esperado SEM_EKU_DE_ASSINATURA (tem Code Signing, falta Developer ID)",
			avaliacaoApple.Aptidao)
	}
}

// Uma CA publica qualquer nao serve para Apple Developer ID.
func TestSigningProviderRecusaCAComumParaApple(t *testing.T) {
	agora := time.Now()
	certificado := certificadoComEmissor(t, "DigiCert Code Signing CA",
		[]x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		agora.Add(-time.Hour), agora.Add(24*time.Hour))

	avaliacao := AvaliarCertificado(certificado, AssinaturaAppleDeveloperID, agora)
	if avaliacao.Apta {
		t.Fatal("uma CA comum foi declarada apta para Apple Developer ID")
	}
	if avaliacao.Aptidao != IdentidadeDeOutraCA {
		t.Fatalf("aptidao = %q; esperado EMISSOR_INESPERADO", avaliacao.Aptidao)
	}
}

// O CERTIFICADO REAL do kit, quando presente, e' avaliado de verdade.
//
// Nao falha se o kit nao estiver nesta maquina (ele esta' fora do repositorio, por desenho).
func TestSigningProviderComCertificadoAppleReal(t *testing.T) {
	caminho := os.Getenv("HADES_CERTIFICADO_APPLE_REAL")
	if caminho == "" {
		t.Skip("HADES_CERTIFICADO_APPLE_REAL nao definido; o kit de credenciais esta' fora do repo")
	}
	if _, err := os.Stat(caminho); err != nil {
		t.Skipf("certificado ausente: %v", err)
	}
	provider := ProviderDeAssinaturaX509{}

	// Como Apple Developer ID: o certificado Apple Distribution TEM Code Signing, mas NAO tem o EKU
	// de Developer ID Application. O adendo diz que um nao substitui o outro:
	//   "Certificado Apple Distribution nao substitui Developer ID Application para distribuicao
	//    direta notarizada."
	// Este teste prova essa frase contra o ficheiro real.
	avaliacao, err := provider.Avaliar(context.Background(), AssinaturaAppleDeveloperID,
		IdentidadeDeAssinatura{CaminhoDoCertificado: caminho, Rotulo: "Apple Distribution (kit)"})
	if err != nil {
		t.Fatalf("avaliar: %v", err)
	}
	t.Logf("Apple Distribution real: aptidao=%s emissor=%q", avaliacao.Aptidao, avaliacao.Emissor)
	if avaliacao.Apta {
		t.Fatalf("o Apple Distribution foi declarado apto para Developer ID: %+v", avaliacao)
	}
	if avaliacao.Aptidao != IdentidadeSemEKU {
		t.Fatalf("aptidao = %q; esperado SEM_EKU_DE_ASSINATURA", avaliacao.Aptidao)
	}

	// E como Authenticode: recusado pelo EMISSOR, apesar de ter o EKU de Code Signing.
	avaliacaoWin, err := provider.Avaliar(context.Background(), AssinaturaAuthenticode,
		IdentidadeDeAssinatura{CaminhoDoCertificado: caminho, Rotulo: "Apple Distribution (kit)"})
	if err != nil {
		t.Fatalf("avaliar: %v", err)
	}
	if avaliacaoWin.Apta {
		t.Fatal("o certificado Apple foi declarado apto a assinar Authenticode")
	}
	if avaliacaoWin.Aptidao != IdentidadeDeOutraCA {
		t.Fatalf("aptidao = %q; esperado EMISSOR_INESPERADO", avaliacaoWin.Aptidao)
	}
}
