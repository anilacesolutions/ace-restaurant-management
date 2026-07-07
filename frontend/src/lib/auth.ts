import { ApiError, serverApi } from "./api";
import type { Me } from "./types";

// Server-side: resolve the current session by hitting /auth/me with the
// incoming request's cookies. Returns null on 401. Use in layouts/pages that
// need to gate by role.
export async function getServerMe(): Promise<Me | null> {
  try {
    return await serverApi<Me>("/api/v1/auth/me");
  } catch (e) {
    // 401 = not logged in. Any other failure (backend down / unreachable,
    // network/DNS error) must NOT crash the SSR page — degrade to "logged
    // out" so the visitor lands on /login instead of a 500.
    if (!(e instanceof ApiError && e.status === 401)) {
      console.error("getServerMe failed:", e);
    }
    return null;
  }
}
