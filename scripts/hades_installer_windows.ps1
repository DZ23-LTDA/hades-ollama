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
# *POR QUE ELE MUDOU* (E-1252/E-1254): a primeira versao escrevia `SOURCE_SHA: <HEAD>` e mais nada.
# Medido na ronda 70: o instalador empacotava **tres origens diferentes** (motor reconstruido do
# HEAD, app desktop de um commit anterior, runner de dois dias antes) e o `PROVENIENCIA.txt`
# declarava **um** SHA. => *Um artefacto composto que declara UM SHA para partes de TRES origens
# mente sobre a sua proveniencia -- e mente mais alto do que um artefacto sem proveniencia nenhuma,
# porque quem le' o PROVENIENCIA acredita.*
#
# Agora o script **mede cada alvo** (SHA-256 do ficheiro, ou a contagem de ficheiros no caso de um
# directorio), **compara a idade de cada um com o commit HEAD** e escreve uma linha por alvo. Se
# qualquer alvo for mais antigo do que o HEAD, o instalador e' declarado **COMPOSICAO**, nao
# "build do HEAD" -- e a palavra aparece no proprio ficheiro, onde quem audita a vai ler.
#
# Uso:
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_installer_windows.ps1
#   (opcional) -OutDir dist\hades-installer -Versao 0.2.0.541 -Iscc <caminho do ISCC.exe>
#   (opcional) -ExigirAlvosDoHead   -> falha (exit 4) se algum alvo nao vier do HEAD
#
# O script FALHA (exit != 0) se qualquer verificacao nao passar.

param(
    [string]$Raiz = "",
    [string]$OutDir = "",
    [string]$Versao = "",
    [string]$Iscc = "",
    [switch]$ExigirAlvosDoHead
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

# *PROVENIENCIA POR ALVO* (E-1254). Cada alvo e' medido em separado e comparado com o HEAD.
#
# O commit HEAD em UTC, para comparar com o `LastWriteTime` dos ficheiros de forma honesta.
$headUtc = (& git log -1 --format=%cI HEAD 2>$null)
$headData = $null
if ($headUtc) { $headData = [DateTimeOffset]::Parse($headUtc).UtcDateTime }
#
# *O `LastWriteTime` e' uma PISTA, nao uma certidao.* Um binario reconstruido de um commit antigo
# mas escrito DEPOIS do HEAD passaria aqui como "do HEAD". A comparacao detecta o caso comum -- o
# alvo esquecido de uma ronda anterior -- e nao promete detectar todos.
# => *um timestamp nao e' uma proveniencia: e' um indicio de proveniencia.*

function Get-ProvenienciaDoAlvo {
    param([string]$Caminho, [string]$Papel, [DateTime]$HeadUtc)
    $completo = Join-Path $raiz $Caminho
    $item = Get-Item $completo
    $linha = @{}
    $linha["papel"] = $Papel
    $linha["caminho"] = $Caminho
    if ($item.PSIsContainer) {
        $ficheiros = @(Get-ChildItem $completo -Recurse -File)
        $linha["tipo"] = "directorio"
        $linha["sha256"] = "(n/a: directorio)"
        $linha["detalhe"] = ("{0} ficheiros, {1} MB" -f $ficheiros.Count,
            [math]::Round((($ficheiros | Measure-Object Length -Sum).Sum) / 1MB, 1))
    } else {
        $linha["tipo"] = "ficheiro"
        $linha["sha256"] = (Get-FileHash $completo -Algorithm SHA256).Hash.ToLower()
        $linha["detalhe"] = ("{0} B" -f $item.Length)
    }
    # *O `LastWriteTime` e' hora local; o commit e' UTC. Comparar os dois sem converter daria uma
    # diferenca de horas que nao existe -- e o defeito de fuso horario ja' apareceu nesta missao.*
    $linha["escrito"] = $item.LastWriteTime
    $linha["maisAntigoQueOHead"] = ($null -ne $HeadUtc) -and ($item.LastWriteTime.ToUniversalTime() -lt $HeadUtc.AddMinutes(-2))
    return $linha
}

$proveniencia = @()
foreach ($a in $alvos) {
    $proveniencia += (Get-ProvenienciaDoAlvo -Caminho $a[0] -Papel $a[1] -HeadUtc $headData)
}
$desatualizados = @($proveniencia | Where-Object { $_.maisAntigoQueOHead })
$composicao = $desatualizados.Count -gt 0

Write-Output "=== PROVENIENCIA DOS ALVOS ==="
foreach ($p in $proveniencia) {
    $marca = ""
    if ($p.maisAntigoQueOHead) { $marca = "  <== MAIS ANTIGO QUE O HEAD" }
    Write-Output ("  {0}" -f $p.papel)
    Write-Output ("     {0}  escrito {1}{2}" -f $p.detalhe, $p.escrito, $marca)
    Write-Output ("     sha256 {0}" -f $p.sha256)
}
if ($composicao) {
    Write-Output ("COMPOSICAO: {0} de {1} alvo(s) sao anteriores ao HEAD." -f $desatualizados.Count, $proveniencia.Count)
    Write-Output "  => o instalador NAO e' um build reproduzivel da fonte: e' uma composicao."
    if ($ExigirAlvosDoHead) {
        Write-Output "  -ExigirAlvosDoHead foi pedido: falho em vez de gerar um artefacto misturado."
        exit 4
    }
} else {
    Write-Output "TODOS OS ALVOS SAO DO HEAD (nenhum mais antigo)."
}
Write-Output ""

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

# *A declaracao de proveniencia, ao lado do artefacto.* => *Sem ela, o .exe nao diz de que estado
# veio.* => **Com uma linha por alvo**, porque um SHA unico para partes de origens diferentes e' uma
# meia-verdade. (E-1254)
$proc = @()
$proc += "HADES - instalador Windows (Inno Setup), NAO assinado"
$proc += ""
$proc += ("HEAD: " + $sha + "   branch: " + (& git rev-parse --abbrev-ref HEAD) + "   dirty: " + ((& git status --porcelain | Measure-Object).Count) + " entrada(s)")
$proc += ("VERSION: " + $Versao)
$proc += ("PLATAFORMA: windows / amd64   ASSINATURA: false")
$proc += ("SHA256: " + $hash)
$proc += ("TAMANHO: " + $item.Length + " B")
$proc += ""
$proc += "COMPOSICAO: " + $(if ($composicao) { "SIM -- " + $desatualizados.Count + " de " + $proveniencia.Count + " alvos sao anteriores ao HEAD" } else { "NAO -- todos os alvos sao do HEAD" })
$proc += ""
$proc += "PROVENIENCIA POR ALVO (medida por este script, nao escrita a mao):"
$proc += "  metodo: SHA-256 do ficheiro + data de escrita comparada com a data do commit HEAD"
$proc += "  LIMITE: a data e' uma PISTA, nao uma certidao -- um alvo reconstruido de um commit"
$proc += "          antigo mas escrito DEPOIS do HEAD apareceria aqui como coerente."
foreach ($p in $proveniencia) {
    $marca = ""
    if ($p.maisAntigoQueOHead) { $marca = "  [ANTERIOR AO HEAD]" }
    $proc += ("  - " + $p.papel)
    $proc += ("      caminho: " + $p.caminho)
    $proc += ("      tipo:    " + $p.tipo + "   " + $p.detalhe + $marca)
    $proc += ("      escrito: " + $p.escrito.ToString("yyyy-MM-dd HH:mm:ss"))
    $proc += ("      sha256:  " + $p.sha256)
}
$proc += ""
$proc += ("COMANDO: & ISCC /O<out> /FHadesSetup-" + $sha + " app\ollama.iss   (com PKG_VERSION=" + $Versao + ", sem KEY_CONTAINER)")
$proc += ""
if ($composicao) {
    $proc += "AVISO: este instalador NAO e' um build reproduzivel da fonte. Ele empacota binarios"
    $proc += "  produzidos em momentos diferentes. Tratar como COMPOSICAO, nunca como 'build do HEAD'."
    $proc += "  Para exigir coerencia total: reconstruir TODOS os alvos no mesmo commit e correr"
    $proc += "  este script com -ExigirAlvosDoHead."
    $proc += ""
}
$proc += "O SmartScreen vai avisar: e' o esperado num artefacto nao assinado, e nao um defeito."
$proc += ("Este instalador NAO foi publicado; existe apenas localmente em " + $OutDir)
$procPath = Join-Path $out "PROVENIENCIA.txt"
[System.IO.File]::WriteAllLines($procPath, $proc, (New-Object System.Text.UTF8Encoding($false)))
Write-Output ("  PROVENIENCIA {0}  ({1} linhas, {2} alvo(s) declarado(s))" -f $procPath, $proc.Count, $proveniencia.Count)
if ($composicao) {
    Write-Output "  *** AVISO GRAVADO: artefacto e' COMPOSICAO, nao build do HEAD ***"
}
Write-Output "INSTALADOR PRONTO (nao assinado, nao publicado)."
exit 0
