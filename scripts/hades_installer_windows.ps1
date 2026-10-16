# Monta o INSTALADOR Windows (Inno Setup) a partir do estado ACTUAL, e verifica-o antes de dizer OK.
#
# *POR QUE ISTO EXISTE* (E-1012): o `app\ollama.iss` tem `OutputBaseFilename="HadesSetup"` e
# `AppName={#MyAppName}` com `MyAppName "Hades"` -- ou seja, o instalador JA' e' `HadesSetup.exe`
# com a marca canonica. Mas compila-lo exige TRES coisas que nao estavam escritas em lugar nenhum:
#
#   1. `PKG_VERSION` NUMERICO de QUATRO campos. => *Medido: `0.2.0-541b1f87c` da'
#      "Error on line 26 ... VersionInfoVersion" -- e a linha 26 e' `VersionInfoVersion={#MyAppVersion}`.
#      Com `0.2.0.541` compila (`Successful compile`).* => *O SHA nao pode entrar na versao; entra no
#      NOME do ficheiro.*
#   2. Os TRES alvos em `dist\`: `windows-ollama-app-amd64.exe`, `windows-amd64\ollama.exe` e
#      `windows-amd64\lib\ollama\`. => *O `.iss` e' DEFENSIVO (`#if FileExists(...)`), logo um alvo
#      em falta **nao** faz o compilador falhar -- faz um instalador INCOMPLETO. E' por isso que este
#      script confere os alvos ANTES, e nao depois.*
#   3. NAO definir `KEY_CONTAINER`. => *O `.iss` linha 78 so' liga `SignTool` e `SignedUninstaller`
#      se essa variavel existir, e o objetivo e' um artefacto **nao assinado**.* => *O script
#      **remove-a** do ambiente em vez de confiar que ela nao esta' la'.*
#
# Uso:
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_installer_windows.ps1
#   (opcional) -OutDir dist\hades-installer -Versao 0.2.0.541 -Iscc <caminho do ISCC.exe>
#
# O script FALHA (exit != 0) se qualquer verificacao nao passar.

param(
    [string]$Raiz = "",
    [string]$OutDir = "",
    [string]$Versao = "",
    [string]$Iscc = ""
)

$ErrorActionPreference = "Stop"

if (-not $Raiz) { $Raiz = (Get-Location).Path }
$raiz = (Resolve-Path $Raiz).Path
Set-Location $raiz

$sha = (& git rev-parse --short HEAD 2>$null)
if (-not $sha) { $sha = "nogit" }

if (-not $Versao) {
    # *Quatro campos NUMERICOS.* O campo de build vem do SHA, mas em decimal, porque
    # `VersionInfoVersion` nao aceita letras. => *A versao fica assim declarada, e o SHA continua
    # visivel no nome do ficheiro e no LEIA-ME que este script escreve.*
    $build = 0
    foreach ($c in $sha.ToCharArray()) { $build = (($build * 31) + [int]$c) % 65535 }
    $Versao = "0.2.0.$build"
}
if ($Versao -notmatch '^\d+\.\d+\.\d+\.\d+$') {
    Write-Output "ERRO: -Versao tem de ser NUMERICA de QUATRO campos (a.b.c.d); recebi '$Versao'."
    Write-Output "  *Medido (E-1012): o `.iss` linha 26 e' `VersionInfoVersion={#MyAppVersion}`, e ele"
    Write-Output "   recusa qualquer coisa que nao seja a.b.c.d -- '0.2.0-<sha>' da' erro de compilacao.*"
    exit 2
}

if (-not $Iscc) { $Iscc = Join-Path $env:LOCALAPPDATA "Programs\Inno Setup 6\ISCC.exe" }
if (-not (Test-Path $Iscc)) {
    Write-Output "BLOCKED_BY_EXTERNAL_DEPENDENCY: ISCC.exe nao encontrado em '$Iscc'."
    Write-Output "  *Nao e' falha do produto: e' ausencia do instrumento, e ela e' declarada.*"
    Write-Output "  Passo de desbloqueio: instalar o Inno Setup 6 (winget install JRSoftware.InnoSetup)."
    exit 3
}

# *Os alvos do `.iss`: o script confere-os ANTES, porque o `#if FileExists` do `.iss` esconderia
# a falta deles num instalador incompleto em vez de um erro.*
$alvos = @(
    @("dist\windows-ollama-app-amd64.exe", "app desktop (DestName: ollama app.exe)"),
    @("dist\windows-amd64\ollama.exe", "motor/servidor"),
    @("dist\windows-amd64\lib\ollama", "bibliotecas do motor")
)
$faltam = @()
foreach ($a in $alvos) {
    if (-not (Test-Path (Join-Path $raiz $a[0]))) { $faltam += ($a[0] + "  (" + $a[1] + ")") }
}
if ($faltam.Count -gt 0) {
    Write-Output "ALVOS EM FALTA -- o instalador sairia INCOMPLETO (o `.iss` e' defensivo e nao falharia):"
    $faltam | ForEach-Object { Write-Output ("  - " + $_) }
    Write-Output "  *Corra primeiro o `scripts\hades_bundle_windows.ps1` (ou o cross-build), e repita.*"
    exit 1
}

if (-not $OutDir) { $OutDir = "dist\hades-installer-$sha" }
$out = Join-Path $raiz $OutDir
New-Item -ItemType Directory -Force -Path $out | Out-Null

# *NAO assinado, por construcao:* o `KEY_CONTAINER` e' removido do ambiente.
if (Test-Path Env:\KEY_CONTAINER) { Remove-Item Env:\KEY_CONTAINER -Force }
$env:PKG_VERSION = $Versao

$nome = "HadesSetup-$sha"
Write-Output "=== INSTALADOR WINDOWS (Inno Setup) ==="
Write-Output ("  HEAD      {0}" -f $sha)
Write-Output ("  versao    {0}  (PKG_VERSION; o SHA vai no nome do ficheiro)" -f $Versao)
Write-Output ("  assinatura NAO (o KEY_CONTAINER foi removido do ambiente)" -f "")
Write-Output ("  ISCC      {0}" -f $Iscc)

$t0 = Get-Date
& $Iscc "/O$out" "/F$nome" (Join-Path $raiz "app\ollama.iss") 2>&1 | Select-Object -Last 3 | ForEach-Object { Write-Output ("  " + $_) }
$code = $LASTEXITCODE
$seg = [int]((Get-Date) - $t0).TotalSeconds
if ($code -ne 0) {
    Write-Output ("  COMPILACAO FALHOU (exit {0}, {1}s)" -f $code, $seg)
    exit 1
}

$exe = Join-Path $out ($nome + ".exe")
if (-not (Test-Path $exe)) { Write-Output "ERRO: o compilador disse OK mas '$exe' nao existe."; exit 1 }

$item = Get-Item $exe
$hash = (Get-FileHash $exe -Algorithm SHA256).Hash
Write-Output ("  COMPILADO  {0}  {1} MB  ({2}s)" -f $item.Name, [math]::Round($item.Length / 1MB, 2), $seg)
Write-Output ("  SHA256     {0}" -f $hash)
Write-Output ("  versao PE  {0} | produto '{1}'" -f $item.VersionInfo.FileVersion, $item.VersionInfo.ProductName.Trim())

# *A declaracao de proveniencia, ao lado do artefacto.* => *Sem ela, o .exe nao diz de que estado veio.*
$proc = @(
    "HADES - instalador Windows (Inno Setup), NAO assinado",
    ("SOURCE_SHA: " + $sha + "  branch: " + (& git rev-parse --abbrev-ref HEAD) + "  dirty: " + ((& git status --porcelain | Measure-Object).Count) + " entrada(s)"),
    ("VERSION: " + $Versao),
    ("PLATAFORMA: windows / amd64   ASSINATURA: false"),
    ("SHA256: " + $hash),
    ("TAMANHO: " + $item.Length + " B"),
    ("COMANDO: & ISCC /O<out> /FHadesSetup-" + $sha + " app\ollama.iss   (com PKG_VERSION=" + $Versao + ", sem KEY_CONTAINER)"),
    "",
    "O SmartScreen vai avisar: e' o esperado num artefacto nao assinado, e nao um defeito.",
    "Este instalador NAO foi publicado; existe apenas localmente em " + $OutDir
)
$procPath = Join-Path $out "PROVENIENCIA.txt"
[System.IO.File]::WriteAllLines($procPath, $proc, (New-Object System.Text.UTF8Encoding($false)))
Write-Output ("  PROVENIENCIA {0}" -f $procPath)
Write-Output "INSTALADOR PRONTO (nao assinado, nao publicado)."
exit 0
