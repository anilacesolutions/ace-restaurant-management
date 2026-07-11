import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// Next 16 renamed Middleware to Proxy (same functionality).
//
// qr.gunguzelbahce.online serves ONLY the public QR menu. Rewrite every request
// on that host to /menu, so the QR subdomain never exposes the admin/waiter app
// (which lives on kasa.gunguzelbahce.online). Other hosts pass through.
export function proxy(request: NextRequest) {
  const host = (request.headers.get("host") ?? "").split(":")[0];
  if (host.startsWith("qr.") && request.nextUrl.pathname !== "/menu") {
    const url = request.nextUrl.clone();
    url.pathname = "/menu";
    return NextResponse.rewrite(url);
  }
  return NextResponse.next();
}

export const config = {
  // Run on everything except Next internals and static files.
  matcher: ["/((?!_next/static|_next/image|favicon.ico|.*\\..*).*)"],
};
