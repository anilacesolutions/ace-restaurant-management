"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { formatTRY } from "@/lib/money";
import type {
  Party,
  PartiesResponse,
  Expense,
  ExpensesResponse,
} from "@/lib/types";

const TZ = "Europe/Istanbul";
const pad = (n: number) => String(n).padStart(2, "0");

function todayISO(): string {
  return new Date().toLocaleDateString("en-CA", { timeZone: TZ });
}

const paidOf = (e: Expense) => (e.payments ?? []).reduce((s, p) => s + p.amount, 0);

type Tab = "cariler" | "hareketler";

export default function ErpCariPage() {
  const [tab, setTab] = useState<Tab>("cariler");
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

  // Net > 0: they owe us (alacak). Net < 0: we owe them (borç).
  const totals = useMemo(() => {
    let alacak = 0;
    let borc = 0;
    for (const p of parties ?? []) {
      if (p.net > 0) alacak += p.net;
      else if (p.net < 0) borc += -p.net;
    }
    return { alacak, borc };
  }, [parties]);

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
          <h1 className="text-2xl font-semibold text-zinc-900">Cari</h1>
        </div>
        <button
          onClick={() => setAdding(true)}
          className="rounded-full bg-amber-700 px-5 py-2.5 text-sm font-semibold text-white shadow-sm active:bg-amber-800"
        >
          + Yeni Cari
        </button>
      </header>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {/* Balance summary */}
      <div className="grid grid-cols-2 gap-2">
        <div className="flex flex-col rounded-2xl border border-green-200 bg-green-50 px-4 py-3">
          <span className="text-xs font-medium text-green-900/70">
            Bize borçlu (alacak)
          </span>
          <span className="text-xl font-bold tabular-nums text-green-700">
            {formatTRY(totals.alacak)}
          </span>
        </div>
        <div className="flex flex-col rounded-2xl border border-red-200 bg-red-50 px-4 py-3">
          <span className="text-xs font-medium text-red-900/70">
            Bizim borcumuz
          </span>
          <span className="text-xl font-bold tabular-nums text-red-700">
            {formatTRY(totals.borc)}
          </span>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 rounded-full border border-zinc-200 bg-white p-1 shadow-sm">
        <TabButton active={tab === "cariler"} onClick={() => setTab("cariler")}>
          Cariler
        </TabButton>
        <TabButton
          active={tab === "hareketler"}
          onClick={() => setTab("hareketler")}
        >
          Hareketler
        </TabButton>
      </div>

      {tab === "cariler" ? (
        <CariList parties={parties} onChanged={load} onError={setError} />
      ) : (
        <MovementTimeline onError={setError} />
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

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex-1 rounded-full py-2.5 text-sm font-semibold transition-colors ${
        active ? "bg-amber-700 text-white" : "text-zinc-600 active:bg-zinc-50"
      }`}
    >
      {children}
    </button>
  );
}

// ---- Cariler tab: net balances -------------------------------------------

function CariList({
  parties,
  onChanged,
  onError,
}: {
  parties: Party[] | null;
  onChanged: () => void | Promise<void>;
  onError: (m: string) => void;
}) {
  async function remove(e: React.MouseEvent, p: Party) {
    e.preventDefault();
    e.stopPropagation();
    if (
      !confirm(`${p.name} silinsin mi? (geçmiş hareketler etkilenmez, isim korunur)`)
    )
      return;
    try {
      await api(`/api/v1/parties/${p.id}`, { method: "DELETE" });
      await onChanged();
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    }
  }

  if (parties === null)
    return <p className="text-sm text-zinc-500">Yukleniyor...</p>;
  if (parties.length === 0)
    return (
      <p className="text-sm text-zinc-500">
        Henüz cari yok. + Yeni Cari ile ekleyin (Ahmet Bey, Ünlü Yapı, Kira,
        Maaş, Fatura, Avans...).
      </p>
    );

  return (
    <ul className="flex flex-col gap-2">
      {parties.map((p) => (
        <li
          key={p.id}
          className="flex items-center gap-3 rounded-2xl border border-zinc-200 bg-white p-4 shadow-sm"
        >
          <Link
            href={`/erp/cari/${p.id}`}
            className="flex min-w-0 flex-1 items-center gap-3 active:opacity-70"
          >
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-sky-100 text-base font-semibold text-sky-800">
              {p.name.slice(0, 1).toUpperCase()}
            </div>
            <div className="flex min-w-0 flex-1 flex-col">
              <span className="truncate text-base font-semibold text-zinc-900">
                {p.name}
              </span>
              <span className="mt-0.5 text-xs text-zinc-500">
                {p.movementCount} hareket
              </span>
            </div>
            <div className="flex shrink-0 flex-col items-end">
              <BalanceAmount net={p.net} />
            </div>
          </Link>
          <button
            onClick={(e) => remove(e, p)}
            className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-zinc-400 active:bg-zinc-100 active:text-red-700"
            aria-label={`${p.name} sil`}
          >
            ×
          </button>
        </li>
      ))}
    </ul>
  );
}

// net > 0 they owe us (alacak, green); net < 0 we owe (borç, red); 0 temiz.
function BalanceAmount({ net }: { net: number }) {
  if (net === 0)
    return (
      <>
        <span className="text-base font-bold tabular-nums text-zinc-400">
          {formatTRY(0)}
        </span>
        <span className="text-[10px] uppercase tracking-wide text-zinc-400">
          temiz
        </span>
      </>
    );
  const owedToUs = net > 0;
  return (
    <>
      <span
        className={`text-base font-bold tabular-nums ${
          owedToUs ? "text-green-700" : "text-red-700"
        }`}
      >
        {formatTRY(Math.abs(net))}
      </span>
      <span className="text-[10px] uppercase tracking-wide text-zinc-400">
        {owedToUs ? "alacak" : "borç"}
      </span>
    </>
  );
}

// ---- Hareketler tab: chronological expense timeline (read-only) -----------

function MovementTimeline({ onError }: { onError: (m: string) => void }) {
  const router = useRouter();
  const [ym, setYm] = useState(() => {
    const [y, m] = todayISO().split("-").map(Number);
    return { y, m };
  });
  const [expenses, setExpenses] = useState<Expense[] | null>(null);

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

  useEffect(() => {
    setExpenses(null);
    api<ExpensesResponse>(`/api/v1/expenses?from=${range.from}&to=${range.to}`)
      .then((r) => setExpenses(r.expenses))
      .catch((e) => onError(e instanceof Error ? e.message : String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [range.from, range.to]);

  const total = (expenses ?? []).reduce((s, e) => s + e.amount, 0);

  function shiftMonth(delta: number) {
    setYm((p) => {
      const idx = p.y * 12 + (p.m - 1) + delta;
      return { y: Math.floor(idx / 12), m: (idx % 12) + 1 };
    });
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between rounded-2xl border border-zinc-200 bg-white p-2 shadow-sm">
        <button
          onClick={() => shiftMonth(-1)}
          className="flex h-11 w-11 items-center justify-center rounded-xl text-xl text-zinc-600 active:bg-zinc-100"
          aria-label="Önceki ay"
        >
          ‹
        </button>
        <div className="flex flex-col items-center">
          <span className="text-base font-semibold capitalize text-zinc-900">
            {monthLabel}
          </span>
          <span className="text-xs text-zinc-500 tabular-nums">
            {formatTRY(total)} gider
          </span>
        </div>
        <button
          onClick={() => shiftMonth(1)}
          className="flex h-11 w-11 items-center justify-center rounded-xl text-xl text-zinc-600 active:bg-zinc-100"
          aria-label="Sonraki ay"
        >
          ›
        </button>
      </div>

      {expenses === null ? (
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      ) : expenses.length === 0 ? (
        <p className="text-sm text-zinc-500">Bu ay gider yok.</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {expenses.map((e) => {
            const remaining = e.amount - paidOf(e);
            return (
              <li key={e.id}>
                <button
                  onClick={() =>
                    e.partyId && router.push(`/erp/cari/${e.partyId}`)
                  }
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
                      {e.partyName || e.category || "Gider"}
                    </span>
                    {e.note && (
                      <span className="truncate text-xs text-zinc-500">
                        {e.note}
                      </span>
                    )}
                  </div>
                  <div className="flex shrink-0 flex-col items-end">
                    <span className="text-base font-semibold tabular-nums text-zinc-900">
                      {formatTRY(e.amount)}
                    </span>
                    {remaining > 0 ? (
                      <span className="text-[10px] font-medium uppercase text-red-600">
                        kalan {formatTRY(remaining)}
                      </span>
                    ) : (
                      <span className="text-[10px] font-medium uppercase text-green-600">
                        ödendi
                      </span>
                    )}
                  </div>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

// ---- New cari sheet -------------------------------------------------------

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
        <h2 className="text-base font-semibold text-zinc-900">Yeni Cari</h2>
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
            placeholder="orn. Ahmet Bey, Ünlü Yapı, Kira, Maaş, Fatura"
            autoFocus
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
