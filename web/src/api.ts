// Thin fetch wrapper that adds the bearer token and throws on non-2xx.

import { getToken, logout } from "./oauth";

export type Tenant = {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
};

export type Kind = {
  id: string;
  tenantId: string;
  name: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
};

export type Model = {
  id: string;
  tenantId: string;
  kindId: string;
  kindVersionId: string;
  slug: string;
  body: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
};

export type FieldType =
  | "text"
  | "number"
  | "integer"
  | "boolean"
  | "date"
  | "enum"
  | "url"
  | "tags";

export type SchemaField = {
  key: string;
  label?: string;
  type: FieldType;
  pinned?: boolean;
  values?: string[];
  min?: number;
  max?: number;
  unit?: string;
};

export type Schema = {
  version?: number;
  fields: SchemaField[];
};

async function call<T>(path: string): Promise<T> {
  const token = getToken();
  if (!token) throw new Error("not signed in");
  const resp = await fetch(path, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (resp.status === 401) {
    logout();
    throw new Error("session expired — please sign in again");
  }
  if (!resp.ok) {
    throw new Error(`${resp.status} ${resp.statusText}`);
  }
  return resp.json() as Promise<T>;
}

export const api = {
  tenant: () => call<Tenant>("/tenant"),
  kinds: () => call<Kind[]>("/kinds"),
  kind: (kindId: string) => call<Kind>(`/kinds/${encodeURIComponent(kindId)}`),
  models: (kindId: string) =>
    call<Model[]>(`/kinds/${encodeURIComponent(kindId)}/models`),
  schema: (kindId: string) =>
    call<Schema>(`/kinds/${encodeURIComponent(kindId)}/schema`),
};
