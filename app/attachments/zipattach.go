package attachments

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Erros que o utilizador TEM de ver. Antes desta funcao existir, o caminho do zip em
// app/cmd/app/webview.go fazia apenas slog.Warn quando ReadZipText falhava e continuava com
// report.Entries vazio: o anexo desaparecia sem mensagem nenhuma e o utilizador concluia que
// "o zip quebrou".
var (
	ErrZipNoTextFiles = errors.New("zip sem ficheiros de texto suportados")
)

// ZipAttachment e' um ficheiro de texto extraido de um arquivo, pronto para virar anexo.
type ZipAttachment struct {
	// Name e' o nome apresentado ao utilizador, prefixado com o nome do zip.
	Name string
	Data []byte
}

// ZipAttachmentResult descreve o que saiu do arquivo, incluindo o que foi ignorado.
// Skipped e' obrigatorio na resposta: silencio sobre o que ficou de fora foi o defeito original.
type ZipAttachmentResult struct {
	Attachments []ZipAttachment
	Skipped     []string
	// Reason explica, em texto para o utilizador, porque o arquivo nao produziu anexos.
	Reason string
}

// BuildZipAttachments le o arquivo e devolve os anexos OU um erro explicavel.
//
// Regras (as mesmas de ReadZipText, agora com erro em vez de silencio):
//   - erro de leitura do zip        -> erro com o motivo;
//   - todas as entradas ignoradas   -> ErrZipNoTextFiles com a contagem;
//   - limites excedidos             -> ErrZipLimit (vem de ReadZipText).
func BuildZipAttachments(zipPath string, allowed []string, limits ZipLimits) (ZipAttachmentResult, error) {
	var resultado ZipAttachmentResult

	report, err := ReadZipText(zipPath, allowed, limits)
	if err != nil {
		return resultado, err
	}

	base := filepath.Base(zipPath)
	for _, entrada := range report.Entries {
		resultado.Attachments = append(resultado.Attachments, ZipAttachment{
			Name: base + "/" + entrada.Name,
			Data: entrada.Data,
		})
	}
	resultado.Skipped = report.Skipped

	if len(resultado.Attachments) == 0 {
		// Mensagem accionavel: diz o que foi ignorado e quantos.
		resultado.Reason = fmt.Sprintf("%s nao tem ficheiros de texto suportados (%d entradas ignoradas)",
			base, len(report.Skipped))
		return resultado, fmt.Errorf("%w: %s", ErrZipNoTextFiles, resultado.Reason)
	}

	if len(report.Skipped) > 0 {
		resultado.Reason = fmt.Sprintf("%d entrada(s) ignorada(s) em %s: %s",
			len(report.Skipped), base, resumirIgnorados(report.Skipped, 5))
	}
	return resultado, nil
}

// resumirIgnorados evita despejar centenas de nomes no chat.
func resumirIgnorados(nomes []string, maximo int) string {
	if len(nomes) <= maximo {
		return strings.Join(nomes, ", ")
	}
	return strings.Join(nomes[:maximo], ", ") + fmt.Sprintf(" (+%d)", len(nomes)-maximo)
}
