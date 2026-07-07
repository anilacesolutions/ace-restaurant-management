import { redirect } from "next/navigation";
import { getServerMe } from "@/lib/auth";

// The root route picks where to send the visitor based on session:
//   - admin → /kasa (main cashier dashboard)
//   - waiter → /garson
//   - no session → /login
// Hitting / is the canonical entrypoint after auth changes.
export default async function RootPage() {
  const me = await getServerMe();
  if (!me) redirect("/login");
  if (me.kind === "admin") redirect("/kasa");
  redirect("/garson");
}
