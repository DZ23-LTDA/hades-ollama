# HADES — T2: O AMBIENTE DA MEDIÇÃO

**Este ficheiro é gerado.** Não edite à mão: corrija a fonte e regenere com
`python -X utf8 scripts/hades_t2_contexto.py`.

## Porque este ficheiro existe

O `HADES_T2_MATRIX.md` e o `HADES_T2_REPETICAO.md` têm os **cenários**, os **modelos**, as
**repetições** (N=3) e o **veredito**. O que **não** tinham era o **ambiente**: sem ele, os
números **não são reproduzíveis** — quem os quiser repetir não sabe em que máquina, com que
versão e com que contexto foram medidos.

⇒ ***um número sem ambiente é uma curiosidade; um número com ambiente é uma medição.***

## Hardware e sistema

| Item | Valor medido |
| --- | --- |
| CPU | `13th Gen Intel(R) Core(TM) i9-13900HX` |
| RAM | `47,7 GB` |
| GPU | `NVIDIA GeForce RTX 4060 Laptop GPU` |
| SO | `Windows 11` (build `10.0.26300`, arch `AMD64`) |
| Python | `3.12.10` |
| Go | `go version go1.26.9 windows/amd64` |
| Disco livre | `276.3 GB` |
| Cores lógicos | `32` |

## Ollama

| Item | Valor medido |
| --- | --- |
| Versão | `0.40.1` |
| Endpoint | `http://127.0.0.1:11434` |

## Modelos instalados (agora)

| Modelo | Tamanho | Digest (curto) |
| --- | --- | --- |
| `Huihui-Qwen3.5-9B-abliterated:latest` | 6.10 GB | `607c4a44546d` |
| `claude-sonnet-4-5-qwen25:latest` | 4.36 GB | `845dbda0ea48` |
| `qwen2.5-coder:7b` | 4.36 GB | `dae161e27b0e` |
| `qwen2.5:7b` | 4.36 GB | `845dbda0ea48` |
| `nomic-embed-text:latest` | 0.26 GB | `0a109f422b47` |
| `llava:7b` | 4.41 GB | `8dd30f6b0cb1` |
| `llama3.2:3b` | 1.88 GB | `a80c4f17acd5` |
| `llama3.1:8b` | 4.58 GB | `46e0c10c039e` |
| `deepseek-r1:8b` | 4.87 GB | `6995872bfe4c` |

## Os dois modelos do T2

- `qwen2.5:7b`: **instalado**
- `llama3.2:3b`: **instalado**

*`AUSENTE agora` não invalida a medição antiga — diz que **este ambiente** já não o tem.*

## O que este ambiente **não** cobre

- **warm/cold**: o T2 e o T2-REPETIÇÃO **não separaram** o tempo de carregamento do modelo do
  tempo de geração. ⇒ a coluna «duração» é **wall time**, não a soma decomposta.
- **separação de tempos**: carregamento, geração, ferramentas e wall time **não** foram medidos
  em separado. ⇒ fica como lacuna declarada, não como número inventado.
- **erro**: o T2 regista veredito por cenário, e **não** um intervalo de confiança — com N=3,
  os números são **contagem**, não probabilidade (o próprio `HADES_T2_REPETICAO.md` o diz).
- **comparação externa**: Claude Code / Codex / Manus ⇒ `BLOCKED_BY_EXTERNAL_DEPENDENCY` ⇒
  **`NAO_MEDIDO`**. *Sem acesso legal ao concorrente, não há comparação — e inventá-la seria
  marketing, não benchmark.*

## Proveniência

| Item | Valor |
| --- | --- |
| Commit | `ed59c5565` |
| Estado do worktree | `limpo` |
| `HADES_T2_MATRIX.md` | sha256 `985ea65c51e94041` |
| `HADES_T2_REPETICAO.md` | sha256 `07efb3893c2acda4` |

⇒ ***o hash dos dois documentos liga ESTE ambiente aos números que ele contextualiza.***
