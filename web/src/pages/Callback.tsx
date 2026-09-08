import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { completeLoginFromURL } from "../oauth";
import { errorStyle, mutedStyle, pageStyle } from "../styles";

export function Callback() {
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    completeLoginFromURL()
      .then((ok) => {
        if (ok) navigate("/", { replace: true });
        else navigate("/", { replace: true });
      })
      .catch((e) => setError(String(e)));
  }, [navigate]);

  return (
    <main style={pageStyle}>
      {error ? (
        <pre style={errorStyle}>{error}</pre>
      ) : (
        <p style={mutedStyle}>signing in…</p>
      )}
    </main>
  );
}
