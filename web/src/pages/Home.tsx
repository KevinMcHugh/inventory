import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { api, type Kind, type Tenant } from "../api";
import {
  buttonStyle,
  codeStyle,
  errorStyle,
  linkStyle,
  listItemStyle,
  listStyle,
  mutedStyle,
  sectionStyle,
} from "../styles";

export function Home() {
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [kinds, setKinds] = useState<Kind[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([api.tenant(), api.kinds()])
      .then(([t, ks]) => {
        setTenant(t);
        setKinds(ks);
      })
      .catch((e) => setError(String(e)));
  }, []);

  return (
    <>
      {error && <pre style={errorStyle}>{error}</pre>}

      {tenant && (
        <section style={sectionStyle}>
          <div style={mutedStyle}>Tenant</div>
          <div style={{ fontSize: "1.1rem" }}>
            <strong>{tenant.name}</strong>
          </div>
        </section>
      )}

      <McpEndpoint />

      <section style={sectionStyle}>
        <h2 style={{ margin: "0 0 0.5rem 0", fontSize: "1rem" }}>Kinds</h2>
        {kinds === null && !error && <p style={mutedStyle}>loading…</p>}
        {kinds && kinds.length === 0 && (
          <p style={mutedStyle}>
            No kinds yet. Ask Claude to create one via the{" "}
            <code style={codeStyle}>create_kind</code> MCP tool.
          </p>
        )}
        {kinds && kinds.length > 0 && (
          <ul style={listStyle}>
            {kinds.map((k) => (
              <li key={k.id} style={listItemStyle}>
                <Link
                  to={`/kinds/${k.id}`}
                  style={{ ...linkStyle, fontWeight: 600 }}
                >
                  {k.name}
                </Link>
                {k.description && (
                  <span style={{ ...mutedStyle, marginLeft: "auto" }}>
                    {k.description}
                  </span>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </>
  );
}

function McpEndpoint() {
  const url = window.location.origin + "/mcp/rpc";
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard blocked — no-op
    }
  }

  return (
    <section style={sectionStyle}>
      <h2 style={{ margin: "0 0 0.5rem 0", fontSize: "1rem" }}>
        Connect an MCP client
      </h2>
      <p style={mutedStyle}>
        Point Claude (or any MCP client) here. Pick <em>Always required</em>{" "}
        and <em>No client ID — register one automatically</em>. You will be
        sent back to this login screen — paste your{" "}
        <code style={codeStyle}>inv_</code> api key to grant access.
      </p>
      <div
        style={{
          display: "flex",
          gap: "0.5rem",
          alignItems: "stretch",
          marginTop: "0.75rem",
        }}
      >
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
