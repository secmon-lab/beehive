// Thin wrapper around openapi-fetch. The generated `paths` type lives
// in src/generated/api.d.ts (run `pnpm gen:api`). When tests run against
// an offline build the generated file may be absent — keep the import
// optional so vitest does not explode.

export async function fetchJSON<T = unknown>(url: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(url, init);
  if (!resp.ok) {
    const body = await resp.text();
    throw new ApiError(resp.status, body);
  }
  return (await resp.json()) as T;
}

export class ApiError extends Error {
  constructor(public status: number, public body: string) {
    super(`HTTP ${status}`);
  }
}
