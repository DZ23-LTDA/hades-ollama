package agent

// Testes do `CredentialProvider` (E-1265).
//
// O que estes testes travam, por ordem de importancia:
//
//  1. **NUNCA abre o que pode ser segredo** -- `.p12`, `.jks`, `.dpapi`, `.p8`, `.mobileprovision`;
//  2. a prova de (1) nao e' "o codigo diz que nao abre": e' que o ficheiro **ilegivel** ainda aparece
//     no inventario com tamanho -- um `os.Open` teria falhado;
//  3. certificados publicos SAO lidos, com fingerprint e validade;
//  4. um ficheiro com extensao de certificado mas conteudo invalido nao inventa fingerprint;
//  5. a varredura tem tecto;
//  6. o aviso de fronteira diz o que NAO foi verificado;
//  7. o `SegredosPresentes` lista exactamente os ficheiros que nao foram abertos.

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// criarKitDeTeste monta um kit com um certificado real e ficheiros que fingem ser segredos.
func criarKitDeTeste(t *testing.T) (raiz string, fingerprintEsperado string) {
	t.Helper()
	raiz = t.TempDir()

	// Um certificado publico verdadeiro.
	chave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("chave: %v", err)
	}
	serie, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("serie: %v", err)
	}
	agora := time.Now()
	modelo := &x509.Certificate{
		SerialNumber:          serie,
		Subject:               pkix.Name{CommonName: "HADES Kit de Teste"},
		NotBefore:             agora.Add(-time.Hour),
		NotAfter:              agora.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, modelo, modelo, &chave.PublicKey, chave)
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	pastaApple := filepath.Join(raiz, "apple")
	if err := os.MkdirAll(pastaApple, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pastaApple, "distribution.cer"), der, 0o600); err != nil {
		t.Fatalf("escrever cer: %v", err)
	}
	soma := sha256.Sum256(der)
	fingerprintEsperado = hex.EncodeToString(soma[:])

	// Ficheiros que FINGEM ser segredos. O conteudo e' uma sentinela reconhecivel.
	const sentinela = "CONTEUDO-SECRETO-QUE-NAO-PODE-SER-LIDO"
	for _, nome := range []string{
		"hades-apple-distribution.p12",
		"hades-apple-distribution.password.dpapi",
		"AuthKey_WBH66TWV84.p8",
		"HADES_App_Store_Distribution.mobileprovision",
	} {
		if err := os.WriteFile(filepath.Join(pastaApple, nome), []byte(sentinela), 0o600); err != nil {
			t.Fatalf("escrever %s: %v", nome, err)
		}
	}
	pastaAndroid := filepath.Join(raiz, "android")
	if err := os.MkdirAll(pastaAndroid, 0o755); err != nil {
		t.Fatalf("mkdir android: %v", err)
	}
	for _, nome := range []string{"hades-upload.jks", "hades-upload.password.dpapi"} {
		if err := os.WriteFile(filepath.Join(pastaAndroid, nome), []byte(sentinela), 0o600); err != nil {
			t.Fatalf("escrever %s: %v", nome, err)
		}
	}
	if err := os.WriteFile(filepath.Join(raiz, "SHA256SUMS.txt"), []byte("hash  ficheiro"), 0o600); err != nil {
		t.Fatalf("escrever sums: %v", err)
	}
	return raiz, fingerprintEsperado
}

func TestCredentialProviderInventariaKit(t *testing.T) {
	raiz, fingerprint := criarKitDeTeste(t)
	provider := ProviderDeCredenciais{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	if inventario.Total == 0 {
		t.Fatal("inventario vazio")
	}
	if inventario.Certificados != 1 {
		t.Fatalf("certificados = %d; esperado 1", inventario.Certificados)
	}

	// O certificado tem fingerprint, sujeito e validade.
	var encontrado bool
	for _, credencial := range inventario.Credenciais {
		if credencial.Tipo != CredencialCertificadoX509 {
			continue
		}
		encontrado = true
		if credencial.FingerprintSHA256 != fingerprint {
			t.Fatalf("fingerprint = %q; esperado %q", credencial.FingerprintSHA256, fingerprint)
		}
		if !credencial.ConteudoLido {
			t.Fatal("conteudo_lido = false num certificado publico")
		}
		if credencial.Sujeito == "" {
			t.Fatal("sujeito vazio num certificado lido")
		}
		if credencial.ValidoAte == "" {
			t.Fatal("valido_ate vazio num certificado lido")
		}
	}
	if !encontrado {
		t.Fatal("o certificado nao apareceu no inventario")
	}
}

// O TESTE QUE IMPORTA: os segredos NAO foram abertos.
func TestCredentialProviderNaoAbreSegredos(t *testing.T) {
	raiz, _ := criarKitDeTeste(t)
	provider := ProviderDeCredenciais{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}

	segredosEsperados := map[string]bool{
		"apple/hades-apple-distribution.p12":                 true,
		"apple/hades-apple-distribution.password.dpapi":      true,
		"apple/AuthKey_WBH66TWV84.p8":                        true,
		"apple/HADES_App_Store_Distribution.mobileprovision": true,
		"android/hades-upload.jks":                           true,
		"android/hades-upload.password.dpapi":                true,
	}
	vistos := map[string]bool{}
	for _, credencial := range inventario.Credenciais {
		if !segredosEsperados[filepath.ToSlash(credencial.CaminhoRelativo)] {
			continue
		}
		vistos[filepath.ToSlash(credencial.CaminhoRelativo)] = true
		if credencial.ConteudoLido {
			t.Fatalf("%s foi ABERTO (conteudo_lido = true)", credencial.CaminhoRelativo)
		}
		if credencial.Nota == "" {
			t.Fatalf("%s sem nota a dizer que o conteudo nao foi lido", credencial.CaminhoRelativo)
		}
		if credencial.FingerprintSHA256 != "" {
			t.Fatalf("%s tem fingerprint -- so' podia ter se tivesse sido lido", credencial.CaminhoRelativo)
		}
	}
	for esperado := range segredosEsperados {
		if !vistos[esperado] {
			t.Fatalf("%s nao apareceu no inventario", esperado)
		}
	}
	if len(inventario.SegredosPresentes) != len(segredosEsperados) {
		t.Fatalf("segredos_presentes = %v; esperado %d entradas",
			inventario.SegredosPresentes, len(segredosEsperados))
	}
}

// A PROVA de que nao abriu: um segredo ILEGIVEL ainda e' inventariado com tamanho.
//
// Se o provider tivesse tentado `os.Open`, este teste falharia com um erro de permissao.
func TestCredentialProviderNaoPrecisaDeLerParaInventariar(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("a correr como root: as permissoes nao impedem a leitura")
	}
	raiz := t.TempDir()
	caminho := filepath.Join(raiz, "hades-upload.jks")
	if err := os.WriteFile(caminho, []byte("segredo-inacessivel"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	// Tira a permissao de leitura: um `os.Open` falharia.
	if err := os.Chmod(caminho, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer func() { _ = os.Chmod(caminho, 0o600) }()

	provider := ProviderDeCredenciais{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	if inventario.Total != 1 {
		t.Fatalf("total = %d; esperado 1 (o ficheiro ilegivel tem de aparecer)", inventario.Total)
	}
	credencial := inventario.Credenciais[0]
	if credencial.ConteudoLido {
		t.Fatal("conteudo_lido = true num ficheiro sem permissao de leitura")
	}
	if credencial.TamanhoBytes != int64(len("segredo-inacessivel")) {
		t.Fatalf("tamanho = %d; esperado %d", credencial.TamanhoBytes, len("segredo-inacessivel"))
	}
}

func TestCredentialProviderNaoInventaFingerprint(t *testing.T) {
	raiz := t.TempDir()
	// Extensao de certificado, conteudo que nao e' um certificado.
	if err := os.WriteFile(filepath.Join(raiz, "falso.cer"), []byte("isto nao e' um certificado"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	provider := ProviderDeCredenciais{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	credencial := inventario.Credenciais[0]
	if credencial.FingerprintSHA256 != "" {
		t.Fatalf("fingerprint inventado: %q", credencial.FingerprintSHA256)
	}
	if credencial.Nota == "" {
		t.Fatal("sem nota a explicar porque o ficheiro nao deu um certificado")
	}
	// E nao conta como certificado valido.
	if inventario.Certificados != 0 {
		t.Fatalf("certificados = %d; esperado 0", inventario.Certificados)
	}
}

func TestCredentialProviderDetectaExpirado(t *testing.T) {
	raiz := t.TempDir()
	chave, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	serie, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	agora := time.Now()
	modelo := &x509.Certificate{
		SerialNumber: serie,
		Subject:      pkix.Name{CommonName: "Expirado"},
		NotBefore:    agora.Add(-48 * time.Hour),
		NotAfter:     agora.Add(-24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, modelo, modelo, &chave.PublicKey, chave)
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "velho.cer"), der, 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	provider := ProviderDeCredenciais{Agora: agora}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	if inventario.Expirados != 1 {
		t.Fatalf("expirados = %d; esperado 1", inventario.Expirados)
	}
	if !inventario.Credenciais[0].Expirado {
		t.Fatal("expirado = false num certificado fora da validade")
	}
}

func TestCredentialProviderRecusaRaizInexistente(t *testing.T) {
	provider := ProviderDeCredenciais{}
	_, err := provider.Inventariar(context.Background(), filepath.Join(t.TempDir(), "nao-existe"))
	if err == nil {
		t.Fatal("erro = nil para uma raiz inexistente")
	}
	if !strings.Contains(err.Error(), ErrKitInexistente.Error()) {
		t.Fatalf("erro = %v; esperado embrulhar ErrKitInexistente", err)
	}
}

func TestCredentialProviderDeclaraAFronteira(t *testing.T) {
	raiz, _ := criarKitDeTeste(t)
	provider := ProviderDeCredenciais{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	// O aviso tem de negar explicitamente as tres coisas que alguem poderia supor.
	for _, frase := range []string{"NAO le' chaves privadas", "Nao valida senhas", "nao exporta nada"} {
		if !strings.Contains(inventario.Aviso, frase) {
			t.Fatalf("aviso nao contem %q: %q", frase, inventario.Aviso)
		}
	}
}

func TestCredentialProviderClassifica(t *testing.T) {
	casos := map[string]TipoDeCredencial{
		"a.p12": CredencialKeystore, "b.jks": CredencialKeystore, "c.keystore": CredencialKeystore,
		"d.password.dpapi": CredencialSenhaProtegida, "e.p8": CredencialChavePrivada,
		"f.mobileprovision": CredencialPerfilDeProvisionamento,
		"g.cer":             CredencialCertificadoX509, "h.der": CredencialCertificadoX509,
		"i.pem":          CredencialCertificadoX509,
		"SHA256SUMS.txt": CredencialChecksums, "j.txt": CredencialDocumentacao,
		"k.bin": CredencialDesconhecida,
	}
	for nome, esperado := range casos {
		if obtido := classificar(nome); obtido != esperado {
			t.Fatalf("classificar(%q) = %q; esperado %q", nome, obtido, esperado)
		}
	}
	// E o mais importante: `DESCONHECIDA` NAO e' legivel. Na duvida, nao se abre.
	if conteudoLido(CredencialDesconhecida) {
		t.Fatal("tipo DESCONHECIDA declarado legivel: na duvida, nao se abre")
	}
	_ = pem.Block{} // mantem o import em uso se os casos mudarem
}

// Inventaria o kit REAL, quando `HADES_KIT_CREDENCIAIS` esta' definido.
//
// Nao falha se o kit nao estiver nesta maquina: ele esta' fora do repositorio, por desenho.
//
// A prova que este teste da' e' dupla: (a) o inventario encontra os ficheiros reais; (b) **nenhum
// ficheiro de segredo e' aberto** -- o que se verifica porque `ConteudoLido` e' falso e o
// `FingerprintSHA256` esta' vazio neles.
func TestCredentialProviderComKitReal(t *testing.T) {
	raiz := os.Getenv("HADES_KIT_CREDENCIAIS")
	if raiz == "" {
		t.Skip("HADES_KIT_CREDENCIAIS nao definido; o kit de credenciais esta' fora do repo")
	}
	if _, err := os.Stat(raiz); err != nil {
		t.Skipf("kit ausente: %v", err)
	}
	provider := ProviderDeCredenciais{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	t.Logf("kit real: %d ficheiros, %d lidos, %d NAO lidos, %d certificados, %d expirados, %d segredos",
		inventario.Total, inventario.Lidos, inventario.NaoLidos,
		inventario.Certificados, inventario.Expirados, len(inventario.SegredosPresentes))

	if inventario.Total == 0 {
		t.Fatal("inventario vazio num kit que existe")
	}
	// O kit tem pelo menos um certificado publico real.
	if inventario.Certificados < 1 {
		t.Fatalf("certificados = %d; o kit tem certificados publicos", inventario.Certificados)
	}
	// E tem segredos: .p12, .jks, .dpapi, .p8, .mobileprovision.
	if len(inventario.SegredosPresentes) < 5 {
		t.Fatalf("segredos_presentes = %v; o kit real tem varios", inventario.SegredosPresentes)
	}
	// NENHUM deles foi aberto.
	for _, credencial := range inventario.Credenciais {
		if !conteudoLido(credencial.Tipo) && credencial.ConteudoLido {
			t.Fatalf("%s (%s) foi ABERTO", credencial.CaminhoRelativo, credencial.Tipo)
		}
	}
	// E os certificados lidos tem fingerprint real.
	for _, credencial := range inventario.Credenciais {
		if credencial.Tipo != CredencialCertificadoX509 {
			continue
		}
		t.Logf("certificado real: %s -> sujeito=%q expira=%s",
			credencial.CaminhoRelativo, credencial.Sujeito, credencial.ValidoAte)
		if credencial.FingerprintSHA256 != "" && len(credencial.FingerprintSHA256) != 64 {
			t.Fatalf("%s: fingerprint com %d caracteres", credencial.CaminhoRelativo,
				len(credencial.FingerprintSHA256))
		}
	}
}
