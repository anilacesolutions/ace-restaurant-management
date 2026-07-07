export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

// In the browser, derive the API host from the page's own host so a phone on
// the LAN hitting http://192.168.x.y:3000 talks to http://192.168.x.y:8080 —
// not its own localhost. NEXT_PUBLIC_API_BASE wins only when it points at a
// non-localhost host (e.g. a real prod URL).
function clientApiBase(): string {
  const configured = process.env.NEXT_PUBLIC_API_BASE;
  if (configured && !/^https?:\/\/(localhost|127\.0\.0\.1)/.test(configured)) {
    return configured;
  }
  return `${window.location.protocol}//${window.location.hostname}:8080`;
}

// Client-side fetch — the browser attaches our cookies via credentials.
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${clientApiBase()}${path}`, {
    credentials: "include",
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
  return handleResponse<T>(res);
}

// Server-side fetch — must run inside a Server Component / Route Handler.
// Reads the incoming request's cookies via next/headers and forwards them
// so the backend sees the same session as the browser.
export async function serverApi<T>(path: string, init: RequestInit = {}): Promise<T> {
  const { cookies } = await import("next/headers");
  const store = await cookies();
  const cookieHeader = store
    .getAll()
    .map((c) => `${c.name}=${c.value}`)
    .join("; ");
  const serverBase = process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080";
  const res = await fetch(`${serverBase}${path}`, {
    cache: "no-store",
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(cookieHeader ? { Cookie: cookieHeader } : {}),
      ...(init.headers ?? {}),
    },
  });
  return handleResponse<T>(res);
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let msg = res.statusText;
    try {
      const body = (await res.json()) as { error?: string };
      if (body?.error) msg = body.error;
    } catch {
      // body wasn't JSON; keep statusText
    }
    throw new ApiError(res.status, msg);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}
