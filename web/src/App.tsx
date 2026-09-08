import { useEffect, useState } from "react";

import { api, type Kind, type Model, type Tenant } from "./api";
import { completeLoginFromURL, getToken, logout, startLogin } from "./oauth";

type AuthState = "checking" | "signed_out" | "signed_in";

export default function App() {
  const [auth, setAuth] = useState<AuthState>("checking");
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [kinds, setKinds] = useState<Kind[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        if (await completeLoginFromURL()) {
          window.history.replaceState({}, "", "/");
        }
      } catch (e) {
        setError(String(e));
      }
      setAuth(getToken() ? "signed_in" : "signed_out");
    })();
  }, []);

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
          <button
            style={buttonStyle}
            onClick={() => {
              logout();
              location.reload();
            }}
          >
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
                No kinds yet. Ask Claude to create one via the{" "}
                <code style={codeStyle}>create_kind</code> MCP tool.
              </p>
            )}
            {kinds && kinds.length > 0 && (
              <ul style={listStyle}>
                {kinds.map((k) => (
                  <KindRow key={k.id} kind={k} />
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
// KindRow: click to expand/collapse; models fetched lazily on first expand.
// -----------------------------------------------------------------------------

function KindRow({ kind }: { kind: Kind }) {
  const [expanded, setExpanded] = useState(false);
  const [models, setModels] = useState<Model[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function toggle() {
    const next = !expanded;
    setExpanded(next);
    if (next && models === null && !loading) {
      setLoading(true);
      try {
        const ms = await api.models(kind.id);
        setModels(ms);
      } catch (e) {
        setErr(String(e));
      } finally {
        setLoading(false);
      }
    }
  }

  return (
    <li style={listItemStyle}>
      <div
        onClick={toggle}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            toggle();
          }
        }}
        style={kindHeaderStyle}
      >
        <span style={chevronStyle}>{expanded ? "▾" : "▸"}</span>
        <strong>{kind.name}</strong>
        <code style={codeStyle}>{kind.id}</code>
        {kind.description && (
          <span style={{ ...mutedStyle, marginLeft: "auto" }}>
            {kind.description}
          </span>
        )}
      </div>
      {expanded && (
        <div style={modelsWrapStyle}>
          {loading && <p style={mutedStyle}>loading models…</p>}
          {err && <pre style={errorStyle}>{err}</pre>}
          {models && models.length === 0 && (
            <p style={mutedStyle}>No models in this kind.</p>
          )}
          {models && models.length > 0 && (
            <ul style={listStyle}>
              {models.map((m) => (
                <li key={m.id} style={modelItemStyle}>
                  <div style={modelHeaderStyle}>
                    <strong>{m.slug}</strong>
                    <code style={codeStyle}>{m.id}</code>
                  </div>
                  <pre style={bodyStyle}>
                    {JSON.stringify(m.body, null, 2)}
                  </pre>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </li>
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
      // clipboard blocked (insecure context, denied permission) — no-op
    }
  }

  return (
    <section style={sectionStyle}>
      <h2 style={{ margin: "0 0 0.5rem 0", fontSize: "1rem" }}>
        Connect an MCP client
      </h2>
      <p style={mutedStyle}>
        Point Claude (or any MCP client) here. Pick <em>Always required</em>{" "}
        and <em>No client ID — register one automatically</em>. You will be sent
        back to this login screen — paste your{" "}
        <code style={codeStyle}>inv_</code> api key to grant access.
      </p>
      <div style={endpointRowStyle}>
        <code
          style={{
            ...codeStyle,
            flex: 1,
            padding: "0.5rem 0.6rem",
            fontSize: "0.9rem",
            overflowX: "auto",
          }}
        >
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
// Styles — layout inline; colors reference CSS vars in index.css so both
// light and dark palettes apply without a toggle.
// -----------------------------------------------------------------------------

const pageStyle: React.CSSProperties = {
  maxWidth: 640,
  margin: "3rem auto",
  padding: "0 1.5rem",
  color: "var(--fg)",
};

const headerStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
};

const sectionStyle: React.CSSProperties = {
  marginTop: "2rem",
  paddingTop: "1rem",
  borderTop: "1px solid var(--border)",
};

const mutedStyle: React.CSSProperties = {
  color: "var(--muted)",
  fontSize: "0.9rem",
};

const codeStyle: React.CSSProperties = {
  background: "var(--code-bg)",
  color: "var(--fg)",
  padding: "0 0.3rem",
  borderRadius: 3,
  fontSize: "0.85rem",
};

const errorStyle: React.CSSProperties = {
  color: "var(--error-fg)",
  background: "var(--error-bg)",
  padding: "0.6rem 0.8rem",
  borderRadius: 6,
  whiteSpace: "pre-wrap",
};

const buttonStyle: React.CSSProperties = {
  background: "var(--btn-bg)",
  color: "var(--btn-fg)",
  padding: "0.4rem 0.8rem",
  border: 0,
  borderRadius: 6,
  cursor: "pointer",
};

const primaryButtonStyle: React.CSSProperties = {
  ...buttonStyle,
  background: "var(--btn-primary-bg)",
  color: "var(--btn-primary-fg)",
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
  borderBottom: "1px solid var(--border-item)",
};

const kindHeaderStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: "0.5rem",
  cursor: "pointer",
  userSelect: "none",
};

const chevronStyle: React.CSSProperties = {
  color: "var(--muted)",
  width: "1rem",
  display: "inline-block",
  textAlign: "center",
};

const modelsWrapStyle: React.CSSProperties = {
  marginTop: "0.6rem",
  paddingLeft: "1.4rem",
};

const modelItemStyle: React.CSSProperties = {
  padding: "0.4rem 0",
  borderBottom: "1px solid var(--border-item)",
};

const modelHeaderStyle: React.CSSProperties = {
  display: "flex",
  gap: "0.5rem",
  alignItems: "center",
};

const bodyStyle: React.CSSProperties = {
  background: "var(--code-bg)",
  color: "var(--fg)",
  padding: "0.5rem 0.6rem",
  borderRadius: 4,
  fontSize: "0.8rem",
  maxHeight: 240,
  overflow: "auto",
  margin: "0.4rem 0 0 0",
};

const endpointRowStyle: React.CSSProperties = {
  display: "flex",
  gap: "0.5rem",
  alignItems: "stretch",
  marginTop: "0.75rem",
};
