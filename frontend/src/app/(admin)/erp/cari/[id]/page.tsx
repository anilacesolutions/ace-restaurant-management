"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { api } from "@/lib/api";
import { formatTRY, fromTL, toTL } from "@/lib/money";
import type { Expense, PartyLedger, Payment, Receivable } from "@/lib/types";

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

const sumPay = (p: Payment[] | null | undefined) =>
  (p ?? []).reduce((s, x) => s + x.amount, 0);

// A cari movement is either a gider (money we owe — settled with ödeme) or an
// alacak (money owed to us — settled with tahsilat). Unified for one timeline.
type Movement =
  | { kind: "gider"; id: string; date: string; e: Expense }
  | { kind: "alacak"; id: string; date: string; r: Receivable };

export default function CariDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;

  const [ledger, setLedger] = useState<PartyLedger | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [addKind, setAddKind] = useState<"gider" | "alacak" | null>(null);
  const [selected, setSelected] = useState<Movement | null>(null);

  const load = useCallback(async () => {
    try {
      const l = await api<PartyLedger>(`/api/v1/parties/${id}`);
      setLedger(l);
      setError(null);
      // keep an open detail sheet fresh
      setSelected((cur) => {
        if (!cur) return cur;
        if (cur.kind === "gider") {
          const e = l.expenses.find((x) => x.id === cur.id);
          return e ? { kind: "gider", id: e.id, date: e.spentAt, e } : null;
        }
        const r = l.receivables.find((x) => x.id === cur.id);
        return r ? { kind: "alacak", id: r.id, date: r.issuedAt, r } : null;
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);

  const movements = useMemo<Movement[]>(() => {
    if (!ledger) return [];
    const g: Movement[] = ledger.expenses.map((e) => ({
      kind: "gider",
      id: e.id,
      date: e.spentAt,
      e,
    }));
    const a: Movement[] = ledger.receivables.map((r) => ({
      kind: "alacak",
      id: r.id,
      date: r.issuedAt,
      r,
    }));
    return [...g, ...a].sort((x, y) => (x.date < y.date ? 1 : -1));
  }, [ledger]);

  return (
    <main className="flex flex-1 flex-col gap-5 p-4 pb-28">
      <header className="flex items-center gap-3">
        <Link
          href="/erp/cari"
          className="flex h-10 w-10 items-center justify-center rounded-full border border-zinc-200 bg-white text-zinc-700 shadow-sm active:bg-zinc-50"
          aria-label="Geri"
        >
          ←
        </Link>
        <h1 className="truncate text-2xl font-semibold text-zinc-900">
          {ledger?.party.name ?? "Cari"}
        </h1>
      </header>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {ledger === null ? (
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      ) : (
        <>
          {/* Net balance */}
          <NetCard ledger={ledger} />

          {/* Add buttons */}
          <div className="grid grid-cols-2 gap-2">
            <button
              onClick={() => setAddKind("gider")}
              className="rounded-xl bg-red-700 py-3.5 text-sm font-semibold text-white shadow-sm active:bg-red-800"
            >
              + Gider (borç)
            </button>
            <button
              onClick={() => setAddKind("alacak")}
              className="rounded-xl bg-green-700 py-3.5 text-sm font-semibold text-white shadow-sm active:bg-green-800"
            >
              + Alacak
            </button>
          </div>

          {/* Movements */}
          <div className="flex flex-col gap-2">
            <span className="text-sm font-medium text-zinc-700">Hareketler</span>
            {movements.length === 0 ? (
              <p className="text-sm text-zinc-400">
                Henüz hareket yok. Gider ya da alacak ekleyin.
              </p>
            ) : (
              <ul className="flex flex-col gap-2">
                {movements.map((m) => (
                  <li key={`${m.kind}-${m.id}`}>
                    <MovementRow m={m} onOpen={() => setSelected(m)} />
                  </li>
                ))}
              </ul>
            )}
          </div>
        </>
      )}

      {addKind === "gider" && ledger && (
        <GiderFormSheet
          partyId={id}
          partyName={ledger.party.name}
          onClose={() => setAddKind(null)}
          onSaved={async () => {
            setAddKind(null);
            await load();
          }}
        />
      )}

      {addKind === "alacak" && ledger && (
        <AlacakFormSheet
          partyId={id}
          onClose={() => setAddKind(null)}
          onSaved={async () => {
            setAddKind(null);
            await load();
          }}
        />
      )}

      {selected && (
        <MovementDetailSheet
          movement={selected}
          onClose={() => setSelected(null)}
          onChanged={load}
          onError={setError}
        />
      )}
    </main>
  );
}

function NetCard({ ledger }: { ledger: PartyLedger }) {
  const { net, borc, alacak } = ledger;
  const owedToUs = net > 0;
  const clean = net === 0;
  // Lifetime cash flow with this cari: aldığımız = collected from them
  // (tahsilat), verdiğimiz = paid to them (ödeme).
  const aldigimiz = ledger.receivables.reduce((s, r) => s + sumPay(r.payments), 0);
  const verdigimiz = ledger.expenses.reduce((s, e) => s + sumPay(e.payments), 0);
  return (
    <div className="rounded-2xl border border-zinc-200 bg-white p-4 shadow-sm">
      <div className="flex items-baseline justify-between">
        <span className="text-sm font-medium text-zinc-700">Net bakiye</span>
        <span
          className={`text-3xl font-bold tabular-nums ${
            clean
              ? "text-zinc-400"
              : owedToUs
                ? "text-green-700"
                : "text-red-700"
          }`}
        >
          {formatTRY(Math.abs(net))}
        </span>
      </div>
      <p className="mt-0.5 text-right text-xs uppercase tracking-wide text-zinc-400">
        {clean ? "temiz" : owedToUs ? "bize borçlu (alacak)" : "bizim borcumuz"}
      </p>
      <div className="mt-3 grid grid-cols-2 gap-2 border-t border-zinc-100 pt-3 text-center">
        <div className="flex flex-col">
          <span className="text-xs text-zinc-500">Bize borçlu</span>
          <span className="text-base font-semibold tabular-nums text-green-700">
            {formatTRY(alacak)}
          </span>
        </div>
        <div className="flex flex-col">
          <span className="text-xs text-zinc-500">Bizim borç</span>
          <span className="text-base font-semibold tabular-nums text-red-700">
            {formatTRY(borc)}
          </span>
        </div>
      </div>

      {/* Lifetime cash flow — what actually changed hands to date. */}
      <div className="mt-3 border-t border-zinc-100 pt-3">
        <p className="mb-2 text-center text-[11px] font-medium uppercase tracking-wide text-zinc-400">
          Bugüne kadar
        </p>
        <div className="grid grid-cols-2 gap-2 text-center">
          <div className="flex flex-col">
            <span className="text-xs text-zinc-500">Aldığımız</span>
            <span className="text-base font-semibold tabular-nums text-zinc-900">
              {formatTRY(aldigimiz)}
            </span>
            <span className="text-[10px] uppercase tracking-wide text-zinc-400">
              tahsil edilen
            </span>
          </div>
          <div className="flex flex-col">
            <span className="text-xs text-zinc-500">Verdiğimiz</span>
            <span className="text-base font-semibold tabular-nums text-zinc-900">
              {formatTRY(verdigimiz)}
            </span>
            <span className="text-[10px] uppercase tracking-wide text-zinc-400">
              ödenen
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

function MovementRow({ m, onOpen }: { m: Movement; onOpen: () => void }) {
  const isGider = m.kind === "gider";
  const doc = isGider ? m.e : m.r;
  const paid = sumPay(doc.payments);
  const remaining = doc.amount - paid;
  const settled = remaining <= 0;
  const payments = doc.payments ?? [];
  const payLabel = isGider ? "Ödeme" : "Tahsilat";

  return (
    <>
    <button
      onClick={onOpen}
      className="flex w-full items-center gap-3 rounded-2xl border border-zinc-200 bg-white p-3 text-left shadow-sm active:bg-zinc-50"
    >
      <div className="flex w-14 shrink-0 flex-col items-center">
        <span className="text-lg font-bold tabular-nums leading-none text-zinc-900">
          {new Date(m.date).toLocaleDateString("tr-TR", {
            timeZone: TZ,
            day: "2-digit",
          })}
        </span>
        <span className="text-[10px] uppercase text-zinc-400">
          {new Date(m.date).toLocaleDateString("tr-TR", {
            timeZone: TZ,
            month: "short",
          })}
        </span>
      </div>
      <div className="flex min-w-0 flex-1 flex-col">
        <span className="flex items-center gap-1.5">
          <span
            className={`rounded-full px-2 py-0.5 text-[11px] font-semibold ${
              isGider
                ? "bg-red-100 text-red-800"
                : "bg-green-100 text-green-800"
            }`}
          >
            {isGider ? "Gider" : "Alacak"}
          </span>
          {settled ? (
            <span className="rounded-full bg-zinc-100 px-2 py-0.5 text-[11px] font-medium text-zinc-600">
              {isGider ? "Ödendi" : "Tahsil edildi"}
            </span>
          ) : paid > 0 ? (
            <span className="rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-800">
              kalan {formatTRY(remaining)}
            </span>
          ) : null}
        </span>
        {doc.note && (
          <span className="mt-0.5 truncate text-xs text-zinc-500">
            {doc.note}
          </span>
        )}
      </div>
      <span
        className={`shrink-0 text-base font-semibold tabular-nums ${
          isGider ? "text-red-700" : "text-green-700"
        }`}
      >
        {isGider ? "−" : "+"}
        {formatTRY(doc.amount)}
      </span>
      <span className="shrink-0 text-zinc-300">›</span>
    </button>

    {/* Ödeme / tahsilat entries under the movement — a real ledger view. */}
    {payments.length > 0 && (
      <ul className="ml-14 mt-1 flex flex-col gap-1 border-l-2 border-zinc-100 pl-3">
        {payments.map((p) => (
          <li
            key={p.id}
            className="flex items-center justify-between text-xs text-zinc-500"
          >
            <span>
              {new Date(p.paidAt).toLocaleDateString("tr-TR", {
                timeZone: TZ,
                day: "2-digit",
                month: "short",
              })}{" "}
              · {payLabel}
            </span>
            <span className="tabular-nums font-medium text-green-700">
              {formatTRY(p.amount)}
            </span>
          </li>
        ))}
      </ul>
    )}
    </>
  );
}

// ---- Movement detail (payments / collections) ----------------------------

function MovementDetailSheet({
  movement,
  onClose,
  onChanged,
  onError,
}: {
  movement: Movement;
  onClose: () => void;
  onChanged: () => void | Promise<void>;
  onError: (m: string) => void;
}) {
  const isGider = movement.kind === "gider";
  const doc = isGider ? movement.e : movement.r;
  const base = isGider ? "expenses" : "receivables";

  const paid = sumPay(doc.payments);
  const remaining = doc.amount - paid;
  const payments = doc.payments ?? [];

  const payLabel = isGider ? "Ödeme" : "Tahsilat";
  const paidLabel = isGider ? "Ödenen" : "Tahsil edilen";

  const [showForm, setShowForm] = useState(false);
  const [amountTL, setAmountTL] = useState("");
  const [paidAt, setPaidAt] = useState(todayISO);
  const [busy, setBusy] = useState(false);

  async function addPayment(amountKurus: number) {
    if (amountKurus <= 0) {
      onError("Gecerli bir tutar gir");
      return;
    }
    setBusy(true);
    try {
      await api(`/api/v1/${base}/${doc.id}/payments`, {
        method: "POST",
        body: JSON.stringify({ amount: amountKurus, paidAt }),
      });
      setShowForm(false);
      setAmountTL("");
      await onChanged();
    } catch (e) {
      onError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function deletePayment(p: Payment) {
    if (!confirm(`${formatTRY(p.amount)} kaydı silinsin mi?`)) return;
    setBusy(true);
    try {
      await api(`/api/v1/${base}/${doc.id}/payments/${p.id}`, {
        method: "DELETE",
      });
      await onChanged();
    } catch (e) {
      onError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function deleteMovement() {
    if (
      !confirm(
        `${isGider ? "Gider" : "Alacak"} — ${formatTRY(doc.amount)} silinsin mi?`,
      )
    )
      return;
    setBusy(true);
    try {
      await api(`/api/v1/${base}/${doc.id}`, { method: "DELETE" });
      onClose();
      await onChanged();
    } catch (e) {
      onError(e instanceof Error ? e.message : String(e));
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
          Kapat
        </button>
        <h2 className="truncate px-2 text-base font-semibold text-zinc-900">
          {isGider ? "Gider" : "Alacak"}
        </h2>
        <button
          onClick={deleteMovement}
          disabled={busy}
          className="rounded-full px-3 py-2 text-sm font-medium text-red-700 active:bg-red-50 disabled:opacity-40"
        >
          Sil
        </button>
      </header>

      <div className="flex flex-1 flex-col gap-5 overflow-y-auto p-4 pb-28">
        <div className="rounded-2xl border border-zinc-200 bg-white p-4 shadow-sm">
          <div className="flex items-baseline justify-between">
            <span className="text-sm text-zinc-500">Toplam</span>
            <span className="text-lg font-semibold tabular-nums text-zinc-900">
              {formatTRY(doc.amount)}
            </span>
          </div>
          <div className="mt-1 flex items-baseline justify-between">
            <span className="text-sm text-zinc-500">{paidLabel}</span>
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
            {fmtDay(movement.date)}
            {doc.note ? ` · ${doc.note}` : ""}
          </p>
        </div>

        <div className="flex flex-col gap-2">
          <span className="text-sm font-medium text-zinc-700">
            {payLabel} geçmişi
          </span>
          {payments.length === 0 ? (
            <p className="text-sm text-zinc-400">Henüz {payLabel.toLowerCase()} yok.</p>
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
                    disabled={busy}
                    className="flex h-8 w-8 items-center justify-center rounded-full text-zinc-400 active:bg-zinc-100 active:text-red-700 disabled:opacity-30"
                    aria-label={`${payLabel} sil`}
                  >
                    ×
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {remaining > 0 &&
          (showForm ? (
            <div className="flex flex-col gap-3 rounded-2xl border border-zinc-200 bg-zinc-50 p-4">
              <div className="flex flex-col">
                <span className="mb-1.5 text-sm font-medium text-zinc-700">
                  {payLabel} tutarı (TL)
                </span>
                <input
                  value={amountTL}
                  onChange={(e) => setAmountTL(e.target.value)}
                  inputMode="decimal"
                  placeholder={`kalan: ${toTL(remaining)}`}
                  autoFocus
                  className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base tabular-nums text-zinc-900 shadow-sm outline-none focus:border-amber-500"
                />
              </div>
              <div className="flex flex-col">
                <span className="mb-1.5 text-sm font-medium text-zinc-700">
                  Tarih
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
                    addPayment(Number.isFinite(tl) && tl > 0 ? fromTL(tl) : 0);
                  }}
                  disabled={busy}
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
                disabled={busy}
                className="flex-1 rounded-xl border border-amber-700 py-3 text-sm font-semibold text-amber-800 active:bg-amber-50 disabled:opacity-50"
              >
                + {payLabel} Ekle
              </button>
              <button
                onClick={() => addPayment(remaining)}
                disabled={busy}
                className="flex-1 rounded-xl bg-amber-700 py-3 text-sm font-semibold text-white active:bg-amber-800 disabled:opacity-50"
              >
                {isGider ? "Tümünü Öde" : "Tümünü Tahsil Et"}
              </button>
            </div>
          ))}
      </div>
    </div>
  );
}

// ---- New gider (default paid + confirm) ----------------------------------

function GiderFormSheet({
  partyId,
  partyName,
  onClose,
  onSaved,
}: {
  partyId: string;
  partyName: string;
  onClose: () => void;
  onSaved: () => void | Promise<void>;
}) {
  const [amountTL, setAmountTL] = useState("");
  const [note, setNote] = useState("");
  const [spentAt, setSpentAt] = useState(todayISO);
  const [paid, setPaid] = useState(true); // default: ödendi
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirming, setConfirming] = useState(false);

  const tl = Number(amountTL.replace(",", "."));
  const amountOk = Number.isFinite(tl) && tl > 0;
  const canSave = !!spentAt && amountOk;

  async function doSave() {
    setError(null);
    if (!amountOk) {
      setError("Gecerli bir tutar gir");
      return;
    }
    setBusy(true);
    try {
      await api("/api/v1/expenses", {
        method: "POST",
        body: JSON.stringify({
          amount: fromTL(tl),
          partyId,
          note,
          spentAt,
          paid,
        }),
      });
      await onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setConfirming(false);
    } finally {
      setBusy(false);
    }
  }

  function onSubmit() {
    setError(null);
    if (!canSave) {
      setError("Gecerli bir tutar gir");
      return;
    }
    // Confirm the auto-settle case; an open borç saves straight through.
    if (paid) setConfirming(true);
    else doSave();
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
          onClick={onSubmit}
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

        <div className="rounded-xl border border-sky-200 bg-sky-50 px-4 py-3 text-sm font-medium text-sky-900">
          {partyName}
        </div>

        <Field label="Tarih">
          <input
            type="date"
            value={spentAt}
            onChange={(e) => setSpentAt(e.target.value)}
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        <Field label="Tutar (TL)">
          <input
            value={amountTL}
            onChange={(e) => setAmountTL(e.target.value)}
            inputMode="decimal"
            placeholder="orn. 10000"
            autoFocus
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base tabular-nums text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        <Field label="Not (opsiyonel)">
          <input
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="orn. fayans ücreti"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        {/* Ödendi mi? — default evet */}
        <button
          type="button"
          onClick={() => setPaid((v) => !v)}
          className="flex items-center justify-between rounded-2xl border border-zinc-200 bg-white p-4 text-left shadow-sm active:bg-zinc-50"
        >
          <div className="flex flex-col">
            <span className="text-base font-semibold text-zinc-900">
              Ödendi mi?
            </span>
            <span className="mt-0.5 text-xs text-zinc-500">
              {paid
                ? "Ödendi olarak kaydedilecek (borç kalmaz)"
                : "Borç olarak kalacak, sonra ödenecek"}
            </span>
          </div>
          <span
            className={`relative h-8 w-14 shrink-0 rounded-full transition-colors ${
              paid ? "bg-green-600" : "bg-zinc-300"
            }`}
          >
            <span
              className={`absolute top-1 h-6 w-6 rounded-full bg-white shadow transition-all ${
                paid ? "left-7" : "left-1"
              }`}
            />
          </span>
        </button>
      </div>

      {confirming && (
        <ConfirmModal
          title="Gider kaydedilsin mi?"
          message={`${partyName} · ${amountOk ? formatTRY(fromTL(tl)) : ""} — ödendi olarak kaydedilecek.`}
          confirmLabel="Onayla"
          busy={busy}
          onCancel={() => setConfirming(false)}
          onConfirm={doSave}
        />
      )}
    </div>
  );
}

// ---- New alacak ----------------------------------------------------------

function AlacakFormSheet({
  partyId,
  onClose,
  onSaved,
}: {
  partyId: string;
  onClose: () => void;
  onSaved: () => void | Promise<void>;
}) {
  const [amountTL, setAmountTL] = useState("");
  const [note, setNote] = useState("");
  const [issuedAt, setIssuedAt] = useState(todayISO);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const tl = Number(amountTL.replace(",", "."));
  const amountOk = Number.isFinite(tl) && tl > 0;
  const canSave = !!issuedAt && amountOk;

  async function save() {
    setError(null);
    if (!amountOk) {
      setError("Gecerli bir tutar gir");
      return;
    }
    setBusy(true);
    try {
      await api("/api/v1/receivables", {
        method: "POST",
        body: JSON.stringify({
          amount: fromTL(tl),
          partyId,
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

        <p className="text-sm text-zinc-500">
          Bu cari bize borçlu — tahsil edildikçe düşülür.
        </p>

        <Field label="Tarih">
          <input
            type="date"
            value={issuedAt}
            onChange={(e) => setIssuedAt(e.target.value)}
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        <Field label="Tutar (TL)">
          <input
            value={amountTL}
            onChange={(e) => setAmountTL(e.target.value)}
            inputMode="decimal"
            placeholder="orn. 1500"
            autoFocus
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base tabular-nums text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        <Field label="Not (opsiyonel)">
          <input
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="orn. veresiye, hesap"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>
      </div>
    </div>
  );
}

// ---- Shared bits ---------------------------------------------------------

function ConfirmModal({
  title,
  message,
  confirmLabel,
  busy,
  onCancel,
  onConfirm,
}: {
  title: string;
  message: string;
  confirmLabel: string;
  busy: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/40 p-6">
      <div className="w-full max-w-sm rounded-2xl bg-white p-5 shadow-xl">
        <h3 className="text-lg font-semibold text-zinc-900">{title}</h3>
        <p className="mt-2 text-sm text-zinc-600">{message}</p>
        <div className="mt-5 flex gap-2">
          <button
            onClick={onCancel}
            disabled={busy}
            className="flex-1 rounded-xl bg-zinc-200 py-3 text-sm font-semibold text-zinc-700 active:bg-zinc-300 disabled:opacity-50"
          >
            Vazgeç
          </button>
          <button
            onClick={onConfirm}
            disabled={busy}
            className="flex-1 rounded-xl bg-green-700 py-3 text-sm font-semibold text-white active:bg-green-800 disabled:opacity-50"
          >
            {busy ? "..." : confirmLabel}
          </button>
        </div>
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
