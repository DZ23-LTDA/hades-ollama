# Estágio 14 — Browser, Desktop e Computer-Use: status por plataforma

**Método:** medir, não presumir. O teste de navegador estava na lista de falhas
ambientais conhecidas (B-09) e a causa **não** era a esperada.

## 1. O defeito encontrado (e corrigido)

O operador escolhia o interpretador Python por **existência no PATH**
(`python3` → `python`). No Windows, esses nomes respondem pelo **atalho da Microsoft
Store**, que existe no PATH e **falha ao executar** (exit 9009). Consequências
medidas:

| Antes | Depois |
| --- | --- |
| Operador adotava o atalho e falhava com `exit status 9009` | Operador resolve o Python **real** (`C:\Program Files\Python312\python.exe`) |
| `PythonOK: true` para um atalho que **não executa nada** | `PythonOK` só é verdadeiro para um interpretador que **responde** (`import sys`) |
| Orientação **errada**: "instale o Playwright" | Orientação correta: Playwright ausente **e** o passo do atalho da Microsoft Store quando nenhum candidato executa |
| E2E de navegador: **FAIL** com erro críptico | E2E de navegador: **SKIP** com causa legível |

A resolução é **compartilhada** entre o status e o operador
(`resolvePythonInterpreter`), para que status e execução não possam divergir — o
mesmo princípio do núcleo de workflow.

## 2. Estado real desta máquina (Windows, medida agora)

```json
{
  "python_path": "C:\\Program Files\\Python312\\python.exe",
  "python_ok": true,
  "playwright_ok": false,
  "chromium_ok": false,
  "ready": false,
  "guidance": ["Instale o Playwright: python -m pip install playwright"],
  "can_auto_setup": true
}
```

## 3. Classificação por plataforma

| Plataforma | Navegador (E2E real) | Desktop/computer-use | Base da classificação |
| --- | --- | --- | --- |
| **Windows (este host)** | `BLOQUEADO_EXTERNAMENTE` — Python OK, **Playwright/Chromium ausentes** | `NAO_VERIFICADO` | probe ao vivo acima |
| **Linux** | `NAO_VERIFICADO` — não executado neste host | `NAO_VERIFICADO` | nenhuma medição |
| **macOS** | `NAO_VERIFICADO` — não executado neste host | `NAO_VERIFICADO` | nenhuma medição |

Nada aqui é declarado homologado por compilação cruzada: **não houve execução**.

## 4. O que JÁ está coberto sem navegador (testes verdes)

- ambiente do processo filho **sem segredos** (credenciais, chaves de nuvem, proxy);
- recusa de escopo de organização sem sandbox estrita;
- eventos de frame do operador emitidos pelo runtime;
- lógica de setup (passos que faltam apenas) e de resolução de interpretador;
- status honesto de prontidão com orientação acionável.

## 5. Procedimento exato de desbloqueio

```
python -m pip install playwright
python -m playwright install chromium
```

(~150 MB de download). O produto já expõe exatamente esses dois passos em
`SetupBrowserEnvironment`, que **só** roda sob pedido explícito do usuário. Este
trabalho **não** executou a instalação: é mudança de máquina com custo relevante e
merece decisão do usuário.

## 6. Efeito na lista B-09

O item de navegador sai de **FAIL** para **SKIP com causa legível**; a dependência
externa permanece declarada em vez de virar falha opaca.
