#!/usr/bin/env bash
# Hades/Ollama Full local-first quickstart.
# Builds the checked-out source, starts backend + web UI, and optionally offers a
# recommended local model. It never downloads binaries or credentials silently.
set -Eeuo pipefail

ROOT=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
INSTALL_DIR=${OLLAMA_INSTALL_DIR:-"$ROOT/bin"}
BINARY=${OLLAMA_BINARY:-"$INSTALL_DIR/ollama-full"}
HOST=${OLLAMA_HOST:-"127.0.0.1:11434"}
UI_PORT=${HADES_UI_PORT:-5173}
UI_HOST=${HADES_UI_HOST:-127.0.0.1}
MODEL=${HADES_RECOMMENDED_MODEL:-"gemma4:e2b"}
NO_MODEL=0

usage() {
  cat <<EOF
Uso: $(basename "$0") [--no-model]

Builda e inicia Hades localmente:
  backend: http://$HOST
  UI:      http://$UI_HOST:$UI_PORT

Variáveis: OLLAMA_INSTALL_DIR, OLLAMA_BINARY, OLLAMA_HOST,
HADES_UI_PORT, HADES_UI_HOST, HADES_UI_ALLOW_LAN, HADES_UI_AUTH_REQUIRED,
HADES_RECOMMENDED_MODEL.
EOF
}
for arg in "$@"; do
  case "$arg" in
    --no-model) NO_MODEL=1 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Argumento desconhecido: $arg" >&2; usage >&2; exit 2 ;;
  esac
done

command -v go >/dev/null 2>&1 || { echo "Go não encontrado. Instale Go conforme o guia do projeto." >&2; exit 1; }
command -v node >/dev/null 2>&1 || { echo "Node.js não encontrado. Instale Node.js conforme o guia do projeto." >&2; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "npm não encontrado. Instale Node.js conforme o guia do projeto." >&2; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "curl não encontrado; ele é necessário para verificar a saúde do backend." >&2; exit 1; }
[[ -f "$ROOT/go.mod" && -f "$ROOT/app/ui/app/package.json" ]] || { echo "Execute este script na árvore clonada do Hades." >&2; exit 1; }

mkdir -p "$INSTALL_DIR"
echo "[1/4] Buildando backend..."
(cd "$ROOT" && go build -trimpath -o "$BINARY" .)
chmod 0755 "$BINARY"

echo "[2/4] Instalando dependências e buildando UI..."
(cd "$ROOT/app/ui/app" && npm ci --no-audit --no-fund && npm run build)

cleanup() {
  local code=$?
  trap - EXIT INT TERM
  [[ -n "${BACKEND_PID:-}" ]] && kill "$BACKEND_PID" 2>/dev/null || true
  [[ -n "${UI_PID:-}" ]] && kill "$UI_PID" 2>/dev/null || true
  exit "$code"
}
trap cleanup EXIT INT TERM

echo "[3/4] Iniciando backend em $HOST..."
OLLAMA_HOST="$HOST" "$BINARY" serve &
BACKEND_PID=$!
echo "Aguardando o backend responder..."
for _ in $(seq 1 40); do
  if curl -fsS "http://${HOST}/api/version" >/dev/null 2>&1; then break; fi
  sleep 0.25
done
curl -fsS "http://${HOST}/api/version" >/dev/null || { echo "Backend não ficou saudável." >&2; exit 1; }
echo "Backend saudável em http://${HOST}"

if [[ "$NO_MODEL" -eq 0 ]]; then
  tags=$(curl -fsS "http://${HOST}/api/tags" || echo '{"models":[]}')
  if [[ "$tags" == *'"models":[]'* || "$tags" == *'"models": []'* ]]; then
    if [[ -t 0 && -t 1 ]]; then
      printf 'Nenhum modelo local encontrado. Baixar %s agora? [s/N] ' "$MODEL"
      read -r answer
      if [[ "$answer" =~ ^[sS]$ ]]; then
        echo "Baixando modelo recomendado: $MODEL"
        "$BINARY" pull "$MODEL"
      else
        echo "Sem modelo: a UI abrirá em estado acionável; você pode instalar um depois com: $BINARY pull $MODEL"
      fi
    else
      echo "Nenhum modelo local encontrado; não baixando em modo não interativo. Use: $BINARY pull $MODEL"
    fi
  fi
fi

echo "[4/4] Iniciando UI em http://$UI_HOST:$UI_PORT ..."
# Do not let Vite silently move to another port: the advertised URL must be
# the URL that is actually serving the UI.
(cd "$ROOT/app/ui/app" && HADES_UI_HOST="$UI_HOST" HADES_UI_ALLOW_LAN="${HADES_UI_ALLOW_LAN:-false}" HADES_UI_AUTH_REQUIRED="${HADES_UI_AUTH_REQUIRED:-false}" npm run preview -- --host "$UI_HOST" --port "$UI_PORT" --strictPort) &
UI_PID=$!
echo
echo "Hades está disponível em http://$UI_HOST:$UI_PORT"
echo "Backend saudável: http://${HOST}"
echo "Se a tela informar que não há modelo, abra Configurações ou execute: $BINARY pull $MODEL"
echo "Pressione Ctrl+C para encerrar backend e UI."
wait "$UI_PID"
