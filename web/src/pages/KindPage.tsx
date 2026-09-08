import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { api, type Kind, type Model } from "../api";
import {
  codeStyle,
  errorStyle,
  linkStyle,
  mutedStyle,
  sectionStyle,
  tableStyle,
  tdStyle,
  thSortableStyle,
  thStyle,
} from "../styles";

type SortKey = "slug" | "createdAt" | "updatedAt";
type SortDir = "asc" | "desc";

export function KindPage() {
  const { kindId = "" } = useParams<{ kindId: string }>();
  const [kind, setKind] = useState<Kind | null>(null);
  const [models, setModels] = useState<Model[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [sort, setSort] = useState<{ key: SortKey; dir: SortDir }>({
    key: "slug",
    dir: "asc",
  });

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

  const sorted = useMemo(() => {
    if (!models) return null;
    const { key, dir } = sort;
    return [...models].sort((a, b) => {
      const av = String(a[key] ?? "");
      const bv = String(b[key] ?? "");
      const cmp = av.localeCompare(bv);
      return dir === "asc" ? cmp : -cmp;
    });
  }, [models, sort]);

  function toggle(key: SortKey) {
    setSort((s) =>
      s.key === key
        ? { key, dir: s.dir === "asc" ? "desc" : "asc" }
        : { key, dir: "asc" },
    );
  }

  function arrow(key: SortKey) {
    if (sort.key !== key) return "";
    return sort.dir === "asc" ? " ↑" : " ↓";
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
        <h2 style={{ margin: "0 0 0.5rem 0", fontSize: "1rem" }}>
          Models{models && ` (${models.length})`}
        </h2>
        {sorted === null && !error && <p style={mutedStyle}>loading…</p>}
        {sorted && sorted.length === 0 && (
          <p style={mutedStyle}>No models in this kind yet.</p>
        )}
        {sorted && sorted.length > 0 && (
          <div style={{ overflowX: "auto" }}>
            <table style={tableStyle}>
              <thead>
                <tr>
                  <th style={thSortableStyle} onClick={() => toggle("slug")}>
                    Slug{arrow("slug")}
                  </th>
                  <th style={thStyle}>ID</th>
                  <th
                    style={thSortableStyle}
                    onClick={() => toggle("createdAt")}
                  >
                    Created{arrow("createdAt")}
                  </th>
                  <th
                    style={thSortableStyle}
                    onClick={() => toggle("updatedAt")}
                  >
                    Updated{arrow("updatedAt")}
                  </th>
                  <th style={thStyle}>Body</th>
                </tr>
              </thead>
              <tbody>
                {sorted.map((m) => (
                  <tr key={m.id}>
                    <td style={tdStyle}>
                      <strong>{m.slug}</strong>
                    </td>
                    <td style={tdStyle}>
                      <code style={codeStyle}>{m.id}</code>
                    </td>
                    <td style={tdStyle}>{fmtDate(m.createdAt)}</td>
                    <td style={tdStyle}>{fmtDate(m.updatedAt)}</td>
                    <td style={tdStyle}>
                      <code style={codeStyle}>{bodyPreview(m.body)}</code>
                    </td>
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

function fmtDate(iso: string): string {
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

function bodyPreview(body: Record<string, unknown>): string {
  if (!body) return "{}";
  const keys = Object.keys(body);
  if (keys.length === 0) return "{}";
  const shown = keys.slice(0, 3).join(", ");
  return keys.length > 3 ? `{ ${shown}, … }` : `{ ${shown} }`;
}
