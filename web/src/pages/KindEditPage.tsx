import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import {
  ApiError,
  api,
  type FieldType,
  type Kind,
  type Schema,
  type SchemaField,
} from "../api";
import {
  buttonStyle,
  errorStyle,
  inputStyle,
  linkStyle,
  mutedStyle,
  sectionStyle,
} from "../styles";

const EMPTY_SCHEMA: Schema = { fields: [] };

const FIELD_TYPES: FieldType[] = [
  "text",
  "number",
  "integer",
  "boolean",
  "date",
  "enum",
  "url",
  "tags",
];

export function KindEditPage() {
  const { kindId = "" } = useParams<{ kindId: string }>();
  const navigate = useNavigate();

  const [kind, setKind] = useState<Kind | null>(null);
  const [fields, setFields] = useState<SchemaField[]>([]);
  const [version, setVersion] = useState<number | undefined>();
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!kindId) return;
    setError(null);
    Promise.all([
      api.kind(kindId),
      api.schema(kindId).catch(() => EMPTY_SCHEMA),
    ])
      .then(([k, s]) => {
        setKind(k);
        setFields(s.fields ? [...s.fields] : []);
        setVersion(s.version);
      })
      .catch((e) => setError(String(e)));
  }, [kindId]);

  function move(i: number, dir: -1 | 1) {
    setFields((fs) => {
      const j = i + dir;
      if (j < 0 || j >= fs.length) return fs;
      const next = [...fs];
      [next[i], next[j]] = [next[j], next[i]];
      return next;
    });
  }

  function remove(i: number) {
    setFields((fs) => fs.filter((_, idx) => idx !== i));
  }

  function add() {
    setFields((fs) => [
      ...fs,
      { key: `field${fs.length + 1}`, type: "text" },
    ]);
  }

  function update(i: number, patch: Partial<SchemaField>) {
    setFields((fs) => fs.map((f, idx) => (idx === i ? { ...f, ...patch } : f)));
  }

  async function onSave() {
    if (!kindId) return;
    setSaving(true);
    setError(null);

    // Basic client-side sanity: unique keys, non-empty type.
    const keys = new Set<string>();
    for (const f of fields) {
      if (!f.key.trim()) {
        setError("every field needs a key");
        setSaving(false);
        return;
      }
      if (keys.has(f.key)) {
        setError(`duplicate field key: ${f.key}`);
        setSaving(false);
        return;
      }
      keys.add(f.key);
    }

    const schema: Schema = {
      version: (version ?? 0) + 1,
      fields,
    };

    try {
      await api.createKindVersion(kindId, schema);
      navigate(`/kinds/${kindId}`);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e));
      setSaving(false);
    }
  }

  return (
    <>
      <nav style={{ marginTop: "1rem" }}>
        <Link to={`/kinds/${kindId}`} style={linkStyle}>
          ← {kind ? kind.name : "back to kind"}
        </Link>
      </nav>

      {error && <pre style={errorStyle}>{error}</pre>}

      {!kind && !error && <p style={mutedStyle}>loading…</p>}

      {kind && (
        <>
          <section style={sectionStyle}>
            <div style={mutedStyle}>Editing schema</div>
            <div style={{ fontSize: "1.3rem", fontWeight: 600 }}>
              {kind.name}
            </div>
            <p style={{ ...mutedStyle, marginTop: "0.4rem" }}>
              Saving posts a new kind version. Existing models keep pointing at
              their old version, so this only affects display and new writes.
            </p>
          </section>

          <section style={sectionStyle}>
            <div style={controlsStyle}>
              <h2 style={{ margin: 0, fontSize: "1rem" }}>
                Fields
                <span style={{ ...mutedStyle, marginLeft: "0.5rem" }}>
                  {fields.length}
                </span>
              </h2>
              <button style={buttonStyle} onClick={add}>
                + Add field
              </button>
            </div>

            {fields.length === 0 && (
              <p style={mutedStyle}>No fields yet. Click Add field to begin.</p>
            )}

            {fields.map((f, i) => (
              <FieldEditor
                key={i}
                index={i}
                total={fields.length}
                field={f}
                onChange={(patch) => update(i, patch)}
                onMoveUp={() => move(i, -1)}
                onMoveDown={() => move(i, 1)}
                onRemove={() => remove(i)}
              />
            ))}
          </section>

          <section style={sectionStyle}>
            <div style={{ display: "flex", gap: "0.5rem" }}>
              <button
                style={primaryButtonStyle}
                onClick={onSave}
                disabled={saving}
              >
                {saving ? "Saving…" : "Save as new version"}
              </button>
              <Link
                to={`/kinds/${kindId}`}
                style={{ ...buttonStyle, textDecoration: "none" }}
              >
                Cancel
              </Link>
            </div>
          </section>
        </>
      )}
    </>
  );
}

// -----------------------------------------------------------------------------
// FieldEditor
// -----------------------------------------------------------------------------

function FieldEditor({
  index,
  total,
  field,
  onChange,
  onMoveUp,
  onMoveDown,
  onRemove,
}: {
  index: number;
  total: number;
  field: SchemaField;
  onChange: (patch: Partial<SchemaField>) => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onRemove: () => void;
}) {
  return (
    <div style={cardStyle}>
      <div style={cardHeaderStyle}>
        <span style={{ ...mutedStyle, minWidth: "1.5rem" }}>#{index + 1}</span>
        <input
          type="text"
          value={field.key}
          onChange={(e) => onChange({ key: e.target.value })}
          placeholder="key"
          style={{ ...inputStyle, fontFamily: "monospace", minWidth: "8rem", flex: 1 }}
        />
        <input
          type="text"
          value={field.label ?? ""}
          onChange={(e) => onChange({ label: e.target.value || undefined })}
          placeholder="Label (falls back to key)"
          style={{ ...inputStyle, flex: 2 }}
        />
        <select
          value={field.type}
          onChange={(e) => onChange({ type: e.target.value as FieldType })}
          style={{ ...inputStyle, minWidth: "6.5rem" }}
        >
          {FIELD_TYPES.map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </select>
        <div style={{ display: "flex", gap: "0.2rem" }}>
          <button
            style={iconBtnStyle}
            onClick={onMoveUp}
            disabled={index === 0}
            aria-label="move up"
            type="button"
          >
            ↑
          </button>
          <button
            style={iconBtnStyle}
            onClick={onMoveDown}
            disabled={index === total - 1}
            aria-label="move down"
            type="button"
          >
            ↓
          </button>
          <button
            style={dangerIconBtnStyle}
            onClick={onRemove}
            aria-label="remove"
            type="button"
          >
            ×
          </button>
        </div>
      </div>

      <div style={cardBodyStyle}>
        <label style={checkLabelStyle}>
          <input
            type="checkbox"
            checked={Boolean(field.pinned)}
            onChange={(e) => onChange({ pinned: e.target.checked || undefined })}
          />
          Pinned (visible by default)
        </label>
        <label style={checkLabelStyle}>
          <input
            type="checkbox"
            checked={Boolean(field.required)}
            onChange={(e) =>
              onChange({ required: e.target.checked || undefined })
            }
          />
          Required (writes without this are rejected)
        </label>

        {field.type === "enum" && (
          <label style={{ display: "block", marginTop: "0.4rem" }}>
            <div style={{ ...mutedStyle, marginBottom: "0.2rem" }}>
              Allowed values (one per line)
            </div>
            <textarea
              value={(field.values ?? []).join("\n")}
              onChange={(e) =>
                onChange({
                  values: e.target.value
                    .split("\n")
                    .map((s) => s.trim())
                    .filter(Boolean),
                })
              }
              style={{ ...inputStyle, minHeight: "6rem", width: "100%" }}
              rows={4}
            />
          </label>
        )}

        {(field.type === "number" || field.type === "integer") && (
          <div style={{ display: "flex", gap: "0.5rem", marginTop: "0.4rem" }}>
            <label style={{ display: "block", flex: 1 }}>
              <div style={{ ...mutedStyle, marginBottom: "0.2rem" }}>Min</div>
              <input
                type="number"
                value={field.min ?? ""}
                onChange={(e) =>
                  onChange({
                    min: e.target.value === "" ? undefined : Number(e.target.value),
                  })
                }
                style={inputStyle}
              />
            </label>
            <label style={{ display: "block", flex: 1 }}>
              <div style={{ ...mutedStyle, marginBottom: "0.2rem" }}>Max</div>
              <input
                type="number"
                value={field.max ?? ""}
                onChange={(e) =>
                  onChange({
                    max: e.target.value === "" ? undefined : Number(e.target.value),
                  })
                }
                style={inputStyle}
              />
            </label>
            <label style={{ display: "block", flex: 1 }}>
              <div style={{ ...mutedStyle, marginBottom: "0.2rem" }}>Unit</div>
              <input
                type="text"
                value={field.unit ?? ""}
                onChange={(e) =>
                  onChange({ unit: e.target.value || undefined })
                }
                style={inputStyle}
                placeholder="e.g. USD"
              />
            </label>
          </div>
        )}
      </div>
    </div>
  );
}

// -----------------------------------------------------------------------------
// Styles
// -----------------------------------------------------------------------------

const controlsStyle = {
  display: "flex",
  gap: "0.5rem",
  alignItems: "center",
  justifyContent: "space-between",
  marginBottom: "0.75rem",
};

const cardStyle = {
  border: "1px solid var(--border)",
  borderRadius: 8,
  padding: "0.6rem 0.75rem",
  marginBottom: "0.6rem",
  background: "var(--bg)",
};

const cardHeaderStyle = {
  display: "flex",
  gap: "0.5rem",
  alignItems: "center",
  flexWrap: "wrap" as const,
};

const cardBodyStyle = {
  marginTop: "0.5rem",
  paddingTop: "0.4rem",
  borderTop: "1px dashed var(--border-item)",
};

const checkLabelStyle = {
  display: "flex",
  gap: "0.4rem",
  alignItems: "center",
  fontSize: "0.85rem",
  marginBottom: "0.3rem",
};

const iconBtnStyle = {
  ...buttonStyle,
  padding: "0.25rem 0.5rem",
  fontSize: "0.85rem",
  minWidth: "1.8rem",
};

const dangerIconBtnStyle = {
  ...iconBtnStyle,
  background: "var(--error-bg)",
  color: "var(--error-fg)",
};

const primaryButtonStyle = {
  ...buttonStyle,
  background: "var(--btn-primary-bg)",
  color: "var(--btn-primary-fg)",
  padding: "0.5rem 1rem",
};
