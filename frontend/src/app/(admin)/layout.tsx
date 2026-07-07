import { redirect } from "next/navigation";
import { getServerMe } from "@/lib/auth";
import { AdminShell } from "./AdminShell";

export default async function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const me = await getServerMe();
  if (!me) redirect("/login");
  if (me.kind !== "admin") redirect("/garson");

  return <AdminShell user={me.user}>{children}</AdminShell>;
}
