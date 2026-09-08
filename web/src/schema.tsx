// Shared helpers around the schema-driven UI. Both the cards list and the
// model detail page render field values the same way, so the formatting
// lives here rather than being duplicated per page.

import type { ReactNode } from "react";

import type { FieldType, Schema, SchemaField, Model } from "./api";
import { codeStyle, linkStyle, mutedStyle } from "./styles";

// unionBodyKeys returns every body key seen across the models list, minus
// the keys the schema already declares. Alphabetized so tail columns are
// stable.
export function unionBodyKeys(models: Model[], schema: Schema | null): string[] {
  const declared = new Set((schema?.fields ?? []).map((f) => f.key));
  const seen = new Set<string>();
  for (const m of models) {
    if (m.body && typeof m.body === "object") {
      for (const k of Object.keys(m.body)) {
        if (!declared.has(k)) seen.add(k);
      }
    }
  }
  return [...seen].sort();
}

// effectiveFields returns the ordered field list for a kind: schema-declared
// fields first (in their authoring order), then any body keys the schema
// does not know about, as untyped text with the key as label.
export function effectiveFields(
  models: Model[],
  schema: Schema | null,
): SchemaField[] {
  const base = schema?.fields ?? [];
  const tail = unionBodyKeys(models, schema).map<SchemaField>((key) => ({
    key,
    type: "text",
  }));
  return [...base, ...tail];
}

// -----------------------------------------------------------------------------
// Value rendering
// -----------------------------------------------------------------------------

export function renderCell(type: FieldType | undefined, v: unknown): ReactNode {
  if (v === null || v === undefined || v === "") {
    return <span style={mutedStyle}>—</span>;
  }
  switch (type) {
    case "date":
      return fmtDate(String(v));
    case "boolean":
      return v ? "✓" : "·";
    case "url": {
      const href = String(v);
      const short = href.replace(/^https?:\/\//, "").slice(0, 40);
      return (
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          style={linkStyle}
        >
          {short}
        </a>
      );
    }
    case "number":
    case "integer":
      if (typeof v === "number") return v.toLocaleString();
      return String(v);
    case "tags":
      if (Array.isArray(v)) return v.join(", ");
      return String(v);
    default:
      if (
        typeof v === "string" ||
        typeof v === "number" ||
        typeof v === "boolean"
      ) {
        return String(v);
      }
      const s = JSON.stringify(v);
      return (
        <code style={codeStyle}>
          {s.length > 80 ? s.slice(0, 80) + "…" : s}
        </code>
      );
  }
}

export function fmtDate(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

export function fmtDateTime(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function fieldLabel(f: SchemaField): string {
  return f.label || f.key;
}
