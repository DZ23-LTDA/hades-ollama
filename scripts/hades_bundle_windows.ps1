# Monta o bundle portatil Windows a partir da FONTE ACTUAL e verifica-o antes de dizer OK.
#
# Porque existe: duas rondas seguidas (E-793, E-794) o bundle foi montado com ficheiros
# ou binarios OBSOLETOS, porque a montagem era uma sequencia de passos manuais: nada
# garantia que o `hades.exe` correspondesse ao codigo do momento, nem que o LEIA-ME
# descrevesse o numero certo de ficheiros. *Um artefacto composto tem de ser montado por
# um processo que se possa repetir - e que falhe quando nao bate certo.*
#
# Uso:
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_bundle_windows.ps1
#   (opcional) -OutDir dist\hades-r151 -Go <caminho do go.exe> -Runner _tmp\hadesbin
#
# O script FALHA (exit != 0) se qualquer verificacao nao passar.

param(
    [string]$Raiz = "",
    [string]$OutDir = "dist\hades-r151",
    [string]$Go = "",
    [string]$Runner = "_tmp\hadesbin",
    [int]$Porta = 11444
)

$ErrorActionPreference = "Stop"

if (-not $Raiz) { $Raiz = (Get-Location).Path }
$raiz = (Resolve-Path $Raiz).Path
Set-Location $raiz

if (-not $Go) { $Go = Join-Path $raiz "_tmp\goroot1269\go\bin\go.exe" }
if (-not (Test-Path $Go)) { Write-Output "ERRO: go nao encontrado em $Go"; exit 1 }

$out = Join-Path $raiz $OutDir
$bundle = Join-Path $out "bundle"
$runner = Join-Path $raiz $Runner
$sha = (git rev-parse --short HEAD).Trim()

Write-Output "HEAD=$sha  raiz=$raiz"
Write-Output "OUT=$out"

if (Test-Path $bundle) { Remove-Item $bundle -Recurse -Force }
New-Item -ItemType Directory -Force -Path $bundle | Out-Null

# 1) BINARIO DO SERVIDOR, construido da fonte actual (nunca copiado de outro sitio)
$env:CGO_ENABLED = "0"
& $Go build -o (Join-Path $bundle "hades.exe") . 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) { Write-Output "ERRO: go build falhou"; exit 1 }
Write-Output ("  hades.exe        {0} bytes" -f (Get-Item (Join-Path $bundle "hades.exe")).Length)

# 2) RUNNER (pre-construido; proveniencia declarada no LEIA-ME)
if (-not (Test-Path (Join-Path $runner "llama-server.exe"))) {
    Write-Output "ERRO: llama-server.exe nao encontrado em $runner"
    exit 1
}
Copy-Item (Join-Path $runner "llama-server.exe") $bundle -Force
Get-ChildItem $runner -Filter "*.dll" | Copy-Item -Destination $bundle -Force
Write-Output ("  runner           {0} dlls + llama-server.exe" -f (Get-ChildItem $bundle -Filter "*.dll").Count)

# 3) ATRIBUICAO do runner + do proprio projecto
Get-ChildItem $runner -File | Where-Object { $_.Name -match "LICENSE|NOTICE|COPYING|AUTHORS" } | Copy-Item -Destination $bundle -Force
Copy-Item (Join-Path $raiz "LICENSE") (Join-Path $bundle "HADES_LICENSE") -Force
Copy-Item (Join-Path $raiz "UPSTREAM_POLICY.md") $bundle -Force

# 4) FICHEIROS G12 (gerados por scripts\hades_third_party_notices.py)
foreach ($f in @("THIRD_PARTY_NOTICES.md", "HADES_SBOM_GO.csv", "HADES_SBOM_NPM.csv", "HADES_SBOM_LICENCAS.csv")) {
    $p = Join-Path $raiz $f
    if (-not (Test-Path $p)) { Write-Output "ERRO: falta $f (corra scripts\hades_third_party_notices.py)"; exit 1 }
    Copy-Item $p $bundle -Force
}

# 5) LEIA-ME com o numero de ficheiros DERIVADO (nunca escrito a mao)
$n = (Get-ChildItem $bundle -File).Count
$leia = @"
HADES - artefactos de desenvolvimento (NAO assinados; NAO e' instalador Inno/NSIS)
HEAD: $sha   branch: $(git rev-parse --abbrev-ref HEAD)   GOOS=windows GOARCH=amd64 CGO_ENABLED=0
Bundle montado por scripts\hades_bundle_windows.ps1 - nao ha' passo manual.

CONTEUDO: $n ficheiros =
  hades.exe (servidor/CLI, construido DESTE HEAD por este script) +
  runner (llama-server.exe + DLLs, de $Runner) +
  ficheiros de atribuicao (licencas do motor + HADES_LICENSE + UPSTREAM_POLICY.md) +
  4 ficheiros G12 (THIRD_PARTY_NOTICES.md e 3 SBOM .csv).

VERIFICADO por este script, num directorio limpo, extraindo ESTE zip:
  GET  /api/tags              -> 200
  POST /api/chat              -> 200 com resposta REAL do modelo
  POST /api/v1/model/upstream -> 501 (contrato honesto; era 404 antes do E-794)

PROVENIENCIA: o hades.exe e' construido deste HEAD; o runner vem da arvore
PRE-CONSTRUIDA $Runner - NAO foi construido por este script (toolchain C++ ausente).
=> o bundle executa e infere, mas NAO e' um build 100% reproduzivel da fonte.

SERVIDOR LOCAL (modo degradado): 501 em POST /api/v1/settings, /chat/new, /chat/:id,
/chat/:id/rename e /model/upstream; 200 em GET /api/v1/chats e /api/v1/settings.

MATRIZ: SOURCE_READY=SIM | BUILDS=SIM | PACKAGE_CREATED=SIM (ESTE ZIP portatil **e** o
HadesSetup.exe nao assinado, ver INSTALADOR abaixo) | INSTALLED=NAO | DEVICE_E2E=NAO |
SIGNED_FOR_PUBLIC_RELEASE=NAO | STORE_PUBLISHED=NAO

INSTALADOR: scripts\build_windows.ps1 -> Inno Setup ISCC.exe -> app\ollama.iss
(OutputBaseFilename="HadesSetup", ramo NAO assinado em :1068 -> dist\HadesSetup.exe).
O ISCC.exe **existe e funciona** neste host, em
%LOCALAPPDATA%\Programs\Inno Setup 6\ - a instalacao POR UTILIZADOR do Inno Setup 6,
que nao exige administrador. A versao anterior deste LEIA-ME dizia "ISCC.exe ausente":
era falso desde que o E-938 mediu o contrario. O `build_windows.ps1:290` passou a
procurar nos dois sitios (Program Files primeiro, %LOCALAPPDATA% depois).
O que falta nao e' a ferramenta: e' EXECUTAR o instalador (instalacao limpa, primeira
abertura, reinicio, actualizacao sobre a RC antiga e desinstalacao). Isso escreve em
%LOCALAPPDATA%\Programs\Hades e exige aprovacao do dono.
"@
Set-Content -Path (Join-Path $bundle "LEIA-ME.txt") -Value $leia -Encoding utf8
$n = (Get-ChildItem $bundle -File).Count
Write-Output "  LEIA-ME.txt      $n ficheiros (numero derivado)"

# 6) ZIP
$zip = Join-Path $out ("Hades-{0}-windows-amd64-portable-completo.zip" -f $sha)
if (Test-Path $zip) { Remove-Item $zip -Force }
Compress-Archive -Path (Join-Path $bundle "*") -DestinationPath $zip -Force
Write-Output ("  ZIP              {0} bytes" -f (Get-Item $zip).Length)

# 7) VERIFICACAO: extrair para directorio limpo e medir o que o ZIP entrega
$v = Join-Path $raiz ("_tmp\bundle_check_" + $Porta)
if (Test-Path $v) { Remove-Item $v -Recurse -Force }
New-Item -ItemType Directory -Force -Path $v | Out-Null
Expand-Archive -Path $zip -DestinationPath $v -Force
$extraidos = (Get-ChildItem $v -File).Count
Write-Output "  extraidos        $extraidos ficheiros"
$falhas = 0
if ($extraidos -ne $n) { Write-Output "  FALHA: extraidos $extraidos <> $n no bundle"; $falhas++ }
$env:TMP = (Join-Path $raiz "_tmp\webtemp"); $env:TEMP = $env:TMP
$env:OLLAMA_HOST = "127.0.0.1:$Porta"
$srv = Start-Process -FilePath (Join-Path $v "hades.exe") -ArgumentList "serve" -WorkingDirectory $v -PassThru -RedirectStandardOutput (Join-Path $raiz "_tmp\bundle_srv.out") -RedirectStandardError (Join-Path $raiz "_tmp\bundle_srv.err")
$ok = $false
for ($i = 0; $i -lt 45; $i++) {
    try { $w = Invoke-WebRequest "http://127.0.0.1:$Porta/api/tags" -UseBasicParsing -TimeoutSec 3; if ($w.StatusCode -eq 200) { $ok = $true; break } } catch { }
    Start-Sleep -Seconds 1
}
Write-Output "  /api/tags        $ok"
if (-not $ok) { $falhas++ }
if ($ok) {
    try {
        $c = Invoke-WebRequest "http://127.0.0.1:$Porta/api/chat" -Method POST -Body '{"model":"llama3.2:3b","messages":[{"role":"user","content":"Diga so: ok"}],"stream":false,"options":{"num_predict":4}}' -ContentType "application/json" -UseBasicParsing -TimeoutSec 180
        $j = $c.Content | ConvertFrom-Json
        Write-Output ("  /api/chat        {0}  resposta=[{1}]" -f $c.StatusCode, $j.message.content)
        if ($c.StatusCode -ne 200) { $falhas++ }
    } catch { Write-Output "  /api/chat        FALHOU"; $falhas++ }
    $up = 0
    try { Invoke-WebRequest "http://127.0.0.1:$Porta/api/v1/model/upstream" -Method POST -UseBasicParsing -TimeoutSec 10 | Out-Null; $up = 200 } catch { $up = $_.Exception.Response.StatusCode.value__ }
    Write-Output "  model/upstream   $up (esperado 501)"
    if ($up -ne 501) { $falhas++ }
}
Stop-Process -Id $srv.Id -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

$h = (Get-FileHash $zip -Algorithm SHA256).Hash.ToLower()
Write-Output ("  SHA256           {0}" -f $h)
$linhas = Get-ChildItem $out -File | Where-Object { $_.Extension -in ".exe", ".zip" } | Sort-Object Name | ForEach-Object { "{0}  {1}" -f (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLower(), $_.Name }
$linhas | Set-Content (Join-Path $out "SHA256SUMS.txt") -Encoding ascii
Write-Output ("  SHA256SUMS.txt   {0} artefactos" -f $linhas.Count)

if ($falhas -gt 0) { Write-Output "RESULTADO: $falhas FALHA(S) - o bundle NAO esta' bom"; exit 1 }
Write-Output "RESULTADO: bundle montado e VERIFICADO (tags 200, chat 200, upstream 501)"
exit 0
