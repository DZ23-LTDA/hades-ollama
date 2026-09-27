#!/bin/sh

set -eu

# Updating a registry manifest is an external publication action. The fork has
# no public image by default, so both the repository and an explicit opt-in are
# required. Never fall back to the upstream ollama/ollama repository.
: "${VERSION:?VERSION is required}"
: "${FINAL_IMAGE_REPO:?FINAL_IMAGE_REPO must name an operator-owned registry}"
if [ "${FINAL_IMAGE_REPO}" = "ollama/ollama" ] || [ "${FINAL_IMAGE_REPO}" = "local/ollama-full" ]; then
    echo "ERROR: refusing to publish an upstream or local placeholder image" >&2
    exit 1
fi
if [ "${OLLAMA_ENABLE_LATEST:-}" != "true" ]; then
    echo "ERROR: set OLLAMA_ENABLE_LATEST=true only after the Ollama Full registry is configured and homologated" >&2
    exit 1
fi

echo "Updating ${FINAL_IMAGE_REPO}:latest -> ${FINAL_IMAGE_REPO}:${VERSION}"
docker buildx imagetools create -t "${FINAL_IMAGE_REPO}:latest" "${FINAL_IMAGE_REPO}:${VERSION}"
echo "Updating ${FINAL_IMAGE_REPO}:rocm -> ${FINAL_IMAGE_REPO}:${VERSION}-rocm"
docker buildx imagetools create -t "${FINAL_IMAGE_REPO}:rocm" "${FINAL_IMAGE_REPO}:${VERSION}-rocm"
