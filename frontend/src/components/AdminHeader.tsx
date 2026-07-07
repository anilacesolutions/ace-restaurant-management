"use client";

import Link from "next/link";
import { useRouter, usePathname } from "next/navigation";
import { api } from "@/lib/api";
import type { User } from "@/lib/types";

const tabs = [
  { href: "/kasa", label: "Kasa" },
  { href: "/erp", label: "Yonetim" },
] as const;

export function AdminHeader({
  user,
  onIssueQR,
}: {
  user: User;
  onIssueQR: () => void;
}) {
  const router = useRouter();
  const pathname = usePathname();

  async function logout() {
    try {
      await api("/api/v1/auth/logout", { method: "POST" });
    } catch {
      // ignore — we'll route to /login regardless
    }
    router.replace("/login");
    router.refresh();
  }

  return (
    <header className="sticky top-0 z-20 flex flex-wrap items-center gap-3 border-b border-zinc-200 bg-white/95 px-4 py-3 backdrop-blur">
      <nav className="flex gap-1">
        {tabs.map((t) => {
          const active = pathname === t.href || pathname.startsWith(t.href + "/");
          return (
            <Link
              key={t.href}
              href={t.href}
              className={
                "rounded-full px-4 py-2 text-sm font-semibold transition-colors " +
                (active
                  ? "bg-zinc-900 text-white"
                  : "border border-zinc-200 bg-white text-zinc-700 active:bg-zinc-50")
              }
            >
              {t.label}
            </Link>
          );
        })}
      </nav>

      <div className="ml-auto flex items-center gap-2">
        <button
          onClick={onIssueQR}
          className="rounded-full bg-amber-700 px-4 py-2 text-sm font-semibold text-white shadow-sm active:bg-amber-800"
        >
          Garson QR
        </button>
        <span className="hidden text-sm text-zinc-500 sm:inline">
          {user.username}
        </span>
        <button
          onClick={logout}
          className="rounded-full border border-zinc-200 bg-white px-4 py-2 text-sm font-medium text-zinc-700 active:bg-zinc-50"
        >
          Cikis
        </button>
      </div>
    </header>
  );
}
