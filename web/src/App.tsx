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
import { KindPage } from "./pages/KindPage";
import { ModelPage } from "./pages/ModelPage";
import { getToken, startLogin } from "./oauth";
import {
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
          path="/kinds/:kindId/models/:slug"
          element={
            <AuthGate>
              <Layout>
                <ModelPage />
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
        <p style={mutedStyle}>
          Sign in with your Inventory api key to view your data.
        </p>
        <button style={primaryButtonStyle} onClick={() => startLogin()}>
          Sign in
        </button>
      </section>
    </main>
  );
}
