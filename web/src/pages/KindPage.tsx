import {
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type FilterFn,
  type SortingState,
  type Table,
  type VisibilityState,
} from "@tanstack/react-table";
import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";

import {
  api,
  type Kind,
  type Model,
  type Schema,
  type SchemaField,
} from "../api";
import {
  buttonStyle,
  codeStyle,
  errorStyle,
  inputStyle,
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

const columnHelper = createColumnHelper<Model>();

const EMPTY_SCHEMA: Schema = { fields: [] };

export function KindPage() {
  const { kindId = "" } = useParams<{ kindId: string }>();
  const [kind, setKind] = useState<Kind | null>(null);
  const [models, setModels] = useState<Model[] | null>(null);
  const [schema, setSchema] = useState<Schema | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [globalFilter, setGlobalFilter] = useState("");
  const [sorting, setSorting] = useState<SortingState>([
    { id: "slug", desc: false },
  ]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [visibilityLoaded, setVisibilityLoaded] = useState(false);

  useEffect(() => {
    if (!kindId) return;
    setKind(null);
    setModels(null);
    setSchema(null);
    setError(null);
    setVisibilityLoaded(false);
    Promise.all([
      api.kind(kindId),
      api.models(kindId),
      api.schema(kindId).catch(() => EMPTY_SCHEMA),
    ])
      .then(([k, ms, s]) => {
        setKind(k);
        setModels(ms);
        setSchema(s ?? EMPTY_SCHEMA);
      })
      .catch((e) => setError(String(e)));
  }, [kindId]);

  const fields = useMemo(
    () => (models && schema ? effectiveFields(models, schema) : []),
    [models, schema],
  );

  const fieldByKey = useMemo(() => {
    const m = new Map<string, SchemaField>();
    for (const f of fields) m.set(f.key, f);
    return m;
  }, [fields]);

  // Seed column visibility once both schema and models arrive.
  useEffect(() => {
    if (visibilityLoaded || !schema || !models) return;
    const stored = loadVisibility(kindId);
    if (stored) {
      setColumnVisibility(stored);
    } else {
      const v: VisibilityState = { slug: true };
      for (const f of schema.fields) {
        v[`body.${f.key}`] = Boolean(f.pinned);
      }
      // any tail body key defaults off
      for (const f of fields) {
        if (!(f.key in (schema.fields.find((sf) => sf.key === f.key) ?? {}))) {
          v[`body.${f.key}`] = v[`body.${f.key}`] ?? false;
        }
      }
      setColumnVisibility(v);
    }
    setVisibilityLoaded(true);
  }, [kindId, schema, models, fields, visibilityLoaded]);

  useEffect(() => {
    if (!visibilityLoaded) return;
    saveVisibility(kindId, columnVisibility);
  }, [kindId, columnVisibility, visibilityLoaded]);

  const columns = useMemo<ColumnDef<Model, any>[]>(() => {
    const base: ColumnDef<Model, any>[] = [
      columnHelper.accessor("slug", { header: "Slug" }),
    ];
    const bodyCols: ColumnDef<Model, any>[] = fields.map((f) => makeBodyColumn(f));
    return [...base, ...bodyCols];
  }, [fields]);

  const table = useReactTable({
    data: models ?? [],
    columns,
    state: { sorting, columnVisibility, globalFilter },
    onSortingChange: setSorting,
    onColumnVisibilityChange: setColumnVisibility,
    onGlobalFilterChange: setGlobalFilter,
    globalFilterFn: fuzzyIncludes,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
  });

  const rows = table.getRowModel().rows;
  const filteredCount = table.getFilteredRowModel().rows.length;

  const hasActiveFilters =
    globalFilter !== "" || table.getState().columnFilters.length > 0;

  function resetFilters() {
    setGlobalFilter("");
    table.resetColumnFilters();
  }

  const filterableFields = fields.filter(
    (f) => f.type === "enum" || f.type === "date" || f.type === "number" || f.type === "integer",
  );

  return (
    <>
      <nav style={{ marginTop: "1rem" }}>
        <Link to="/" style={linkStyle}>
          ← All kinds
        </Link>
      </nav>

      {error && <pre style={errorStyle}>{error}</pre>}

      <section style={sectionStyle}>
        {kind === null && !error && <p style={mutedStyle}>loading…</p>}
        {kind && (
          <>
            <div style={mutedStyle}>Kind</div>
            <div style={kindHeaderStyle}>
              <div style={{ fontSize: "1.3rem", fontWeight: 600 }}>
                {kind.name} <code style={codeStyle}>{kind.id}</code>
              </div>
              <Link
                to={`/kinds/${kindId}/edit`}
                style={{ ...buttonStyle, textDecoration: "none" }}
              >
                Edit schema
              </Link>
            </div>
            {kind.description && (
              <p style={{ ...mutedStyle, marginTop: "0.4rem" }}>
                {kind.description}
              </p>
            )}
          </>
        )}
      </section>

      <section style={sectionStyle}>
        <div style={controlsRowStyle}>
          <h2 style={{ margin: 0, fontSize: "1rem" }}>
            Models
            {models && (
              <span style={{ ...mutedStyle, marginLeft: "0.5rem" }}>
                {filteredCount === models.length
                  ? models.length
                  : `${filteredCount} / ${models.length}`}
              </span>
            )}
          </h2>
          <input
            type="search"
            placeholder="Search…"
            value={globalFilter}
            onChange={(e) => setGlobalFilter(e.target.value)}
            style={{ ...inputStyle, flex: 1, maxWidth: 260 }}
          />
          <SortControl table={table} fields={fields} />
          <FieldMenu table={table} fields={fields} />
          {hasActiveFilters && (
            <button style={buttonStyle} onClick={resetFilters}>
              Reset filters
            </button>
          )}
          <Link
            to={`/kinds/${kindId}/models/new`}
            style={{ ...primaryLinkStyle, textDecoration: "none" }}
          >
            + New
          </Link>
        </div>

        {filterableFields.length > 0 && (
          <div style={filterStripStyle}>
            {filterableFields.map((f) => (
              <FilterChip key={f.key} field={f} table={table} />
            ))}
          </div>
        )}

        {models === null && !error && <p style={mutedStyle}>loading…</p>}
        {models && models.length === 0 && (
          <p style={mutedStyle}>No models in this kind yet.</p>
        )}
        {models && models.length > 0 && (
          <div style={gridStyle}>
            {rows.map((row) => (
              <Card
                key={row.original.id}
                model={row.original}
                fields={fields}
                fieldByKey={fieldByKey}
                visibility={columnVisibility}
                kindId={kindId}
              />
            ))}
          </div>
        )}
      </section>
    </>
  );
}

// -----------------------------------------------------------------------------
// Card
// -----------------------------------------------------------------------------

function Card({
  model,
  fields,
  fieldByKey,
  visibility,
  kindId,
}: {
  model: Model;
  fields: SchemaField[];
  fieldByKey: Map<string, SchemaField>;
  visibility: VisibilityState;
  kindId: string;
}) {
  const visibleFields = fields.filter(
    (f) => visibility[`body.${f.key}`] !== false,
  );
  const body = model.body ?? {};
  return (
    <article style={cardStyle}>
      <header style={cardHeaderStyle}>
        <Link
          to={`/kinds/${kindId}/models/${encodeURIComponent(model.slug)}`}
          style={{ ...linkStyle, fontWeight: 600, fontSize: "0.95rem" }}
        >
          {model.slug}
        </Link>
      </header>
      <div style={cardBodyStyle}>
        {visibleFields.map((f) => (
          <div key={f.key} style={fieldRowStyle}>
            <div style={fieldLabelStyle}>{fieldLabel(f)}</div>
            <div style={fieldValueStyle}>
              {renderCell(fieldByKey.get(f.key)?.type ?? f.type, body[f.key])}
            </div>
          </div>
        ))}
      </div>
      <footer style={cardFooterStyle}>
        <span>updated {fmtDateTime(model.updatedAt)}</span>
      </footer>
    </article>
  );
}

// -----------------------------------------------------------------------------
// TanStack column builder (needed for filtering/sorting even without a table)
// -----------------------------------------------------------------------------

function makeBodyColumn(field: SchemaField): ColumnDef<Model, any> {
  const filterFn =
    field.type === "enum"
      ? enumFilterFn
      : field.type === "date"
        ? dateFilterFn
        : field.type === "number" || field.type === "integer"
          ? numberFilterFn
          : undefined;
  return columnHelper.accessor((row) => (row.body ?? {})[field.key], {
    id: `body.${field.key}`,
    header: fieldLabel(field),
    filterFn,
    sortingFn:
      field.type === "number" || field.type === "integer" ? numericSort : "auto",
  });
}

// -----------------------------------------------------------------------------
// Sort control
// -----------------------------------------------------------------------------

function SortControl({
  table,
  fields,
}: {
  table: Table<Model>;
  fields: SchemaField[];
}) {
  const state = table.getState().sorting[0];
  const currentId = state?.id ?? "slug";
  const currentDir: "asc" | "desc" = state?.desc ? "desc" : "asc";

  const sortOptions: { id: string; label: string }[] = [
    { id: "slug", label: "Slug" },
    ...fields.map((f) => ({ id: `body.${f.key}`, label: fieldLabel(f) })),
  ];

  function setSortId(id: string) {
    table.setSorting([{ id, desc: currentDir === "desc" }]);
  }
  function toggleDir() {
    table.setSorting([{ id: currentId, desc: currentDir === "asc" }]);
  }

  return (
    <div style={{ display: "flex", gap: "0.25rem", alignItems: "center" }}>
      <select
        value={currentId}
        onChange={(e) => setSortId(e.target.value)}
        style={{ ...inputStyle, padding: "0.3rem 0.4rem" }}
      >
        {sortOptions.map((o) => (
          <option key={o.id} value={o.id}>
            Sort by {o.label}
          </option>
        ))}
      </select>
      <button style={buttonStyle} onClick={toggleDir} aria-label="toggle direction">
        {currentDir === "asc" ? "↑" : "↓"}
      </button>
    </div>
  );
}

// -----------------------------------------------------------------------------
// Field visibility menu (renamed from Columns → Fields)
// -----------------------------------------------------------------------------

function FieldMenu({
  table,
  fields,
}: {
  table: Table<Model>;
  fields: SchemaField[];
}) {
  return (
    <details style={{ position: "relative" }}>
      <summary style={filterSummaryStyle}>Fields</summary>
      <div style={filterMenuStyle}>
        <label style={filterCheckLabelStyle}>
          <input
            type="checkbox"
            checked={table.getIsAllColumnsVisible()}
            ref={(el) => {
              if (el)
                el.indeterminate =
                  !!table.getIsSomeColumnsVisible() &&
                  !table.getIsAllColumnsVisible();
            }}
            onChange={table.getToggleAllColumnsVisibilityHandler()}
          />
          <span style={{ fontWeight: 600 }}>All</span>
        </label>
        <hr style={hrStyle} />
        {fields.map((f) => {
          const col = table.getColumn(`body.${f.key}`);
          if (!col) return null;
          return (
            <label key={f.key} style={filterCheckLabelStyle}>
              <input
                type="checkbox"
                checked={col.getIsVisible()}
                onChange={col.getToggleVisibilityHandler()}
              />
              <span>{fieldLabel(f)}</span>
            </label>
          );
        })}
      </div>
    </details>
  );
}

// -----------------------------------------------------------------------------
// Per-field filter chip
// -----------------------------------------------------------------------------

function FilterChip({ field, table }: { field: SchemaField; table: Table<Model> }) {
  const col = table.getColumn(`body.${field.key}`);
  if (!col) return null;
  if (field.type === "enum") {
    const value = (col.getFilterValue() as string[] | undefined) ?? [];
    const options = field.values ?? [];
    const summary =
      value.length === 0
        ? fieldLabel(field)
        : value.length === 1
          ? `${fieldLabel(field)}: ${value[0]}`
          : `${fieldLabel(field)}: ${value.length}`;
    return (
      <details style={{ position: "relative" }}>
        <summary style={{ ...filterSummaryStyle, background: value.length ? "var(--btn-primary-bg)" : "var(--btn-bg)", color: value.length ? "var(--btn-primary-fg)" : "var(--btn-fg)" }}>
          {summary}
        </summary>
        <div style={filterMenuStyle}>
          {options.map((opt) => {
            const checked = value.includes(opt);
            return (
              <label key={opt} style={filterCheckLabelStyle}>
                <input
                  type="checkbox"
                  checked={checked}
                  onChange={(e) => {
                    const next = e.target.checked
                      ? [...value, opt]
                      : value.filter((v) => v !== opt);
                    col.setFilterValue(next.length ? next : undefined);
                  }}
                />
                {opt}
              </label>
            );
          })}
          {value.length > 0 && (
            <button
              type="button"
              style={filterClearStyle}
              onClick={() => col.setFilterValue(undefined)}
            >
              Clear
            </button>
          )}
        </div>
      </details>
    );
  }
  if (field.type === "date") {
    const v = (col.getFilterValue() as { from?: string; to?: string } | undefined) ?? {};
    return (
      <div style={rangeChipStyle}>
        <span style={mutedStyle}>{fieldLabel(field)}</span>
        <input
          type="date"
          value={v.from ?? ""}
          onChange={(e) => {
            const next = { ...v, from: e.target.value || undefined };
            const empty = !next.from && !next.to;
            col.setFilterValue(empty ? undefined : next);
          }}
          style={rangeInputStyle}
        />
        <span>–</span>
        <input
          type="date"
          value={v.to ?? ""}
          onChange={(e) => {
            const next = { ...v, to: e.target.value || undefined };
            const empty = !next.from && !next.to;
            col.setFilterValue(empty ? undefined : next);
          }}
          style={rangeInputStyle}
        />
      </div>
    );
  }
  if (field.type === "number" || field.type === "integer") {
    const v = (col.getFilterValue() as { from?: number; to?: number } | undefined) ?? {};
    return (
      <div style={rangeChipStyle}>
        <span style={mutedStyle}>{fieldLabel(field)}</span>
        <input
          type="number"
          placeholder="min"
          value={v.from ?? ""}
          onChange={(e) => {
            const num = e.target.value === "" ? undefined : Number(e.target.value);
            const next = { ...v, from: num };
            const empty = next.from === undefined && next.to === undefined;
            col.setFilterValue(empty ? undefined : next);
          }}
          style={rangeInputStyle}
        />
        <span>–</span>
        <input
          type="number"
          placeholder="max"
          value={v.to ?? ""}
          onChange={(e) => {
            const num = e.target.value === "" ? undefined : Number(e.target.value);
            const next = { ...v, to: num };
            const empty = next.from === undefined && next.to === undefined;
            col.setFilterValue(empty ? undefined : next);
          }}
          style={rangeInputStyle}
        />
      </div>
    );
  }
  return null;
}

// -----------------------------------------------------------------------------
// Filter fns
// -----------------------------------------------------------------------------

const fuzzyIncludes: FilterFn<Model> = (row, columnId, filterValue) => {
  if (!filterValue) return true;
  const v = row.getValue(columnId);
  if (v === null || v === undefined) return false;
  const s =
    typeof v === "string"
      ? v
      : typeof v === "number" || typeof v === "boolean"
        ? String(v)
        : JSON.stringify(v);
  return s.toLowerCase().includes(String(filterValue).toLowerCase());
};

const enumFilterFn: FilterFn<Model> = (row, columnId, filterValue) => {
  if (!Array.isArray(filterValue) || filterValue.length === 0) return true;
  const v = row.getValue(columnId);
  return filterValue.includes(String(v));
};

const dateFilterFn: FilterFn<Model> = (row, columnId, filterValue) => {
  const f = filterValue as { from?: string; to?: string };
  if (!f?.from && !f?.to) return true;
  const raw = row.getValue(columnId);
  if (raw == null) return false;
  const t = new Date(String(raw)).getTime();
  if (Number.isNaN(t)) return false;
  if (f.from && t < new Date(f.from).getTime()) return false;
  if (f.to && t > new Date(f.to + "T23:59:59").getTime()) return false;
  return true;
};

const numberFilterFn: FilterFn<Model> = (row, columnId, filterValue) => {
  const f = filterValue as { from?: number; to?: number };
  if (f?.from === undefined && f?.to === undefined) return true;
  const raw = row.getValue(columnId);
  const n = typeof raw === "number" ? raw : Number(raw);
  if (Number.isNaN(n)) return false;
  if (f.from !== undefined && n < f.from) return false;
  if (f.to !== undefined && n > f.to) return false;
  return true;
};

function numericSort(a: any, b: any, colId: string): number {
  const av = Number(a.getValue(colId));
  const bv = Number(b.getValue(colId));
  if (Number.isNaN(av) && Number.isNaN(bv)) return 0;
  if (Number.isNaN(av)) return -1;
  if (Number.isNaN(bv)) return 1;
  return av - bv;
}

// -----------------------------------------------------------------------------
// Column visibility persistence
// -----------------------------------------------------------------------------

function storageKey(kindId: string) {
  return `inv.kind.${kindId}.colvis`;
}
function loadVisibility(kindId: string): VisibilityState | null {
  if (!kindId) return null;
  try {
    const raw = localStorage.getItem(storageKey(kindId));
    if (raw) return JSON.parse(raw);
  } catch {}
  return null;
}
function saveVisibility(kindId: string, v: VisibilityState) {
  if (!kindId) return;
  try {
    localStorage.setItem(storageKey(kindId), JSON.stringify(v));
  } catch {}
}

// -----------------------------------------------------------------------------
// Styles
// -----------------------------------------------------------------------------

const controlsRowStyle = {
  display: "flex",
  gap: "0.5rem",
  alignItems: "center",
  marginBottom: "0.75rem",
  flexWrap: "wrap" as const,
};

const kindHeaderStyle = {
  display: "flex",
  gap: "0.75rem",
  alignItems: "center",
  justifyContent: "space-between",
  flexWrap: "wrap" as const,
};

const primaryLinkStyle = {
  ...buttonStyle,
  background: "var(--btn-primary-bg)",
  color: "var(--btn-primary-fg)",
};

const filterStripStyle = {
  display: "flex",
  gap: "0.5rem",
  flexWrap: "wrap" as const,
  marginBottom: "0.75rem",
  padding: "0.5rem 0",
  borderTop: "1px solid var(--border-item)",
  borderBottom: "1px solid var(--border-item)",
};

const gridStyle = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fill, minmax(300px, 1fr))",
  gap: "0.75rem",
  marginTop: "0.25rem",
};

const cardStyle = {
  border: "1px solid var(--border)",
  borderRadius: 8,
  padding: "0.75rem 0.9rem",
  background: "var(--bg)",
  display: "flex",
  flexDirection: "column" as const,
  gap: "0.6rem",
};

const cardHeaderStyle = {
  display: "flex",
  justifyContent: "space-between",
  gap: "0.5rem",
  alignItems: "baseline",
};

const cardBodyStyle = {
  display: "grid",
  gridTemplateColumns: "auto 1fr",
  columnGap: "0.75rem",
  rowGap: "0.2rem",
};

const fieldRowStyle = {
  display: "contents",
};

const fieldLabelStyle = {
  color: "var(--muted)",
  fontSize: "0.75rem",
  whiteSpace: "nowrap" as const,
  textTransform: "uppercase" as const,
  letterSpacing: "0.02em",
};

const fieldValueStyle = {
  fontSize: "0.9rem",
  wordBreak: "break-word" as const,
  minWidth: 0,
};

const cardFooterStyle = {
  color: "var(--muted)",
  fontSize: "0.75rem",
  borderTop: "1px solid var(--border-item)",
  paddingTop: "0.4rem",
};

const filterSummaryStyle = {
  cursor: "pointer",
  padding: "0.35rem 0.6rem",
  background: "var(--btn-bg)",
  color: "var(--btn-fg)",
  borderRadius: 6,
  fontSize: "0.8rem",
  listStyle: "none" as const,
  display: "inline-block",
  whiteSpace: "nowrap" as const,
};

const filterMenuStyle = {
  position: "absolute" as const,
  left: 0,
  top: "calc(100% + 4px)",
  background: "var(--bg)",
  border: "1px solid var(--border)",
  borderRadius: 6,
  padding: "0.5rem 0.6rem",
  minWidth: 200,
  maxHeight: 320,
  overflowY: "auto" as const,
  boxShadow: "0 4px 12px rgba(0,0,0,0.25)",
  zIndex: 20,
};

const filterCheckLabelStyle = {
  display: "flex",
  gap: "0.4rem",
  alignItems: "center",
  padding: "0.15rem 0",
  fontSize: "0.85rem",
  cursor: "pointer",
  whiteSpace: "nowrap" as const,
};

const filterClearStyle = {
  marginTop: "0.4rem",
  background: "transparent",
  color: "var(--link)",
  border: 0,
  padding: 0,
  cursor: "pointer",
  fontSize: "0.8rem",
};

const rangeChipStyle = {
  display: "flex",
  gap: "0.3rem",
  alignItems: "center",
  padding: "0.25rem 0.5rem",
  background: "var(--btn-bg)",
  borderRadius: 6,
  fontSize: "0.8rem",
};

const rangeInputStyle = {
  ...inputStyle,
  padding: "0.15rem 0.35rem",
  fontSize: "0.8rem",
  maxWidth: "7rem",
};

const hrStyle = {
  border: 0,
  borderTop: "1px solid var(--border-item)",
  margin: "0.35rem 0",
};
