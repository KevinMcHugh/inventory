// Thin fetch wrapper that adds the bearer token and throws on non-2xx.

import { getToken as _getToken, logout as _logout } from "./oauth";

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
  model: (kindId: string, slug: string) =>
    call<Model>(
      `/kinds/${encodeURIComponent(kindId)}/models/${encodeURIComponent(slug)}`,
    ),
  schema: (kindId: string) =>
    call<Schema>(`/kinds/${encodeURIComponent(kindId)}/schema`),
  createModel: (
    kindId: string,
    body: { slug: string; body: Record<string, unknown>; kindVersionId?: string },
  ) =>
    write<Model>(`/kinds/${encodeURIComponent(kindId)}/models`, "POST", body),
  updateModel: (
    kindId: string,
    slug: string,
    body: { body: Record<string, unknown>; kindVersionId?: string },
  ) =>
    write<Model>(
      `/kinds/${encodeURIComponent(kindId)}/models/${encodeURIComponent(slug)}`,
      "PUT",
      body,
    ),
  deleteModel: (kindId: string, slug: string) =>
    write<void>(
      `/kinds/${encodeURIComponent(kindId)}/models/${encodeURIComponent(slug)}`,
      "DELETE",
    ),
  createKindVersion: (kindId: string, schema: unknown) =>
    write<{ id: string; kindId: string; schema: unknown }>(
      `/kinds/${encodeURIComponent(kindId)}/versions`,
      "POST",
      { schema },
    ),
};

// FieldError: the shape returned in a 400 body.fields when a write is
// rejected by schema validation.
export type FieldError = { field: string; message: string };

export class ApiError extends Error {
  status: number;
  fields?: FieldError[];
  constructor(status: number, message: string, fields?: FieldError[]) {
    super(message);
    this.status = status;
    this.fields = fields;
  }
}

async function write<T>(
  path: string,
  method: "POST" | "PUT" | "DELETE",
  body?: unknown,
): Promise<T> {
  const token = getToken();
  if (!token) throw new Error("not signed in");
  const resp = await fetch(path, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (resp.status === 401) {
    logout();
    throw new ApiError(401, "session expired — please sign in again");
  }
  if (resp.status === 204) {
    return undefined as T;
  }
  if (!resp.ok) {
    let msg = `${resp.status} ${resp.statusText}`;
    let fields: FieldError[] | undefined;
    try {
      const data = await resp.json();
      if (data.message) msg = data.message;
      if (data.fields) fields = data.fields;
    } catch {
      // non-JSON body, keep the status text
    }
    throw new ApiError(resp.status, msg, fields);
  }
  return resp.json() as Promise<T>;
}

function getToken(): string | null {
  return _getToken();
}
function logout(): void {
  _logout();
}
