import { useEffect, useState } from "react";

type Tenant = {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
};

export default function App() {
  const [tenants, setTenants] = useState<Tenant[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch("/tenants")
      .then((r) => (r.ok ? r.json() : Promise.reject(r.statusText)))
      .then(setTenants)
      .catch((e) => setError(String(e)));
  }, []);

  return (
    <main style={{ fontFamily: "system-ui, sans-serif", padding: "2rem" }}>
      <h1>Inventory</h1>
      {error && <pre style={{ color: "crimson" }}>{error}</pre>}
      {tenants === null && !error && <p>loading…</p>}
      {tenants && (
        <ul>
          {tenants.map((t) => (
            <li key={t.id}>
              <strong>{t.name}</strong> <code>{t.id}</code>
            </li>
          ))}
          {tenants.length === 0 && <li>(no tenants yet)</li>}
        </ul>
      )}
    </main>
  );
}
