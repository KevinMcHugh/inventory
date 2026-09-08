import type { ReactNode } from "react";
import { Link } from "react-router-dom";

import { logout } from "./oauth";
import {
  buttonStyle,
  headerStyle,
  linkStyle,
  pageStyle,
} from "./styles";

export function Layout({ children }: { children: ReactNode }) {
  return (
    <main style={pageStyle}>
      <header style={headerStyle}>
        <Link to="/" style={{ ...linkStyle, ...headerTitleStyle }}>
          Inventory
        </Link>
        <button
          style={buttonStyle}
          onClick={() => {
            logout();
            location.reload();
          }}
        >
          Sign out
        </button>
      </header>
      {children}
    </main>
  );
}

const headerTitleStyle = {
  margin: 0,
  fontSize: "1.4rem",
  fontWeight: 600,
} as const;
