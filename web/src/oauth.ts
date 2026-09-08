// Browser-side OAuth 2.1 client for the inventory web UI.
//
// The web app is a public OAuth client on the same origin as the AS. On first
// use it dynamically registers itself (POST /oauth/register), caches the
// client_id in localStorage, and drives a PKCE-protected authorization-code
// flow. The resulting access_token also lives in localStorage and is sent as
// Authorization: Bearer on every API call.

const STORAGE = {
  clientId: "inv.oauth.client_id",
  token: "inv.oauth.token",
  verifier: "inv.oauth.pkce_verifier",
  state: "inv.oauth.state",
};

const REDIRECT_URI = window.location.origin + "/callback";

// -----------------------------------------------------------------------------
// PKCE utilities
// -----------------------------------------------------------------------------

function base64urlFromBytes(bytes: Uint8Array): string {
  let bin = "";
  bytes.forEach((b) => (bin += String.fromCharCode(b)));
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function randomBase64url(nBytes: number): string {
  const buf = new Uint8Array(nBytes);
  crypto.getRandomValues(buf);
  return base64urlFromBytes(buf);
}

async function pkceChallenge(verifier: string): Promise<string> {
  const digest = await crypto.subtle.digest(
    "SHA-256",
    new TextEncoder().encode(verifier),
  );
  return base64urlFromBytes(new Uint8Array(digest));
}

// -----------------------------------------------------------------------------
// Client registration (DCR)
// -----------------------------------------------------------------------------

async function ensureClientId(): Promise<string> {
  const cached = localStorage.getItem(STORAGE.clientId);
  if (cached) return cached;
  const resp = await fetch("/oauth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      client_name: "Inventory Web",
      redirect_uris: [REDIRECT_URI],
    }),
  });
  if (!resp.ok) throw new Error(`register failed: ${resp.status}`);
  const body = await resp.json();
  localStorage.setItem(STORAGE.clientId, body.client_id);
  return body.client_id;
}

// -----------------------------------------------------------------------------
// Public API
// -----------------------------------------------------------------------------

export function getToken(): string | null {
  return localStorage.getItem(STORAGE.token);
}

export function logout(): void {
  localStorage.removeItem(STORAGE.token);
}

export async function startLogin(): Promise<void> {
  const clientId = await ensureClientId();
  const verifier = randomBase64url(32);
  const challenge = await pkceChallenge(verifier);
  const state = randomBase64url(16);
  sessionStorage.setItem(STORAGE.verifier, verifier);
  sessionStorage.setItem(STORAGE.state, state);

  const params = new URLSearchParams({
    response_type: "code",
    client_id: clientId,
    redirect_uri: REDIRECT_URI,
    state,
    code_challenge: challenge,
    code_challenge_method: "S256",
  });
  window.location.href = `/oauth/authorize?${params.toString()}`;
}

// completeLoginFromURL runs on the callback page. If a ?code=... is present it
// exchanges the code for an access token and returns true; the caller should
// then clear the URL query.
export async function completeLoginFromURL(): Promise<boolean> {
  const url = new URL(window.location.href);
  const code = url.searchParams.get("code");
  const state = url.searchParams.get("state");
  if (!code) return false;

  const expected = sessionStorage.getItem(STORAGE.state);
  if (expected && state !== expected) {
    throw new Error("oauth state mismatch");
  }
  const verifier = sessionStorage.getItem(STORAGE.verifier);
  const clientId = localStorage.getItem(STORAGE.clientId);
  if (!verifier || !clientId) {
    throw new Error("missing pkce verifier or client_id");
  }

  const resp = await fetch("/oauth/token", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "authorization_code",
      code,
      code_verifier: verifier,
      client_id: clientId,
      redirect_uri: REDIRECT_URI,
    }),
  });
  if (!resp.ok) {
    const body = await resp.text();
    throw new Error(`token exchange failed: ${resp.status} ${body}`);
  }
  const body = await resp.json();
  localStorage.setItem(STORAGE.token, body.access_token);
  sessionStorage.removeItem(STORAGE.verifier);
  sessionStorage.removeItem(STORAGE.state);
  return true;
}
