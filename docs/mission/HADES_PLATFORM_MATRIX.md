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
| **Windows** | `cmd/`, `app/`, `server/` (Go) | `dz23-windows-installer.yaml` | `dist\HadesSetup.exe` | 24.3 MB | `a70b51073d89b7f65463f4f46d5a7a89` | NÃO MEDIDO (INSTALLED=NAO) | NÃO MEDIDO | não assinado (unsigned) |
| **Linux** | `cmd/`, `app/`, `server/` (Go) | `dz23-linux-package.yaml` | `AUSENTE` | - | `AUSENTE` | NÃO MEDIDO (INSTALLED=NAO) | NÃO MEDIDO | não assinado (unsigned) |
| **macOS** | `cmd/`, `app/`, `server/` (Go) | `dz23-macos-build.yaml` | `AUSENTE` | - | `AUSENTE` | NÃO MEDIDO | NÃO MEDIDO | não assinado (unsigned) |
| **Android** | `apps/mobile-agentic/` (Expo/RN + `android/`) | `dz23-e2e.yaml` | `AUSENTE` | - | `AUSENTE` | NÃO MEDIDO | NÃO MEDIDO | APK de desenvolvimento (não publicável) |
| **iOS** | `apps/mobile-agentic/` (Expo/RN) | `AUSENTE` | `AUSENTE` | - | `AUSENTE` | NÃO MEDIDO | NÃO MEDIDO | não assinado (unsigned) |
| **Web/PWA** | `app/ui/app/` (React+TS+Vite) | `dz23-e2e.yaml` | `app\ui\app\dist\index.html` | 9 KB | `0daa2a2f233dc3d58c8699f4c72e0738` | N/A (build de produção) | NÃO MEDIDO (PWA_INSTALADO_NA_UI=NAO) | N/A |

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

- **Windows**: o SmartScreen é limite esperado
- **Linux**: cross-build Windows não é teste Linux
- **macOS**: **BLOCKED_EXTERNAL**: exige macOS/Xcode, e não ha'
- **Android**: **o `app.json`/`eas.json` existem; o APK não**
- **iOS**: **BLOCKED_EXTERNAL**: exige macOS/Xcode + conta Apple
- **Web/PWA**: servido por HTTP; o `manifest` existe

## Os workflows, contados

**15 workflows** em `.github/workflows/`:

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
- `test-install.yaml`
- `test-llamacpp-update.yaml`
- `test.yaml`

## A conclusão honesta

⇒ ***o inventário é, sobretudo, uma lista do que falta:*** *nenhuma plataforma tem
`INSTALL` medido, nenhuma tem `DEVICE_E2E`, e as que têm `PACKAGE` têm-no **não assinado**.
Isso **não** é uma falha do produto: é o estado **medido**, e é o que impede
`NOT_RELEASE_READY` de virar `RELEASE_READY` sem homologação real.
