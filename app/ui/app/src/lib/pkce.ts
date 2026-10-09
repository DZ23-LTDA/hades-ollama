// PKCE helpers (RFC 7636) built on the Web Crypto API. The code_verifier is a
// high-entropy random base64url string between 43 and 128 characters; the
// code_challenge is the base64url-encoded SHA-256 of the verifier.

function base64UrlEncode(bytes: Uint8Array): string {
  let binary = "";
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

// generateCodeVerifier returns a random code_verifier whose length is clamped
// to the PKCE-mandated 43–128 character range (base64url alphabet only).
export function generateCodeVerifier(length = 64): string {
  const size = Math.min(128, Math.max(43, Math.floor(length)));
  const bytes = new Uint8Array(size);
  crypto.getRandomValues(bytes);
  // base64url of `size` bytes is longer than `size` chars, so trimming to
  // `size` keeps the result within bounds and well above the 43-char minimum.
  return base64UrlEncode(bytes).slice(0, size);
}

// generateCodeChallenge returns the S256 challenge for a verifier.
export async function generateCodeChallenge(verifier: string): Promise<string> {
  const data = new TextEncoder().encode(verifier);
  const digest = await crypto.subtle.digest("SHA-256", data);
  return base64UrlEncode(new Uint8Array(digest));
}
