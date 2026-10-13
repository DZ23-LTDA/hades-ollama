package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ESTÁGIO 42.1 — CENTRAL DE PRODUÇÃO: VALIDAÇÃO E WORKER (E-334).
//
// **FFmpeg NÃO está disponível neste ambiente** (medido: `Get-Command ffmpeg` não encontrado). Isso NÃO vira desculpa
// para teste frouxo: a ausência é um caso previsto — *declarada* — e o worker é testado com um transcodificador de
// teste que permite provar o que importa sem depender de binário externo.

// --- utilitários ---

func arquivoComConteudo(t *testing.T, dir, nome string, conteudo []byte) string {
	t.Helper()
	caminho := filepath.Join(dir, nome)
	if err := os.WriteFile(caminho, conteudo, 0o600); err != nil {
		t.Fatal(err)
	}
	return caminho
}

func cabecalhoMP4() []byte {
	// `....ftyp` + marcação mínima: é o que faz um arquivo ser reconhecido como MP4 pelos magic bytes.
	conteudo := make([]byte, 64)
	copy(conteudo[0:4], []byte{0x00, 0x00, 0x00, 0x20})
	copy(conteudo[4:8], []byte("ftyp"))
	copy(conteudo[8:12], []byte("isom"))
	return conteudo
}

func cabecalhoPNG() []byte {
	conteudo := make([]byte, 64)
	copy(conteudo[0:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
	return conteudo
}

type transcoderDeTeste struct {
	disponivel bool
	escrita    []byte
	falhar     error
	bloquear   time.Duration
	forcarArgs []string
	recebidos  [][]string
}

func (t *transcoderDeTeste) Available() bool { return t.disponivel }

func (t *transcoderDeTeste) Args(r MediaTranscodeRequest) []string {
	if t.forcarArgs != nil {
		return t.forcarArgs
	}
	// VETOR de argumentos: nada de string de comando concatenada.
	return []string{"-nostdin", "-y", "-i", r.InputPath, "-c:v", "libx264", r.OutputPath}
}

func (t *transcoderDeTeste) Run(ctx context.Context, r MediaTranscodeRequest, args []string) error {
	t.recebidos = append(t.recebidos, args)
	if t.bloquear > 0 {
		select {
		case <-time.After(t.bloquear):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if t.falhar != nil {
		return t.falhar
	}
	if err := os.WriteFile(r.OutputPath, t.escrita, 0o600); err != nil {
		return err
	}
	return nil
}

// --- validação por CONTEÚDO ---

func TestUploadRecusaConteudoDiferenteDoDeclarado(t *testing.T) {
	dir := t.TempDir()
	// Um arquivo chamado .mp4 que na verdade é PNG: a extensão é AFIRMAÇÃO do usuário, não fato.
	origem := arquivoComConteudo(t, dir, "video.mp4", cabecalhoPNG())
	destino := filepath.Join(dir, "uploads")
	if _, err := ValidateMediaUpload(origem, destino, "video.mp4", "video/mp4", DefaultMediaLimits()); !errors.Is(err, ErrMediaContentMismatch) {
		t.Fatalf("conteúdo PNG declarado como video/mp4 precisa ser RECUSADO: %v", err)
	}
	// Conteúdo desconhecido (texto) também é recusado: não se adivinha tipo.
	texto := arquivoComConteudo(t, dir, "nota.mp4", []byte("isto e apenas texto, nao um video"))
	if _, err := ValidateMediaUpload(texto, destino, "nota.mp4", "video/mp4", DefaultMediaLimits()); !errors.Is(err, ErrMediaContentMismatch) {
		t.Fatalf("conteúdo não reconhecido precisa ser recusado: %v", err)
	}
	// MP4 de verdade é aceito e copiado com nome seguro.
	bom := arquivoComConteudo(t, dir, "bom.mp4", cabecalhoMP4())
	validado, err := ValidateMediaUpload(bom, destino, "bom video.mp4", "video/mp4", DefaultMediaLimits())
	if err != nil {
		t.Fatalf("MP4 válido precisa ser aceito: %v", err)
	}
	if validado.ActualType != "video/mp4" || validado.SafeName != "bom_video.mp4" {
		t.Fatalf("validação precisa devolver tipo real e nome seguro: %+v", validado)
	}
	if _, err := os.Stat(validado.StoredPath); err != nil {
		t.Fatalf("o arquivo validado precisa existir no destino: %v", err)
	}
}

func TestUploadRecusaNomePerigoso(t *testing.T) {
	perigosos := []string{
		"../../etc/passwd",
		"..\\..\\windows\\system32\\config",
		"/etc/shadow",
		"C:\\Windows\\system.ini",
		"pasta/arquivo.mp4",
		"..",
		"arquivo\x00.mp4",
		"linha\nquebrada.mp4",
	}
	for _, nome := range perigosos {
		if _, err := SanitizeMediaFilename(nome); !errors.Is(err, ErrMediaPathUnsafe) {
			t.Fatalf("nome perigoso precisa ser RECUSADO: %q (err=%v)", nome, err)
		}
	}
	// Nome comum com espaços e acentos é aceito e normalizado.
	limpo, err := SanitizeMediaFilename("Minha Gravação Ação 01.mp4")
	if err != nil {
		t.Fatalf("nome comum precisa ser aceito: %v", err)
	}
	if strings.ContainsAny(limpo, "/\\ ") {
		t.Fatalf("nome normalizado não pode conter separador nem espaço: %q", limpo)
	}
}

func TestUploadRecusaAcimaDoLimite(t *testing.T) {
	dir := t.TempDir()
	grande := arquivoComConteudo(t, dir, "grande.mp4", cabecalhoMP4())
	limites := DefaultMediaLimits()
	limites.MaxBytes = 10 // limite artificialmente pequeno: o excesso tem de ser DETECTADO, não truncado
	if _, err := ValidateMediaUpload(grande, filepath.Join(dir, "u"), "grande.mp4", "video/mp4", limites); !errors.Is(err, ErrMediaInvalid) {
		t.Fatalf("arquivo acima do limite precisa ser recusado: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "u", "grande.mp4")); err == nil {
		t.Fatal("nada pode ficar no destino quando a validação falha")
	}
}

// --- worker: sem shell, com validação de saída, timeout e limpeza ---

func TestSemTranscodificadorOResultadoEhDeclaradoENadaEhCriado(t *testing.T) {
	dir := t.TempDir()
	saida := filepath.Join(dir, "out.mp4")
	// Este é o caso REAL deste ambiente: sem FFmpeg.
	indisponivel := &transcoderDeTeste{disponivel: false}
	resultado, err := TranscodeMedia(context.Background(), indisponivel, MediaTranscodeRequest{
		InputPath: arquivoComConteudo(t, dir, "in.mp4", cabecalhoMP4()), OutputPath: saida, TargetFormat: "mp4",
	}, DefaultMediaLimits())
	if !errors.Is(err, ErrMediaTranscoderUnavailable) {
		t.Fatalf("ausência de transcodificador precisa ser DECLARADA: %v", err)
	}
	if resultado.Produced {
		t.Fatal("não pode existir Produced=true sem transcodificador")
	}
	if _, err := os.Stat(saida); err == nil {
		t.Fatal("ARMADILHA: nenhum arquivo de saída pode ser criado quando não há transcodificador")
	}
}

func TestArgvNaoEhLinhaDeComando(t *testing.T) {
	dir := t.TempDir()
	// Nome hostil: se o pipeline concatenasse shell, isto viraria execução de comando.
	saida := filepath.Join(dir, "out.mp4")
	transcoder := &transcoderDeTeste{disponivel: true, escrita: cabecalhoMP4()}
	entrada := filepath.Join(dir, "entrada;rm -rf tudo.mp4")
	if _, err := TranscodeMedia(context.Background(), transcoder, MediaTranscodeRequest{
		InputPath: entrada, OutputPath: saida, TargetFormat: "mp4"}, DefaultMediaLimits()); err != nil {
		t.Fatalf("transcodificação de teste: %v", err)
	}
	if len(transcoder.recebidos) != 1 {
		t.Fatalf("o transcodificador precisa ter recebido exatamente uma chamada: %d", len(transcoder.recebidos))
	}
	// ARMADILHA: o caminho hostil precisa estar em UM argumento, intacto — nunca dividido em pedaços de comando.
	achouInteiro := false
	for _, arg := range transcoder.recebidos[0] {
		if arg == entrada {
			achouInteiro = true
		}
	}
	if !achouInteiro {
		t.Fatalf("o caminho precisa viajar como UM argumento: %v", transcoder.recebidos[0])
	}
}

func TestSaidaInvalidaEhRecusadaERemovida(t *testing.T) {
	dir := t.TempDir()
	saida := filepath.Join(dir, "out.mp4")
	// Transcodificador que "termina bem" mas escreve LIXO — exatamente o "MP4 simulado" que o complemento proíbe.
	mentiroso := &transcoderDeTeste{disponivel: true, escrita: []byte("isto nao e um video")}
	if _, err := TranscodeMedia(context.Background(), mentiroso, MediaTranscodeRequest{
		InputPath: arquivoComConteudo(t, dir, "in.mp4", cabecalhoMP4()), OutputPath: saida, TargetFormat: "mp4"},
		DefaultMediaLimits()); !errors.Is(err, ErrMediaOutputInvalid) {
		t.Fatalf("saída que não é vídeo precisa ser RECUSADA: %v", err)
	}
	if _, err := os.Stat(saida); err == nil {
		t.Fatal("ARMADILHA: a saída inválida precisa ser REMOVIDA do disco, não deixada como resultado")
	}
	// Saída vazia também é recusada (existir não é ser mídia).
	vazio := &transcoderDeTeste{disponivel: true, escrita: []byte{}}
	if _, err := TranscodeMedia(context.Background(), vazio, MediaTranscodeRequest{
		InputPath: arquivoComConteudo(t, dir, "in2.mp4", cabecalhoMP4()), OutputPath: saida, TargetFormat: "mp4"},
		DefaultMediaLimits()); !errors.Is(err, ErrMediaOutputInvalid) {
		t.Fatalf("saída vazia precisa ser recusada: %v", err)
	}
}

func TestTimeoutCancelaELimpa(t *testing.T) {
	dir := t.TempDir()
	saida := filepath.Join(dir, "out.mp4")
	lento := &transcoderDeTeste{disponivel: true, escrita: cabecalhoMP4(), bloquear: 2 * time.Second}
	limites := DefaultMediaLimits()
	limites.Timeout = 50 * time.Millisecond
	inicio := time.Now()
	if _, err := TranscodeMedia(context.Background(), lento, MediaTranscodeRequest{
		InputPath: arquivoComConteudo(t, dir, "in.mp4", cabecalhoMP4()), OutputPath: saida, TargetFormat: "mp4"},
		limites); !errors.Is(err, ErrMediaInvalid) {
		t.Fatalf("estouro de tempo precisa ser recusado: %v", err)
	}
	if decorrido := time.Since(inicio); decorrido > time.Second {
		t.Fatalf("o tempo limite não foi respeitado: %v", decorrido)
	}
	if _, err := os.Stat(saida); err == nil {
		t.Fatal("em falha por tempo a saída parcial precisa ser removida")
	}
}

func TestCancelamentoEhRespeitado(t *testing.T) {
	dir := t.TempDir()
	saida := filepath.Join(dir, "out.mp4")
	bloqueante := &transcoderDeTeste{disponivel: true, escrita: cabecalhoMP4(), bloquear: 5 * time.Second}
	ctx, cancelar := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancelar()
	}()
	inicio := time.Now()
	if _, err := TranscodeMedia(ctx, bloqueante, MediaTranscodeRequest{
		InputPath: arquivoComConteudo(t, dir, "in.mp4", cabecalhoMP4()), OutputPath: saida, TargetFormat: "mp4"},
		DefaultMediaLimits()); !errors.Is(err, ErrMediaInvalid) {
		t.Fatalf("cancelamento precisa ser reportado: %v", err)
	}
	if decorrido := time.Since(inicio); decorrido > time.Second {
		t.Fatalf("o cancelamento não foi ágil: %v", decorrido)
	}
}

func TestTranscodificacaoBemSucedidaExigeSaidaDeVideoReal(t *testing.T) {
	dir := t.TempDir()
	saida := filepath.Join(dir, "out.mp4")
	bom := &transcoderDeTeste{disponivel: true, escrita: cabecalhoMP4()}
	resultado, err := TranscodeMedia(context.Background(), bom, MediaTranscodeRequest{
		InputPath: arquivoComConteudo(t, dir, "in.mp4", cabecalhoMP4()), OutputPath: saida, TargetFormat: "mp4"},
		DefaultMediaLimits())
	if err != nil {
		t.Fatalf("transcodificação válida: %v", err)
	}
	if !resultado.Produced || resultado.MediaType != "video/mp4" || resultado.Bytes <= 0 {
		t.Fatalf("resultado precisa descrever mídia real: %+v", resultado)
	}
	if !strings.Contains(strings.Join(resultado.Args, " "), "-i") {
		t.Fatalf("os argumentos usados precisam ficar registrados: %+v", resultado.Args)
	}
}
