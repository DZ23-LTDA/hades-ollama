# HADES — ADRs DE INTEGRAÇÃO (decisão por referência)

**Exigido por:** ADENDO DE PARIDADE §4 (*uma estratégia documentada por referência*) e §6.
**Evidência de criação:** E-318.
**Regra:** **nada é copiado antes da licença ser lida.** Licença não lida = `NAO_VERIFICADA` = **nenhum trecho entra no
Hades**, nem por "inspiração" de código.

Estratégias possíveis (§4): **a** capacidade nativa · **b** adaptador oficial substituível · **c** benchmark/referência
sem dependência · **d** rejeitado com justificativa.

---

## 1. Licenças efetivamente LIDAS nesta rodada

Comando reproduzível (texto e blob SHA lidos via API/raw do GitHub, em `HEAD` no momento da consulta):

```bash
# texto
curl -s https://raw.githubusercontent.com/<owner>/<repo>/HEAD/LICENSE
# blob SHA (prova de QUAL arquivo foi lido)
# via API de conteúdos: GET /repos/<owner>/<repo>/contents/LICENSE
```

| ref_id | referência | licença **medida** | blob SHA do arquivo lido | o que isso permite/obriga |
| --- | --- | --- | --- | --- |
| REF-R21 | `OleLehmann/youtube-automation-agent` (TUBEGENT) | **MIT** | `de3dcc3f02b8eacb200f033687dbb0fb46576af5` | uso, cópia, modificação e distribuição permitidos **com aviso de copyright e licença** |
| REF-R22 | `morluto/rea` | **MIT** | `2d476d24fa5a9ae2ab35e5fd121494049eb2ee5d` | idem; **atribuição obrigatória** |
| REF-R01 | `Dpro-at/Tel-Agent` | ⚠️ **AGPL-3.0** | `be3f7b28e564e7dd05eaf59d64adba1a4065ac0e` | **copyleft de rede**: a §13 exige oferecer o *Corresponding Source* aos usuários que interagem remotamente |
| REF-R11 | `BerriAI/litellm` | `NAO_VERIFICADA` | `3bfef5bae9b48c334acf426d5b7f21bc1913aab9` | arquivo localizado; **texto não lido** ⇒ nada é copiado |
| REF-R15 | `tldraw/tldraw` | `NAO_VERIFICADA` | `346511b1cc5338abdca005a4c5cb49eec183f483` (`LICENSE.md`) | idem |
| REF-R16 | `xyflow/xyflow` | `NAO_VERIFICADA` | `072ff46d7c23dd8e18ccba11aa6563f14e524312` | idem |

> **Nota de método:** o blob SHA prova **qual** arquivo foi lido, **não** o conteúdo. Para as três referências em que o
> texto não foi lido, a coluna continua `NAO_VERIFICADA` — *registrar um SHA não é ler uma licença*.

---

## 2. Decisões (ADRs)

### ADR-TEL-AGENT-001 — Tel-Agent entra como **processo separado via adaptador** (estratégia **b**), **nunca como código embutido**

- **Contexto:** `Dpro-at/Tel-Agent` é **AGPL-3.0** (medido). O V2.2 já pedia *"Tel-Agent/telefonia via adapter"*.
- **Decisão:** integrar por **adaptador de protocolo** (o Tel-Agent roda como serviço próprio, com fronteira de
  processo/rede), trocável, sem incorporar seu código ao binário do Hades. Se algum trecho for reaproveitado, será
  tratado como obra derivada com as obrigações da AGPL **explicitamente aceitas pelo proprietário**.
- **Consequência honesta:** a AGPL **não** é "só um detalhe" — a §13 pode obrigar a publicar o Corresponding Source de
  versões modificadas oferecidas por rede. **A decisão de aceitar ou não esse acoplamento é do proprietário** e está
  registrada como decisão **pendente de aprovação** para uso comercial fechado.
- **Teste de aceite (futuro):** sessão de voz/telefonia pelo adaptador, com o Tel-Agent em processo separado; sem
  dependência de código AGPL no binário Hades.
- **Bloqueio:** provedor/SIP/número ⇒ `BLOCKED_BY_EXTERNAL_DEPENDENCY` (independente da licença).

### ADR-CREATOR-001 — Creator Studio é **capacidade nativa** inspirada no TUBEGENT (estratégia **a** + **c**), com atribuição

- **Contexto:** `OleLehmann/youtube-automation-agent` é **MIT** (medido) e tem 7 papéis úteis (estratégia, roteiro,
  thumbnail, SEO, produção, publicação, analytics).
- **Decisão:** **não** clonar o projeto nem criar segundo orquestrador. Implementar os 7 papéis **dentro** do Mission
  Control/Tool Runtime do Hades (estratégia **a**), usando o TUBEGENT como **referência de comportamento**
  (estratégia **c**). Se algum trecho for adaptado, manter **aviso de copyright MIT e a licença** no código derivado.
- **Alerta herdado (V2.2 §8), tratado como requisito:** a referência continha caminhos de **simulação**
  (`getVideoStream` devolvendo JSON). ⇒ No Hades, o aceite exige **MP4 realmente decodificável** (codec/duração/áudio
  verificáveis por sonda) e **upload binário real**. *Simulação não conta como sucesso.*

### ADR-REA-001 — REA via **MCP/skill governada** (estratégia **b**), sobre artefato próprio/autorizado

- **Contexto:** `morluto/rea` é **MIT** (medido) e o V2.2 pede integração por MCP/skill, não vendorização maciça.
- **Decisão:** expor as capacidades de análise como **servidor MCP + skill versionada**, sob Policy Engine (sandbox,
  egress, DLP, timeouts, auditoria). Ferramentas comerciais (Ghidra/Hopper/JADX) exigem **licença do operador**.
- **Limite absoluto:** nada de burlar DRM, licença, autenticação ou controle de acesso; análise só em artefato
  **próprio ou autorizado**.

### ADR-GATEWAY-001 — litellm/9router/OmniRoute: **benchmark de arquitetura** (estratégia **c**) enquanto a licença não for lida

- **Decisão:** o Model Gateway do Hades já cobre roteamento, fallback, custo e LOCAL_ONLY. Não se adiciona dependência
  externa cuja licença **não foi lida** (litellm: `NAO_VERIFICADA`). Antes de qualquer adoção, ler a licença e decidir.
- **Evita explicitamente:** roteadores em cascata (proibido pelo adendo §1) e executor duplicado.

### ADR-CANVAS-001 — canvas (tldraw/xyflow): **licença primeiro, adoção depois**

- **Decisão:** nenhuma adoção antes da leitura (`NAO_VERIFICADA`). O canvas é P2 e a capacidade nativa já tem testes de
  artefato; a escolha só acontece com licença compatível com distribuição do Hades.

### ADR-BUILDER-001 — builders comerciais (Lovable/Bolt/v0/Base44/…) permanecem **referência** (estratégia **c**)

- **Decisão:** nenhum código proprietário é copiado; comparar **comportamento observável** por workflows autorizados
  (T3) e melhorar o Builder nativo. *Não se copia marca, interface protegida ou material sem direito.*

### ADR-HARNESS-001 — harnesses CLI (Codex/Claude Code/Gemini…) como **adaptadores de sessão** (estratégia **b**)

- **Decisão:** contrato único de sessão (descoberta, streaming, cancelamento, custo, limites, retomada). **Nunca**
  confundir API de modelo com harness completo. Hoje `BLOCKED_BY_EXTERNAL_DEPENDENCY` por **login** — e login ausente
  **não** é teste aprovado (T2 fica `NAO_MEDIDO`, não "ok").

### ADR-MULTIAGENT-001 — multiagente é **nativo** (estratégia **a**); MetaGPT/CrewAI/AutoGen/LangGraph/AgentScope são referência

- **Decisão:** orquestração, cotas, profundidade e worktrees já existem no Hades; o aceite é o **T4** (3 especialistas em
  worktrees isolados com merge controlado). Frameworks externos entram como **referência de comportamento**, evitando
  segundo orquestrador (proibido pelo adendo §1).

---

## 3. Tabela de decisão consolidada

| Referência | Estratégia | Licença | Próxima ação |
| --- | --- | --- | --- |
| TUBEGENT | **a** + **c** (com atribuição) | MIT ✓ | implementar 7 papéis nativos + T8 (MP4 real) |
| REA | **b** (MCP/skill) | MIT ✓ | skill governada + T9 (artefato próprio) |
| Tel-Agent | **b** (processo separado) | **AGPL-3.0** ⚠️ | adaptador; **decisão do proprietário** sobre acoplamento AGPL |
| litellm | **c** (até ler licença) | NAO_VERIFICADA | ler licença antes de qualquer adoção |
| tldraw / xyflow | **c** (até ler licença) | NAO_VERIFICADA | ler licença antes de qualquer adoção |
| Lovable/Bolt/v0/Base44/… | **c** | proprietárias | comparar comportamento; **não** copiar |
| Codex/Claude Code/Gemini | **b** | CLIs de terceiros | login (bloqueio externo) ⇒ T2 `NAO_MEDIDO` |
| MetaGPT/CrewAI/AutoGen/LangGraph | **c** | a verificar se adotado | T4 nativo primeiro |

## 4. O que este ADR **não** afirma

- Não afirma que qualquer referência foi **integrada** — só que a **decisão** existe e a **licença foi lida** onde
  indicado.
- Não afirma superioridade: **benchmark comparativo segue `NAO_MEDIDO`**.
- Não autoriza cópia: cada adoção futura reabre a leitura da licença **na versão então vigente** (licenças mudam).
