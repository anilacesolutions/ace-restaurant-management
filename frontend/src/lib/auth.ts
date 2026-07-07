import { ApiError, serverApi } from "./api";
import type { Me } from "./types";

// Server-side: resolve the current session by hitting /auth/me with the
// incoming request's cookies. Returns null on 401. Use in layouts/pages that
// need to gate by role.
export async function getServerMe(): Promise<Me | null> {
  try {
    return await serverApi<Me>("/api/v1/auth/me");
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) return null;
    throw e;
  }
}
