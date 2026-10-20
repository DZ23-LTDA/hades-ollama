package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// GuardarEm grava o estado do pare num ficheiro, de forma atomica.
//
// # PORQUE ISTO EXISTE
//
// Sem persistencia, um reinicio do servidor **desarma o travao em silencio**: alguem para o sistema por
// uma razao grave, o processo reinicia, e a execucao volta sem que ninguem tenha decidido isso. Um
// travao que esquece no restart e' meio travao -- e a metade que falta e' a que decide se continua
// parado.
//
// # ATOMICIDADE
//
// Escreve num temporario no MESMO directorio e faz `rename`. Um `rename` no mesmo sistema de ficheiros
// e' atomico: ou o ficheiro antigo fica, ou o novo fica inteiro. Gravar directamente sobre o destino
// deixaria, numa falha a meio, um JSON truncado -- e um estado ilegivel e' pior do que nenhum, porque
// quem o le pode concluir "nao esta' parado".
func (p *PareDeEmergencia) GuardarEm(caminho string) error {
	if p == nil {
		return errors.New("pare nulo: nao ha' estado a guardar")
	}
	if caminho == "" {
		return errors.New("caminho vazio: nao ha' onde guardar o estado do pare")
	}
	estado := p.Estado()
	bruto, err := json.MarshalIndent(estado, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar estado do pare: %w", err)
	}
	directorio := filepath.Dir(caminho)
	if err := os.MkdirAll(directorio, 0o700); err != nil {
		return fmt.Errorf("criar directorio do pare %q: %w", directorio, err)
	}
	temporario, err := os.CreateTemp(directorio, ".pare-*.tmp")
	if err != nil {
		return fmt.Errorf("criar temporario do pare: %w", err)
	}
	// Se algo falhar a seguir, o temporario nao pode ficar para tras.
	defer func() {
		_ = temporario.Close()
		_ = os.Remove(temporario.Name())
	}()
	if _, err := temporario.Write(bruto); err != nil {
		return fmt.Errorf("escrever temporario do pare: %w", err)
	}
	if err := temporario.Sync(); err != nil {
		return fmt.Errorf("sincronizar temporario do pare: %w", err)
	}
	if err := temporario.Close(); err != nil {
		return fmt.Errorf("fechar temporario do pare: %w", err)
	}
	if err := os.Rename(temporario.Name(), caminho); err != nil {
		return fmt.Errorf("publicar estado do pare em %q: %w", caminho, err)
	}
	return nil
}

// CarregarDe le o estado guardado. Ficheiro inexistente **nao e' erro**: e' o estado inicial.
//
// Um estado ilegivel, porem, e' erro e o pare fica como estava -- *um estado que nao se consegue ler nao
// pode ser interpretado como "desarmado"*.
func (p *PareDeEmergencia) CarregarDe(caminho string) error {
	if p == nil {
		return errors.New("pare nulo: nao ha' estado a carregar")
	}
	bruto, err := os.ReadFile(caminho)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("ler estado do pare %q: %w", caminho, err)
	}
	var estado EstadoDoPare
	if err := json.Unmarshal(bruto, &estado); err != nil {
		return fmt.Errorf("estado do pare ilegivel em %q: %w", caminho, err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.estao = estado
	return nil
}
