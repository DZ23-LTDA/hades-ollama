# HADES — MATRIZ DE PLATAFORMAS (medida)

**Este ficheiro é gerado.** Não edite à mão: corrija a fonte e regenere com
`python -X utf8 scripts/hades_platform_matrix.py`.

## Porque este ficheiro existe

O §ESTÁGIO 1 exige «atualizar inventário de **código, plataformas, workflows, pacotes e
lacunas**», e o prompt exige distinguir **SOURCE, BUILD, PACKAGE, INSTALL, DEVICE_E2E,
assinatura e hash** por plataforma.

⇒ ***cross-compile não é teste nativo; bundle web não é aplicativo móvel; configuração EAS
não é APK.***

## A matriz

| Plataforma | SOURCE | BUILD (workflow) | PACKAGE | Tamanho | Hash (SHA-256[:32]) | INSTALL | DEVICE_E2E | Assinatura |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| **Windows** | `cmd/`, `app/`, `server/` (Go) | `dz23-windows-installer.yaml` | `dist\hades-installer-6d59c217e\HadesSetup-6d59c217e.exe` | 24.3 MB | `97a6b3450980405a8fc5bc578c936f39` | NÃO MEDIDO (INSTALLED=NAO) | NÃO MEDIDO | não assinado (unsigned) |
| **Linux** | `cmd/`, `app/`, `server/` (Go) | `dz23-linux-package.yaml` | `dist\hades-linux-amd64\hades-linux-amd64-UNSIGNED.tar.gz` | 16.2 MB | `1b7ee3046a887138f75298054f81f1ea` | NÃO MEDIDO (INSTALLED=NAO) | NÃO MEDIDO | não assinado (unsigned) |
| **macOS** | `cmd/`, `app/`, `server/` (Go) | `dz23-macos-build.yaml` | `AUSENTE` | - | `AUSENTE` | NÃO MEDIDO | NÃO MEDIDO | não assinado (unsigned) |
| **Android** | `apps/mobile-agentic/` (Expo/RN + `android/`) | `dz23-e2e.yaml` | `AUSENTE` | - | `AUSENTE` | NÃO MEDIDO | NÃO MEDIDO | APK de desenvolvimento (não publicável) |
| **iOS** | `apps/mobile-agentic/` (Expo/RN) | `AUSENTE` | `AUSENTE` | - | `AUSENTE` | NÃO MEDIDO | NÃO MEDIDO | não assinado (unsigned) |
| **Web/PWA** | `app/ui/app/` (React+TS+Vite) | `dz23-e2e.yaml` | `app\ui\app\dist\index.html` | 9 KB | `d2108fa2d0aa3bbc8e6b04a97e2ed9f8` | N/A (build de produção) | NÃO MEDIDO (PWA_INSTALADO_NA_UI=NAO) | N/A |

## Artefactos de COMPONENTE (nao sao uma das 6 plataformas)

Pacotes que vivem em `dist/` e nao correspondem a um alvo de produto. *Sem esta seccao, um pacote que existe nao aparece em inventario nenhum.*

| Componente | Pacote | Tamanho | Hash (SHA-256[:32]) | Proveniencia | Escrito em |
| --- | --- | --- | --- | --- | --- |
| `browser-bridge` | `dist\browser-bridge\hades-browser-bridge-0.1.0-unsigned.zip` | 112 KB | `3507621561c87d263bca626bf07a00ac` | a313019e48ef | 2026-10-17 18:22 |
| `browser-bridge-firefox` | `dist\browser-bridge-firefox\hades-browser-bridge-firefox-UNSIGNED.zip` | 108 KB | `dcd66d17482b2e1ad26313ab0206e520` | a313019e48ef | 2026-10-17 18:23 |


## O que cada eixo significa, e o que NÃO conta

| Eixo | Conta como MEDIDO | NÃO conta |
| --- | --- | --- |
| **SOURCE** | o código existe e compila | «o repositório menciona a plataforma» |
| **BUILD** | um workflow **executou** o build | o workflow existir por si |
| **PACKAGE** | o ficheiro **existe**, com tamanho e hash | «o workflow geraria» |
| **INSTALL** | alguém **instalou** e abriu | o `dist/` ter o instalador |
| **DEVICE_E2E** | teste em **dispositivo/emulador** | o Playwright no browser |
| **Assinatura** | certificado + notarização | «não assinado durante o desenvolvimento» |

## As lacunas, por plataforma

- **Windows**: artefacto do candidato (SOURCE_SHA 6d59c217e declarado no PROVENIENCIA.txt)
- **Linux**: artefacto de SOURCE_SHA 9a0408e31905 (declarado no PROVENIENCIA.txt DENTRO do tarball); cross-compile Windows→Linux com CGO=0; **NAO** e' build nativo, **NAO** foi instalado, e **NAO** e' byte-a-byte o artefacto do `dz23-linux-package.yaml` (que usa CGO=1)
- **macOS**: **BLOCKED_EXTERNAL (medido, nao suposto)**: `GOOS=darwin CGO_ENABLED=0` falha com `undefined: GetCPUMem` -- o `GetCPUMem` do darwin vive em `discover/gpu_darwin.go`, que usa `import "C"` (Metal/Foundation), e o Go exclui ficheiros com `import "C"` quando CGO=0. Desbloqueio: o `dz23-macos-build.yaml` em `macos-latest` (Xcode + CGO).
- **Android**: **o `app.json`/`eas.json` existem; o APK não**
- **iOS**: **BLOCKED_EXTERNAL**: exige macOS/Xcode + conta Apple
- **Web/PWA**: bundle auto-coerente: 5/5 assets referenciados presentes; build de 2026-10-18 11:46 (sem SHA de fonte -- Vite nao o declara); servido por HTTP; o `manifest` existe

## Os workflows, contados

**16 workflows** em `.github/workflows/`:

- `class-a-plus-integrity.yaml`
- `dz23-agentic-quality.yaml`
- `dz23-e2e.yaml`
- `dz23-linux-package.yaml`
- `dz23-macos-build.yaml`
- `dz23-multi-provider.yaml`
- `dz23-provider-smoke.yaml`
- `dz23-windows-installer.yaml`
- `latest.yaml`
- `pr-gate.yaml`
- `release-readiness.yaml`
- `release.yaml`
- `secret-scan.yaml`
- `test-install.yaml`
- `test-llamacpp-update.yaml`
- `test.yaml`

## A conclusão honesta

⇒ ***o inventário é, sobretudo, uma lista do que falta:*** *nenhuma plataforma tem
`INSTALL` medido, nenhuma tem `DEVICE_E2E`, e as que têm `PACKAGE` têm-no **não assinado**.
Isso **não** é uma falha do produto: é o estado **medido**, e é o que impede
`NOT_RELEASE_READY` de virar `RELEASE_READY` sem homologação real.
