#!/usr/bin/env bash
set -euo pipefail

# Verifies the release manifest and its detached RSA signature, and — when the
# caller anchors trust — that the public key and the release identity are the
# expected ones.
#
# Why the anchors matter: the public key is distributed WITH the release, so a
# detached signature alone is self-referential — an attacker who replaces the
# manifest, the signature AND the public key together still verifies. Build
# provenance/attestation closes this, but it is optional. These anchors give a
# cert-free way to pin trust:
#   OLLAMA_RELEASE_PUBKEY_SHA256   expected SHA-256 of the public key PEM,
#                                  published out-of-band (repo/docs), not taken
#                                  from the release assets.
#   OLLAMA_RELEASE_EXPECT_REPOSITORY / _COMMIT / _REF
#                                  expected identity in release-metadata.json,
#                                  which is itself covered by the signed manifest.
#   OLLAMA_RELEASE_METADATA        path to the metadata file (default below).
manifest=${1:-sha256sum.txt}
signature=${2:-sha256sum.txt.sig}
public_key=${3:-release-signing-public.pem}
expected_fingerprint=${OLLAMA_RELEASE_PUBKEY_SHA256:-${4:-}}
metadata=${OLLAMA_RELEASE_METADATA:-release-metadata.json}

[[ -s "$manifest" && -s "$signature" && -s "$public_key" ]] || { echo 'release signature inputs are missing' >&2; exit 1; }

# Anchor 1: pin the public key to a known fingerprint so a swapped key cannot
# pass. Without it the signature check is not anchored to any identity.
if [[ -n "$expected_fingerprint" ]]; then
  actual_fingerprint=$(sha256sum "$public_key" | awk '{print $1}')
  normalized_expected=$(printf '%s' "$expected_fingerprint" | tr '[:upper:]' '[:lower:]' | tr -d ' :')
  if [[ "$actual_fingerprint" != "$normalized_expected" ]]; then
    echo "release public key fingerprint mismatch: expected ${normalized_expected}, got ${actual_fingerprint}" >&2
    exit 1
  fi
else
  echo 'warning: release public key is not pinned (set OLLAMA_RELEASE_PUBKEY_SHA256 to anchor trust to a known key)' >&2
fi

sha256sum -c "$manifest"
openssl dgst -sha256 -verify "$public_key" -signature "$signature" "$manifest"

# Anchor 2: pin the release identity. Only runs when at least one expected value
# is provided; then the metadata must exist, be covered by the signed manifest,
# and match the expected repository/commit/ref.
if [[ -n "${OLLAMA_RELEASE_EXPECT_REPOSITORY:-}${OLLAMA_RELEASE_EXPECT_COMMIT:-}${OLLAMA_RELEASE_EXPECT_REF:-}" ]]; then
  [[ -s "$metadata" ]] || { echo "release metadata ${metadata} is missing but identity anchoring was requested" >&2; exit 1; }
  metadata_base=$(basename "$metadata")
  metadata_pattern=$(printf '%s' "$metadata_base" | sed -E 's/[.[\*^$]/\\&/g')
  if ! grep -qE "(^|[[:space:]*/])${metadata_pattern}\$" "$manifest"; then
    echo "release manifest does not cover ${metadata_base}; identity cannot be trusted" >&2
    exit 1
  fi
  check_metadata_field() {
    local field=$1 expected=$2
    [[ -n "$expected" ]] || return 0
    local actual
    actual=$(grep -oE "\"${field}\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" "$metadata" | head -n1 | sed -E 's/.*:[[:space:]]*"([^"]*)"/\1/')
    if [[ "$actual" != "$expected" ]]; then
      echo "release metadata ${field} mismatch: expected '${expected}', got '${actual:-<missing>}'" >&2
      exit 1
    fi
  }
  check_metadata_field repository "${OLLAMA_RELEASE_EXPECT_REPOSITORY:-}"
  check_metadata_field commit "${OLLAMA_RELEASE_EXPECT_COMMIT:-}"
  check_metadata_field ref "${OLLAMA_RELEASE_EXPECT_REF:-}"
fi
