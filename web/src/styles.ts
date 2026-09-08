// Shared inline style objects. Colors reference CSS vars declared in
// index.css so both light and dark palettes apply without a toggle.

import type { CSSProperties } from "react";

export const pageStyle: CSSProperties = {
  maxWidth: 720,
  margin: "3rem auto",
  padding: "0 1.5rem",
  color: "var(--fg)",
};

export const headerStyle: CSSProperties = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
};

export const sectionStyle: CSSProperties = {
  marginTop: "2rem",
  paddingTop: "1rem",
  borderTop: "1px solid var(--border)",
};

export const mutedStyle: CSSProperties = {
  color: "var(--muted)",
  fontSize: "0.9rem",
};

export const codeStyle: CSSProperties = {
  background: "var(--code-bg)",
  color: "var(--fg)",
  padding: "0 0.3rem",
  borderRadius: 3,
  fontSize: "0.85rem",
};

export const errorStyle: CSSProperties = {
  color: "var(--error-fg)",
  background: "var(--error-bg)",
  padding: "0.6rem 0.8rem",
  borderRadius: 6,
  whiteSpace: "pre-wrap",
};

export const buttonStyle: CSSProperties = {
  background: "var(--btn-bg)",
  color: "var(--btn-fg)",
  padding: "0.4rem 0.8rem",
  border: 0,
  borderRadius: 6,
  cursor: "pointer",
};

export const primaryButtonStyle: CSSProperties = {
  ...buttonStyle,
  background: "var(--btn-primary-bg)",
  color: "var(--btn-primary-fg)",
  marginTop: "1rem",
  padding: "0.6rem 1.2rem",
};

export const linkStyle: CSSProperties = {
  color: "var(--link)",
  textDecoration: "none",
};

export const listStyle: CSSProperties = {
  listStyle: "none",
  padding: 0,
  margin: 0,
};

export const listItemStyle: CSSProperties = {
  padding: "0.6rem 0",
  borderBottom: "1px solid var(--border-item)",
  display: "flex",
  alignItems: "center",
  gap: "0.5rem",
};

export const tableStyle: CSSProperties = {
  width: "100%",
  borderCollapse: "collapse",
  marginTop: "0.5rem",
};

export const thStyle: CSSProperties = {
  textAlign: "left",
  padding: "0.35rem 0.5rem",
  borderBottom: "2px solid var(--border)",
  fontWeight: 600,
  fontSize: "0.78rem",
  color: "var(--muted)",
  userSelect: "none",
  whiteSpace: "nowrap",
};

export const thSortableStyle: CSSProperties = {
  ...thStyle,
  cursor: "pointer",
};

export const tdStyle: CSSProperties = {
  padding: "0.28rem 0.5rem",
  borderBottom: "1px solid var(--border-item)",
  fontSize: "0.83rem",
  verticalAlign: "top",
  whiteSpace: "nowrap",
  maxWidth: 260,
  overflow: "hidden",
  textOverflow: "ellipsis",
};

export const inputStyle: CSSProperties = {
  padding: "0.4rem 0.6rem",
  fontSize: "0.85rem",
  border: "1px solid var(--input-border)",
  background: "var(--input-bg)",
  color: "var(--fg)",
  borderRadius: 6,
  minWidth: 0,
};
