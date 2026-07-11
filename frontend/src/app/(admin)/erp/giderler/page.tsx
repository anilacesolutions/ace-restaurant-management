"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatTRY, fromTL, toTL } from "@/lib/money";
import type {
  Expense,
  Party,
  PartiesResponse,
  Payment,
  ExpensesResponse,
} from "@/lib/types";

const CATEGORIES = [
  "Sebze-Meyve",
  "Et/Tavuk/Balık",
  "İçecek",
  "Bakliyat/Kuru Gıda",
  "Süt Ürünleri",
  "Temizlik",
  "Kira",
  "Personel",
  "Fatura",
  "Diğer",
] as const;

const TZ = "Europe/Istanbul";
const pad = (n: number) => String(n).padStart(2, "0");

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

const paymentsOf = (e: Expense): Payment[] => e.payments ?? [];
const paidOf = (e: Expense) => paymentsOf(e).reduce((s, p) => s + p.amount, 0);
const remainingOf = (e: Expense) => e.amount - paidOf(e);

export default function ErpGiderlerPage() {
  const [ym, setYm] = useState(() => {
    const [y, m] = todayISO().split("-").map(Number);
    return { y, m };
  });
  const [expenses, setExpenses] = useState<Expense[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [adding, setAdding] = useState(false);
  const [selected, setSelected] = useState<Expense | null>(null);

  const range = useMemo(() => {
    const from = `${ym.y}-${pad(ym.m)}-01`;
    const next = ym.m === 12 ? { y: ym.y + 1, m: 1 } : { y: ym.y, m: ym.m + 1 };
    const to = `${next.y}-${pad(next.m)}-01`;
    return { from, to };
  }, [ym]);

  const monthLabel = useMemo(
    () =>
      new Date(ym.y, ym.m - 1, 1).toLocaleDateString("tr-TR", {
        month: "long",
        year: "numeric",
      }),
    [ym],
  );

  async function load() {
    try {
      const r = await api<ExpensesResponse>(
        `/api/v1/expenses?from=${range.from}&to=${range.to}`,
      );
      setExpenses(r.expenses);
      // keep the open detail sheet in sync with fresh data
      setSelected((cur) =>
        cur ? (r.expenses.find((e) => e.id === cur.id) ?? null) : cur,
      );
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    setExpenses(null);
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [range.from, range.to]);

  const totals = useMemo(() => {
    const list = expenses ?? [];
    const total = list.reduce((s, e) => s + e.amount, 0);
    const paid = list.reduce((s, e) => s + paidOf(e), 0);
    return { total, paid, remaining: total - paid };
  }, [expenses]);

  const byCategory = useMemo(() => {
    const m = new Map<string, number>();
    for (const e of expenses ?? []) {
      m.set(e.category, (m.get(e.category) ?? 0) + e.amount);
    }
    return [...m.entries()].sort((a, b) => b[1] - a[1]);
  }, [expenses]);

  function shiftMonth(delta: number) {
    setYm((p) => {
      const idx = p.y * 12 + (p.m - 1) + delta;
      return { y: Math.floor(idx / 12), m: (idx % 12) + 1 };
    });
  }

  async function deleteExpense(e: Expense) {
    if (!confirm(`${e.category} — ${formatTRY(e.amount)} gideri silinsin mi?`))
      return;
    setBusy(true);
    try {
      await api(`/api/v1/expenses/${e.id}`, { method: "DELETE" });
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
          <h1 className="text-2xl font-semibold text-zinc-900">Giderler</h1>
        </div>
        <button
          onClick={() => setAdding(true)}
          className="rounded-full bg-amber-700 px-5 py-2.5 text-sm font-semibold text-white shadow-sm active:bg-amber-800"
        >
          + Yeni Gider
        </button>
      </header>

      {/* Month navigation */}
      <div className="flex items-center justify-between rounded-2xl border border-zinc-200 bg-white p-2 shadow-sm">
        <button
          onClick={() => shiftMonth(-1)}
          className="flex h-11 w-11 items-center justify-center rounded-xl text-xl text-zinc-600 active:bg-zinc-100"
          aria-label="Önceki ay"
        >
          ‹
        </button>
        <span className="text-base font-semibold capitalize text-zinc-900">
          {monthLabel}
        </span>
        <button
          onClick={() => shiftMonth(1)}
          className="flex h-11 w-11 items-center justify-center rounded-xl text-xl text-zinc-600 active:bg-zinc-100"
          aria-label="Sonraki ay"
        >
          ›
        </button>
      </div>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {/* Summary */}
      <div className="rounded-2xl border border-amber-200 bg-amber-50 p-4">
        <div className="grid grid-cols-3 gap-2 text-center">
          <SummaryCell label="Toplam gider" value={totals.total} tone="zinc" />
          <SummaryCell label="Ödenen" value={totals.paid} tone="green" />
          <SummaryCell
            label="Kalan borç"
            value={totals.remaining}
            tone={totals.remaining > 0 ? "red" : "green"}
          />
        </div>
        {byCategory.length > 0 && (
          <ul className="mt-4 flex flex-col gap-1.5 border-t border-amber-200 pt-3">
            {byCategory.map(([cat, amt]) => (
              <li
                key={cat}
                className="flex items-center justify-between text-sm text-amber-900/80"
              >
                <span>{cat}</span>
                <span className="tabular-nums">{formatTRY(amt)}</span>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* List */}
      {expenses === null ? (
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      ) : expenses.length === 0 ? (
        <p className="text-sm text-zinc-500">
          Bu ay gider yok. + Yeni Gider ile ekleyin.
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {expenses.map((e) => (
            <li key={e.id}>
              <button
                onClick={() => setSelected(e)}
                className="flex w-full items-center gap-3 rounded-2xl border border-zinc-200 bg-white p-3 text-left shadow-sm active:bg-zinc-50"
              >
                <div className="flex w-14 shrink-0 flex-col items-center">
                  <span className="text-lg font-bold tabular-nums leading-none text-zinc-900">
                    {new Date(e.spentAt).toLocaleDateString("tr-TR", {
                      timeZone: TZ,
                      day: "2-digit",
                    })}
                  </span>
                  <span className="text-[10px] uppercase text-zinc-400">
                    {new Date(e.spentAt).toLocaleDateString("tr-TR", {
                      timeZone: TZ,
                      month: "short",
                    })}
                  </span>
                </div>
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-base font-semibold text-zinc-900">
                    {e.category}
                  </span>
                  <span className="mt-0.5 flex flex-wrap items-center gap-1.5">
                    {e.partyName && (
                      <span className="rounded-full bg-sky-100 px-2 py-0.5 text-[11px] font-medium text-sky-800">
                        {e.partyName}
                      </span>
                    )}
                    <StatusBadge expense={e} />
                  </span>
                </div>
                <span className="shrink-0 text-base font-semibold tabular-nums text-zinc-900">
                  {formatTRY(e.amount)}
                </span>
                <span className="shrink-0 text-zinc-300">›</span>
              </button>
            </li>
          ))}
        </ul>
      )}

      {adding && (
        <ExpenseFormSheet
          onClose={() => setAdding(false)}
          onSaved={async () => {
            setAdding(false);
            await load();
          }}
        />
      )}

      {selected && (
        <ExpenseDetailSheet
          expense={selected}
          busy={busy}
          onClose={() => setSelected(null)}
          onChanged={load}
          onDelete={() => deleteExpense(selected)}
        />
      )}
    </main>
  );
}

function SummaryCell({
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

function StatusBadge({ expense }: { expense: Expense }) {
  const remaining = remainingOf(expense);
  const paid = paidOf(expense);
  if (remaining <= 0) {
    return (
      <span className="rounded-full bg-green-100 px-2 py-0.5 text-[11px] font-medium text-green-800">
        Ödendi
      </span>
    );
  }
  if (paid > 0) {
    return (
      <span className="rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-800">
        Kalan {formatTRY(remaining)}
      </span>
    );
  }
  return (
    <span className="rounded-full bg-red-100 px-2 py-0.5 text-[11px] font-medium text-red-800">
      Ödenmedi
    </span>
  );
}

// ---- Detail sheet (payments) --------------------------------------------

function ExpenseDetailSheet({
  expense,
  busy,
  onClose,
  onChanged,
  onDelete,
}: {
  expense: Expense;
  busy: boolean;
  onClose: () => void;
  onChanged: () => void | Promise<void>;
  onDelete: () => void;
}) {
  const remaining = remainingOf(expense);
  const paid = paidOf(expense);
  const payments = paymentsOf(expense);

  const [showForm, setShowForm] = useState(false);
  const [amountTL, setAmountTL] = useState("");
  const [paidAt, setPaidAt] = useState(todayISO);
  const [localBusy, setLocalBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function addPayment(amountKurus: number) {
    setError(null);
    if (amountKurus <= 0) {
      setError("Gecerli bir tutar gir");
      return;
    }
    setLocalBusy(true);
    try {
      await api(`/api/v1/expenses/${expense.id}/payments`, {
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

  async function deletePayment(p: Payment) {
    if (!confirm(`${formatTRY(p.amount)} ödemesi silinsin mi?`)) return;
    setLocalBusy(true);
    try {
      await api(`/api/v1/expenses/${expense.id}/payments/${p.id}`, {
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
          {expense.category}
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
        {/* Balance header */}
        <div className="rounded-2xl border border-zinc-200 bg-white p-4 shadow-sm">
          <div className="flex items-baseline justify-between">
            <span className="text-sm text-zinc-500">Toplam</span>
            <span className="text-lg font-semibold tabular-nums text-zinc-900">
              {formatTRY(expense.amount)}
            </span>
          </div>
          <div className="mt-1 flex items-baseline justify-between">
            <span className="text-sm text-zinc-500">Ödenen</span>
            <span className="tabular-nums text-green-700">
              {formatTRY(paid)}
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
            {fmtDay(expense.spentAt)}
            {expense.partyName ? ` · ${expense.partyName}` : ""}
            {expense.supplier ? ` · ${expense.supplier}` : ""}
            {expense.note ? ` · ${expense.note}` : ""}
          </p>
        </div>

        {error && (
          <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
            {error}
          </div>
        )}

        {/* Payments */}
        <div className="flex flex-col gap-2">
          <span className="text-sm font-medium text-zinc-700">Ödemeler</span>
          {payments.length === 0 ? (
            <p className="text-sm text-zinc-400">Henüz ödeme yok.</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {payments.map((p) => (
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
                    onClick={() => deletePayment(p)}
                    disabled={localBusy}
                    className="flex h-8 w-8 items-center justify-center rounded-full text-zinc-400 active:bg-zinc-100 active:text-red-700 disabled:opacity-30"
                    aria-label="Ödeme sil"
                  >
                    ×
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Add payment */}
        {remaining > 0 &&
          (showForm ? (
            <div className="flex flex-col gap-3 rounded-2xl border border-zinc-200 bg-zinc-50 p-4">
              <div className="flex flex-col">
                <span className="mb-1.5 text-sm font-medium text-zinc-700">
                  Ödeme tutarı (TL)
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
                  Ödeme tarihi
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
                    addPayment(
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
                + Ödeme Ekle
              </button>
              <button
                onClick={() => addPayment(remaining)}
                disabled={localBusy}
                className="flex-1 rounded-xl bg-amber-700 py-3 text-sm font-semibold text-white active:bg-amber-800 disabled:opacity-50"
              >
                Tümünü Öde
              </button>
            </div>
          ))}
      </div>
    </div>
  );
}

// ---- New expense sheet ---------------------------------------------------

function ExpenseFormSheet({
  onClose,
  onSaved,
}: {
  onClose: () => void;
  onSaved: () => void | Promise<void>;
}) {
  const [category, setCategory] = useState("");
  const [amountTL, setAmountTL] = useState("");
  const [supplier, setSupplier] = useState("");
  const [note, setNote] = useState("");
  const [spentAt, setSpentAt] = useState(todayISO);
  const [parties, setParties] = useState<Party[]>([]);
  const [party, setParty] = useState<Party | null>(null);
  const [pickingParty, setPickingParty] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api<PartiesResponse>("/api/v1/parties")
      .then((r) => setParties(r.parties))
      .catch(() => setParties([]));
  }, []);

  async function save() {
    setError(null);
    if (!category.trim()) {
      setError("Kategori sec");
      return;
    }
    const tl = Number(amountTL.replace(",", "."));
    if (!Number.isFinite(tl) || tl <= 0) {
      setError("Gecerli bir tutar gir");
      return;
    }
    setBusy(true);
    try {
      await api("/api/v1/expenses", {
        method: "POST",
        body: JSON.stringify({
          category,
          amount: fromTL(tl),
          supplier,
          partyId: party?.id ?? "",
          note,
          spentAt,
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
        <h2 className="text-base font-semibold text-zinc-900">Yeni Gider</h2>
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

        <Field label="Tarih">
          <input
            type="date"
            value={spentAt}
            onChange={(e) => setSpentAt(e.target.value)}
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        <Field label="Kategori">
          <div className="flex flex-wrap gap-2">
            {CATEGORIES.map((c) => (
              <button
                key={c}
                onClick={() => setCategory(c)}
                className={`rounded-full px-3.5 py-2 text-sm font-medium active:bg-amber-100 ${
                  category === c
                    ? "bg-amber-700 text-white"
                    : "bg-zinc-100 text-zinc-700"
                }`}
              >
                {c}
              </button>
            ))}
          </div>
        </Field>

        <Field label="Tutar (TL)">
          <input
            value={amountTL}
            onChange={(e) => setAmountTL(e.target.value)}
            inputMode="decimal"
            placeholder="orn. 10000"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base tabular-nums text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        <Field label="Kişi / Cari (opsiyonel)">
          {party ? (
            <div className="flex items-center gap-2 rounded-xl border border-sky-200 bg-sky-50 px-4 py-3">
              <span className="flex-1 truncate text-base font-medium text-sky-900">
                {party.name}
              </span>
              <button
                onClick={() => setParty(null)}
                className="rounded-lg px-3 py-1.5 text-sm font-medium text-sky-800 active:bg-sky-100"
              >
                Kaldır
              </button>
            </div>
          ) : (
            <button
              onClick={() => setPickingParty(true)}
              className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-left text-base text-zinc-500 shadow-sm active:bg-zinc-50"
            >
              Kişi seç (borç, avans...)
            </button>
          )}
        </Field>

        <Field label="Tedarikci / yer (opsiyonel)">
          <input
            value={supplier}
            onChange={(e) => setSupplier(e.target.value)}
            placeholder="orn. Hal Manavi"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        <Field label="Not (opsiyonel)">
          <input
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="orn. haftalik sebze meyve"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>
      </div>

      {pickingParty && (
        <PartyPicker
          parties={parties}
          onPick={(p) => {
            setParty(p);
            setPickingParty(false);
          }}
          onClose={() => setPickingParty(false)}
        />
      )}
    </div>
  );
}

// ---- Party picker (full-screen sheet, no dropdown) -----------------------

function PartyPicker({
  parties,
  onPick,
  onClose,
}: {
  parties: Party[];
  onPick: (p: Party) => void;
  onClose: () => void;
}) {
  const [q, setQ] = useState("");
  const norm = (s: string) => s.toLocaleLowerCase("tr");
  const filtered = q.trim()
    ? parties.filter((p) => norm(p.name).includes(norm(q.trim())))
    : parties;

  return (
    <div className="fixed inset-0 z-[60] flex flex-col bg-white">
      <header className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
        <button
          onClick={onClose}
          className="rounded-full px-3 py-2 text-sm font-medium text-zinc-600 active:bg-zinc-100"
        >
          Kapat
        </button>
        <h2 className="text-base font-semibold text-zinc-900">Kişi Seç</h2>
        <span className="w-16" />
      </header>

      <div className="p-4">
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Kişi ara"
          className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
        />
      </div>

      <div className="flex-1 overflow-y-auto px-4 pb-8">
        {parties.length === 0 ? (
          <p className="text-sm text-zinc-500">
            Henüz kişi yok. Kişiler / Cari ekranından ekleyebilirsin.
          </p>
        ) : filtered.length === 0 ? (
          <p className="text-sm text-zinc-500">Eşleşen kişi yok.</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {filtered.map((p) => (
              <li key={p.id}>
                <button
                  onClick={() => onPick(p)}
                  className="flex w-full items-center justify-between gap-3 rounded-2xl border border-zinc-200 bg-white p-4 text-left shadow-sm active:bg-zinc-50"
                >
                  <span className="truncate text-base font-medium text-zinc-900">
                    {p.name}
                  </span>
                  {p.remaining > 0 && (
                    <span className="shrink-0 rounded-full bg-red-100 px-2 py-0.5 text-[11px] font-medium text-red-800">
                      kalan {formatTRY(p.remaining)}
                    </span>
                  )}
                </button>
              </li>
            ))}
          </ul>
        )}
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
