# B12 -- medicao de performance do Hades (arranque + latencia nucleo).
# ASCII puro: o PowerShell 5.1 le .ps1 sem BOM como Windows-1252.
param([int]$Corridas = 3, [int]$Porta = 11437)

$repo = 'C:\Users\DZ23\Documents\deepseek-harness\default-workspace\hades-ollama'
Set-Location $repo
New-Item -ItemType Directory -Force -Path "$repo\_tmp\b12-store","$repo\_tmp\b12-ws" | Out-Null
$env:OLLAMA_HOST = "127.0.0.1:$Porta"
$env:OLLAMA_AGENT_STORE = "$repo\_tmp\b12-store"
$env:OLLAMA_AGENT_ROOT = "$repo\_tmp\b12-ws"

$arranques = @()
for ($i = 1; $i -le $Corridas; $i++) {
    $out = "$repo\_tmp\b12_$i.out"
    $err = "$repo\_tmp\b12_$i.err"
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    $p = Start-Process -FilePath "$repo\_tmp\hades_b12.exe" -ArgumentList 'serve' -PassThru `
        -RedirectStandardOutput $out -RedirectStandardError $err -WindowStyle Hidden
    $ok = $false
    while ($sw.Elapsed.TotalSeconds -lt 45) {
        try {
            $r = Invoke-WebRequest -Uri "http://127.0.0.1:$Porta/api/version" -TimeoutSec 2 -UseBasicParsing
            if ($r.StatusCode -eq 200) { $ok = $true; break }
        } catch {
            Start-Sleep -Milliseconds 25
        }
    }
    $sw.Stop()
    $ms = [Math]::Round($sw.Elapsed.TotalMilliseconds, 0)

    if ($ok -and $i -eq 1) {
        Write-Output "  --- latencia de um endpoint nucleo (20 chamadas) ---"
        $lat = @()
        foreach ($k in 1..20) {
            $s2 = [System.Diagnostics.Stopwatch]::StartNew()
            try { $null = Invoke-WebRequest -Uri "http://127.0.0.1:$Porta/api/agent/v1/projects" -TimeoutSec 5 -UseBasicParsing } catch { }
            $s2.Stop()
            $lat += [Math]::Round($s2.Elapsed.TotalMilliseconds, 1)
        }
        $o = $lat | Sort-Object
        Write-Output ("    min={0} ms  p50={1} ms  p95={2} ms  max={3} ms" -f $o[0], $o[9], $o[18], $o[19])
    }

    Write-Output ("  arranque #{0} -> {1} ms (ok={2})" -f $i, $ms, $ok)
    $arranques += $ms
    if (-not $p.HasExited) { Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue }
    Start-Sleep -Milliseconds 900
}

$o = $arranques | Sort-Object
Write-Output "  === RESUMO (n=$Corridas) ==="
Write-Output ("    arranque: min={0} ms  mediana={1} ms  max={2} ms" -f $o[0], $o[[int]($Corridas / 2)], $o[-1])
