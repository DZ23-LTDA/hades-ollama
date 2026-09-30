#!/usr/bin/env bash
set -euo pipefail

# Verifies the release manifest and its detached RSA signature. The public key
# must be distributed with the release; GitHub's build-provenance attestation
# authenticates who produced that key and manifest.
manifest=${1:-sha256sum.txt}
signature=${2:-sha256sum.txt.sig}
public_key=${3:-release-signing-public.pem}
[[ -s "$manifest" && -s "$signature" && -s "$public_key" ]] || { echo 'release signature inputs are missing' >&2; exit 1; }
sha256sum -c "$manifest"
openssl dgst -sha256 -verify "$public_key" -signature "$signature" "$manifest"
