import Link from "next/link";
import { redirect } from "next/navigation";
import { serverApi } from "@/lib/api";
import { getServerMe } from "@/lib/auth";
import type { TablesResponse } from "@/lib/types";

export const dynamic = "force-dynamic";

export default async function GarsonHomePage() {
  const me = await getServerMe();
  if (!me) redirect("/garson/oturum-bitti");
  if (me.kind === "admin") {
    // Admin landing on /garson is unusual but harmless — bounce to /kasa.
    redirect("/kasa");
  }

  const { tables } = await serverApi<TablesResponse>("/api/v1/tables");

  return (
    <main className="flex flex-1 flex-col gap-6 p-4 pb-24">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-zinc-900">Masalar</h1>
        <span className="text-sm text-zinc-500">{me.waiter.name}</span>
      </header>

      {tables.length === 0 ? (
        <p className="text-sm text-zinc-500">
          Henuz masa tanimli degil.
        </p>
      ) : (
        <ul className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5">
          {tables.map((t) => (
            <li key={t.id}>
              <Link
                href={`/garson/masa/${t.number}`}
                className="flex aspect-square flex-col items-center justify-center rounded-2xl border border-zinc-200 bg-white text-center shadow-sm active:bg-zinc-50"
              >
                <span className="text-xs uppercase tracking-wide text-zinc-500">
                  Masa
                </span>
                <span className="text-3xl font-semibold tabular-nums text-zinc-900">
                  {t.number}
                </span>
                {t.label && (
                  <span className="mt-1 text-[10px] text-zinc-500">
                    {t.label}
                  </span>
                )}
              </Link>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
