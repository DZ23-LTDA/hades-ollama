package agent

import (
	"sort"
	"strings"
)

// E-542 (Estagio 47 - Universal Artifact Drive): a DEDUPLICACAO.
//
// ***A PERGUNTA QUE ESTE FICHEIRO RESPONDE, E A PERGUNTA QUE ELE **RECUSA** RESPONDER.***
//
// Responde: *«dentro desta organizacao, que ficheiros sao o **mesmo conteudo** em sitios diferentes?»* - e para isso
// agrupa pelo `SHA256`, que e' a unica prova de conteudo que o `E-531`/`E-539` aceitam (o **tamanho** nao decide).
//
// ***E RECUSA RESPONDER:*** *«esta organizacao tem o mesmo ficheiro que **outra**?»*
//
// ⇒ E recusa **por desenho, nao por falta de tempo**: ***saber que outro tenant tem o mesmo ficheiro e' INFORMACAO sobre
// esse tenant*** - *diz que ele tem um contrato, uma minuta, um binario especifico, e ate' uma copia de um ficheiro
// nosso.* *Uma funcionalidade de «dedup entre organizacoes» seria, na pratica, um **canal lateral** entre clientes* ✗✓✓.
//
// *E' a mesma fronteira do `E-531`, do `E-537` e do `E-539`, no quinto sitio onde ela se afirma — e a primeira onde
// ela **recusa uma funcionalidade** em vez de recusar um acesso.*
//
// ***LIMITE DECLARADO.*** O agrupamento e' por `SHA256`: dois ficheiros com o mesmo conteudo e o **mesmo hash** sao
// duplicados; dois ficheiros **parecidos** (uma versao com uma linha a mais) **nao** sao - e *deteccao de similaridade e'
// outro problema, com outros custos, e nao se finge faze-lo aqui.*

// DuplicadoDeArtefacto e' um grupo de caminhos com **o mesmo conteudo** (`SHA256`) dentro de um tenant.
type DuplicadoDeArtefacto struct {
	SHA256 string `json:"sha256"`
	// Caminhos vem ordenados, para que a mesma entrada produza sempre a mesma saida.
	Caminhos []string `json:"caminhos"`
	// Versoes diz, por caminho, qual a versao corrente - *para que um duplicado em versoes diferentes seja visivel.*
	Versoes map[string]int `json:"versoes"`
	// TamanhoBytes e' o tamanho de UMA copia. DesperdicioBytes e' o que se pouparia se so' ficasse uma.
	TamanhoBytes     int64 `json:"tamanho_bytes"`
	DesperdicioBytes int64 `json:"desperdicio_bytes"`
}

// ArtefactoDetetarDuplicados agrupa o conteudo repetido **dentro de um tenant**, e **apenas o que o sujeito pode ver**.
//
// *Os dois filtros, e a ordem deles, sao a mesma disciplina dos ficheiros anteriores:* **visibilidade primeiro**, e o
// agrupamento so' sobre o que sobrou. *Assim, a lista de duplicados de alguem **nunca** revela a existencia de um
// ficheiro que essa pessoa nao podia ver - nem sequer indirectamente, pelo tamanho do grupo.*
//
// Um artefacto **sem `SHA256`** e' **ignorado**: *nao se agrupa por «nao sei» - dois desconhecidos nao sao o mesmo.*
func ArtefactoDetetarDuplicados(indice []ArtifactManifest, tenantID, sujeito string) []DuplicadoDeArtefacto {
	// Visibilidade primeiro - e o tenant afirmado outra vez, como nos outros ficheiros.
	visiveis := make([]ArtifactManifest, 0, len(indice))
	for _, m := range indice {
		if !ArtifactVisivelPara(m, tenantID, sujeito) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(m.TenantID), strings.TrimSpace(tenantID)) {
			continue
		}
		if strings.TrimSpace(m.SHA256) == "" {
			// *Ignorado, e nao agrupado num balde «vazio»: dois ficheiros sem hash conhecido nao sao o mesmo ficheiro.*
			continue
		}
		visiveis = append(visiveis, m)
	}

	// Agrupa por hash.
	grupos := map[string][]ArtifactManifest{}
	for _, m := range visiveis {
		grupos[m.SHA256] = append(grupos[m.SHA256], m)
	}

	saida := make([]DuplicadoDeArtefacto, 0)
	for sha, membros := range grupos {
		if len(membros) < 2 {
			continue // *um so' nao e' duplicado.*
		}
		d := DuplicadoDeArtefacto{SHA256: sha, Versoes: map[string]int{}}
		for _, m := range membros {
			d.Caminhos = append(d.Caminhos, m.Path)
			// *Guarda a MAIOR versao por caminho, para que o numero mostrado seja o corrente.*
			if atual, existe := d.Versoes[m.Path]; !existe || m.Version > atual {
				d.Versoes[m.Path] = m.Version
			}
			d.TamanhoBytes = m.Size // *o tamanho e' o do conteudo, logo igual em todas as copias.*
		}
		sort.Strings(d.Caminhos)
		// *Desperdicio = (copias - 1) x tamanho. E' o que se pouparia se so' ficasse uma - **e nao uma recomendacao de
		// apagar**: qual das copias fica e' decisao de quem conhece o uso.*
		d.DesperdicioBytes = d.TamanhoBytes * int64(len(d.Caminhos)-1)
		saida = append(saida, d)
	}

	// Ordem determinista: maior desperdicio primeiro, e o hash como desempate.
	sort.SliceStable(saida, func(i, j int) bool {
		if saida[i].DesperdicioBytes == saida[j].DesperdicioBytes {
			return saida[i].SHA256 < saida[j].SHA256
		}
		return saida[i].DesperdicioBytes > saida[j].DesperdicioBytes
	})
	return saida
}
