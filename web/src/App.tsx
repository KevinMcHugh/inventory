import { useEffect, useState } from "react";

import { api, type Kind, type Tenant } from "./api";
import { completeLoginFromURL, getToken, logout, startLogin } from "./oauth";

type AuthState = "checking" | "signed_out" | "signed_in";

export default function App() {
  const [auth, setAuth] = useState<AuthState>("checking");
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [kinds, setKinds] = useState<Kind[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  // On mount: finish any pending OAuth callback, then decide auth state.
  useEffect(() => {
    (async () => {
      try {
        if (await completeLoginFromURL()) {
          // Strip the ?code=&state=... query from the URL bar.
          window.history.replaceState({}, "", "/");
        }
      } catch (e) {
        setError(String(e));
      }
      setAuth(getToken() ? "signed_in" : "signed_out");
    })();
  }, []);

  // Fetch tenant + kinds once signed in.
  useEffect(() => {
    if (auth !== "signed_in") return;
    Promise.all([api.tenant(), api.kinds()])
      .then(([t, ks]) => {
        setTenant(t);
        setKinds(ks);
      })
      .catch((e) => setError(String(e)));
  }, [auth]);

  return (
    <main style={pageStyle}>
      <header style={headerStyle}>
        <h1 style={{ margin: 0, fontSize: "1.4rem" }}>Inventory</h1>
        {auth === "signed_in" && (
          <button style={buttonStyle} onClick={() => { logout(); location.reload(); }}>
            Sign out
          </button>
        )}
      </header>

      {error && <pre style={errorStyle}>{error}</pre>}

      {auth === "checking" && <p style={mutedStyle}>loading…</p>}

      {auth === "signed_out" && (
        <section style={{ marginTop: "2rem" }}>
          <p style={mutedStyle}>
            Sign in with your Inventory api key to view your data.
          </p>
          <button style={primaryButtonStyle} onClick={() => startLogin()}>
            Sign in
          </button>
        </section>
      )}

      {auth === "signed_in" && (
        <>
          {tenant && (
            <section style={sectionStyle}>
              <div style={mutedStyle}>Tenant</div>
              <div style={{ fontSize: "1.1rem" }}>
                <strong>{tenant.name}</strong>{" "}
                <code style={codeStyle}>{tenant.id}</code>
              </div>
            </section>
          )}

          <McpEndpoint />

          <section style={sectionStyle}>
            <h2 style={{ margin: "0 0 0.5rem 0", fontSize: "1rem" }}>Kinds</h2>
            {kinds === null && <p style={mutedStyle}>loading…</p>}
            {kinds && kinds.length === 0 && (
              <p style={mutedStyle}>
                No kinds yet. Ask Claude to create one via the <code style={codeStyle}>create_kind</code> MCP tool.
              </p>
            )}
            {kinds && kinds.length > 0 && (
              <ul style={listStyle}>
                {kinds.map((k) => (
                  <li key={k.id} style={listItemStyle}>
                    <strong>{k.name}</strong>{" "}
                    <code style={codeStyle}>{k.id}</code>
                    {k.description && (
                      <div style={mutedStyle}>{k.description}</div>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </section>
        </>
      )}
    </main>
  );
}

// -----------------------------------------------------------------------------
// MCP endpoint card
// -----------------------------------------------------------------------------

function McpEndpoint() {
  const url = window.location.origin + "/mcp/rpc";
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore — clipboard blocked (http, insecure context, etc.)
    }
  }

  return (
    <section style={sectionStyle}>
      <h2 style={{ margin: "0 0 0.5rem 0", fontSize: "1rem" }}>Connect an MCP client</h2>
      <p style={mutedStyle}>
        Point Claude (or any MCP client) here. Pick <em>Always required</em> and
        {" "}<em>No client ID — register one automatically</em>. You will be sent
        back to this login screen — paste your <code style={codeStyle}>inv_</code> api key to grant access.
      </p>
      <div style={endpointRowStyle}>
        <code style={{ ...codeStyle, flex: 1, padding: "0.5rem 0.6rem", fontSize: "0.9rem", overflowX: "auto" }}>
          {url}
        </code>
        <button style={buttonStyle} onClick={copy}>
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
    </section>
  );
}

// -----------------------------------------------------------------------------
// Styles — inline so this stays a single file for now.
// -----------------------------------------------------------------------------

const pageStyle: React.CSSProperties = {
  fontFamily: "system-ui, -apple-system, sans-serif",
  maxWidth: 640,
  margin: "3rem auto",
  padding: "0 1.5rem",
  color: "#1a1a1a",
};

const headerStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
};

const sectionStyle: React.CSSProperties = {
  marginTop: "2rem",
  paddingTop: "1rem",
  borderTop: "1px solid #eee",
};

const mutedStyle: React.CSSProperties = { color: "#666", fontSize: "0.9rem" };

const codeStyle: React.CSSProperties = {
  background: "#f4f4f4",
  padding: "0 0.3rem",
  borderRadius: 3,
  fontSize: "0.85rem",
};

const errorStyle: React.CSSProperties = {
  color: "#b00020",
  background: "#ffe8ec",
  padding: "0.6rem 0.8rem",
  borderRadius: 6,
  whiteSpace: "pre-wrap",
};

const buttonStyle: React.CSSProperties = {
  background: "#eee",
  color: "#333",
  padding: "0.4rem 0.8rem",
  border: 0,
  borderRadius: 6,
  cursor: "pointer",
};

const primaryButtonStyle: React.CSSProperties = {
  ...buttonStyle,
  background: "#222",
  color: "white",
  marginTop: "1rem",
  padding: "0.6rem 1.2rem",
};

const listStyle: React.CSSProperties = {
  listStyle: "none",
  padding: 0,
  margin: 0,
};

const listItemStyle: React.CSSProperties = {
  padding: "0.6rem 0",
  borderBottom: "1px solid #f0f0f0",
};

const endpointRowStyle: React.CSSProperties = {
  display: "flex",
  gap: "0.5rem",
  alignItems: "stretch",
  marginTop: "0.75rem",
};
