import {
  useEffect,
  useMemo,
  useState,
  type ChangeEvent,
  type FormEvent,
} from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import {
  ApiError,
  api,
  type Kind,
  type Model,
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
import { effectiveFields, fieldLabel } from "../schema";

const EMPTY_SCHEMA: Schema = { fields: [] };

export function ModelEditPage() {
  const { kindId = "", slug: slugParam } = useParams<{
    kindId: string;
    slug: string;
  }>();
  const isCreate = !slugParam;
  const navigate = useNavigate();

  const [kind, setKind] = useState<Kind | null>(null);
  const [schema, setSchema] = useState<Schema | null>(null);
  const [existing, setExisting] = useState<Model | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [slug, setSlug] = useState(slugParam ?? "");
  const [values, setValues] = useState<Record<string, unknown>>({});
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!kindId) return;
    setKind(null);
    setSchema(null);
    setExisting(null);
    setError(null);
    setFieldErrors({});
    const jobs: Promise<any>[] = [
      api.kind(kindId),
      api.schema(kindId).catch(() => EMPTY_SCHEMA),
    ];
    if (!isCreate) jobs.push(api.model(kindId, slugParam!));
    Promise.all(jobs)
      .then((results) => {
        setKind(results[0] as Kind);
        setSchema(results[1] as Schema);
        if (!isCreate) {
          const m = results[2] as Model;
          setExisting(m);
          setValues({ ...m.body });
        }
      })
      .catch((e) => setError(String(e)));
  }, [kindId, isCreate, slugParam]);

  const fields = useMemo(
    () => (schema ? effectiveFields(existing ? [existing] : [], schema) : []),
    [schema, existing],
  );

  function setValue(key: string, v: unknown) {
    setValues((prev) => ({ ...prev, [key]: v }));
    setFieldErrors((prev) => {
      if (!(key in prev)) return prev;
      const { [key]: _, ...rest } = prev;
      return rest;
    });
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!kindId) return;
    setError(null);
    setFieldErrors({});
    setSaving(true);

    // Client-side required-field check (server also enforces).
    const localErrors: Record<string, string> = {};
    for (const f of fields) {
      if (f.required && isBlank(values[f.key])) {
        localErrors[f.key] = "required";
      }
    }
    if (Object.keys(localErrors).length > 0) {
      setFieldErrors(localErrors);
      setSaving(false);
      return;
    }

    // Coerce body values to their declared types before sending.
    const body = coerce(fields, values);

    try {
      if (isCreate) {
        const created = await api.createModel(kindId, { slug, body });
        navigate(`/kinds/${kindId}/models/${encodeURIComponent(created.slug)}`);
      } else {
        await api.updateModel(kindId, slug, { body });
        navigate(`/kinds/${kindId}/models/${encodeURIComponent(slug)}`);
      }
    } catch (e) {
      if (e instanceof ApiError && e.fields && e.fields.length > 0) {
        const map: Record<string, string> = {};
        for (const fe of e.fields) map[fe.field] = fe.message;
        setFieldErrors(map);
        setError(e.message);
      } else {
        setError(e instanceof ApiError ? e.message : String(e));
      }
      setSaving(false);
    }
  }

  return (
    <>
      <nav style={{ marginTop: "1rem" }}>
        <Link
          to={isCreate ? `/kinds/${kindId}` : `/kinds/${kindId}/models/${encodeURIComponent(slug)}`}
          style={linkStyle}
        >
          ← {isCreate ? (kind?.name ?? "back to kind") : slug}
        </Link>
      </nav>

      {error && <pre style={errorStyle}>{error}</pre>}

      {(!kind || !schema) && !error && <p style={mutedStyle}>loading…</p>}

      {kind && schema && (
        <form onSubmit={onSubmit}>
          <section style={sectionStyle}>
            <div style={mutedStyle}>{isCreate ? "New model" : "Editing"}</div>
            <div style={{ fontSize: "1.3rem", fontWeight: 600 }}>
              {kind.name}
            </div>
            {kind.description && (
              <p style={{ ...mutedStyle, marginTop: "0.4rem" }}>
                {kind.description}
              </p>
            )}
          </section>

          <section style={sectionStyle}>
            <FormRow
              label="Slug"
              required
              hint="URL-safe. Unique within this kind."
              error={fieldErrors.__slug}
            >
              <input
                type="text"
                required
                value={slug}
                disabled={!isCreate}
                onChange={(e) => setSlug(e.target.value)}
                style={inputStyle}
                pattern="[a-z0-9][a-z0-9\-]*"
                placeholder="unique-slug"
              />
            </FormRow>
          </section>

          <section style={sectionStyle}>
            <h2 style={{ margin: "0 0 0.75rem 0", fontSize: "1rem" }}>Fields</h2>
            {fields.length === 0 && (
              <p style={mutedStyle}>
                No fields authored. Author some in{" "}
                <Link to={`/kinds/${kindId}/edit`} style={linkStyle}>
                  the kind editor
                </Link>{" "}
                first.
              </p>
            )}
            {fields.map((f) => (
              <FormRow
                key={f.key}
                label={fieldLabel(f)}
                required={f.required}
                hint={f.unit ? f.unit : undefined}
                error={fieldErrors[f.key]}
              >
                <FieldInput
                  field={f}
                  value={values[f.key]}
                  onChange={(v) => setValue(f.key, v)}
                />
              </FormRow>
            ))}
          </section>

          <section style={sectionStyle}>
            <div style={{ display: "flex", gap: "0.5rem" }}>
              <button
                type="submit"
                style={primaryButtonStyle}
                disabled={saving}
              >
                {saving ? "Saving…" : isCreate ? "Create" : "Save"}
              </button>
              <Link
                to={
                  isCreate
                    ? `/kinds/${kindId}`
                    : `/kinds/${kindId}/models/${encodeURIComponent(slug)}`
                }
                style={{ ...buttonStyle, textDecoration: "none" }}
              >
                Cancel
              </Link>
            </div>
            {null}
          </section>
        </form>
      )}
    </>
  );
}

// -----------------------------------------------------------------------------
// FormRow: label + input + inline error/hint
// -----------------------------------------------------------------------------

function FormRow({
  label,
  required,
  hint,
  error,
  children,
}: {
  label: string;
  required?: boolean;
  hint?: string;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <div style={{ marginBottom: "0.9rem" }}>
      <label style={rowLabelStyle}>
        {label}
        {required && <span style={{ color: "var(--error-fg)" }}> *</span>}
      </label>
      {children}
      {hint && !error && (
        <div style={{ ...mutedStyle, fontSize: "0.75rem", marginTop: "0.2rem" }}>
          {hint}
        </div>
      )}
      {error && (
        <div
          style={{
            color: "var(--error-fg)",
            fontSize: "0.8rem",
            marginTop: "0.2rem",
          }}
        >
          {error}
        </div>
      )}
    </div>
  );
}

// -----------------------------------------------------------------------------
// FieldInput: renders the right widget per field type
// -----------------------------------------------------------------------------

function FieldInput({
  field,
  value,
  onChange,
}: {
  field: SchemaField;
  value: unknown;
  onChange: (v: unknown) => void;
}) {
  const s = value == null ? "" : String(value);

  function textChange(e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) {
    onChange(e.target.value === "" ? undefined : e.target.value);
  }

  switch (field.type) {
    case "enum":
      return (
        <select
          value={s}
          onChange={(e) => onChange(e.target.value || undefined)}
          style={inputStyle}
        >
          <option value="">— none —</option>
          {(field.values ?? []).map((opt) => (
            <option key={opt} value={opt}>
              {opt}
            </option>
          ))}
        </select>
      );
    case "date":
      return (
        <input
          type="date"
          value={s}
          onChange={textChange}
          style={inputStyle}
        />
      );
    case "boolean":
      return (
        <label
          style={{
            display: "inline-flex",
            gap: "0.4rem",
            alignItems: "center",
          }}
        >
          <input
            type="checkbox"
            checked={value === true}
            onChange={(e) => onChange(e.target.checked)}
          />
          <span style={mutedStyle}>
            {value === true ? "true" : "false"}
          </span>
        </label>
      );
    case "url":
      return (
        <input
          type="url"
          value={s}
          onChange={textChange}
          style={inputStyle}
          placeholder="https://…"
        />
      );
    case "number":
    case "integer":
      return (
        <input
          type="number"
          step={field.type === "integer" ? "1" : "any"}
          min={field.min}
          max={field.max}
          value={s}
          onChange={(e) => {
            if (e.target.value === "") onChange(undefined);
            else onChange(Number(e.target.value));
          }}
          style={inputStyle}
        />
      );
    case "tags": {
      const arr = Array.isArray(value)
        ? value.join(", ")
        : typeof value === "string"
          ? value
          : "";
      return (
        <input
          type="text"
          value={arr}
          onChange={(e) => {
            const raw = e.target.value;
            if (raw.trim() === "") return onChange(undefined);
            onChange(raw.split(",").map((s) => s.trim()).filter(Boolean));
          }}
          style={inputStyle}
          placeholder="tag1, tag2"
        />
      );
    }
    default:
      // long text if likely long, single-line otherwise
      if (field.key === "notes" || field.key === "description") {
        return (
          <textarea
            value={s}
            onChange={textChange}
            style={{ ...inputStyle, minHeight: "5rem", width: "100%" }}
            rows={4}
          />
        );
      }
      return (
        <input
          type="text"
          value={s}
          onChange={textChange}
          style={inputStyle}
        />
      );
  }
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

function isBlank(v: unknown): boolean {
  if (v === null || v === undefined) return true;
  if (typeof v === "string") return v.trim() === "";
  if (Array.isArray(v)) return v.length === 0;
  return false;
}

// coerce ensures every declared field emits its declared type on save,
// dropping undefined/empty entries so the server sees a tidy JSONB body.
function coerce(
  fields: SchemaField[],
  values: Record<string, unknown>,
): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const f of fields) {
    const v = values[f.key];
    if (isBlank(v)) continue;
    switch (f.type) {
      case "number":
      case "integer":
        out[f.key] = typeof v === "number" ? v : Number(v);
        break;
      case "boolean":
        out[f.key] = Boolean(v);
        break;
      case "tags":
        out[f.key] = Array.isArray(v)
          ? v
          : String(v).split(",").map((s) => s.trim()).filter(Boolean);
        break;
      default:
        out[f.key] = v;
    }
  }
  // Preserve any body keys the schema does not know about (unusual, but keeps
  // the invariant that we never silently drop authored data on edit).
  for (const key of Object.keys(values)) {
    if (!(key in out) && !isBlank(values[key]) && !fields.some((f) => f.key === key)) {
      out[key] = values[key];
    }
  }
  return out;
}

// -----------------------------------------------------------------------------
// Styles
// -----------------------------------------------------------------------------

const rowLabelStyle = {
  display: "block",
  fontSize: "0.8rem",
  fontWeight: 600,
  color: "var(--muted)",
  marginBottom: "0.25rem",
};

const primaryButtonStyle = {
  ...buttonStyle,
  background: "var(--btn-primary-bg)",
  color: "var(--btn-primary-fg)",
  padding: "0.5rem 1rem",
};
