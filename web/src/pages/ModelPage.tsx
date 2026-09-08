import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { ApiError, api, type Kind, type Model, type Schema } from "../api";
import {
  buttonStyle,
  codeStyle,
  errorStyle,
  linkStyle,
  mutedStyle,
  sectionStyle,
} from "../styles";
import {
  effectiveFields,
  fieldLabel,
  fmtDateTime,
  renderCell,
} from "../schema";

export function ModelPage() {
  const { kindId = "", slug = "" } = useParams<{ kindId: string; slug: string }>();
  const navigate = useNavigate();
  const [kind, setKind] = useState<Kind | null>(null);
  const [model, setModel] = useState<Model | null>(null);
  const [schema, setSchema] = useState<Schema | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    if (!kindId || !slug) return;
    setKind(null);
    setModel(null);
    setSchema(null);
    setError(null);
    Promise.all([
      api.kind(kindId),
      api.model(kindId, slug),
      api.schema(kindId).catch(() => ({ fields: [] }) as Schema),
    ])
      .then(([k, m, s]) => {
        setKind(k);
        setModel(m);
        setSchema(s);
      })
      .catch((e) => setError(String(e)));
  }, [kindId, slug]);

  async function onDelete() {
    if (!confirm(`Delete model "${slug}"? This is a soft delete.`)) return;
    setDeleting(true);
    try {
      await api.deleteModel(kindId, slug);
      navigate(`/kinds/${kindId}`);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e));
      setDeleting(false);
    }
  }

  const fields = model && schema ? effectiveFields([model], schema) : [];
  const body = model?.body ?? {};

  return (
    <>
      <nav style={{ marginTop: "1rem" }}>
        <Link to={`/kinds/${kindId}`} style={linkStyle}>
          ← {kind ? kind.name : "back to kind"}
        </Link>
      </nav>

      {error && <pre style={errorStyle}>{error}</pre>}

      {model === null && !error && <p style={mutedStyle}>loading…</p>}

      {model && (
        <>
          <section style={sectionStyle}>
            <div style={headerRowStyle}>
              <div>
                <div style={mutedStyle}>Model</div>
                <div style={{ fontSize: "1.3rem", fontWeight: 600 }}>
                  {model.slug}
                </div>
                <div style={{ ...mutedStyle, marginTop: "0.2rem" }}>
                  <code style={codeStyle}>{model.id}</code> · pinned to{" "}
                  <code style={codeStyle}>{model.kindVersionId}</code>
                </div>
              </div>
              <div style={{ display: "flex", gap: "0.5rem" }}>
                <Link
                  to={`/kinds/${kindId}/models/${encodeURIComponent(slug)}/edit`}
                  style={{ ...buttonStyle, textDecoration: "none" }}
                >
                  Edit
                </Link>
                <button
                  style={dangerButtonStyle}
                  onClick={onDelete}
                  disabled={deleting}
                >
                  {deleting ? "Deleting…" : "Delete"}
                </button>
              </div>
            </div>
          </section>

          <section style={sectionStyle}>
            <h2 style={{ margin: "0 0 0.75rem 0", fontSize: "1rem" }}>Fields</h2>
            {fields.length === 0 && (
              <p style={mutedStyle}>No fields authored for this kind yet.</p>
            )}
            {fields.length > 0 && (
              <dl style={dlStyle}>
                {fields.map((f) => (
                  <div key={f.key} style={dlRowStyle}>
                    <dt style={dtStyle}>{fieldLabel(f)}</dt>
                    <dd style={ddStyle}>{renderCell(f.type, body[f.key])}</dd>
                  </div>
                ))}
              </dl>
            )}
          </section>

          <section style={sectionStyle}>
            <div style={mutedStyle}>Timestamps</div>
            <div style={{ fontSize: "0.9rem" }}>
              Created {fmtDateTime(model.createdAt)}
              <br />
              Updated {fmtDateTime(model.updatedAt)}
            </div>
          </section>
        </>
      )}
    </>
  );
}

const headerRowStyle = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "flex-start",
  gap: "1rem",
  flexWrap: "wrap" as const,
};

const dangerButtonStyle = {
  ...buttonStyle,
  background: "var(--error-bg)",
  color: "var(--error-fg)",
};

const dlStyle = {
  margin: 0,
  display: "grid",
  gridTemplateColumns: "minmax(140px, max-content) 1fr",
  columnGap: "1rem",
  rowGap: "0.4rem",
};

const dlRowStyle = {
  display: "contents",
};

const dtStyle = {
  color: "var(--muted)",
  fontSize: "0.85rem",
  fontWeight: 600,
};

const ddStyle = {
  margin: 0,
  fontSize: "0.95rem",
  wordBreak: "break-word" as const,
  minWidth: 0,
};
