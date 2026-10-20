# hades_vault_e2e.ps1 -- E2E real das rotas governadas do vault.
#
# PORQUE ESTE FICHEIRO EXISTE
#
#   As rotas POST /memory/vault/append e GET /memory/vault/read existem e tem testes
#   em processo. O que faltava era exercita-las contra o SERVIDOR REAL, com token
#   emitido pela rota de desenvolvimento e duas identidades em organizacoes diferentes.
#
#   O cliente e' System.Net.Http.HttpClient, nao `Invoke-WebRequest` nem `curl.exe`:
#     - IWR tenta PEDIR CREDENCIAIS em modo nao interativo (Read and Prompt indisponivel);
#     - curl.exe recebe o JSON SEM AS ASPAS (mangling de argumentos nativos do PS 5.1);
#   Nenhum dos dois e' defeito do produto -- eram defeitos do instrumento.
#
# ASCII puro de proposito: o PowerShell 5.1 le .ps1 sem BOM como Windows-1252.

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Net.Http
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$repo = Split-Path -Parent $PSScriptRoot
if (-not (Test-Path (Join-Path $repo 'go.mod'))) { $repo = (Get-Location).Path }
# O binario do servidor e' construido a partir do codigo desta arvore. Se nao existir,
# o script diz o comando exacto (nao adivinha, nao descarrega nada).
$bin = Join-Path $repo '_tmp\hades-server-e2e.exe'
if (-not (Test-Path $bin)) { $bin = Join-Path $repo '_tmp\hades-server-2d30.exe' }
if (-not (Test-Path $bin)) {
    Write-Host 'FALHA: falta o binario do servidor. Construa-o com:'
    Write-Host '  go build -o _tmp\hades-server-e2e.exe .'
    exit 1
}

$vault = Join-Path $repo '_tmp\e2e_vault'
$dados = Join-Path $repo '_tmp\appdata-e2e'
foreach ($d in @($vault, $dados)) { if (Test-Path $d) { Remove-Item $d -Recurse -Force }; New-Item -ItemType Directory -Force -Path $d | Out-Null }

$env:APPDATA = $dados
$env:OLLAMA_HOST = '127.0.0.1:11500'
$env:OLLAMA_AGENT_AUTH_REQUIRED = 'true'
$env:OLLAMA_AGENT_AUTH_DEV = 'true'
$env:OLLAMA_AGENT_AUTH_DEV_SECRET = [guid]::NewGuid().ToString('N')
$env:HADES_VAULT_ROOT = $vault
$env:HADES_VAULT_ACL = ''
$env:OLLAMA_MODELS = Join-Path $repo '_tmp\models'
New-Item -ItemType Directory -Force -Path $env:OLLAMA_MODELS | Out-Null

$base = 'http://127.0.0.1:11500/api/agent/v1'
$script:proc = $null

function Arrancar {
    $script:proc = Start-Process -FilePath $bin -ArgumentList 'serve' -PassThru -NoNewWindow `
        -RedirectStandardOutput (Join-Path $repo '_tmp\e2e_vault.log') `
        -RedirectStandardError (Join-Path $repo '_tmp\e2e_vault.err')
    $cliente = New-Object Net.Http.HttpClient
    $cliente.Timeout = [TimeSpan]::FromSeconds(3)
    for ($i = 0; $i -lt 20; $i++) {
        Start-Sleep -Seconds 2
        try { $r = $cliente.GetAsync('http://127.0.0.1:11500/').GetAwaiter().GetResult(); return $true } catch { }
    }
    return $false
}

function Parar { if ($script:proc) { Stop-Process -Id $script:proc.Id -Force -ErrorAction SilentlyContinue; Start-Sleep -Seconds 2 } }

function Pedido($metodo, $url, $token, $corpoTexto) {
    $cliente = New-Object Net.Http.HttpClient
    $cliente.Timeout = [TimeSpan]::FromSeconds(15)
    $pedido = New-Object Net.Http.HttpRequestMessage ([Net.Http.HttpMethod]::$metodo), $url
    if ($token) { $pedido.Headers.Add('Authorization', 'Bearer ' + $token) }
    if ($null -ne $corpoTexto) {
        $pedido.Content = New-Object Net.Http.StringContent ($corpoTexto, [Text.Encoding]::UTF8, 'application/json')
    }
    $resp = $cliente.SendAsync($pedido).GetAwaiter().GetResult()
    $texto = $resp.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    return [pscustomobject]@{ Codigo = [int]$resp.StatusCode; Corpo = $texto }
}

function CriarUtilizador($email, $nome, $org) {
    $corpo = '{"email":"' + $email + '","name":"' + $nome + '","organization":"' + $org + '"}'
    $cliente = New-Object Net.Http.HttpClient
    $pedido = New-Object Net.Http.HttpRequestMessage ([Net.Http.HttpMethod]::Post), "$base/auth/dev/token"
    $pedido.Headers.Add('X-Ollama-Agent-Dev-Secret', $env:OLLAMA_AGENT_AUTH_DEV_SECRET)
    $pedido.Content = New-Object Net.Http.StringContent ($corpo, [Text.Encoding]::UTF8, 'application/json')
    $resp = $cliente.SendAsync($pedido).GetAwaiter().GetResult()
    $texto = $resp.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    if ([int]$resp.StatusCode -ne 201) { Write-Host ('  dev/token ' + $email + ' -> ' + [int]$resp.StatusCode + ' ' + $texto); return $null }
    return ($texto | ConvertFrom-Json)
}

$falhas = 0
function Exigir($condicao, $descricao) {
    if ($condicao) { Write-Host ('  OK   ' + $descricao) } else { Write-Host ('  FALHA ' + $descricao); $script:falhas++ }
}

Write-Host '== E2E do vault governado =='
Parar
Exigir (Arrancar) 'servidor real arranca e responde em loopback'

$a = CriarUtilizador 'usera@local' 'User A' 'Org A'
$b = CriarUtilizador 'usera2@local' 'User B' 'Org B'
Exigir ($null -ne $a) 'token de dev emitido para A (201)'
Exigir ($null -ne $b) 'token de dev emitido para B (201)'
if ($null -eq $a -or $null -eq $b) { Parar; exit 1 }
Exigir ($a.organization.id -ne $b.organization.id) 'A e B ficam em organizacoes DIFERENTES'

$acl = '[{"identidade":"' + $a.user.id + '","escopo":"TENANT","permissoes":["ler","escrever"]},' +
       '{"identidade":"' + $b.user.id + '","escopo":"TENANT","permissoes":["ler"]}]'
$env:HADES_VAULT_ACL = $acl
Parar
Exigir (Arrancar) 'servidor reinicia com ACL das identidades reais'

$tokA = $a.access_token
$tokB = $b.access_token

$escrita = Pedido 'Post' "$base/memory/vault/append" $tokA '{"relativo":"notas/decisao.md","conteudo":"decisao tomada no chat A","rotulo":"FACT_VERIFIED","fonte":"chat-A"}'
Exigir ($escrita.Codigo -eq 201) ('append com A -> 201 (veio ' + $escrita.Codigo + ')')

$leituraA = Pedido 'Get' "$base/memory/vault/read?relativo=notas/decisao.md" $tokA $null
Exigir ($leituraA.Codigo -eq 200) ('read com A -> 200 (veio ' + $leituraA.Codigo + ')')
Exigir ($leituraA.Corpo -match 'decisao tomada no chat A') 'o conteudo voltou igual'
Exigir ($leituraA.Corpo -match 'chat-A') 'a proveniencia (fonte) voltou'

$outraOrg = Pedido 'Get' "$base/memory/vault/read?relativo=notas/decisao.md" $tokB $null
$inexistente = Pedido 'Get' "$base/memory/vault/read?relativo=notas/nao-existe.md" $tokB $null
Exigir ($outraOrg.Codigo -eq 403) ('read com B de nota de outra organizacao -> 403 (veio ' + $outraOrg.Codigo + ')')
Exigir ($inexistente.Codigo -eq 403) ('read com B de nota inexistente -> 403 (veio ' + $inexistente.Codigo + ')')
Exigir ($outraOrg.Corpo -eq $inexistente.Corpo) 'as duas respostas sao IDENTICAS (nao revela existencia)'
Exigir (-not ($outraOrg.Corpo -match 'decisao tomada no chat A')) 'a resposta nao vaza o conteudo alheio'

$escritaB = Pedido 'Post' "$base/memory/vault/append" $tokB '{"relativo":"notas/de-b.md","conteudo":"x","rotulo":"FACT_VERIFIED","fonte":"chat-B"}'
Exigir ($escritaB.Codigo -eq 403) ('append com B (so leitura) -> 403 (veio ' + $escritaB.Codigo + ')')

$semToken = Pedido 'Post' "$base/memory/vault/append" $null '{"relativo":"notas/x.md"}'
Exigir ($semToken.Codigo -eq 401) ('append sem token -> 401 (veio ' + $semToken.Codigo + ')')

# --- fase 2: concorrencia optimista na MESMA nota -------------------------------
# `versao_esperada` e' o mecanismo: um segundo escritor com versao desactualizada
# tem de ser RECUSADO com conflito explicito, nunca sobrescrever em silencio.
$v0 = Pedido 'Post' "$base/memory/vault/append" $tokA '{"relativo":"notas/versao.md","conteudo":"primeira","rotulo":"FACT_VERIFIED","fonte":"chat-A","versao_esperada":0}'
Exigir ($v0.Codigo -eq 201) ('1a escrita com versao_esperada=0 -> 201 (veio ' + $v0.Codigo + ')')

$vDesactualizada = Pedido 'Post' "$base/memory/vault/append" $tokA '{"relativo":"notas/versao.md","conteudo":"segunda-com-versao-velha","rotulo":"FACT_VERIFIED","fonte":"chat-A","versao_esperada":0}'
Exigir ($vDesactualizada.Codigo -eq 409) ('2a escrita com a MESMA versao_esperada -> 409 (veio ' + $vDesactualizada.Codigo + ')')

$confirmacao = Pedido 'Get' "$base/memory/vault/read?relativo=notas/versao.md" $tokA $null
Exigir (-not ($confirmacao.Corpo -match 'segunda-com-versao-velha')) 'a escrita recusada NAO entrou na nota'

$vCorrecta = Pedido 'Post' "$base/memory/vault/append" $tokA '{"relativo":"notas/versao.md","conteudo":"terceira-com-versao-certa","rotulo":"FACT_VERIFIED","fonte":"chat-A","versao_esperada":1}'
Exigir ($vCorrecta.Codigo -eq 201) ('escrita com versao_esperada=1 -> 201 (veio ' + $vCorrecta.Codigo + ')')
$ficheiros = @(Get-ChildItem $vault -Recurse -File -ErrorAction SilentlyContinue)
Exigir ($ficheiros.Count -eq 2) ('o vault tem exactamente 2 ficheiros: decisao.md e versao.md (tem ' + $ficheiros.Count + ')')

Parar
Write-Host ('== fim: ' + $falhas + ' falha(s) ==')
if ($falhas -gt 0) { exit 1 }
exit 0
