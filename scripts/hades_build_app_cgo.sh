#!/usr/bin/env bash
# Build do shell grafico do Hades para Windows COM cgo.
#
# PORQUE ISTO EXISTE (medido em 2026-10-21):
#   O instalador entregue trazia `ollama app.exe` compilado com CGO_ENABLED=0. Nesse modo o
#   app/webview cai no `webview_stub.go` (`//go:build !cgo`), cujo Run() e' vazio e cujo Window()
#   devolve nil -- ou seja, NAO EXISTE JANELA. Ao mesmo tempo o go-sqlite3 (app/store) devolve
#   "Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work. This is a stub",
#   e a app nao consegue abrir settings/onboarding.
#   Resultado observado: icone na bandeja, nenhuma janela, e um binario de ~44,7 MB.
#
# ATENCAO A UMA ARMADILHA REAL: o mingw-w64 fornece `eventtoken.h` em minusculas, mas o WebView2
#   inclui `EventToken.h`. Em Linux o sistema de ficheiros distingue maiusculas, logo o build falha
#   com "EventToken.h: No such file or directory". Os symlinks abaixo resolvem isso.
#
# USO (no WSL):  bash scripts/hades_build_app_cgo.sh [saida.exe]
set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${1:-$REPO/_tmp/hades-app-cgo.exe}"
INC=/usr/x86_64-w64-mingw32/include

cd "$REPO"

echo "== 1. toolchain =="
export PATH=/usr/local/go/bin:/usr/bin:/bin:/usr/sbin:/sbin
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.10}"
export GOFLAGS=-mod=mod
command -v go >/dev/null || { echo "FALHA: go nao encontrado"; exit 1; }
echo "    go: $(go version)"

if [ ! -x /usr/bin/x86_64-w64-mingw32-gcc ] || [ ! -x /usr/bin/x86_64-w64-mingw32-g++ ]; then
  echo "    a instalar mingw-w64 (gcc e g++)..."
  apt-get update -qq >/dev/null 2>&1
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64 >/dev/null 2>&1
fi
echo "    cc:  $(x86_64-w64-mingw32-gcc --version | head -1)"
echo "    cxx: $(x86_64-w64-mingw32-g++ --version | head -1)"

echo "== 2. aliases de capitalizacao (mingw tem minusculas, WebView2 inclui CamelCase) =="
for par in "eventtoken.h:EventToken.h" "wrl.h:Wrl.h" "rpc.h:Rpc.h" "rpcndr.h:RpcNdr.h"; do
  de="${par%%:*}"; para="${par##*:}"
  if [ -f "$INC/$de" ] && [ ! -e "$INC/$para" ]; then
    ln -sf "$de" "$INC/$para" && echo "    $de -> $para"
  fi
done

echo "== 3. build (CGO_ENABLED=1 windows/amd64) =="
mkdir -p "$(dirname "$OUT")"
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
  go build -o "$OUT" ./app/cmd/app

echo "== 4. guarda anti-stub =="
TAM=$(stat -c %s "$OUT")
echo "    tamanho: $TAM bytes"
# O binario com webview real tem ~85 MB; o stub tem ~44,7 MB. Um fallback para o stub ja' aconteceu
# em producao, por isso o build falha alto em vez de entregar um app sem janela.
if [ "$TAM" -lt 60000000 ]; then
  echo "FALHA: binario com $TAM bytes parece ser o STUB (sem cgo). Nao entregar."
  exit 1
fi
echo "OK: $OUT"
