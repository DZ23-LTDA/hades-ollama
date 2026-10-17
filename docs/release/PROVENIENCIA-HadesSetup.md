PROVENIENCIA -- HADES instalador Windows (Inno Setup), NAO ASSINADO
===================================================================

***LEIA ISTO ANTES DE USAR.*** *Este manifesto foi escrito **depois** do artefacto, porque o
`HadesSetup.exe` foi gerado **antes** de a regra «todo o pacote tem de registar `SOURCE_SHA`» existir.*

O QUE ESTE ARTEFACTO E':
  - um instalador Inno Setup, NAO ASSINADO;
  - `dist\HadesSetup.exe`, 24,26 MB, gerado em **2026-10-16 22:58** (hora local);
  - SHA-256: a70b51073d89b7f65463f4f46d5a7a89... (o hash completo esta' em `docs/audit/ARTEFATOS.sha256`
    e na linha `Windows` do `docs/mission/HADES_PLATFORM_MATRIX.md`);
  - contem `windows-ollama-app-amd64.exe` (o app) e `windows-amd64\ollama.exe` + `windows-amd64\lib\ollama\*`
    (o motor), conforme o `[Files]` do `app\ollama.iss`.

O QUE ESTE ARTEFACTO **NAO** E', e e' o que importa:
  - ***NAO registou `SOURCE_SHA`.*** *O `PROVENIENCIA.txt` nao existia quando ele foi gerado* => *e eu **nao o
    posso reconstruir**: inventar um SHA seria pior do que declarar que nao se sabe* => **`SOURCE_SHA:
    DESCONHECIDO`.***
  - ***NAO e' do candidato actual.*** *O HEAD do candidato e' `5f6fc87f3`, e este instalador e' **anterior** e
    **diferente** do que tem proveniencia* (ver abaixo);
  - **NAO e' publicado**, **NAO e' assinado**, e **NAO foi instalado** (`INSTALL=NAO` no
    `HADES_PLATFORM_MATRIX.md`).

E O OUTRO INSTALADOR, que **tem** proveniencia -- e nao e' este:
  - `dist\hades-installer-541b1f87c\HadesSetup-541b1f87c.exe`;
  - `SOURCE_SHA: 541b1f87c` (branch `fix/checker-relogio`, `dirty: 14 entrada(s)`);
  - **SHA-256: d6fcc44400fd1d54d8a86ddd016eea2b3b1665f0a9dab2f6fb47b3e4919ba2af** (confere com o manifesto
    dele);
  - *e' um artefacto **diferente** deste* => ***dois instaladores no `dist/`, um com proveniencia e outro sem:
    o que faltava nao era um instalador, era saber de onde vem cada um.***

O QUE FALTA PARA UM ARTEFACTO DO **CANDIDATO ACTUAL**:
  - *o `[Files]` do `app\ollama.iss` exige `dist\windows-ollama-app-amd64.exe`, `dist\windows-amd64\ollama.exe`
    e `dist\windows-amd64\lib\ollama\*`* => *ou seja, o **build completo do Windows** (app + motor + libs)*;
  - *esse build e' o que o workflow `dz23-windows-installer.yaml` faz em `windows-latest`* => *e o passo exacto
    de desbloqueio e' **correr esse workflow no SHA candidato**, ou reproduzi-lo aqui com a toolchain completa
    (Inno Setup 6.7.3 esta' em `%LOCALAPPDATA%\Programs\Inno Setup 6\ISCC.exe`)*;
  - ***isso e' um limite NOMEADO, nao um bloqueio externo*** => *e o `E-1123` registou-o com o comando exacto.*

COMO FOI GERADO (o comando, reconstruido do `app\ollama.iss`):
  & ISCC /O<out> /FHadesSetup-<sha> app\ollama.iss   (com PKG_VERSION=<versao>, sem KEY_CO...)

O SmartScreen vai avisar: e' o esperado num artefacto nao assinado, e **nao** e' um defeito.
Este instalador **NAO** foi publicado; existe apenas localmente em `dist\`.
