"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatTRY } from "@/lib/money";
import type {
  ActiveOrdersResponse,
  Order,
  Table,
  TablesResponse,
} from "@/lib/types";

const POLL_MS = 5000;

export default function KasaPage() {
  const [tables, setTables] = useState<Table[] | null>(null);
  const [orders, setOrders] = useState<Order[]>([]);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    try {
      const [t, o] = await Promise.all([
        api<TablesResponse>("/api/v1/tables"),
        api<ActiveOrdersResponse>("/api/v1/orders/active"),
      ]);
      setTables(t.tables);
      setOrders(o.orders);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  // Poll so a waiter's new order surfaces on the cashier within a few seconds.
  useEffect(() => {
    load();
    const id = setInterval(load, POLL_MS);
    return () => clearInterval(id);
  }, []);

  const orderByTable = useMemo(() => {
    const m = new Map<number, Order>();
    for (const o of orders) m.set(o.tableNumber, o);
    return m;
  }, [orders]);

  const openCount = orders.length;
  const openTotal = useMemo(
    () => orders.reduce((s, o) => s + o.grandTotal, 0),
    [orders],
  );

  return (
    <main className="flex flex-1 flex-col gap-5 p-4 pb-24">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-zinc-900">Masalar</h1>
      </header>

      {/* Kasa toplamı — sum of all open (unpaid) tables */}
      <div className="flex items-center justify-between rounded-2xl border border-amber-200 bg-amber-50 px-5 py-4">
        <div className="flex flex-col">
          <span className="text-sm font-medium text-amber-900">
            Kasa Toplamı
          </span>
          <span className="text-xs text-amber-800/70">
            {openCount} açık masa
          </span>
        </div>
        <span className="text-3xl font-bold tabular-nums text-amber-900">
          {formatTRY(openTotal)}
        </span>
      </div>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {tables === null ? (
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      ) : tables.length === 0 ? (
        <p className="text-sm text-zinc-500">
          Henuz masa tanimli degil. ERP → Masalar bolumunden ekleyin.
        </p>
      ) : (
        <ul className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5">
          {tables.map((t) => {
            const order = orderByTable.get(t.number);
            const open = !!order;
            const itemCount = order
              ? order.items
                  .filter((it) => it.status !== "voided" && it.status !== "refunded")
                  .reduce((s, it) => s + it.qty, 0)
              : 0;
            return (
              <li key={t.id}>
                <Link
                  href={`/kasa/masa/${t.number}`}
                  className={`flex aspect-square flex-col items-center justify-center rounded-2xl border text-center shadow-sm active:opacity-90 ${
                    open
                      ? "border-amber-300 bg-amber-50"
                      : "border-zinc-200 bg-white active:bg-zinc-50"
                  }`}
                >
                  <span className="text-[10px] uppercase tracking-wide text-zinc-500">
                    Masa
                  </span>
                  <span className="text-3xl font-semibold tabular-nums text-zinc-900">
                    {t.number}
                  </span>
                  {open ? (
                    <>
                      <span className="mt-1 text-base font-bold tabular-nums text-amber-800">
                        {formatTRY(order!.grandTotal)}
                      </span>
                      <span className="text-[10px] text-amber-700">
                        {itemCount} ürün
                      </span>
                    </>
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
