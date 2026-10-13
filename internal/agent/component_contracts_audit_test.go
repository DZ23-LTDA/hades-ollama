package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// E-278 — AUDITORIA DE PRESENÇA DOS CONTRATOS. O limite declarado em E-276/E-277 era "o registro DESCREVE, não
// AUDITA". Esta camada audita a EXISTÊNCIA do ponto de entrada de cada componente.
//
// ESCOPO HONESTO: NÃO é prova de compatibilidade semântica. Dizer "conformidade verificada" tendo checado só
// existência de arquivo inflaria a evidência.
//
// Propriedades travadas:
//  1. todo componente tem ponto de entrada DECLARADO (sem ele a auditoria não pode dizer "presente");
//  2. os 16 pontos de entrada existem de fato no pacote;
//  3. a auditoria ACUSA ausência — uma auditoria que só sabe confirmar não audita nada.
func TestAuditoriaDePresencaDosContratosAcusaPontoDeEntradaAusente(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "contratos")
	registro, err := NewContractRegistry(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SeedCanonicalContracts(registro, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	// 1 e 2. Os 16 pontos de entrada EXISTEM no pacote real.
	relatorio, err := AuditContractPresence(registro)
	if err != nil {
		t.Fatal(err)
	}
	if relatorio.Audited != len(registro.Contracts()) {
		t.Fatalf("a auditoria precisa medir TODOS os contratos: mediu %d de %d", relatorio.Audited, len(registro.Contracts()))
	}
	// CONSISTÊNCIA mapa x auditoria: todo componente COM ponto declarado precisa aparecer como presente; todo
	// componente SEM ponto declarado precisa ser ACUSADO. Nada é presumido.
	pontos := ComponentEntryPoints()
	for _, contrato := range registro.Contracts() {
		achado := ContractPresence{}
		for _, f := range relatorio.Findings {
			if f.Component == contrato.Component {
				achado = f
			}
		}
		_, declarado := pontos[contrato.Component]
		if declarado && !achado.Present {
			t.Fatalf("componente com ponto de entrada declarado (%s -> %s) precisa estar PRESENTE: %+v", contrato.Component, pontos[contrato.Component], achado)
		}
		if !declarado && achado.Present {
			t.Fatalf("componente SEM ponto de entrada declarado não pode ser presumido presente: %s", contrato.Component)
		}
	}
	// INVARIANTE (E-279): os 16 canônicos têm ponto de entrada DECLARADO e PRESENTE. Renomear ou remover qualquer
	// um deles quebra este teste — a auditoria deixa de ser relatório e passa a ser trava.
	if len(pontos) != len(CanonicalComponentIDs()) {
		t.Fatalf("os %d canônicos precisam ter ponto de entrada declarado, há %d", len(CanonicalComponentIDs()), len(pontos))
	}
	if len(relatorio.Missing) != 0 {
		t.Fatalf("nenhum componente canônico pode ficar sem ponto de entrada verificado: %+v", relatorio.Missing)
	}
	if relatorio.Present != len(CanonicalComponentIDs()) {
		t.Fatalf("os %d pontos precisam existir no disco, presentes=%d", len(CanonicalComponentIDs()), relatorio.Present)
	}

	// 3. CAMINHO NEGATIVO: arquivo inexistente é ACUSADO.
	ausente := &ContractRegistry{contracts: map[string]ComponentContract{
		"hades_core": {Component: "hades_core", Version: "1.0", Owner: "cto"},
	}}
	negativo, err := auditContractPresenceAt(ausente, map[string]string{"hades_core": "arquivo_que_nao_existe.go"}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if negativo.Present != 0 || len(negativo.Missing) != 1 {
		t.Fatalf("ponto de entrada inexistente precisa ser ACUSADO: %+v", negativo)
	}

	// 4. Componente SEM ponto declarado também é acusado: "não sei" não vira "presente".
	semMapa := &ContractRegistry{contracts: map[string]ComponentContract{
		"hades_core": {Component: "hades_core", Version: "1.0", Owner: "cto"},
	}}
	cego, err := auditContractPresenceAt(semMapa, map[string]string{}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cego.Present != 0 || len(cego.Missing) != 1 {
		t.Fatalf("componente sem ponto declarado precisa ser acusado, não presumido presente: %+v", cego)
	}
}
