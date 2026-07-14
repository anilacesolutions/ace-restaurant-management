"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatTRY, fromTL, toTL } from "@/lib/money";
import type { Payment, Receivable, ReceivablesResponse } from "@/lib/types";

const TZ = "Europe/Istanbul";

function todayISO(): string {
  return new Date().toLocaleDateString("en-CA", { timeZone: TZ });
}
function fmtDay(iso: string): string {
  return new Date(iso).toLocaleDateString("tr-TR", {
    timeZone: TZ,
    day: "2-digit",
    month: "long",
  });
}

const collectionsOf = (r: Receivable): Payment[] => r.payments ?? [];
const collectedOf = (r: Receivable) =>
  collectionsOf(r).reduce((s, p) => s + p.amount, 0);
const remainingOf = (r: Receivable) => r.amount - collectedOf(r);

export default function ErpAlacaklarPage() {
  const [items, setItems] = useState<Receivable[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [adding, setAdding] = useState(false);
  const [selected, setSelected] = useState<Receivable | null>(null);

  async function load() {
    try {
      const r = await api<ReceivablesResponse>("/api/v1/receivables");
      setItems(r.receivables);
      setSelected((cur) =>
        cur ? (r.receivables.find((x) => x.id === cur.id) ?? null) : cur,
      );
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    load();
  }, []);

  const totals = useMemo(() => {
    const list = items ?? [];
    const total = list.reduce((s, r) => s + r.amount, 0);
    const collected = list.reduce((s, r) => s + collectedOf(r), 0);
    return { total, collected, remaining: total - collected };
  }, [items]);

  async function deleteReceivable(r: Receivable) {
    if (
      !confirm(
        `${r.personName} — ${formatTRY(r.amount)} alacağı silinsin mi?`,
      )
    )
      return;
    setBusy(true);
    try {
      await api(`/api/v1/receivables/${r.id}`, { method: "DELETE" });
      setSelected(null);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
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
          <h1 className="text-2xl font-semibold text-zinc-900">Alacaklar</h1>
        </div>
        <button
          onClick={() => setAdding(true)}
          className="rounded-full bg-amber-700 px-5 py-2.5 text-sm font-semibold text-white shadow-sm active:bg-amber-800"
        >
          + Yeni Alacak
        </button>
      </header>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {/* Toplam kalan alacak */}
      <div className="rounded-2xl border border-amber-200 bg-amber-50 p-4">
        <div className="grid grid-cols-3 gap-2 text-center">
          <Cell label="Toplam alacak" value={totals.total} tone="zinc" />
          <Cell label="Tahsil edilen" value={totals.collected} tone="green" />
          <Cell
            label="Kalan alacak"
            value={totals.remaining}
            tone={totals.remaining > 0 ? "red" : "green"}
          />
        </div>
      </div>

      {/* List */}
      {items === null ? (
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      ) : items.length === 0 ? (
        <p className="text-sm text-zinc-500">
          Alacak yok. + Yeni Alacak ile ekleyin (örn. Ahmet Bey&apos;den
          4.000₺).
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {items.map((r) => (
            <li key={r.id}>
              <button
                onClick={() => setSelected(r)}
                className="flex w-full items-center gap-3 rounded-2xl border border-zinc-200 bg-white p-3 text-left shadow-sm active:bg-zinc-50"
              >
                <div className="flex w-14 shrink-0 flex-col items-center">
                  <span className="text-lg font-bold tabular-nums leading-none text-zinc-900">
                    {new Date(r.issuedAt).toLocaleDateString("tr-TR", {
                      timeZone: TZ,
                      day: "2-digit",
                    })}
                  </span>
                  <span className="text-[10px] uppercase text-zinc-400">
                    {new Date(r.issuedAt).toLocaleDateString("tr-TR", {
                      timeZone: TZ,
                      month: "short",
                    })}
                  </span>
                </div>
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-base font-semibold text-zinc-900">
                    {r.personName}
                  </span>
                  <span className="mt-0.5">
                    <StatusBadge receivable={r} />
                  </span>
                </div>
                <span className="shrink-0 text-base font-semibold tabular-nums text-zinc-900">
                  {formatTRY(r.amount)}
                </span>
                <span className="shrink-0 text-zinc-300">›</span>
              </button>
            </li>
          ))}
        </ul>
      )}

      {adding && (
        <ReceivableFormSheet
          onClose={() => setAdding(false)}
          onSaved={async () => {
            setAdding(false);
            await load();
          }}
        />
      )}

      {selected && (
        <ReceivableDetailSheet
          receivable={selected}
          busy={busy}
          onClose={() => setSelected(null)}
          onChanged={load}
          onDelete={() => deleteReceivable(selected)}
        />
      )}
    </main>
  );
}

function Cell({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: "zinc" | "green" | "red";
}) {
  const color =
    tone === "green"
      ? "text-green-700"
      : tone === "red"
        ? "text-red-700"
        : "text-zinc-900";
  return (
    <div className="flex flex-col">
      <span className="text-xs font-medium text-amber-900/70">{label}</span>
      <span className={`text-lg font-bold tabular-nums ${color}`}>
        {formatTRY(value)}
      </span>
    </div>
  );
}

function StatusBadge({ receivable }: { receivable: Receivable }) {
  const remaining = remainingOf(receivable);
  const collected = collectedOf(receivable);
  if (remaining <= 0) {
    return (
      <span className="rounded-full bg-green-100 px-2 py-0.5 text-[11px] font-medium text-green-800">
        Tahsil edildi
      </span>
    );
  }
  if (collected > 0) {
    return (
      <span className="rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-800">
        Kalan {formatTRY(remaining)}
      </span>
    );
  }
  return (
    <span className="rounded-full bg-red-100 px-2 py-0.5 text-[11px] font-medium text-red-800">
      Bekliyor
    </span>
  );
}

// ---- Detail sheet (collections) ------------------------------------------

function ReceivableDetailSheet({
  receivable,
  busy,
  onClose,
  onChanged,
  onDelete,
}: {
  receivable: Receivable;
  busy: boolean;
  onClose: () => void;
  onChanged: () => void | Promise<void>;
  onDelete: () => void;
}) {
  const remaining = remainingOf(receivable);
  const collected = collectedOf(receivable);
  const collections = collectionsOf(receivable);

  const [showForm, setShowForm] = useState(false);
  const [amountTL, setAmountTL] = useState("");
  const [paidAt, setPaidAt] = useState(todayISO);
  const [localBusy, setLocalBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function addCollection(amountKurus: number) {
    setError(null);
    if (amountKurus <= 0) {
      setError("Gecerli bir tutar gir");
      return;
    }
    setLocalBusy(true);
    try {
      await api(`/api/v1/receivables/${receivable.id}/payments`, {
        method: "POST",
        body: JSON.stringify({ amount: amountKurus, paidAt }),
      });
      setShowForm(false);
      setAmountTL("");
      await onChanged();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLocalBusy(false);
    }
  }

  async function deleteCollection(p: Payment) {
    if (!confirm(`${formatTRY(p.amount)} tahsilatı silinsin mi?`)) return;
    setLocalBusy(true);
    try {
      await api(`/api/v1/receivables/${receivable.id}/payments/${p.id}`, {
        method: "DELETE",
      });
      await onChanged();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLocalBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-white">
      <header className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
        <button
          onClick={onClose}
          className="rounded-full px-3 py-2 text-sm font-medium text-zinc-600 active:bg-zinc-100"
        >
          Kapat
        </button>
        <h2 className="truncate px-2 text-base font-semibold text-zinc-900">
          {receivable.personName}
        </h2>
        <button
          onClick={onDelete}
          disabled={busy || localBusy}
          className="rounded-full px-3 py-2 text-sm font-medium text-red-700 active:bg-red-50 disabled:opacity-40"
        >
          Sil
        </button>
      </header>

      <div className="flex flex-1 flex-col gap-5 overflow-y-auto p-4 pb-28">
        <div className="rounded-2xl border border-zinc-200 bg-white p-4 shadow-sm">
          <div className="flex items-baseline justify-between">
            <span className="text-sm text-zinc-500">Alacak</span>
            <span className="text-lg font-semibold tabular-nums text-zinc-900">
              {formatTRY(receivable.amount)}
            </span>
          </div>
          <div className="mt-1 flex items-baseline justify-between">
            <span className="text-sm text-zinc-500">Tahsil edilen</span>
            <span className="tabular-nums text-green-700">
              {formatTRY(collected)}
            </span>
          </div>
          <div className="mt-2 flex items-baseline justify-between border-t border-zinc-100 pt-2">
            <span className="text-sm font-medium text-zinc-700">Kalan</span>
            <span
              className={`text-2xl font-bold tabular-nums ${
                remaining > 0 ? "text-red-700" : "text-green-700"
              }`}
            >
              {formatTRY(remaining)}
            </span>
          </div>
          <p className="mt-2 text-xs text-zinc-400">
            {fmtDay(receivable.issuedAt)}
            {receivable.note ? ` · ${receivable.note}` : ""}
          </p>
        </div>

        {error && (
          <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
            {error}
          </div>
        )}

        {/* Tahsilatlar */}
        <div className="flex flex-col gap-2">
          <span className="text-sm font-medium text-zinc-700">Tahsilatlar</span>
          {collections.length === 0 ? (
            <p className="text-sm text-zinc-400">Henüz tahsilat yok.</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {collections.map((p) => (
                <li
                  key={p.id}
                  className="flex items-center gap-3 rounded-xl border border-zinc-200 bg-white p-3 shadow-sm"
                >
                  <span className="flex-1 text-sm text-zinc-600">
                    {fmtDay(p.paidAt)}
                  </span>
                  <span className="tabular-nums font-semibold text-green-700">
                    {formatTRY(p.amount)}
                  </span>
                  <button
                    onClick={() => deleteCollection(p)}
                    disabled={localBusy}
                    className="flex h-8 w-8 items-center justify-center rounded-full text-zinc-400 active:bg-zinc-100 active:text-red-700 disabled:opacity-30"
                    aria-label="Tahsilat sil"
                  >
                    ×
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Tahsilat ekle */}
        {remaining > 0 &&
          (showForm ? (
            <div className="flex flex-col gap-3 rounded-2xl border border-zinc-200 bg-zinc-50 p-4">
              <div className="flex flex-col">
                <span className="mb-1.5 text-sm font-medium text-zinc-700">
                  Tahsilat tutarı (TL)
                </span>
                <input
                  value={amountTL}
                  onChange={(e) => setAmountTL(e.target.value)}
                  inputMode="decimal"
                  placeholder={`kalan: ${toTL(remaining)}`}
                  className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base tabular-nums text-zinc-900 shadow-sm outline-none focus:border-amber-500"
                />
              </div>
              <div className="flex flex-col">
                <span className="mb-1.5 text-sm font-medium text-zinc-700">
                  Tahsilat tarihi
                </span>
                <input
                  type="date"
                  value={paidAt}
                  onChange={(e) => setPaidAt(e.target.value)}
                  className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
                />
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => setShowForm(false)}
                  className="flex-1 rounded-xl bg-zinc-200 py-3 text-sm font-semibold text-zinc-700 active:bg-zinc-300"
                >
                  Vazgeç
                </button>
                <button
                  onClick={() => {
                    const tl = Number(amountTL.replace(",", "."));
                    addCollection(
                      Number.isFinite(tl) && tl > 0 ? fromTL(tl) : 0,
                    );
                  }}
                  disabled={localBusy}
                  className="flex-1 rounded-xl bg-amber-700 py-3 text-sm font-semibold text-white active:bg-amber-800 disabled:opacity-50"
                >
                  Kaydet
                </button>
              </div>
            </div>
          ) : (
            <div className="flex gap-2">
              <button
                onClick={() => setShowForm(true)}
                disabled={localBusy}
                className="flex-1 rounded-xl border border-amber-700 py-3 text-sm font-semibold text-amber-800 active:bg-amber-50 disabled:opacity-50"
              >
                + Tahsilat Ekle
              </button>
              <button
                onClick={() => addCollection(remaining)}
                disabled={localBusy}
                className="flex-1 rounded-xl bg-amber-700 py-3 text-sm font-semibold text-white active:bg-amber-800 disabled:opacity-50"
              >
                Tümünü Tahsil Et
              </button>
            </div>
          ))}
      </div>
    </div>
  );
}

// ---- New receivable sheet ------------------------------------------------

function ReceivableFormSheet({
  onClose,
  onSaved,
}: {
  onClose: () => void;
  onSaved: () => void | Promise<void>;
}) {
  const [personName, setPersonName] = useState("");
  const [amountTL, setAmountTL] = useState("");
  const [note, setNote] = useState("");
  const [issuedAt, setIssuedAt] = useState(todayISO);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const tl = Number(amountTL.replace(",", "."));
  const amountOk = Number.isFinite(tl) && tl > 0;
  const canSave = !!personName.trim() && amountOk && !!issuedAt;

  async function save() {
    setError(null);
    if (!personName.trim()) {
      setError("Kim borçlu, isim gir");
      return;
    }
    if (!amountOk) {
      setError("Gecerli bir tutar gir");
      return;
    }
    setBusy(true);
    try {
      await api("/api/v1/receivables", {
        method: "POST",
        body: JSON.stringify({
          personName,
          amount: fromTL(tl),
          note,
          issuedAt,
        }),
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
        <h2 className="text-base font-semibold text-zinc-900">Yeni Alacak</h2>
        <button
          onClick={save}
          disabled={busy || !canSave}
          className="rounded-full bg-amber-700 px-4 py-2 text-sm font-semibold text-white active:bg-amber-800 disabled:opacity-40"
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

        <Field label="Tarih">
          <input
            type="date"
            value={issuedAt}
            onChange={(e) => setIssuedAt(e.target.value)}
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        <Field label="Kim borçlu?">
          <input
            value={personName}
            onChange={(e) => setPersonName(e.target.value)}
            placeholder="orn. Ahmet Bey"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        <Field label="Alacak tutarı (TL)">
          <input
            value={amountTL}
            onChange={(e) => setAmountTL(e.target.value)}
            inputMode="decimal"
            placeholder="orn. 4000"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base tabular-nums text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        <Field label="Not (opsiyonel)">
          <input
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="orn. masa hesabı, sonra ödeyecek"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>
      </div>
    </div>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col">
      <span className="mb-1.5 text-sm font-medium text-zinc-700">{label}</span>
      {children}
    </div>
  );
}
