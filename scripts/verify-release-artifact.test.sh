#!/usr/bin/env bash
# Self-contained test for verify-release-artifact.sh. Requires openssl and
# sha256sum (both ship with Git for Windows and standard Linux runners).
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
script="$here/verify-release-artifact.sh"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cd "$work"

# Build a fake-but-real release: keypair, metadata, SBOM, a signed manifest.
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out private.pem 2>/dev/null
openssl rsa -in private.pem -pubout -out release-signing-public.pem 2>/dev/null
cat > release-metadata.json <<'JSON'
{
  "repository": "DZ23-LTDA/hades-ollama",
  "ref": "refs/tags/v1.2.3",
  "commit": "abc123",
  "version": "1.2.3"
}
JSON
echo 'sbom' > ollama-classe-a-plus-sbom.cdx.json
echo 'binary' > HadesSetup.exe
sha256sum release-metadata.json ollama-classe-a-plus-sbom.cdx.json HadesSetup.exe > sha256sum.txt
openssl dgst -sha256 -sign private.pem -out sha256sum.txt.sig sha256sum.txt
fingerprint=$(sha256sum release-signing-public.pem | awk '{print $1}')

pass=0
fail=0
expect_ok() { if "$@" >/dev/null 2>&1; then pass=$((pass+1)); else echo "FAIL (expected success): $*"; fail=$((fail+1)); fi; }
expect_fail() { if "$@" >/dev/null 2>&1; then echo "FAIL (expected failure): $*"; fail=$((fail+1)); else pass=$((pass+1)); fi; }

# 1. Pinned fingerprint + correct identity verifies.
expect_ok env OLLAMA_RELEASE_PUBKEY_SHA256="$fingerprint" \
  OLLAMA_RELEASE_EXPECT_REPOSITORY="DZ23-LTDA/hades-ollama" \
  OLLAMA_RELEASE_EXPECT_COMMIT="abc123" \
  OLLAMA_RELEASE_EXPECT_REF="refs/tags/v1.2.3" \
  bash "$script"

# 2. Wrong public-key fingerprint is rejected (swapped-key attack).
expect_fail env OLLAMA_RELEASE_PUBKEY_SHA256="0000000000000000000000000000000000000000000000000000000000000000" \
  bash "$script"

# 3. Wrong repository is rejected (identity anchor).
expect_fail env OLLAMA_RELEASE_PUBKEY_SHA256="$fingerprint" \
  OLLAMA_RELEASE_EXPECT_REPOSITORY="attacker/evil" \
  bash "$script"

# 4. Manifest that does not cover the metadata is rejected.
sha256sum HadesSetup.exe > sha256sum.txt
openssl dgst -sha256 -sign private.pem -out sha256sum.txt.sig sha256sum.txt
expect_fail env OLLAMA_RELEASE_PUBKEY_SHA256="$fingerprint" \
  OLLAMA_RELEASE_EXPECT_REPOSITORY="DZ23-LTDA/hades-ollama" \
  bash "$script"

# 5. Tampered manifest (bad checksum) is rejected even with correct pins.
printf '%s  HadesSetup.exe\n' "0000000000000000000000000000000000000000000000000000000000000000" > sha256sum.txt
openssl dgst -sha256 -sign private.pem -out sha256sum.txt.sig sha256sum.txt
expect_fail env OLLAMA_RELEASE_PUBKEY_SHA256="$fingerprint" bash "$script"

echo "verify-release-artifact: ${pass} passed, ${fail} failed"
[[ "$fail" -eq 0 ]]
