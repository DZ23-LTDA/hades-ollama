package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// E-972: O FORMATO ABERTO `hades-plugin.json` — versao 1.0.
//
// **PORQUE EXISTE**: um plugin do HADES precisa de declarar O QUE OFERECE ao utilizador
// (instrucoes, capacidades, apresentacao) sem transformar isso no nucleo executavel. O
// `PluginManifest` canonico resolve identidade, assinatura, scopes e entrypoint; nao
// resolve "o que este plugin ensina" nem "como se chama na interface".
//
// **E O QUE O HADES *NAO* COPIOU**: ao ler os quatro adaptadores de plugin das fontes do
// adendo (`plugin.json`, `.claude-plugin/`, `.codex-plugin/`), a diferenca ficou medida —
// nenhum deles tem `signing_key_id`, `content_sha256` nem `signature`. *O formato dos
// outros descreve; o do HADES comprova.* Este formato mantem essa vantagem e acrescenta
// as duas secoes que faltavam.
//
// **O QUE ELE REUTILIZA**: o nucleo (`id`, `version`, `name`, `kind`, `scopes`,
// `dependencies`, ...) e' validado pelo **mesmo** `validatePluginManifest` que ja' existia
// (`plugin_registry.go:183`). *Um segundo validador para o mesmo contrato e' como se
// comeca a divergir* — e seria a duplicacao que o `§ESTAGIO 3` proibe.

// HadesPluginFormatVersion e' a versao do FORMATO (nao do plugin).
const HadesPluginFormatVersion = "1.0"

// HadesPluginSchemaID identifica o schema publico deste formato.
const HadesPluginSchemaID = "https://hades.dev/schemas/hades-plugin-1.0.json"

// Erros do formato. *Tipados, para que a recusa seja programavel e nao uma string.*
var (
	ErrHadesPluginFormat    = errors.New("hades-plugin: formato invalido")
	ErrHadesPluginVersion   = errors.New("hades-plugin: format_version nao suportada")
	ErrHadesPluginSkillPath = errors.New("hades-plugin: caminho de skill invalido")
	ErrHadesPluginInterface = errors.New("hades-plugin: interface invalida")
)

// safeSkillNamePattern espelha a regra de nome ja' usada nos plugins: minusculas,
// digitos e hifen, comecando por alfanumerico.
var safeSkillNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// safeRelativePath recusa caminho absoluto, travessia e separador de Windows.
//
// **E-972, e um erro meu que o teste apanhou ANTES do commit**: a primeira versao usou
// `^(?!/)(?!.*\.\.)[^\\]+$` — e o `regexp` do **Go e' RE2, que NAO suporta lookahead**.
// *O `MustCompile` entrou em PANICO no arranque do pacote*, o que derrubaria o
// `internal/agent` inteiro no CI.
//
// ⇒ ***«raciocinei em sintaxe PCRE, e o motor era RE2»*** — e o sintoma nao foi um erro
// devolvido, foi um panico. A cura e' **codigo explicito**, que o RE2 le' e um humano
// audita sem decorar dialetos de regex.
//
// *A mesma defesa que `preparePluginWorkspace` aplica na execucao* — mas aqui, mais cedo:
// um manifesto que aponta para fora do bundle e' recusado na **instalacao**.
func safeRelativePath(path string) bool {
	if path == "" {
		return false
	}
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, `\`) {
		return false // absoluto
	}
	if strings.Contains(path, `\`) {
		return false // separador de Windows: exigimos `/` para o caminho ser portavel
	}
	if strings.Contains(path, "//") {
		return false // segmento vazio, que alguns sistemas colapsam de formas diferentes
	}
	for _, segmento := range strings.Split(path, "/") {
		if segmento == ".." {
			return false // travessia
		}
	}
	if len(path) >= 2 && path[1] == ':' {
		return false // drive letter: absoluto disfarcado
	}
	return true
}

// HadesPluginSkill e' uma instrucao que o plugin oferece.
//
// *O carregamento e' sob demanda*: declarar 95 skills no manifesto **nao** as carrega no
// contexto — o que o adendo mediu nas fontes (`googleworkspace/cli` tem 95) e o que o
// `skill_activation.go` do HADES ja' implementa por `TRIAGADOS`/activacao.
type HadesPluginSkill struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

// HadesPluginInterface sao os metadados de APRESENTACAO.
//
// *Separados do nucleo de proposito*: um plugin pode mudar de nome visivel sem mudar de
// identidade, e sem invalidar a assinatura do conteudo.
type HadesPluginInterface struct {
	DisplayName      string   `json:"display_name,omitempty"`
	ShortDescription string   `json:"short_description,omitempty"`
	LongDescription  string   `json:"long_description,omitempty"`
	Category         string   `json:"category,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	DefaultPrompt    []string `json:"default_prompt,omitempty"`
	Icon             string   `json:"icon,omitempty"`
}

// HadesPluginManifest e' o `hades-plugin.json` completo.
//
// O nucleo e' EMBUTIDO (`PluginManifest`) em vez de reescrito: *assim o manifesto do
// formato novo **e'** um manifesto do runtime, e nao uma copia que envelhece ao lado.*
type HadesPluginManifest struct {
	Schema         string                `json:"$schema,omitempty"`
	FormatVersion  string                `json:"format_version"`
	PluginManifest                       // nucleo canonico, validado pelo validador existente
	Skills         []HadesPluginSkill    `json:"skills,omitempty"`
	Interface      *HadesPluginInterface `json:"interface,omitempty"`
}

// ParseHadesPluginManifest le e valida um `hades-plugin.json`.
//
// **Recusa campos desconhecidos** (`DisallowUnknownFields`): o mesmo rigor que a rota de
// execucao tem (`E-966`) — *um manifesto com um campo que o runtime nao conhece e' um
// manifesto cujo efeito nao se pode prever.*
func ParseHadesPluginManifest(data []byte, policy CapabilityPolicy) (HadesPluginManifest, error) {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var manifest HadesPluginManifest
	if err := decoder.Decode(&manifest); err != nil {
		return HadesPluginManifest{}, fmt.Errorf("%w: %v", ErrHadesPluginFormat, err)
	}
	return ValidateHadesPluginManifest(manifest, policy)
}

// ValidateHadesPluginManifest valida o formato e delega o nucleo no validador canonico.
func ValidateHadesPluginManifest(manifest HadesPluginManifest, policy CapabilityPolicy) (HadesPluginManifest, error) {
	if version := strings.TrimSpace(manifest.FormatVersion); version != HadesPluginFormatVersion {
		return HadesPluginManifest{}, fmt.Errorf("%w: %q (esperado %q)", ErrHadesPluginVersion, manifest.FormatVersion, HadesPluginFormatVersion)
	}
	// O NUCLEO vai pelo validador que ja' existia. *Nao ha' segundo contrato.*
	nucleo, err := validatePluginManifest(manifest.PluginManifest, policy)
	if err != nil {
		return HadesPluginManifest{}, err
	}
	manifest.PluginManifest = nucleo

	// As skills declaradas: nome seguro e caminho RELATIVO dentro do bundle.
	seen := map[string]struct{}{}
	for _, skill := range manifest.Skills {
		name := strings.TrimSpace(skill.Name)
		if !safeSkillNamePattern.MatchString(name) {
			return HadesPluginManifest{}, fmt.Errorf("%w: nome %q nao e' um slug seguro", ErrHadesPluginSkillPath, skill.Name)
		}
		if _, duplicado := seen[name]; duplicado {
			return HadesPluginManifest{}, fmt.Errorf("%w: skill %q declarada duas vezes", ErrHadesPluginSkillPath, name)
		}
		seen[name] = struct{}{}
		path := strings.TrimSpace(skill.Path)
		if !safeRelativePath(path) {
			return HadesPluginManifest{}, fmt.Errorf("%w: %q tem de ser um caminho relativo dentro do bundle", ErrHadesPluginSkillPath, skill.Path)
		}
	}
	// Um `kind: "skill"` sem skills declaradas e' contraditorio: *declara ensinar e nao
	// traz nada.*
	if manifest.Kind == PluginKindSkill && len(manifest.Skills) == 0 {
		return HadesPluginManifest{}, fmt.Errorf("%w: kind=skill exige ao menos uma entrada em skills", ErrHadesPluginSkillPath)
	}
	if iface := manifest.Interface; iface != nil {
		icon := strings.TrimSpace(iface.Icon)
		if icon != "" && !safeRelativePath(icon) {
			// *Um icone remoto quebraria o funcionamento offline.*
			return HadesPluginManifest{}, fmt.Errorf("%w: icon tem de ser caminho relativo dentro do bundle", ErrHadesPluginInterface)
		}
	}
	return manifest, nil
}

// InstallHadesPluginForOrganization instala um plugin declarado no formato aberto.
//
// **A ligacao que faltava**: o `E-972` criou o formato; isto **liga-o** ao registo
// canonico. *Um formato que ninguem consegue instalar e' uma especificacao, nao uma
// capacidade* — e o `§REGRA-DE-IMPLEMENTACAO` item 3 diz exactamente isso: *«se existe
// isolada, conecte»*.
//
// **E o que NAO se duplica**: a instalacao em si — tenant isolation, recusa de versao ja'
// instalada, digest, `Trusted=false`, `RequestedScopes` vs `GrantedScopes` — e' **toda**
// do `InstallForOrganization` (`plugin_registry.go:337`). *Este metodo valida o formato e
// delega; nao reimplementa uma segunda instalacao.*
//
// **As `skills` e a `interface` sao PERSISTIDAS**: sem isso, o manifesto seria aceito e a
// informacao perdida no momento de gravar — *e um plugin que ensina 25 skills apareceria
// sem nenhuma.*
func (r *PluginRegistry) InstallHadesPluginForOrganization(organizationID string, manifest HadesPluginManifest, policy CapabilityPolicy) (PluginInstallation, error) {
	validated, err := ValidateHadesPluginManifest(manifest, policy)
	if err != nil {
		return PluginInstallation{}, err
	}
	// Primeiro o caminho canonico: *ele valida o tenant, recusa versao repetida, calcula o
	// digest e grava.* Nao ha' segunda instalacao.
	if _, err := r.InstallForOrganization(organizationID, validated.PluginManifest, policy); err != nil {
		return PluginInstallation{}, err
	}
	// **E aqui esta' a armadilha que eu evitei**, e vale a pena escreve-la: o
	// `InstallForOrganization` termina com `return clonePluginInstallation(*installation)`
	// (`plugin_registry.go:383`). *Mutar o valor devolvido alteraria um CLONE, e o registo
	// ficaria sem as skills* — o manifesto seria aceito e a informacao perdida no momento
	// de gravar.
	//
	// Por isso a anexacao e' feita **no mapa, sob o mesmo lock**, e **persistida de novo**.
	// Se a segunda gravacao falhar, os campos sao revertidos: *uma instalacao sem as skills
	// que o manifesto declarava seria um estado que nunca existiu.*
	trimmed := strings.TrimSpace(organizationID)
	r.mu.Lock()
	defer r.mu.Unlock()
	scoped := r.byOrg[trimmed]
	if scoped == nil {
		return PluginInstallation{}, fmt.Errorf("%w: %s", ErrPluginNotFound, validated.ID)
	}
	stored, ok := scoped[validated.ID]
	if !ok {
		return PluginInstallation{}, fmt.Errorf("%w: %s", ErrPluginNotFound, validated.ID)
	}
	previousSkills, previousInterface := stored.Skills, stored.Interface
	stored.Skills = append([]HadesPluginSkill(nil), validated.Skills...)
	if validated.Interface != nil {
		copied := *validated.Interface
		stored.Interface = &copied
	}
	stored.UpdatedAt = time.Now().UTC()
	if err := r.persistLocked(trimmed); err != nil {
		stored.Skills, stored.Interface = previousSkills, previousInterface
		return PluginInstallation{}, err
	}
	return clonePluginInstallation(*stored), nil
}

// SkillsDeclaradas devolve as skills que uma instalacao oferece.
//
// *Leitura pura, do estado persistido*: a lista vem da instalacao, e nao de um pedido.
func (i PluginInstallation) SkillsDeclaradas() []HadesPluginSkill {
	return append([]HadesPluginSkill(nil), i.Skills...)
}

// NomeVisivel devolve o nome de APRESENTACAO, com o nome do nucleo como fallback.
//
// *A separacao importa*: um plugin pode mudar de nome visivel sem mudar de identidade —
// e sem invalidar a assinatura do conteudo. Quando nao ha' `interface`, o nome do nucleo
// e' a resposta honesta.
func (i PluginInstallation) NomeVisivel() string {
	if i.Interface != nil {
		if display := strings.TrimSpace(i.Interface.DisplayName); display != "" {
			return display
		}
	}
	return i.Manifest.Name
}
