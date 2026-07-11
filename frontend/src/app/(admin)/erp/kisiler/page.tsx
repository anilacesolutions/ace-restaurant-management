"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatTRY } from "@/lib/money";
import type { Party, PartiesResponse } from "@/lib/types";

export default function ErpKisilerPage() {
  const [parties, setParties] = useState<Party[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);

  async function load() {
    try {
      const r = await api<PartiesResponse>("/api/v1/parties");
      setParties(r.parties);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    load();
  }, []);

  const totalRemaining = (parties ?? []).reduce((s, p) => s + p.remaining, 0);

  async function remove(p: Party) {
    if (
      !confirm(
        `${p.name} kişisi silinsin mi? (geçmiş giderler etkilenmez, isim korunur)`,
      )
    )
      return;
    try {
      await api(`/api/v1/parties/${p.id}`, { method: "DELETE" });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <main className="flex flex-1 flex-col gap-5 p-4 pb-24">
      <header className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link
            href="/erp"
            className="flex h-10 w-10 items-center justify-center rounded-full border border-zinc-200 bg-white text-zinc-700 shadow-sm active:bg-zinc-50"
            aria-label="Geri"
          >
            ←
          </Link>
          <h1 className="text-2xl font-semibold text-zinc-900">
            Kişiler / Cari
          </h1>
        </div>
        <button
          onClick={() => setAdding(true)}
          className="rounded-full bg-amber-700 px-5 py-2.5 text-sm font-semibold text-white shadow-sm active:bg-amber-800"
        >
          + Yeni Kişi
        </button>
      </header>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {/* Toplam kalan borç */}
      <div className="flex items-baseline justify-between rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3">
        <span className="text-sm font-medium text-amber-900/80">
          Toplam kalan borç
        </span>
        <span
          className={`text-2xl font-bold tabular-nums ${
            totalRemaining > 0 ? "text-red-700" : "text-green-700"
          }`}
        >
          {formatTRY(totalRemaining)}
        </span>
      </div>

      {parties === null ? (
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      ) : parties.length === 0 ? (
        <p className="text-sm text-zinc-500">
          Henüz kişi yok. + Yeni Kişi ile ekleyin (Ahmet Bey, garson avansı...).
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {parties.map((p) => (
            <li
              key={p.id}
              className="flex items-center gap-3 rounded-2xl border border-zinc-200 bg-white p-4 shadow-sm"
            >
              <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-sky-100 text-base font-semibold text-sky-800">
                {p.name.slice(0, 1).toUpperCase()}
              </div>
              <div className="flex min-w-0 flex-1 flex-col">
                <span className="truncate text-base font-semibold text-zinc-900">
                  {p.name}
                </span>
                <span className="mt-0.5 text-xs text-zinc-500">
                  {p.expenseCount} kayıt · ödenen {formatTRY(p.paid)}
                </span>
              </div>
              <div className="flex shrink-0 flex-col items-end">
                <span
                  className={`text-base font-bold tabular-nums ${
                    p.remaining > 0 ? "text-red-700" : "text-green-700"
                  }`}
                >
                  {formatTRY(p.remaining)}
                </span>
                <span className="text-[10px] uppercase tracking-wide text-zinc-400">
                  {p.remaining > 0 ? "kalan borç" : "temiz"}
                </span>
              </div>
              <button
                onClick={() => remove(p)}
                className="ml-1 flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-zinc-400 active:bg-zinc-100 active:text-red-700"
                aria-label={`${p.name} sil`}
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}

      {adding && (
        <PartyFormSheet
          onClose={() => setAdding(false)}
          onSaved={async () => {
            setAdding(false);
            await load();
          }}
        />
      )}
    </main>
  );
}

function PartyFormSheet({
  onClose,
  onSaved,
}: {
  onClose: () => void;
  onSaved: () => void | Promise<void>;
}) {
  const [name, setName] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function save() {
    setError(null);
    if (!name.trim()) {
      setError("İsim gir");
      return;
    }
    setBusy(true);
    try {
      await api("/api/v1/parties", {
        method: "POST",
        body: JSON.stringify({ name, note }),
      });
      await onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-white">
      <header className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
        <button
          onClick={onClose}
          className="rounded-full px-3 py-2 text-sm font-medium text-zinc-600 active:bg-zinc-100"
        >
          Iptal
        </button>
        <h2 className="text-base font-semibold text-zinc-900">Yeni Kişi</h2>
        <button
          onClick={save}
          disabled={busy}
          className="rounded-full bg-amber-700 px-4 py-2 text-sm font-semibold text-white active:bg-amber-800 disabled:opacity-50"
        >
          {busy ? "..." : "Kaydet"}
        </button>
      </header>

      <div className="flex flex-1 flex-col gap-5 overflow-y-auto p-4 pb-28">
        {error && (
          <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
            {error}
          </div>
        )}

        <div className="flex flex-col">
          <span className="mb-1.5 text-sm font-medium text-zinc-700">
            Ad / Ünvan
          </span>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="orn. Ahmet Bey, Garson Mehmet"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </div>

        <div className="flex flex-col">
          <span className="mb-1.5 text-sm font-medium text-zinc-700">
            Not (opsiyonel)
          </span>
          <input
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="orn. manav, telefon no..."
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </div>
      </div>
    </div>
  );
}
