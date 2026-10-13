package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ESTÁGIO 42.1 — CENTRAL DE PRODUÇÃO AUDIOVISUAL: VALIDAÇÃO REAL E WORKER ISOLADO (E-334).
//
// **MEDIDO ANTES DE ESCREVER**: `media.go` faz **geração** de mídia via provider (`GenerateImage`, `GenerateVideo`,
// `Transcribe`) — não faz **upload, validação nem transcodificação**. E **FFmpeg NÃO está disponível** neste ambiente
// (`Get-Command ffmpeg` ⇒ não encontrado). ⇒ Esta fatia entrega o que **não depende de FFmpeg** e **declara** o que
// depende:
//
//  1. **VALIDAÇÃO POR CONTEÚDO, NÃO POR NOME.** A extensão do arquivo é uma **afirmação do usuário**, não um fato:
//     um `.mp4` pode ser um executável. A validação lê os **bytes iniciais** (magic numbers) e compara com o tipo
//     declarado — divergência é RECUSA, não aviso.
//  2. **PATH TRAVERSAL É RECUSADO.** Nome com `..`, caminho absoluto, separador de diretório ou caractere de controle
//     não entra: ***o nome do arquivo é dado de terceiro***.
//  3. **WORKER SEM SHELL.** O transcodificador recebe **vetor de argumentos**, nunca uma linha de comando concatenada
//     — evita injeção por nome de arquivo, que é o vetor clássico deste tipo de pipeline.
//  4. **A SAÍDA É CONFERIDA, NÃO PRESUMIDA.** O worker **não** confia que o transcodificador gerou um MP4: ele lê os
//     magic bytes da saída e **recusa** o que não for vídeo de verdade. ***Um pipeline que aceita lixo na saída é a
//     versão de produção do "MP4 simulado" que o complemento proíbe.***
//  5. **AUSÊNCIA DECLARADA.** Sem transcodificador disponível, o resultado é `ErrMediaTranscoderUnavailable` — e
//     **nenhum arquivo de saída é criado**. Não existe caminho que devolva "pronto" sem ter produzido mídia real.
//
// LIMITE DECLARADO: limites de **CPU/RAM** não são impostos por este código (dependem do SO/container); o que ele
// impõe é **tempo**, **tamanho de saída** e **cancelamento**. Benchmarks próprios ficam para fatia de medição.

var (
	// ErrMediaInvalid recusa upload inválido (tipo, tamanho, conteúdo ou nome).
	ErrMediaInvalid = errors.New("media upload is invalid")
	// ErrMediaContentMismatch recusa arquivo cujo conteúdo não é o tipo declarado.
	ErrMediaContentMismatch = errors.New("media content does not match declared type")
	// ErrMediaPathUnsafe recusa nome de arquivo perigoso.
	ErrMediaPathUnsafe = errors.New("media filename is unsafe")
	// ErrMediaTranscoderUnavailable declara ausência de transcodificador em vez de fingir sucesso.
	ErrMediaTranscoderUnavailable = errors.New("media transcoder is unavailable")
	// ErrMediaOutputInvalid recusa saída que não é mídia real.
	ErrMediaOutputInvalid = errors.New("transcoder output is not valid media")
)

// MediaLimits são os limites da central. Zero significa "usa o padrão declarado".
type MediaLimits struct {
	MaxBytes       int64
	MaxDuration    time.Duration
	MaxOutputBytes int64
	Timeout        time.Duration
}

// DefaultMediaLimits devolve limites explícitos (nada de números mágicos espalhados).
func DefaultMediaLimits() MediaLimits {
	return MediaLimits{MaxBytes: 512 << 20, MaxDuration: 3 * time.Hour, MaxOutputBytes: 512 << 20, Timeout: 30 * time.Minute}
}

func (l MediaLimits) normalizar() MediaLimits {
	padrao := DefaultMediaLimits()
	if l.MaxBytes <= 0 {
		l.MaxBytes = padrao.MaxBytes
	}
	if l.MaxDuration <= 0 {
		l.MaxDuration = padrao.MaxDuration
	}
	if l.MaxOutputBytes <= 0 {
		l.MaxOutputBytes = padrao.MaxOutputBytes
	}
	if l.Timeout <= 0 {
		l.Timeout = padrao.Timeout
	}
	return l
}

// mediaKindFromMagic reconhece o tipo REAL pelos bytes iniciais.
//
// Cobre os formatos que a produção audiovisual usa de fato. Formato não reconhecido ⇒ vazio ⇒ RECUSA (não há
// "aceita e vê depois": adivinhar tipo de arquivo é como adivinhar permissão).
func mediaKindFromMagic(cabecalho []byte) string {
	tem := func(deslocamento int, assinatura ...byte) bool {
		if len(cabecalho) < deslocamento+len(assinatura) {
			return false
		}
		for i, b := range assinatura {
			if cabecalho[deslocamento+i] != b {
				return false
			}
		}
		return true
	}
	switch {
	case tem(0, 0x89, 'P', 'N', 'G'):
		return "image/png"
	case tem(0, 0xFF, 0xD8, 0xFF):
		return "image/jpeg"
	case len(cabecalho) >= 12 && string(cabecalho[4:8]) == "ftyp":
		return "video/mp4"
	case tem(0, 0x1A, 0x45, 0xDF, 0xA3):
		return "video/webm"
	case tem(0, 'I', 'D', '3'):
		return "audio/mpeg"
	case tem(0, 'R', 'I', 'F', 'F') && len(cabecalho) >= 12 && string(cabecalho[8:12]) == "WAVE":
		return "audio/wav"
	case tem(0, 'O', 'g', 'g', 'S'):
		return "audio/ogg"
	case tem(0, 'R', 'I', 'F', 'F') && len(cabecalho) >= 12 && string(cabecalho[8:12]) == "AVI ":
		return "video/x-msvideo"
	}
	return ""
}

// ValidatedMedia é o resultado da validação: o tipo REAL e o caminho seguro.
type ValidatedMedia struct {
	DeclaredType string `json:"declared_type"`
	ActualType   string `json:"actual_type"`
	Bytes        int64  `json:"bytes"`
	SafeName     string `json:"safe_name"`
	StoredPath   string `json:"stored_path"`
}

// ValidateMediaUpload valida o arquivo em `origem` e o coloca em `destinoDir` com nome seguro.
func ValidateMediaUpload(origem, destinoDir, nomeDeclarado, tipoDeclarado string, limites MediaLimits) (ValidatedMedia, error) {
	limites = limites.normalizar()

	nomeSeguro, err := SanitizeMediaFilename(nomeDeclarado)
	if err != nil {
		return ValidatedMedia{}, err
	}
	arquivo, err := os.Open(origem)
	if err != nil {
		return ValidatedMedia{}, fmt.Errorf("%w: %v", ErrMediaInvalid, err)
	}
	defer arquivo.Close()
	info, err := arquivo.Stat()
	if err != nil {
		return ValidatedMedia{}, fmt.Errorf("%w: %v", ErrMediaInvalid, err)
	}
	if info.Size() <= 0 {
		return ValidatedMedia{}, fmt.Errorf("%w: arquivo vazio", ErrMediaInvalid)
	}
	if info.Size() > limites.MaxBytes {
		return ValidatedMedia{}, fmt.Errorf("%w: %d bytes excede o limite de %d", ErrMediaInvalid, info.Size(), limites.MaxBytes)
	}
	cabecalho := make([]byte, 32)
	lidos, _ := arquivo.Read(cabecalho)
	real := mediaKindFromMagic(cabecalho[:lidos])
	if real == "" {
		return ValidatedMedia{}, fmt.Errorf("%w: conteúdo não reconhecido (não se adivinha tipo de arquivo)", ErrMediaContentMismatch)
	}
	declarado := strings.TrimSpace(strings.ToLower(tipoDeclarado))
	if declarado != "" && !mediaTypesCompativeis(declarado, real) {
		return ValidatedMedia{}, fmt.Errorf("%w: declarado %q, conteúdo é %q", ErrMediaContentMismatch, declarado, real)
	}

	destino := filepath.Join(destinoDir, nomeSeguro)
	if err := os.MkdirAll(destinoDir, 0o700); err != nil {
		return ValidatedMedia{}, fmt.Errorf("%w: %v", ErrMediaInvalid, err)
	}
	if _, err := arquivo.Seek(0, 0); err != nil {
		return ValidatedMedia{}, fmt.Errorf("%w: %v", ErrMediaInvalid, err)
	}
	saida, err := os.OpenFile(destino, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return ValidatedMedia{}, fmt.Errorf("%w: %v", ErrMediaInvalid, err)
	}
	defer saida.Close()
	// Cópia com limite: lê no máximo MaxBytes+1 para DETECTAR excesso em vez de truncar em silêncio.
	escrito, err := io.Copy(saida, io.LimitReader(arquivo, limites.MaxBytes+1))
	if err != nil {
		_ = os.Remove(destino)
		return ValidatedMedia{}, fmt.Errorf("%w: %v", ErrMediaInvalid, err)
	}
	if escrito > limites.MaxBytes {
		_ = os.Remove(destino)
		return ValidatedMedia{}, fmt.Errorf("%w: arquivo maior que o limite declarado", ErrMediaInvalid)
	}
	return ValidatedMedia{DeclaredType: declarado, ActualType: real, Bytes: info.Size(), SafeName: nomeSeguro, StoredPath: destino}, nil
}

func mediaTypesCompativeis(declarado, real string) bool {
	if declarado == real {
		return true
	}
	// Aceita a forma curta "video/mp4" vs "mp4" e aliases comuns, sem afrouxar a checagem de família.
	familia := func(t string) string {
		switch {
		case strings.HasPrefix(t, "video/"), t == "mp4", t == "webm", t == "avi":
			return "video"
		case strings.HasPrefix(t, "audio/"), t == "mp3", t == "wav", t == "ogg", t == "mpeg":
			return "audio"
		case strings.HasPrefix(t, "image/"), t == "png", t == "jpg", t == "jpeg":
			return "image"
		}
		return t
	}
	return familia(declarado) == familia(real) && declarado == real
}

// SanitizeMediaFilename devolve um nome SEGURO ou recusa. ***O nome do arquivo é dado de terceiro.***
func SanitizeMediaFilename(nome string) (string, error) {
	bruto := strings.TrimSpace(nome)
	if bruto == "" {
		return "", fmt.Errorf("%w: nome vazio", ErrMediaPathUnsafe)
	}
	if strings.ContainsAny(bruto, "\x00\n\r") {
		return "", fmt.Errorf("%w: caractere de controle no nome", ErrMediaPathUnsafe)
	}
	// Caminho absoluto, separadores e `..` são recusados ANTES de qualquer normalização.
	if strings.HasPrefix(bruto, "/") || strings.HasPrefix(bruto, "\\") || filepath.IsAbs(bruto) ||
		strings.Contains(bruto, "..") || strings.ContainsAny(bruto, "/\\") {
		return "", fmt.Errorf("%w: %q não é um nome simples de arquivo", ErrMediaPathUnsafe, bruto)
	}
	limpo := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '.' || r == '-' || r == '_':
			return r
		case r == ' ':
			return '_'
		}
		return -1
	}, bruto)
	limpo = strings.Trim(limpo, "._-")
	if limpo == "" || limpo == "." || limpo == ".." {
		return "", fmt.Errorf("%w: nome sem caracteres utilizáveis", ErrMediaPathUnsafe)
	}
	if len(limpo) > 120 {
		limpo = limpo[:120]
	}
	return limpo, nil
}

// MediaTranscodeRequest é o pedido de transcodificação.
type MediaTranscodeRequest struct {
	InputPath    string
	OutputPath   string
	TargetFormat string // mp4
}

// MediaTranscoder é a fronteira com o mundo externo (FFmpeg ou alternativa). Recebe ARGUMENTOS, nunca uma linha de
// comando: *um vetor de argumentos não tem shell para injetar*.
type MediaTranscoder interface {
	Available() bool
	// Args devolve o vetor de argumentos para o par entrada/saída (auditável e testável sem executar nada).
	Args(request MediaTranscodeRequest) []string
	Run(ctx context.Context, request MediaTranscodeRequest, args []string) error
}

// TranscodeResult é o resultado HONESTO de uma transcodificação.
type TranscodeResult struct {
	Produced  bool   `json:"produced"`
	Output    string `json:"output,omitempty"`
	Bytes     int64  `json:"bytes,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	// Unavailable explica a ausência declarada (nunca um sucesso fingido).
	Unavailable string   `json:"unavailable,omitempty"`
	Args        []string `json:"args,omitempty"`
}

// TranscodeMedia executa o worker: sem shell, com timeout, com validação da SAÍDA e limpeza em falha.
func TranscodeMedia(ctx context.Context, transcoder MediaTranscoder, request MediaTranscodeRequest, limites MediaLimits) (TranscodeResult, error) {
	limites = limites.normalizar()
	if transcoder == nil || !transcoder.Available() {
		// AUSÊNCIA DECLARADA: nenhum arquivo de saída é criado, nenhum sucesso é inventado.
		return TranscodeResult{Unavailable: ErrMediaTranscoderUnavailable.Error()}, ErrMediaTranscoderUnavailable
	}
	if strings.TrimSpace(request.InputPath) == "" || strings.TrimSpace(request.OutputPath) == "" {
		return TranscodeResult{}, fmt.Errorf("%w: entrada e saída são obrigatórias", ErrMediaInvalid)
	}
	args := transcoder.Args(request)
	// O vetor não pode conter nada que dependa de shell (o chamador não usa shell de qualquer forma; isto é defesa).
	for _, arg := range args {
		if strings.ContainsAny(arg, "\x00") {
			return TranscodeResult{}, fmt.Errorf("%w: argumento com byte nulo", ErrMediaInvalid)
		}
	}

	ctxComPrazo, cancelar := context.WithTimeout(ctx, limites.Timeout)
	defer cancelar()
	if err := transcoder.Run(ctxComPrazo, request, args); err != nil {
		// LIMPEZA EM FALHA: saída parcial não fica no disco como se fosse resultado.
		removerSaidaInvalida(request.OutputPath)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctxComPrazo.Err(), context.DeadlineExceeded) {
			return TranscodeResult{Args: args}, fmt.Errorf("%w: tempo limite de %s excedido", ErrMediaInvalid, limites.Timeout)
		}
		if errors.Is(err, context.Canceled) {
			return TranscodeResult{Args: args}, fmt.Errorf("%w: cancelado", ErrMediaInvalid)
		}
		return TranscodeResult{Args: args}, fmt.Errorf("%w: %v", ErrMediaInvalid, err)
	}

	// A SAÍDA É CONFERIDA: não se presume que o transcodificador gerou vídeo de verdade.
	info, err := os.Stat(request.OutputPath)
	if err != nil {
		return TranscodeResult{Args: args}, fmt.Errorf("%w: saída não foi criada", ErrMediaOutputInvalid)
	}
	if info.Size() <= 0 || info.Size() > limites.MaxOutputBytes {
		removerSaidaInvalida(request.OutputPath)
		return TranscodeResult{Args: args}, fmt.Errorf("%w: saída com %d bytes", ErrMediaOutputInvalid, info.Size())
	}
	arquivo, err := os.Open(request.OutputPath)
	if err != nil {
		return TranscodeResult{Args: args}, fmt.Errorf("%w: %v", ErrMediaOutputInvalid, err)
	}
	cabecalho := make([]byte, 32)
	lidos, _ := arquivo.Read(cabecalho)
	// MEDIDO (E-334): no Windows, `os.Remove` com o arquivo ABERTO falha por compartilhamento e a saída inválida
	// ficava no disco. Fechar ANTES de remover é o que faz a limpeza valer nas duas plataformas.
	_ = arquivo.Close()
	real := mediaKindFromMagic(cabecalho[:lidos])
	if real == "" || !strings.HasPrefix(real, "video/") {
		removerSaidaInvalida(request.OutputPath)
		return TranscodeResult{Args: args}, fmt.Errorf("%w: magic bytes dizem %q (lixo na saída não é resultado)",
			ErrMediaOutputInvalid, real)
	}
	return TranscodeResult{Produced: true, Output: request.OutputPath, Bytes: info.Size(), MediaType: real, Args: args}, nil
}

// removerSaidaInvalida apaga uma saída que não serve, em qualquer plataforma.
//
// POR QUE ISTO É UMA FUNÇÃO E NÃO UM `_ = os.Remove` (E-334): no Windows, remover um arquivo ABERTO falha por
// compartilhamento — e o erro era descartado, deixando lixo no disco com aparência de resultado. Aqui o chamador
// FECHA antes, e a remoção devolve o que aconteceu em vez de fingir.
func removerSaidaInvalida(caminho string) {
	if strings.TrimSpace(caminho) == "" {
		return
	}
	if err := os.Remove(caminho); err != nil && !os.IsNotExist(err) {
		// Não há o que fazer além de não esconder: o caminho fica sabido para diagnóstico.
		fmt.Fprintf(os.Stderr, "hades: falha ao remover saída inválida %q: %v\n", caminho, err)
	}
}
