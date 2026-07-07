"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import type { Table, TablesResponse } from "@/lib/types";

export default function ErpMasalarPage() {
  const [tables, setTables] = useState<Table[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    try {
      const r = await api<TablesResponse>("/api/v1/tables");
      setTables(r.tables);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function addTable() {
    setBusy(true);
    try {
      await api<Table>("/api/v1/tables", { method: "POST" });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function deleteTable(t: Table) {
    if (!confirm(`Masa ${t.number} silinsin mi?`)) return;
    setBusy(true);
    try {
      await api(`/api/v1/tables/${t.id}`, { method: "DELETE" });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="flex flex-1 flex-col gap-6 p-4 pb-24">
      <header className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link
            href="/erp"
            className="flex h-10 w-10 items-center justify-center rounded-full border border-zinc-200 bg-white text-zinc-700 shadow-sm active:bg-zinc-50"
            aria-label="Geri"
          >
            ←
          </Link>
          <h1 className="text-2xl font-semibold text-zinc-900">Masa Yonetimi</h1>
        </div>
        <button
          onClick={addTable}
          disabled={busy}
          className="rounded-full bg-amber-700 px-5 text-sm font-semibold text-white shadow-sm active:bg-amber-800 disabled:opacity-50"
        >
          + Yeni Masa
        </button>
      </header>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {tables === null ? (
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      ) : tables.length === 0 ? (
        <p className="text-sm text-zinc-500">Henuz masa yok. + Yeni Masa ile ekleyin.</p>
      ) : (
        <ul className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5">
          {tables.map((t) => (
            <li
              key={t.id}
              className="relative flex aspect-square flex-col items-center justify-center rounded-2xl border border-zinc-200 bg-white text-center shadow-sm"
            >
              <span className="text-xs uppercase tracking-wide text-zinc-500">
                Masa
              </span>
              <span className="text-3xl font-semibold tabular-nums text-zinc-900">
                {t.number}
              </span>
              <button
                onClick={() => deleteTable(t)}
                disabled={busy}
                className="absolute right-2 top-2 flex h-9 w-9 items-center justify-center rounded-full text-zinc-400 active:bg-zinc-100 active:text-red-700 disabled:opacity-30"
                aria-label={`Masa ${t.number} sil`}
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
