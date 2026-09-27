#!/usr/bin/env bash
# Packaging Linux (CPU) do Ollama Full.
# Produz um tarball reproduzivel com o binario, VERSION, metadata, install/
# uninstall e checksum. NAO cobre GPU (CUDA/ROCm/Vulkan) — esses ficam no
# pipeline de release com runner apropriado.
#
# Uso: scripts/package_linux.sh [outdir]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

OUTDIR="${1:-dist}"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) GOARCH="amd64"; ARCH="amd64" ;;
  aarch64|arm64) GOARCH="arm64"; ARCH="arm64" ;;
  *) echo "arch nao suportada: $ARCH" >&2; exit 1 ;;
esac

# Versao derivada do build (tag), sem hardcode.
if [ -z "${VERSION:-}" ]; then
  VERSION="$(git describe --tags --match 'v*' --first-parent --abbrev=7 --long --dirty 2>/dev/null | sed 's/^v//')"
  [ -n "$VERSION" ] || VERSION="0.0.0-dev+$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)"
fi
COMMIT="$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)"

PKG="ollama-full-${VERSION}-linux-${ARCH}"
STAGE="${OUTDIR}/${PKG}"
mkdir -p "${STAGE}/bin"

echo ">> build ollama (CPU) ${VERSION} ${GOARCH}"
CGO_ENABLED=1 GOARCH="${GOARCH}" go build -trimpath \
  -ldflags "-s -w -X=github.com/ollama/ollama/version.Version=${VERSION} -X=github.com/ollama/ollama/server.mode=release" \
  -o "${STAGE}/bin/ollama" .

# Metadata versionada dentro do pacote.
cat > "${STAGE}/VERSION" <<META
product=Ollama Full
version=${VERSION}
commit=${COMMIT}
arch=${ARCH}
built_at_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)
signed=false
META

# Instalador local (sem root obrigatorio): prefixo configuravel.
cat > "${STAGE}/install.sh" <<'INSTALL'
#!/usr/bin/env bash
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PREFIX="${PREFIX:-$HOME/.local}"
install -Dm755 "${HERE}/bin/ollama" "${PREFIX}/bin/ollama"
install -Dm644 "${HERE}/VERSION" "${PREFIX}/share/ollama-full/VERSION"
echo "Ollama Full instalado em ${PREFIX}/bin/ollama"
echo "Garanta que ${PREFIX}/bin esta no PATH. Rode: ollama --version"
INSTALL
chmod +x "${STAGE}/install.sh"

cat > "${STAGE}/uninstall.sh" <<'UNINSTALL'
#!/usr/bin/env bash
set -euo pipefail
PREFIX="${PREFIX:-$HOME/.local}"
rm -f "${PREFIX}/bin/ollama"
rm -rf "${PREFIX}/share/ollama-full"
echo "Ollama Full removido de ${PREFIX}. Dados do usuario em ~/.ollama nao foram tocados."
UNINSTALL
chmod +x "${STAGE}/uninstall.sh"

# Tarball reproduzivel (ordem estavel, mtime/owners normalizados).
TARBALL="${OUTDIR}/${PKG}.tgz"
echo ">> empacotando ${TARBALL}"
tar --sort=name --owner=0 --group=0 --numeric-owner \
    --mtime="@${SOURCE_DATE_EPOCH:-0}" \
    -czf "${TARBALL}" -C "${OUTDIR}" "${PKG}"

# Checksum.
( cd "${OUTDIR}" && sha256sum "${PKG}.tgz" > "${PKG}.tgz.sha256" )
echo ">> pronto:"
ls -l "${TARBALL}" "${TARBALL}.sha256"
cat "${TARBALL}.sha256"
