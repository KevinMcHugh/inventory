import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
  type VisibilityState,
} from "@tanstack/react-table";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";

import { api, type Kind, type Model } from "../api";
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

export function KindPage() {
  const { kindId = "" } = useParams<{ kindId: string }>();
  const [kind, setKind] = useState<Kind | null>(null);
  const [models, setModels] = useState<Model[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [globalFilter, setGlobalFilter] = useState("");
  const [sorting, setSorting] = useState<SortingState>([
    { id: "slug", desc: false },
  ]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(
    () => loadVisibility(kindId),
  );

  useEffect(() => {
    if (!kindId) return;
    setKind(null);
    setModels(null);
    setError(null);
    Promise.all([api.kind(kindId), api.models(kindId)])
      .then(([k, ms]) => {
        setKind(k);
        setModels(ms);
      })
      .catch((e) => setError(String(e)));
  }, [kindId]);

  useEffect(() => {
    saveVisibility(kindId, columnVisibility);
  }, [kindId, columnVisibility]);

  // Union of all body keys across models becomes the dynamic column set.
  const bodyKeys = useMemo(() => {
    if (!models) return [];
    const set = new Set<string>();
    for (const m of models) {
      if (m.body && typeof m.body === "object") {
        for (const k of Object.keys(m.body)) set.add(k);
      }
    }
    return [...set].sort();
  }, [models]);

  const columns = useMemo<ColumnDef<Model, any>[]>(() => {
    const base: ColumnDef<Model, any>[] = [
      columnHelper.accessor("slug", {
        header: "Slug",
        cell: (info) => <strong>{info.getValue()}</strong>,
      }),
    ];
    const bodyCols: ColumnDef<Model, any>[] = bodyKeys.map((key) =>
      columnHelper.accessor((row) => (row.body ?? {})[key], {
        id: `body.${key}`,
        header: key,
        cell: (info) => renderCell(info.getValue()),
        sortingFn: bodySortingFn,
      }),
    );
    const tail: ColumnDef<Model, any>[] = [
      columnHelper.accessor("createdAt", {
        header: "Created",
        cell: (info) => (
          <span style={mutedStyle}>{fmtDate(info.getValue())}</span>
        ),
      }),
      columnHelper.accessor("updatedAt", {
        header: "Updated",
        cell: (info) => (
          <span style={mutedStyle}>{fmtDate(info.getValue())}</span>
        ),
      }),
      columnHelper.accessor("id", {
        header: "ID",
        cell: (info) => <code style={codeStyle}>{info.getValue()}</code>,
      }),
    ];
    return [...base, ...bodyCols, ...tail];
  }, [bodyKeys]);

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
          <ColumnMenu table={table} />
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
// ColumnMenu: <details> dropdown of visibility checkboxes.
// -----------------------------------------------------------------------------

function ColumnMenu({ table }: { table: ReturnType<typeof useReactTable<Model>> }) {
  return (
    <details style={detailsStyle}>
      <summary style={summaryStyle}>Columns</summary>
      <div style={columnMenuStyle}>
        <label style={rowCheckLabelStyle}>
          <input
            type="checkbox"
            checked={table.getIsAllColumnsVisible()}
            ref={(el) => {
              if (el) el.indeterminate = !!table.getIsSomeColumnsVisible() && !table.getIsAllColumnsVisible();
            }}
            onChange={table.getToggleAllColumnsVisibilityHandler()}
          />
          <span style={{ fontWeight: 600 }}>All</span>
        </label>
        <hr style={hrStyle} />
        {table.getAllLeafColumns().map((col) => (
          <label key={col.id} style={rowCheckLabelStyle}>
            <input
              type="checkbox"
              checked={col.getIsVisible()}
              onChange={col.getToggleVisibilityHandler()}
            />
            <span>{columnLabel(col.id, col.columnDef.header)}</span>
          </label>
        ))}
      </div>
    </details>
  );
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

function renderCell(v: unknown): ReactNode {
  if (v === null || v === undefined || v === "") {
    return <span style={mutedStyle}>—</span>;
  }
  if (typeof v === "string" || typeof v === "number" || typeof v === "boolean") {
    return String(v);
  }
  const s = JSON.stringify(v);
  const trimmed = s.length > 80 ? s.slice(0, 80) + "…" : s;
  return <code style={codeStyle}>{trimmed}</code>;
}

function fmtDate(iso: string): string {
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

// TanStack default global filter is string-only. This one stringifies each
// row value so it also matches numbers/booleans/objects.
function fuzzyIncludes(row: any, columnId: string, filterValue: string): boolean {
  if (!filterValue) return true;
  const v = row.getValue(columnId);
  if (v === null || v === undefined) return false;
  const s =
    typeof v === "string"
      ? v
      : typeof v === "number" || typeof v === "boolean"
        ? String(v)
        : JSON.stringify(v);
  return s.toLowerCase().includes(filterValue.toLowerCase());
}

// Body values might be numbers, strings, or arbitrary JSON. Compare numerics
// as numbers when both sides parse; otherwise fall back to string compare.
function bodySortingFn(rowA: any, rowB: any, colId: string): number {
  const a = rowA.getValue(colId);
  const b = rowB.getValue(colId);
  if (a === b) return 0;
  if (a === null || a === undefined) return -1;
  if (b === null || b === undefined) return 1;
  const na = typeof a === "number" ? a : Number(a);
  const nb = typeof b === "number" ? b : Number(b);
  if (!Number.isNaN(na) && !Number.isNaN(nb)) return na - nb;
  return String(a).localeCompare(String(b));
}

function columnLabel(id: string, header: unknown): string {
  if (typeof header === "string") return header;
  return id.startsWith("body.") ? id.slice(5) : id;
}

// -----------------------------------------------------------------------------
// Column visibility persistence, per-kind.
// -----------------------------------------------------------------------------

function storageKey(kindId: string) {
  return `inv.kind.${kindId}.colvis`;
}

function loadVisibility(kindId: string): VisibilityState {
  if (!kindId) return {};
  try {
    const raw = localStorage.getItem(storageKey(kindId));
    if (raw) return JSON.parse(raw);
  } catch {
    // fall through
  }
  return {};
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

const detailsStyle = {
  position: "relative" as const,
};

const summaryStyle = {
  cursor: "pointer",
  padding: "0.4rem 0.7rem",
  background: "var(--btn-bg)",
  color: "var(--btn-fg)",
  borderRadius: 6,
  fontSize: "0.85rem",
  listStyle: "none" as const,
};

const columnMenuStyle = {
  position: "absolute" as const,
  right: 0,
  top: "calc(100% + 4px)",
  background: "var(--bg)",
  border: "1px solid var(--border)",
  borderRadius: 6,
  padding: "0.5rem 0.6rem",
  minWidth: 200,
  maxHeight: 360,
  overflowY: "auto" as const,
  boxShadow: "0 4px 12px rgba(0,0,0,0.15)",
  zIndex: 10,
};

const rowCheckLabelStyle = {
  display: "flex",
  gap: "0.5rem",
  alignItems: "center",
  padding: "0.2rem 0",
  fontSize: "0.85rem",
  cursor: "pointer",
};

const hrStyle = {
  border: 0,
  borderTop: "1px solid var(--border-item)",
  margin: "0.35rem 0",
};
