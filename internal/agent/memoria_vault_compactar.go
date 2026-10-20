package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	ErrNadaACompactar      = errors.New("nothing to compact: no readable notes in the tenant")
	ErrCompactacaoInsegura = errors.New("compaction refused: destination or archive collides with a source")
)

// PastaDeArquivo e' onde as notas originais passam a viver depois da compactacao. Elas NAO sao
// apagadas: compactar preserva o bruto (o resumo e' um indice, nao um substituto).
const PastaDeArquivo = "arquivo"

// UmaFonte e' o que o resumo cita: identificador, caminho e hash do que foi compactado.
type UmaFonte struct {
	MemoryID    string
	Relativo    string
	ContentHash string
	ArquivadaEm string
}

// RelatorioDeCompactacao e' a prova do que aconteceu.
type RelatorioDeCompactacao struct {
	Fontes          []UmaFonte
	ResumoRelativo  string
	PastaDeArquivo  string
	TotalCompactado int
}

// CompactacaoGovernada move notas para o arquivo e escreve um resumo que as CITA.
// Nao resume conteudo por modelo: recebe o texto do resumo do chamador, para nao inventar nada.
type CompactacaoGovernada struct {
	Raiz     string
	Politica PoliticaDeAcesso
	Agora    func() time.Time
}

type PedidoDeCompactacao struct {
	Identidade     string
	Tenant         string
	Pasta          string // subpasta a compactar, relativa a' raiz (ex.: "notas")
	ResumoRelativo string // onde escrever o resumo (ex.: "notas/RESUMO.md")
	TextoDoResumo  string
}

func (c CompactacaoGovernada) agora() time.Time {
	if c.Agora != nil {
		return c.Agora()
	}
	return time.Now().UTC()
}

// Compactar e' NAO DESTRUTIVA: cada nota do tenant e' movida para `arquivo/<instante>/`, com os
// mesmos bytes, e o resumo lista memory_id, caminho e hash de cada uma. Qualquer colisao entre
// origem, resumo e arquivo recusa a operacao ANTES de mover nada.
func (c CompactacaoGovernada) Compactar(p PedidoDeCompactacao) (RelatorioDeCompactacao, error) {
	if strings.TrimSpace(c.Raiz) == "" {
		return RelatorioDeCompactacao{}, ErrRaizDoVaultAusente
	}
	identidade := strings.TrimSpace(p.Identidade)
	if identidade == "" {
		return RelatorioDeCompactacao{}, ErrIdentidadeDeSessaoAusente
	}
	tenant := strings.TrimSpace(p.Tenant)
	if tenant == "" {
		return RelatorioDeCompactacao{}, ErrTenantDaSessaoAusente
	}
	if err := c.Politica.Autorizar(identidade, EscopoTenant, PermissaoEscrever); err != nil {
		return RelatorioDeCompactacao{}, err
	}
	pasta := strings.TrimSpace(p.Pasta)
	if pasta == "" {
		return RelatorioDeCompactacao{}, fmt.Errorf("%w: pasta vazia", ErrForaDaRaiz)
	}
	resumoRel := strings.TrimSpace(p.ResumoRelativo)
	if resumoRel == "" {
		return RelatorioDeCompactacao{}, fmt.Errorf("%w: resumo sem destino", ErrForaDaRaiz)
	}

	pastaAbs, err := CaminhoAutorizado(c.Raiz, pasta)
	if err != nil {
		return RelatorioDeCompactacao{}, err
	}
	resumoAbs, err := CaminhoAutorizado(c.Raiz, resumoRel)
	if err != nil {
		return RelatorioDeCompactacao{}, err
	}
	instante := c.agora()
	arquivoRel := filepath.ToSlash(filepath.Join(pasta, PastaDeArquivo, instante.Format("20060102T150405Z")))
	arquivoAbs, err := CaminhoAutorizado(c.Raiz, arquivoRel)
	if err != nil {
		return RelatorioDeCompactacao{}, err
	}

	entradas, err := os.ReadDir(pastaAbs)
	if err != nil {
		return RelatorioDeCompactacao{}, fmt.Errorf("%w: %v", ErrNadaACompactar, err)
	}

	fontes := make([]UmaFonte, 0, len(entradas))
	for _, entrada := range entradas {
		if entrada.IsDir() || strings.HasSuffix(entrada.Name(), ".lock") {
			continue
		}
		relativo := filepath.ToSlash(filepath.Join(pasta, entrada.Name()))
		caminho := filepath.Join(pastaAbs, entrada.Name())
		if caminho == resumoAbs {
			continue // o resumo nao compacta a si mesmo
		}
		dados, errLeitura := os.ReadFile(caminho)
		if errLeitura != nil {
			continue
		}
		meta, _, errNota := LerNota(dados)
		if errNota != nil {
			continue // nao se move o que nao se compreende
		}
		if strings.TrimSpace(meta.Tenant) != tenant {
			continue // nunca se move nota de outro tenant
		}
		fontes = append(fontes, UmaFonte{
			MemoryID:    meta.MemoryID,
			Relativo:    relativo,
			ContentHash: meta.ContentHash,
			ArquivadaEm: filepath.ToSlash(filepath.Join(arquivoRel, entrada.Name())),
		})
	}
	if len(fontes) == 0 {
		return RelatorioDeCompactacao{}, ErrNadaACompactar
	}
	sort.Slice(fontes, func(i, j int) bool { return fontes[i].Relativo < fontes[j].Relativo })

	if err := os.MkdirAll(arquivoAbs, 0o700); err != nil {
		return RelatorioDeCompactacao{}, err
	}

	// Mover PRIMEIRO (bytes identicos) e so' depois escrever o resumo: se algo falhar a meio, o
	// bruto ja' esta' no arquivo e nenhuma nota se perde.
	for i := range fontes {
		origem := filepath.Join(c.Raiz, filepath.FromSlash(fontes[i].Relativo))
		destino := filepath.Join(c.Raiz, filepath.FromSlash(fontes[i].ArquivadaEm))
		if origem == destino {
			return RelatorioDeCompactacao{}, ErrCompactacaoInsegura
		}
		if _, statErr := os.Stat(destino); statErr == nil {
			return RelatorioDeCompactacao{}, ErrCompactacaoInsegura
		}
		if err := os.Rename(origem, destino); err != nil {
			return RelatorioDeCompactacao{}, err
		}
	}

	cabeca := fmt.Sprintf("# Resumo de compactacao (%s)\n\nInstante: %s\nTenant: %s\nNotas compactadas: %d\n\n",
		pasta, instante.Format(time.RFC3339), tenant, len(fontes))
	var corpo strings.Builder
	corpo.WriteString(cabeca)
	if texto := strings.TrimSpace(p.TextoDoResumo); texto != "" {
		corpo.WriteString(texto + "\n\n")
	}
	corpo.WriteString("## Fontes\n\n")
	for _, f := range fontes {
		corpo.WriteString(fmt.Sprintf("- `%s` | %s | hash=%s | arquivada em `%s`\n",
			f.MemoryID, f.Relativo, f.ContentHash, f.ArquivadaEm))
	}
	if err := os.MkdirAll(filepath.Dir(resumoAbs), 0o700); err != nil {
		return RelatorioDeCompactacao{}, err
	}
	if err := os.WriteFile(resumoAbs, []byte(corpo.String()), 0o600); err != nil {
		return RelatorioDeCompactacao{}, err
	}

	return RelatorioDeCompactacao{
		Fontes:          fontes,
		ResumoRelativo:  filepath.ToSlash(resumoRel),
		PastaDeArquivo:  arquivoRel,
		TotalCompactado: len(fontes),
	}, nil
}
