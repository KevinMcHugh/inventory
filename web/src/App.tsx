import { useEffect, useState, type ReactNode } from "react";
import {
  BrowserRouter,
  Navigate,
  Route,
  Routes,
} from "react-router-dom";

import { Layout } from "./Layout";
import { Callback } from "./pages/Callback";
import { Home } from "./pages/Home";
import { KindEditPage } from "./pages/KindEditPage";
import { KindPage } from "./pages/KindPage";
import { ModelEditPage } from "./pages/ModelEditPage";
import { ModelPage } from "./pages/ModelPage";
import { getToken, saveToken, startLogin } from "./oauth";
import {
  buttonStyle,
  mutedStyle,
  pageStyle,
  primaryButtonStyle,
} from "./styles";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/callback" element={<Callback />} />
        <Route
          path="/"
          element={
            <AuthGate>
              <Layout>
                <Home />
              </Layout>
            </AuthGate>
          }
        />
        <Route
          path="/kinds/:kindId"
          element={
            <AuthGate>
              <Layout wide>
                <KindPage />
              </Layout>
            </AuthGate>
          }
        />
        <Route
          path="/kinds/:kindId/edit"
          element={
            <AuthGate>
              <Layout wide>
                <KindEditPage />
              </Layout>
            </AuthGate>
          }
        />
        <Route
          path="/kinds/:kindId/models/new"
          element={
            <AuthGate>
              <Layout>
                <ModelEditPage />
              </Layout>
            </AuthGate>
          }
        />
        <Route
          path="/kinds/:kindId/models/:slug"
          element={
            <AuthGate>
              <Layout>
                <ModelPage />
              </Layout>
            </AuthGate>
          }
        />
        <Route
          path="/kinds/:kindId/models/:slug/edit"
          element={
            <AuthGate>
              <Layout>
                <ModelEditPage />
              </Layout>
            </AuthGate>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

// -----------------------------------------------------------------------------
// AuthGate — renders sign-in prompt if no token, otherwise its children.
// -----------------------------------------------------------------------------

function AuthGate({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(false);
  const [signedIn, setSignedIn] = useState(false);

  useEffect(() => {
    // SSO callback lands here with the raw token in the URL fragment.
    // Pull it into localStorage and strip the fragment so a browser refresh
    // does not re-parse it.
    if (window.location.hash.startsWith("#access_token=")) {
      const raw = decodeURIComponent(
        window.location.hash.slice("#access_token=".length),
      );
      if (raw) saveToken(raw);
      window.history.replaceState(
        {},
        "",
        window.location.pathname + window.location.search,
      );
    }
    setSignedIn(Boolean(getToken()));
    setReady(true);
  }, []);

  if (!ready) {
    return (
      <main style={pageStyle}>
        <p style={mutedStyle}>loading…</p>
      </main>
    );
  }
  if (!signedIn) return <SignIn />;
  return <>{children}</>;
}

function SignIn() {
  return (
    <main style={pageStyle}>
      <h1 style={{ margin: 0, fontSize: "1.4rem" }}>Inventory</h1>
      <section style={{ marginTop: "2rem" }}>
        <p style={mutedStyle}>Sign in to view your data.</p>
        <div style={{ display: "grid", gap: "0.6rem", maxWidth: 320 }}>
          <a
            href={
              "/auth/google?return_to=" +
              encodeURIComponent(window.location.pathname || "/")
            }
            style={{
              ...primaryButtonStyle,
              textDecoration: "none",
              textAlign: "center",
            }}
          >
            Sign in with Google
          </a>
          <div
            style={{
              ...mutedStyle,
              textAlign: "center",
              fontSize: "0.85rem",
            }}
          >
            or
          </div>
          <button style={buttonStyle} onClick={() => startLogin()}>
            Use an inv_ api key
          </button>
        </div>
      </section>
    </main>
  );
}
