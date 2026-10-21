# HADES Local Companion -- instalador/desinstalador do native messaging host.
#
# ASCII PURO. O `powershell.exe` deste host e' 5.1 Desktop e le' `.ps1` sem BOM como
# Windows-1252: um acento aqui vira lixo e pode partir o parser.
#
# REVERSIVEL POR CONSTRUCAO: tudo o que este script cria fica registado num manifesto
# de instalacao em `%LOCALAPPDATA%\HADES\NativeHost\INSTALACAO.json`, e `-Desinstalar`
# remove exactamente o que esta' listado. Nada de "apagar a pasta e rezar".
#
# O QUE ELE **NAO** FAZ: nao publica, nao assina, nao altera o sistema fora do HKCU,
# nao pede elevacao e nao apaga projetos do usuario.
#
# NAO PUBLICADO, NAO ASSINADO.

[CmdletBinding()]
param(
    [switch]$Desinstalar,
    [switch]$Diagnostico,
    [string]$Navegador = 'chrome,edge'
)

$ErrorActionPreference = 'Stop'

$NomeDoHost = 'com.dz23ltd.hades.bridge'
$Raiz = Split-Path -Parent $PSScriptRoot
$Bridge = Join-Path $Raiz 'apps\hades-browser-bridge'
$Companion = Join-Path $Bridge 'native-host\companion.mjs'
$Gerador = Join-Path $PSScriptRoot 'hades_native_host.py'
$Destino = Join-Path $env:LOCALAPPDATA 'HADES\NativeHost'
$Manifesto = Join-Path $Destino ($NomeDoHost + '.json')
# O `path` do manifesto aponta para o LANCADOR, nao para o `.mjs`: medido no `E-1032`,
# o Windows nao executa `.mjs` directamente e o Chrome responde
# "Error when communicating with the native messaging host".
$Lancador = Join-Path $Destino ($NomeDoHost + '.cmd')
$Registo = Join-Path $Destino 'INSTALACAO.json'

# Chaves do registo do Windows, por navegador. Firefox NAO usa registo -- nao esta' aqui.
$Chaves = @{
    'chrome' = 'HKCU:\Software\Google\Chrome\NativeMessagingHosts\' + $NomeDoHost
    'edge'   = 'HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\' + $NomeDoHost
    'brave'  = 'HKCU:\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\' + $NomeDoHost
}

function Escrever($t) { Write-Host ('  ' + $t) }

function Diagnostico {
    Escrever 'DIAGNOSTICO (nada e'' alterado)'
    Escrever ('  companion      : ' + $Companion)
    Escrever ('  existe         : ' + (Test-Path -LiteralPath $Companion))
    Escrever ('  destino        : ' + $Destino)
    Escrever ('  manifesto      : ' + $Manifesto)
    Escrever ('  manifesto ok   : ' + (Test-Path -LiteralPath $Manifesto))
    Escrever ('  registo gravado: ' + (Test-Path -LiteralPath $Registo))
    foreach ($n in $Chaves.Keys) {
        $existe = Test-Path -LiteralPath $Chaves[$n]
        $valor = ''
        if ($existe) { $valor = (Get-ItemProperty -LiteralPath $Chaves[$n] -Name '(default)' -ErrorAction SilentlyContinue).'(default)' }
        Escrever ('  ' + $n.PadRight(8) + ' registo=' + $existe + ' -> ' + $valor)
    }
    Escrever ''
    Escrever 'VERIFICACAO DO ID (fonte: a chave publica versionada):'
    & python -X utf8 $Gerador --check
    return
}

function Instalar {
    if (-not (Test-Path -LiteralPath $Companion)) { throw ('companion ausente: ' + $Companion) }
    $id = (& python -X utf8 $Gerador --derivar ((Get-Content -LiteralPath (Join-Path $Bridge 'native-host\chave-publica.b64') -Raw -Encoding UTF8).Trim())).Trim()
    if ($id -notmatch '^[a-p]{32}$') { throw ('ID invalido derivado: ' + $id) }

    if (-not (Test-Path -LiteralPath $Destino)) { New-Item -ItemType Directory -Path $Destino -Force | Out-Null }

    # O manifesto TEM de ter caminho absoluto, logo e' especifico da maquina: por isso
    # e' gerado aqui e nao versionado.
    & python -X utf8 $Gerador --escrever $Companion
    if (-not (Test-Path -LiteralPath $Manifesto)) { throw ('manifesto nao foi escrito: ' + $Manifesto) }

    $gravados = @()
    foreach ($n in $Navegador.Split(',')) {
        $n = $n.Trim().ToLowerInvariant()
        if (-not $Chaves.ContainsKey($n)) { Escrever ('AVISO: navegador desconhecido ignorado: ' + $n); continue }
        $chave = $Chaves[$n]
        $pai = Split-Path -Parent $chave
        if (-not (Test-Path -LiteralPath $pai)) { New-Item -Path $pai -Force | Out-Null }
        New-Item -Path $chave -Force | Out-Null
        Set-ItemProperty -LiteralPath $chave -Name '(default)' -Value $Manifesto
        $gravados += $chave
        Escrever ('registado: ' + $n + ' -> ' + $chave)
    }

    $estado = [ordered]@{
        nomeDoHost   = $NomeDoHost
        idDaExtensao = $id
        manifesto    = $Manifesto
        lancador     = $Lancador
        companion    = $Companion
        chavesRegisto = $gravados
        instaladoEm  = (Get-Date).ToString('o')
        assinado     = $false
        publicado    = $false
    }
    $estado | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $Registo -Encoding UTF8
    Escrever ('estado gravado em ' + $Registo)
    Escrever ''
    Escrever ('PRONTO. Abra chrome://extensions, ative o Modo de programador, carregue')
    Escrever ('a pasta ' + $Bridge + ' e confirme que o ID mostrado e'':')
    Escrever ('  ' + $id)
    Escrever 'Se o ID for outro, o `allowed_origins` nao corresponde e o host recusa.'
}

function Desinstalar {
    if (-not (Test-Path -LiteralPath $Registo)) {
        Escrever 'nada instalado (sem INSTALACAO.json). Nada a remover.'
        return
    }
    $estado = Get-Content -LiteralPath $Registo -Raw -Encoding UTF8 | ConvertFrom-Json
    foreach ($chave in $estado.chavesRegisto) {
        if (Test-Path -LiteralPath $chave) {
            Remove-Item -LiteralPath $chave -Recurse -Force
            Escrever ('removido: ' + $chave)
        }
    }
    if (Test-Path -LiteralPath $Manifesto) { Remove-Item -LiteralPath $Manifesto -Force; Escrever ('removido: ' + $Manifesto) }
    # O LANCADOR tambem: *um desinstalador que esquece um ficheiro que o instalador criou
    # nao desinstala -- deixa residuo, e o `--check` seguinte acusa.* Foi medido no `E-1032`.
    if (Test-Path -LiteralPath $Lancador) { Remove-Item -LiteralPath $Lancador -Force; Escrever ('removido: ' + $Lancador) }
    if (Test-Path -LiteralPath $Registo) { Remove-Item -LiteralPath $Registo -Force; Escrever ('removido: ' + $Registo) }
    # SO' apaga a pasta se ficou vazia: se o usuario pos la' algo, preserva-se.
    if ((Test-Path -LiteralPath $Destino) -and -not (Get-ChildItem -LiteralPath $Destino -Force)) {
        Remove-Item -LiteralPath $Destino -Force
        Escrever ('removido (vazio): ' + $Destino)
    }
    Escrever ''
    Escrever 'DESINSTALADO. Os projetos do usuario nao foram tocados.'
}

if ($Diagnostico) { Diagnostico; exit 0 }
if ($Desinstalar) { Desinstalar; exit 0 }
Instalar
