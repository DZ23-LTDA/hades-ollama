package agent

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// HISTÓRICO BRUTO IMUTÁVEL (§2 do complemento universal de memória).
//
// POR QUE ESTA PEÇA VEM PRIMEIRO: o §2 manda «persistir primeiro o histórico bruto
// completo e imutável/auditável, DEPOIS gravar resumo versionado». Medido: o
// `ContextStore` só tem `AddMemory*` — **não existe** persistência de conversa, evento
// ou transcrição. Sem este substrato, "resumir" seria resumir algo que ninguém guardou,
// e "recuperar" seria impossível por construção. É também por isso que os compactadores
// que existem são stateless.
//
// AS TRÊS PROPRIEDADES, E COMO CADA UMA É OBTIDA SEM CONFIANÇA NO CHAMADOR:
//  1. APPEND-ONLY: o ficheiro é aberto com O_APPEND e cada registo é UMA linha. Nada é
//     reescrito nem apagado — não há API de update nem de delete, de propósito.
//  2. ENCADEADO: cada registo guarda o hash do anterior e o seu próprio. Alterar uma
//     linha antiga quebra a cadeia a partir dali, e `VerificarCadeia` diz ONDE.
//  3. RETOMÁVEL: abrir de novo continua a numeração e a cadeia a partir do que está em
//     disco. Um processo que morreu não reinicia a história nem a duplica.
//
// LIMITE DECLARADO: a cadeia deteta alteração e reordenação, NÃO impede o truncamento
// do fim do ficheiro (remover as últimas linhas deixa a cadeia válida). Detetar isso
// exige uma testemunha externa (hash publicado noutro sítio) — fica por fazer, e é
// preferível declará-lo a chamar a isto "prova de integridade".

// RegistroDeHistorico é uma unidade imutável do histórico. Depois de gravado, não muda.
type RegistroDeHistorico struct {
	Seq       int       `json:"seq"`
	ProjetoID string    `json:"projeto_id,omitempty"`
	MissaoID  string    `json:"missao_id,omitempty"`
	Papel     string    `json:"papel"`
	Conteudo  string    `json:"conteudo"`
	Quando    time.Time `json:"quando"`
	// HashAnterior é "" no primeiro registo: a origem da cadeia é explícita.
	HashAnterior string `json:"hash_anterior"`
	Hash         string `json:"hash"`
}

// HistoricoBruto é o ficheiro JSONL append-only, com uma cadeia de hashes.
type HistoricoBruto struct {
	mu         sync.Mutex
	caminho    string
	arquivo    *os.File
	seq        int
	ultimoHash string
}

// AbrirHistoricoBruto abre (ou cria) o histórico e RETOMA a cadeia do que está em
// disco. Se uma linha estiver corrompida, a abertura falha em vez de a saltar: um
// histórico que ignora silenciosamente uma linha má não é auditável.
func AbrirHistoricoBruto(caminho string) (*HistoricoBruto, error) {
	caminho = strings.TrimSpace(caminho)
	if caminho == "" {
		return nil, errors.New("caminho do histórico é obrigatório")
	}
	if dir := filepath.Dir(caminho); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("criar diretório do histórico: %w", err)
		}
	}
	h := &HistoricoBruto{caminho: caminho}
	registos, err := h.lerFicheiro()
	if err != nil {
		return nil, err
	}
	if len(registos) > 0 {
		if err := VerificarCadeia(registos); err != nil {
			return nil, err
		}
		ultimo := registos[len(registos)-1]
		h.seq = ultimo.Seq
		h.ultimoHash = ultimo.Hash
	}
	arquivo, err := os.OpenFile(caminho, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("abrir histórico para acrescentar: %w", err)
	}
	h.arquivo = arquivo
	return h, nil
}

// Anexar grava UM registo e devolve-o preenchido (Seq, HashAnterior e Hash). O
// chamador não decide o Seq nem o hash — quem decide é o histórico, porque é ele que
// conhece a cadeia.
func (h *HistoricoBruto) Anexar(reg RegistroDeHistorico) (RegistroDeHistorico, error) {
	if strings.TrimSpace(reg.Papel) == "" {
		return RegistroDeHistorico{}, errors.New("papel é obrigatório no registo de histórico")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.arquivo == nil {
		return RegistroDeHistorico{}, errors.New("histórico fechado")
	}
	reg.Seq = h.seq + 1
	reg.HashAnterior = h.ultimoHash
	if reg.Quando.IsZero() {
		reg.Quando = time.Now().UTC()
	}
	reg.Hash = HashDoRegistro(reg)
	linha, err := json.Marshal(reg)
	if err != nil {
		return RegistroDeHistorico{}, err
	}
	if _, err := h.arquivo.Write(append(linha, '\n')); err != nil {
		return RegistroDeHistorico{}, fmt.Errorf("gravar registo %d: %w", reg.Seq, err)
	}
	// Sync explícito: o §2 exige histórico durável ANTES do resumo. Sem isto, um
	// resumo podia sobreviver a uma queda e o material que ele cobre não.
	if err := h.arquivo.Sync(); err != nil {
		return RegistroDeHistorico{}, fmt.Errorf("sincronizar registo %d: %w", reg.Seq, err)
	}
	h.seq = reg.Seq
	h.ultimoHash = reg.Hash
	return reg, nil
}

// LerTudo devolve o histórico inteiro, na ordem. É a operação que torna possível
// "reconstruir a partir das fontes originais" (§2).
func (h *HistoricoBruto) LerTudo() ([]RegistroDeHistorico, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lerFicheiro()
}

// Fechar liberta o ficheiro. O histórico em disco permanece.
func (h *HistoricoBruto) Fechar() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.arquivo == nil {
		return nil
	}
	err := h.arquivo.Close()
	h.arquivo = nil
	return err
}

func (h *HistoricoBruto) lerFicheiro() ([]RegistroDeHistorico, error) {
	arquivo, err := os.Open(h.caminho)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer arquivo.Close()
	var registos []RegistroDeHistorico
	leitor := bufio.NewScanner(arquivo)
	leitor.Buffer(make([]byte, 0, 64*1024), 8<<20)
	numero := 0
	for leitor.Scan() {
		numero++
		linha := strings.TrimSpace(leitor.Text())
		if linha == "" {
			continue
		}
		var reg RegistroDeHistorico
		if err := json.Unmarshal([]byte(linha), &reg); err != nil {
			return nil, fmt.Errorf("linha %d do histórico está corrompida: %w", numero, err)
		}
		registos = append(registos, reg)
	}
	if err := leitor.Err(); err != nil {
		return nil, err
	}
	return registos, nil
}

// HashDoRegistro é o digest canónico do registo. Usa separadores explícitos para que
// dois registos diferentes não possam produzir a mesma string concatenada (o mesmo
// cuidado do envelope do E-561, e pela mesma razão).
func HashDoRegistro(reg RegistroDeHistorico) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s",
		reg.Seq, reg.ProjetoID, reg.MissaoID, reg.Papel,
		reg.Quando.UTC().Format(time.RFC3339Nano), reg.HashAnterior, reg.Conteudo)
	soma := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(soma[:])
}

// VerificarCadeia confere seq crescente, encadeamento e hash de cada registo. Devolve
// o PRIMEIRO ponto onde a cadeia quebra — dizendo ONDE, não só que quebrou.
func VerificarCadeia(registos []RegistroDeHistorico) error {
	anterior := ""
	for i, reg := range registos {
		if reg.Seq != i+1 {
			return fmt.Errorf("histórico fora de ordem na posição %d: seq=%d (esperado %d)", i, reg.Seq, i+1)
		}
		if reg.HashAnterior != anterior {
			return fmt.Errorf("cadeia quebrada no seq %d: hash_anterior=%q, esperado %q", reg.Seq, reg.HashAnterior, anterior)
		}
		if esperado := HashDoRegistro(reg); reg.Hash != esperado {
			return fmt.Errorf("conteúdo alterado no seq %d: hash gravado %q difere do recalculado %q", reg.Seq, reg.Hash, esperado)
		}
		anterior = reg.Hash
	}
	return nil
}
