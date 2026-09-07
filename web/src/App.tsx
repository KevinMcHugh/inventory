import { useEffect, useState } from "react";

type Kind = {
  id: string;
  tenantId: string;
  name: string;
  description?: string;
};

// TODO: the API requires Authorization: Bearer <api key>. Wire up a real auth
// story for the web app (persist a key in localStorage or fetch a signed
// session from the server) before this fetch will succeed. Until then this
// page exists mainly to verify the /kinds route is reachable.
const API_KEY = import.meta.env.VITE_INVENTORY_KEY as string | undefined;

export default function App() {
  const [kinds, setKinds] = useState<Kind[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!API_KEY) {
      setError("set VITE_INVENTORY_KEY in .env.local to fetch");
      return;
    }
    fetch("/kinds", { headers: { Authorization: `Bearer ${API_KEY}` } })
      .then((r) => (r.ok ? r.json() : Promise.reject(r.statusText)))
      .then(setKinds)
      .catch((e) => setError(String(e)));
  }, []);

  return (
    <main style={{ fontFamily: "system-ui, sans-serif", padding: "2rem" }}>
      <h1>Inventory</h1>
      {error && <pre style={{ color: "crimson" }}>{error}</pre>}
      {kinds === null && !error && <p>loading…</p>}
      {kinds && (
        <ul>
          {kinds.map((k) => (
            <li key={k.id}>
              <strong>{k.name}</strong> <code>{k.id}</code>
              {k.description && <> — {k.description}</>}
            </li>
          ))}
          {kinds.length === 0 && <li>(no kinds yet)</li>}
        </ul>
      )}
    </main>
  );
}
