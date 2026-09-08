import {
  createColumnHelper,
  flexRender,
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
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";

import {
  api,
  type Kind,
  type Model,
  type Schema,
  type SchemaField,
} from "../api";
import {
  codeStyle,
  errorStyle,
  inputStyle,
  linkStyle,
  mutedStyle,
  sectionStyle,
  tableStyle,
  tdStyle,
  thSortableStyle,
  thStyle,
} from "../styles";

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

  // Compute effective column list (schema-declared + observed body keys).
  const fieldById = useMemo(() => {
    const map = new Map<string, SchemaField>();
    for (const f of schema?.fields ?? []) map.set(f.key, f);
    return map;
  }, [schema]);

  const orderedFieldKeys = useMemo(() => {
    if (!schema) return [];
    return schema.fields.map((f) => f.key);
  }, [schema]);

  const tailBodyKeys = useMemo(() => {
    if (!models) return [];
    const set = new Set<string>();
    for (const m of models) {
      if (m.body && typeof m.body === "object") {
        for (const k of Object.keys(m.body)) {
          if (!fieldById.has(k)) set.add(k);
        }
      }
    }
    return [...set].sort();
  }, [models, fieldById]);

  // Once schema + models arrive, seed columnVisibility with the pinned
  // defaults unless localStorage already has a per-kind override.
  useEffect(() => {
    if (visibilityLoaded || !schema || !models) return;
    const stored = loadVisibility(kindId);
    if (stored) {
      setColumnVisibility(stored);
    } else {
      const v: VisibilityState = { slug: true, id: false, createdAt: false, updatedAt: false };
      for (const f of schema.fields) {
        v[`body.${f.key}`] = Boolean(f.pinned);
      }
      for (const k of tailBodyKeys) {
        v[`body.${k}`] = false;
      }
      setColumnVisibility(v);
    }
    setVisibilityLoaded(true);
  }, [kindId, schema, models, tailBodyKeys, visibilityLoaded]);

  useEffect(() => {
    if (!visibilityLoaded) return;
    saveVisibility(kindId, columnVisibility);
  }, [kindId, columnVisibility, visibilityLoaded]);

  const columns = useMemo<ColumnDef<Model, any>[]>(() => {
    const base: ColumnDef<Model, any>[] = [
      columnHelper.accessor("slug", {
        header: "Slug",
        cell: (info) => <strong>{info.getValue()}</strong>,
      }),
    ];
    const bodyCols: ColumnDef<Model, any>[] = [...orderedFieldKeys, ...tailBodyKeys].map(
      (key) => makeBodyColumn(key, fieldById.get(key)),
    );
    const tail: ColumnDef<Model, any>[] = [
      columnHelper.accessor("createdAt", {
        header: "Created",
        cell: (info) => (
          <span style={mutedStyle}>{fmtDateTime(info.getValue())}</span>
        ),
      }),
      columnHelper.accessor("updatedAt", {
        header: "Updated",
        cell: (info) => (
          <span style={mutedStyle}>{fmtDateTime(info.getValue())}</span>
        ),
      }),
      columnHelper.accessor("id", {
        header: "ID",
        cell: (info) => <code style={codeStyle}>{info.getValue()}</code>,
      }),
    ];
    return [...base, ...bodyCols, ...tail];
  }, [orderedFieldKeys, tailBodyKeys, fieldById]);

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

  const filteredCount = table.getFilteredRowModel().rows.length;

  const hasActiveFilters =
    globalFilter !== "" || table.getState().columnFilters.length > 0;

  function resetFilters() {
    setGlobalFilter("");
    table.resetColumnFilters();
  }

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
            <div style={{ fontSize: "1.3rem", fontWeight: 600 }}>
              {kind.name} <code style={codeStyle}>{kind.id}</code>
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
            placeholder="Filter…"
            value={globalFilter}
            onChange={(e) => setGlobalFilter(e.target.value)}
            style={{ ...inputStyle, flex: 1, maxWidth: 320 }}
          />
          {hasActiveFilters && (
            <button
              style={{
                background: "var(--btn-bg)",
                color: "var(--btn-fg)",
                border: 0,
                borderRadius: 6,
                padding: "0.4rem 0.7rem",
                fontSize: "0.85rem",
                cursor: "pointer",
              }}
              onClick={resetFilters}
            >
              Reset filters
            </button>
          )}
          <ColumnMenu table={table} fieldById={fieldById} />
        </div>

        {models === null && !error && <p style={mutedStyle}>loading…</p>}
        {models && models.length === 0 && (
          <p style={mutedStyle}>No models in this kind yet.</p>
        )}
        {models && models.length > 0 && (
          <div style={{ overflowX: "auto" }}>
            <table style={tableStyle}>
              <thead>
                {table.getHeaderGroups().map((hg) => (
                  <tr key={hg.id}>
                    {hg.headers.map((h) => {
                      const canSort = h.column.getCanSort();
                      const dir = h.column.getIsSorted();
                      return (
                        <th
                          key={h.id}
                          style={canSort ? thSortableStyle : thStyle}
                          onClick={
                            canSort
                              ? h.column.getToggleSortingHandler()
                              : undefined
                          }
                        >
                          {flexRender(
                            h.column.columnDef.header,
                            h.getContext(),
                          )}
                          {dir === "asc" && " ↑"}
                          {dir === "desc" && " ↓"}
                        </th>
                      );
                    })}
                  </tr>
                ))}
                <tr>
                  {table.getVisibleLeafColumns().map((col) => (
                    <th key={col.id} style={filterCellStyle}>
                      <ColumnFilter column={col} fieldById={fieldById} />
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {table.getRowModel().rows.map((row) => (
                  <tr key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <td key={cell.id} style={tdStyle}>
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext(),
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </>
  );
}

// -----------------------------------------------------------------------------
// Body columns — one per known field, driven by the field's declared type.
// -----------------------------------------------------------------------------

function makeBodyColumn(
  key: string,
  field: SchemaField | undefined,
): ColumnDef<Model, any> {
  const type = field?.type ?? "text";
  const header = field?.label || key;
  const filterFn =
    type === "enum"
      ? enumFilterFn
      : type === "date"
        ? dateFilterFn
        : type === "number" || type === "integer"
          ? numberFilterFn
          : undefined;
  return columnHelper.accessor((row) => (row.body ?? {})[key], {
    id: `body.${key}`,
    header,
    cell: (info) => renderCellFor(type, info.getValue()),
    sortingFn: type === "number" || type === "integer" ? numericSort : "auto",
    filterFn,
  });
}

// -----------------------------------------------------------------------------
// Per-column filter widgets
// -----------------------------------------------------------------------------

function ColumnFilter({
  column,
  fieldById,
}: {
  column: any;
  fieldById: Map<string, SchemaField>;
}) {
  const id: string = column.id;
  if (!id.startsWith("body.")) return null;
  const key = id.slice(5);
  const field = fieldById.get(key);
  const type = field?.type;
  if (type === "enum") return <EnumFilter column={column} field={field!} />;
  if (type === "date") return <DateFilter column={column} />;
  if (type === "number" || type === "integer") return <NumberFilter column={column} />;
  return null;
}

function EnumFilter({
  column,
  field,
}: {
  column: any;
  field: SchemaField;
}) {
  const value = (column.getFilterValue() as string[] | undefined) ?? [];
  const options = field.values ?? [];
  const summary =
    value.length === 0
      ? "Any"
      : value.length === 1
        ? value[0]
        : `${value.length} selected`;
  return (
    <details style={{ position: "relative" }}>
      <summary style={filterSummaryStyle}>{summary}</summary>
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
                  column.setFilterValue(next.length ? next : undefined);
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
            onClick={() => column.setFilterValue(undefined)}
          >
            Clear
          </button>
        )}
      </div>
    </details>
  );
}

function DateFilter({ column }: { column: any }) {
  const v = (column.getFilterValue() as { from?: string; to?: string } | undefined) ?? {};
  function update(patch: { from?: string; to?: string }) {
    const next = { ...v, ...patch };
    const empty = !next.from && !next.to;
    column.setFilterValue(empty ? undefined : next);
  }
  return (
    <div style={rangeWrapStyle}>
      <input
        type="date"
        value={v.from ?? ""}
        onChange={(e) => update({ from: e.target.value || undefined })}
        style={rangeInputStyle}
      />
      <input
        type="date"
        value={v.to ?? ""}
        onChange={(e) => update({ to: e.target.value || undefined })}
        style={rangeInputStyle}
      />
    </div>
  );
}

function NumberFilter({ column }: { column: any }) {
  const v = (column.getFilterValue() as { from?: number; to?: number } | undefined) ?? {};
  function update(patch: { from?: number; to?: number }) {
    const next = { ...v, ...patch };
    const empty = next.from === undefined && next.to === undefined;
    column.setFilterValue(empty ? undefined : next);
  }
  return (
    <div style={rangeWrapStyle}>
      <input
        type="number"
        placeholder="min"
        value={v.from ?? ""}
        onChange={(e) =>
          update({ from: e.target.value === "" ? undefined : Number(e.target.value) })
        }
        style={rangeInputStyle}
      />
      <input
        type="number"
        placeholder="max"
        value={v.to ?? ""}
        onChange={(e) =>
          update({ to: e.target.value === "" ? undefined : Number(e.target.value) })
        }
        style={rangeInputStyle}
      />
    </div>
  );
}

// -----------------------------------------------------------------------------
// Column visibility dropdown
// -----------------------------------------------------------------------------

function ColumnMenu({
  table,
  fieldById,
}: {
  table: Table<Model>;
  fieldById: Map<string, SchemaField>;
}) {
  function label(col: any) {
    const id: string = col.id;
    if (id.startsWith("body.")) {
      const key = id.slice(5);
      return fieldById.get(key)?.label || key;
    }
    if (typeof col.columnDef.header === "string") return col.columnDef.header;
    return id;
  }
  return (
    <details style={{ position: "relative" }}>
      <summary style={filterSummaryStyle}>Columns</summary>
      <div style={filterMenuStyle}>
        <label style={filterCheckLabelStyle}>
          <input
            type="checkbox"
            checked={table.getIsAllColumnsVisible()}
            ref={(el) => {
              if (el) el.indeterminate =
                !!table.getIsSomeColumnsVisible() && !table.getIsAllColumnsVisible();
            }}
            onChange={table.getToggleAllColumnsVisibilityHandler()}
          />
          <span style={{ fontWeight: 600 }}>All</span>
        </label>
        <hr style={{ border: 0, borderTop: "1px solid var(--border-item)", margin: "0.35rem 0" }} />
        {table.getAllLeafColumns().map((col) => (
          <label key={col.id} style={filterCheckLabelStyle}>
            <input
              type="checkbox"
              checked={col.getIsVisible()}
              onChange={col.getToggleVisibilityHandler()}
            />
            <span>{label(col)}</span>
          </label>
        ))}
      </div>
    </details>
  );
}

// -----------------------------------------------------------------------------
// Cell renderers
// -----------------------------------------------------------------------------

function renderCellFor(type: string, v: unknown): ReactNode {
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
      if (typeof v === "string" || typeof v === "number" || typeof v === "boolean") {
        return String(v);
      }
      const s = JSON.stringify(v);
      return <code style={codeStyle}>{s.length > 80 ? s.slice(0, 80) + "…" : s}</code>;
  }
}

function fmtDate(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function fmtDateTime(iso: string): string {
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

// -----------------------------------------------------------------------------
// Filter functions
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
// Column visibility persistence, per-kind.
// -----------------------------------------------------------------------------

function storageKey(kindId: string) {
  return `inv.kind.${kindId}.colvis`;
}

function loadVisibility(kindId: string): VisibilityState | null {
  if (!kindId) return null;
  try {
    const raw = localStorage.getItem(storageKey(kindId));
    if (raw) return JSON.parse(raw);
  } catch {
    // fall through
  }
  return null;
}

function saveVisibility(kindId: string, v: VisibilityState) {
  if (!kindId) return;
  try {
    localStorage.setItem(storageKey(kindId), JSON.stringify(v));
  } catch {
    // ignore
  }
}

// -----------------------------------------------------------------------------
// Styles
// -----------------------------------------------------------------------------

const controlsRowStyle = {
  display: "flex",
  gap: "0.75rem",
  alignItems: "center",
  marginBottom: "0.75rem",
  flexWrap: "wrap" as const,
};

const filterCellStyle = {
  padding: "0.25rem 0.4rem",
  borderBottom: "1px solid var(--border)",
  background: "var(--bg)",
};

const filterSummaryStyle = {
  cursor: "pointer",
  padding: "0.25rem 0.5rem",
  background: "var(--btn-bg)",
  color: "var(--btn-fg)",
  borderRadius: 4,
  fontSize: "0.75rem",
  listStyle: "none" as const,
  display: "inline-block",
  minWidth: "3.5rem",
};

const filterMenuStyle = {
  position: "absolute" as const,
  left: 0,
  top: "calc(100% + 4px)",
  background: "var(--bg)",
  border: "1px solid var(--border)",
  borderRadius: 6,
  padding: "0.5rem 0.6rem",
  minWidth: 180,
  maxHeight: 320,
  overflowY: "auto" as const,
  boxShadow: "0 4px 12px rgba(0,0,0,0.2)",
  zIndex: 20,
};

const filterCheckLabelStyle = {
  display: "flex",
  gap: "0.4rem",
  alignItems: "center",
  padding: "0.15rem 0",
  fontSize: "0.8rem",
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
  fontSize: "0.75rem",
};

const rangeWrapStyle = {
  display: "flex",
  gap: "0.2rem",
  alignItems: "center",
};

const rangeInputStyle = {
  ...inputStyle,
  padding: "0.15rem 0.3rem",
  fontSize: "0.75rem",
  maxWidth: "6rem",
};
