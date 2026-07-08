"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { formatTRY } from "@/lib/money";
import type {
  ActiveOrdersResponse,
  Me,
  Order,
  Table,
  TablesResponse,
} from "@/lib/types";

// Refresh cadence so an order placed on one table shows as "open" here quickly,
// even without a manual reload.
const POLL_MS = 4000;

export default function GarsonHomePage() {
  const router = useRouter();
  const [tables, setTables] = useState<Table[] | null>(null);
  const [orders, setOrders] = useState<Order[]>([]);
  const [waiterName, setWaiterName] = useState("");
  const [error, setError] = useState<string | null>(null);

  async function load() {
    try {
      const [me, t, o] = await Promise.all([
        api<Me>("/api/v1/auth/me"),
        api<TablesResponse>("/api/v1/tables"),
        api<ActiveOrdersResponse>("/api/v1/orders/active"),
      ]);
      if (me.kind === "admin") {
        router.replace("/kasa");
        return;
      }
      setWaiterName(me.waiter.name);
      setTables(t.tables);
      setOrders(o.orders);
      setError(null);
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) {
        router.replace("/garson/oturum-bitti");
        return;
      }
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  // Load on mount, poll, and refresh when returning to the tab / navigating
  // back from a table — so open tables appear without a manual reload.
  useEffect(() => {
    load();
    const id = setInterval(load, POLL_MS);
    const onVisible = () => {
      if (document.visibilityState === "visible") load();
    };
    document.addEventListener("visibilitychange", onVisible);
    window.addEventListener("focus", load);
    return () => {
      clearInterval(id);
      document.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("focus", load);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const orderByTable = useMemo(() => {
    const m = new Map<number, Order>();
    for (const o of orders) m.set(o.tableNumber, o);
    return m;
  }, [orders]);

  return (
    <main className="flex flex-1 flex-col gap-6 p-4 pb-24">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-zinc-900">Masalar</h1>
        {waiterName && (
          <span className="text-sm text-zinc-500">{waiterName}</span>
        )}
      </header>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {tables === null ? (
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      ) : tables.length === 0 ? (
        <p className="text-sm text-zinc-500">Henuz masa tanimli degil.</p>
      ) : (
        <ul className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5">
          {tables.map((t) => {
            const order = orderByTable.get(t.number);
            const open = !!order;
            return (
              <li key={t.id}>
                <Link
                  href={`/garson/masa/${t.number}`}
                  className={`flex aspect-square flex-col items-center justify-center rounded-2xl border text-center shadow-sm active:opacity-90 ${
                    open
                      ? "border-amber-300 bg-amber-50"
                      : "border-zinc-200 bg-white active:bg-zinc-50"
                  }`}
                >
                  <span className="text-xs uppercase tracking-wide text-zinc-500">
                    Masa
                  </span>
                  <span className="text-3xl font-semibold tabular-nums text-zinc-900">
                    {t.number}
                  </span>
                  {open ? (
                    <span className="mt-1 text-sm font-bold tabular-nums text-amber-800">
                      {formatTRY(order!.grandTotal)}
                    </span>
                  ) : (
                    <span className="mt-1 text-[10px] uppercase tracking-wide text-zinc-400">
                      Boş
                    </span>
                  )}
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </main>
  );
}
