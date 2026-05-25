// Thin wrappers around fetch for the in-app REST surface. The OpenAPI
// types live in src/generated/api.d.ts (run `pnpm gen:api`).

export async function fetchJSON<T = unknown>(url: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(url, init);
  if (!resp.ok) {
    const body = await resp.text();
    throw new ApiError(resp.status, body);
  }
  return (await resp.json()) as T;
}

// mutateJSON sends a JSON-bodied request (POST / PUT / DELETE) and
// parses the JSON response. 204 No Content is allowed and returns
// `undefined`.
export async function mutateJSON<T = unknown>(
  url: string,
  init: RequestInit & { json?: unknown } = {},
): Promise<T | undefined> {
  const { json, headers, ...rest } = init;
  const resp = await fetch(url, {
    method: "POST",
    ...rest,
    headers: {
      ...(json !== undefined ? { "Content-Type": "application/json" } : {}),
      ...headers,
    },
    body: json !== undefined ? JSON.stringify(json) : rest.body,
  });
  if (!resp.ok) {
    const body = await resp.text();
    throw new ApiError(resp.status, body);
  }
  if (resp.status === 204) return undefined;
  const text = await resp.text();
  if (!text) return undefined;
  return JSON.parse(text) as T;
}

// ApiError carries both the HTTP status and the response body so the
// UI can surface server-side problem details (e.g. "catalog not
// initialised"). `message` is `HTTP <status>: <best-effort summary>`.
export class ApiError extends Error {
  constructor(public status: number, public body: string) {
    super(`HTTP ${status}${summariseBody(body) ? `: ${summariseBody(body)}` : ""}`);
  }
}

function summariseBody(body: string): string {
  if (!body) return "";
  try {
    const parsed = JSON.parse(body) as { message?: string; code?: string };
    if (parsed?.message) return parsed.message;
    if (parsed?.code) return parsed.code;
  } catch {
    /* not JSON — fall through */
  }
  return body.length > 160 ? body.slice(0, 160) + "…" : body;
}
