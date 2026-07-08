"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { formatTRY } from "@/lib/money";
import { ADISYON_LOGO } from "@/lib/adisyonLogo";
import type {
  CategoryWithItems,
  MenuItem,
  MenuResponse,
  Order,
  OrderResponse,
} from "@/lib/types";

interface DraftLine {
  item: MenuItem;
  qty: number;
  note?: string;
}

// TableOrderEntry is the table screen shared by the waiter (/garson/masa/[n])
// and the cashier (/kasa/masa/[n]). It opens on the current adisyon (the orders
// already placed); "Yeni Sipariş" switches to the full menu to add more.
export function TableOrderEntry({
  tableNumber,
  backHref,
  unauthorizedHref,
  cashier = false,
}: {
  tableNumber: number;
  backHref: string;
  unauthorizedHref: string;
  cashier?: boolean;
}) {
  const router = useRouter();

  const [menu, setMenu] = useState<CategoryWithItems[] | null>(null);
  const [order, setOrder] = useState<Order | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [view, setView] = useState<"adisyon" | "menu">("adisyon");
  const [draft, setDraft] = useState<Record<string, DraftLine>>({});
  const [picking, setPicking] = useState<MenuItem | null>(null);
  const [showDraft, setShowDraft] = useState(false);
  const [showClose, setShowClose] = useState(false);
  const [sending, setSending] = useState(false);
  const [closing, setClosing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const draftKey = `order-draft-masa-${tableNumber}`;

  function guard(e: unknown) {
    if (e instanceof ApiError && e.status === 401) {
      router.replace(unauthorizedHref);
      return;
    }
    setError(e instanceof Error ? e.message : String(e));
  }

  useEffect(() => {
    (async () => {
      try {
        const [m, o] = await Promise.all([
          api<MenuResponse>("/api/v1/menu"),
          api<OrderResponse>(`/api/v1/orders/table/${tableNumber}`),
        ]);
        setMenu(m.categories);
        setOrder(o.order);
        setLoaded(true);
      } catch (e) {
        guard(e);
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tableNumber]);

  // Restore any unsent draft for this table (survives reload / signal drop).
  useEffect(() => {
    try {
      const raw = localStorage.getItem(draftKey);
      if (raw) setDraft(JSON.parse(raw));
    } catch {
      /* ignore corrupt draft */
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tableNumber]);

  useEffect(() => {
    try {
      localStorage.setItem(draftKey, JSON.stringify(draft));
    } catch {
      /* storage full / disabled — draft stays in memory */
    }
  }, [draft, draftKey]);

  const draftLines = useMemo(() => Object.values(draft), [draft]);
  const draftCount = useMemo(
    () => draftLines.reduce((s, l) => s + l.qty, 0),
    [draftLines],
  );
  const draftTotal = useMemo(
    () => draftLines.reduce((s, l) => s + l.item.price * l.qty, 0),
    [draftLines],
  );

  function addToDraft(item: MenuItem, qty: number, note?: string) {
    setDraft((prev) => {
      const existing = prev[item.id];
      const nextQty = (existing?.qty ?? 0) + qty;
      return {
        ...prev,
        [item.id]: { item, qty: nextQty, note: note || existing?.note },
      };
    });
  }

  function setDraftQty(itemId: string, qty: number) {
    setDraft((prev) => {
      if (qty <= 0) {
        const rest = { ...prev };
        delete rest[itemId];
        return rest;
      }
      return { ...prev, [itemId]: { ...prev[itemId], qty } };
    });
  }

  async function send() {
    if (draftLines.length === 0) return;
    setSending(true);
    setError(null);
    try {
      const r = await api<OrderResponse>(
        `/api/v1/orders/table/${tableNumber}/items`,
        {
          method: "POST",
          body: JSON.stringify({
            items: draftLines.map((l) => ({
              menuItemId: l.item.id,
              qty: l.qty,
              note: l.note ?? "",
            })),
          }),
        },
      );
      setOrder(r.order);
      setDraft({});
      setShowDraft(false);
      setView("adisyon"); // back to the tab so the fresh items are visible
    } catch (e) {
      guard(e);
    } finally {
      setSending(false);
    }
  }

  async function closeTable(paymentMethod: string) {
    setClosing(true);
    setError(null);
    try {
      await api(`/api/v1/orders/table/${tableNumber}/close`, {
        method: "POST",
        body: JSON.stringify({ paymentMethod }),
      });
      // Clear any leftover draft and leave — the table is free now.
      try {
        localStorage.removeItem(draftKey);
      } catch {
        /* ignore */
      }
      router.replace(backHref);
    } catch (e) {
      setClosing(false);
      guard(e);
    }
  }

  if (!loaded && !error) {
    return (
      <main className="flex flex-1 items-center justify-center p-8">
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      </main>
    );
  }

  if (view === "menu") {
    return (
      <MenuView
        tableNumber={tableNumber}
        menu={menu ?? []}
        order={order}
        draft={draft}
        draftCount={draftCount}
        draftTotal={draftTotal}
        sending={sending}
        error={error}
        onBack={() => setView("adisyon")}
        onPick={setPicking}
        onOpenDraft={() => setShowDraft(true)}
        onSend={send}
        picking={picking}
        onClosePick={() => setPicking(null)}
        onAdd={(qty, note) => {
          if (picking) addToDraft(picking, qty, note);
          setPicking(null);
        }}
        showDraft={showDraft}
        draftLines={draftLines}
        onCloseDraft={() => setShowDraft(false)}
        onQty={setDraftQty}
      />
    );
  }

  return (
    <>
      <AdisyonView
        tableNumber={tableNumber}
        order={order}
        backHref={backHref}
        error={error}
        cashier={cashier}
        onNewOrder={() => setView("menu")}
        onPrint={() => order && printAdisyon(order, tableNumber)}
        onClose={() => setShowClose(true)}
      />
      {showClose && order && (
        <CloseModal
          order={order}
          tableNumber={tableNumber}
          closing={closing}
          onCancel={() => setShowClose(false)}
          onConfirm={closeTable}
        />
      )}
    </>
  );
}

// ---- Adisyon (default) view ----------------------------------------------

function AdisyonView({
  tableNumber,
  order,
  backHref,
  error,
  cashier,
  onNewOrder,
  onPrint,
  onClose,
}: {
  tableNumber: number;
  order: Order | null;
  backHref: string;
  error: string | null;
  cashier: boolean;
  onNewOrder: () => void;
  onPrint: () => void;
  onClose: () => void;
}) {
  const items = (order?.items ?? []).filter(
    (it) => it.status !== "voided" && it.status !== "refunded",
  );
  const hasOrder = items.length > 0;
  const kdvLines = Object.entries(order?.kdvBreakdown ?? {}).filter(
    ([, v]) => v > 0,
  );

  return (
    <main className="flex flex-1 flex-col gap-5 p-4 pb-32">
      <header className="flex items-center gap-3">
        <Link
          href={backHref}
          className="flex h-10 w-10 items-center justify-center rounded-full border border-zinc-200 bg-white text-zinc-700 shadow-sm active:bg-zinc-50"
          aria-label="Geri"
        >
          ←
        </Link>
        <div>
          <span className="block text-xs uppercase tracking-wide text-zinc-500">
            Masa
          </span>
          <h1 className="text-2xl font-semibold tabular-nums text-zinc-900">
            {tableNumber}
          </h1>
        </div>
      </header>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {items.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-2 text-center">
          <p className="text-base text-zinc-500">
            Bu masada henüz sipariş yok.
          </p>
          <p className="text-sm text-zinc-400">
            Yeni Sipariş ile menüden ürün ekleyin.
          </p>
        </div>
      ) : (
        <>
          <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
            Verilen Siparişler
          </h2>
          <ul className="flex flex-col gap-2">
            {items.map((it) => (
              <li
                key={it.id}
                className="flex items-center gap-3 rounded-2xl border border-zinc-200 bg-white p-3 shadow-sm"
              >
                <span className="flex h-9 min-w-9 items-center justify-center rounded-full bg-zinc-100 px-2 text-base font-bold tabular-nums text-zinc-700">
                  {it.qty}
                </span>
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-base text-zinc-900">
                    {it.name}
                  </span>
                  {it.note && (
                    <span className="truncate text-xs text-amber-700">
                      {it.note}
                    </span>
                  )}
                </div>
                <span className="tabular-nums text-zinc-900">
                  {formatTRY(it.unitPrice * it.qty)}
                </span>
              </li>
            ))}
          </ul>

          <div className="mt-1 rounded-2xl border border-zinc-200 bg-white p-4 shadow-sm">
            {kdvLines.map(([rate, amt]) => (
              <div
                key={rate}
                className="flex items-baseline justify-between text-sm text-zinc-500"
              >
                <span>KDV %{rate}</span>
                <span className="tabular-nums">{formatTRY(amt)}</span>
              </div>
            ))}
            <div className="mt-2 flex items-baseline justify-between border-t border-zinc-100 pt-2">
              <span className="text-base font-semibold text-zinc-700">
                Toplam
              </span>
              <span className="text-2xl font-bold tabular-nums text-zinc-900">
                {formatTRY(order?.grandTotal ?? 0)}
              </span>
            </div>
          </div>
        </>
      )}

      {/* Actions */}
      <div className="fixed inset-x-0 bottom-0 z-30 border-t border-zinc-200 bg-white p-3 shadow-[0_-4px_12px_rgba(0,0,0,0.06)]">
        <div className="mx-auto flex max-w-3xl flex-col gap-2">
          {cashier && hasOrder && (
            <div className="flex gap-2">
              <button
                onClick={onPrint}
                className="flex-1 rounded-full border border-zinc-300 bg-white py-3.5 text-base font-semibold text-zinc-700 active:bg-zinc-50"
              >
                Adisyon Bas
              </button>
              <button
                onClick={onClose}
                className="flex-1 rounded-full bg-green-700 py-3.5 text-base font-semibold text-white active:bg-green-800"
              >
                Masa Kapat
              </button>
            </div>
          )}
          <button
            onClick={onNewOrder}
            className="w-full rounded-full bg-amber-700 py-3.5 text-base font-semibold text-white shadow-sm active:bg-amber-800"
          >
            + Yeni Sipariş
          </button>
        </div>
      </div>
    </main>
  );
}

// ---- Close table modal (payment) -----------------------------------------

function CloseModal({
  order,
  tableNumber,
  closing,
  onCancel,
  onConfirm,
}: {
  order: Order;
  tableNumber: number;
  closing: boolean;
  onCancel: () => void;
  onConfirm: (paymentMethod: string) => void;
}) {
  const [method, setMethod] = useState("nakit");
  const methods = [
    { key: "nakit", label: "Nakit" },
    { key: "kart", label: "Kart" },
  ];
  return (
    <div
      className="fixed inset-0 z-[60] flex items-end justify-center bg-black/40 sm:items-center"
      onClick={onCancel}
    >
      <div
        className="w-full max-w-md rounded-t-3xl bg-white p-5 sm:rounded-3xl"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="text-lg font-semibold text-zinc-900">
          Masa {tableNumber} Kapat
        </h3>
        <div className="mt-4 flex items-baseline justify-between rounded-2xl bg-zinc-50 px-4 py-3">
          <span className="text-sm text-zinc-600">Toplam</span>
          <span className="text-2xl font-bold tabular-nums text-zinc-900">
            {formatTRY(order.grandTotal)}
          </span>
        </div>

        <span className="mt-5 mb-2 block text-sm font-medium text-zinc-700">
          Ödeme yöntemi
        </span>
        <div className="flex gap-2">
          {methods.map((m) => (
            <button
              key={m.key}
              onClick={() => setMethod(m.key)}
              className={`flex-1 rounded-xl py-3 text-base font-semibold active:opacity-90 ${
                method === m.key
                  ? "bg-amber-700 text-white"
                  : "bg-zinc-100 text-zinc-700"
              }`}
            >
              {m.label}
            </button>
          ))}
        </div>

        <div className="mt-6 flex gap-3">
          <button
            onClick={onCancel}
            disabled={closing}
            className="flex-1 rounded-xl bg-zinc-200 py-3.5 text-base font-semibold text-zinc-700 active:bg-zinc-300 disabled:opacity-50"
          >
            Vazgeç
          </button>
          <button
            onClick={() => onConfirm(method)}
            disabled={closing}
            className="flex-[2] rounded-xl bg-green-700 py-3.5 text-base font-semibold text-white active:bg-green-800 disabled:opacity-50"
          >
            {closing ? "Kapatılıyor..." : "Hesabı Kapat"}
          </button>
        </div>
      </div>
    </div>
  );
}

// ---- Print adisyon (browser print — NOT the thermal bridge) --------------

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

function printAdisyon(order: Order, tableNumber: number) {
  const items = order.items.filter(
    (it) => it.status !== "voided" && it.status !== "refunded",
  );
  const now = new Date().toLocaleString("tr-TR", {
    timeZone: "Europe/Istanbul",
  });
  const rows = items
    .map(
      (it) =>
        `<tr><td class="q">${it.qty}×</td><td class="n">${escapeHtml(
          it.name,
        )}</td><td class="r">${formatTRY(it.unitPrice * it.qty)}</td></tr>`,
    )
    .join("");
  const kdv = Object.entries(order.kdvBreakdown ?? {})
    .filter(([, v]) => v > 0)
    .map(
      ([rate, v]) =>
        `<div class="row sm"><span>KDV %${rate}</span><span>${formatTRY(
          v,
        )}</span></div>`,
    )
    .join("");

  const html = `<!doctype html><html lang="tr"><head><meta charset="utf-8">
<title>Adisyon - Masa ${tableNumber}</title>
<style>
  * { box-sizing: border-box; }
  body { font-family: -apple-system, system-ui, sans-serif; margin: 0; padding: 12px; color: #111; width: 300px; }
  .logo { display: block; margin: 0 auto 4px; width: 190px; max-width: 80%; }
  .sub { text-align: center; font-size: 11px; color: #555; margin-bottom: 10px; }
  hr { border: none; border-top: 1px dashed #999; margin: 8px 0; }
  table { width: 100%; border-collapse: collapse; font-size: 12px; }
  td { padding: 3px 0; vertical-align: top; }
  td.q { width: 28px; font-weight: 700; }
  td.r { text-align: right; white-space: nowrap; font-variant-numeric: tabular-nums; }
  .row { display: flex; justify-content: space-between; font-size: 12px; padding: 2px 0; }
  .row.sm { color: #555; }
  .total { font-size: 16px; font-weight: 800; }
  .foot { text-align: center; font-size: 11px; color: #555; margin-top: 12px; }
</style></head>
<body onload="window.print()">
  <img class="logo" src="${ADISYON_LOGO}" alt="Gün Güzelbahçe">
  <div class="sub">Masa ${tableNumber} · ${now}</div>
  <hr>
  <table>${rows}</table>
  <hr>
  ${kdv}
  <div class="row total"><span>TOPLAM</span><span>${formatTRY(
    order.grandTotal,
  )}</span></div>
  <div class="foot">Afiyet olsun · Bu bir adisyondur, mali belge değildir.</div>
</body></html>`;

  const w = window.open("", "_blank", "width=340,height=640");
  if (!w) return;
  w.document.write(html);
  w.document.close();
}

// ---- Menu view (add items) -----------------------------------------------

function MenuView({
  tableNumber,
  menu,
  order,
  draft,
  draftCount,
  draftTotal,
  sending,
  error,
  onBack,
  onPick,
  onOpenDraft,
  onSend,
  picking,
  onClosePick,
  onAdd,
  showDraft,
  draftLines,
  onCloseDraft,
  onQty,
}: {
  tableNumber: number;
  menu: CategoryWithItems[];
  order: Order | null;
  draft: Record<string, DraftLine>;
  draftCount: number;
  draftTotal: number;
  sending: boolean;
  error: string | null;
  onBack: () => void;
  onPick: (it: MenuItem) => void;
  onOpenDraft: () => void;
  onSend: () => void;
  picking: MenuItem | null;
  onClosePick: () => void;
  onAdd: (qty: number, note?: string) => void;
  showDraft: boolean;
  draftLines: DraftLine[];
  onCloseDraft: () => void;
  onQty: (itemId: string, qty: number) => void;
}) {
  return (
    <main className="flex flex-1 flex-col gap-5 p-4 pb-32">
      <header className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button
            onClick={onBack}
            className="flex h-10 w-10 items-center justify-center rounded-full border border-zinc-200 bg-white text-zinc-700 shadow-sm active:bg-zinc-50"
            aria-label="Adisyona dön"
          >
            ←
          </button>
          <div>
            <span className="block text-xs uppercase tracking-wide text-zinc-500">
              Masa {tableNumber} · Yeni Sipariş
            </span>
            <h1 className="text-xl font-semibold text-zinc-900">Menü</h1>
          </div>
        </div>
        {order && order.items.length > 0 && (
          <div className="flex flex-col items-end">
            <span className="text-[10px] uppercase tracking-wide text-zinc-500">
              Adisyon
            </span>
            <span className="text-base font-bold tabular-nums text-zinc-900">
              {formatTRY(order.grandTotal)}
            </span>
          </div>
        )}
      </header>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      <nav className="flex gap-2 overflow-x-auto pb-1">
        {menu.map((c) => (
          <a
            key={c.id}
            href={`#cat-${c.id}`}
            className="shrink-0 rounded-full border border-zinc-200 bg-white px-4 py-2 text-sm font-medium text-zinc-700 shadow-sm active:bg-zinc-50"
          >
            {c.name}
          </a>
        ))}
      </nav>

      <div className="flex flex-col gap-8">
        {menu.map((cat) => (
          <section
            key={cat.id}
            id={`cat-${cat.id}`}
            className="flex scroll-mt-4 flex-col gap-3"
          >
            <h2 className="text-lg font-semibold text-zinc-900">{cat.name}</h2>
            {cat.items.length === 0 ? (
              <p className="text-sm text-zinc-500">Bu kategoride urun yok.</p>
            ) : (
              <ul className="grid gap-2 sm:grid-cols-2">
                {cat.items.map((it) => {
                  const inDraft = draft[it.id]?.qty ?? 0;
                  return (
                    <li key={it.id}>
                      <button
                        onClick={() => onPick(it)}
                        className="flex min-h-[72px] w-full items-center justify-between gap-3 rounded-2xl border border-zinc-200 bg-white px-4 py-3 text-left shadow-sm active:bg-zinc-50"
                      >
                        <div className="flex flex-col">
                          <span className="text-base font-medium text-zinc-900">
                            {it.name}
                          </span>
                          <span className="mt-0.5 text-xs text-zinc-500">
                            KDV %{it.kdvOrani}
                          </span>
                        </div>
                        <div className="flex items-center gap-2">
                          {inDraft > 0 && (
                            <span className="flex h-7 min-w-7 items-center justify-center rounded-full bg-amber-700 px-2 text-sm font-bold text-white">
                              {inDraft}
                            </span>
                          )}
                          <span className="text-base font-semibold tabular-nums text-zinc-900">
                            {formatTRY(it.price)}
                          </span>
                        </div>
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </section>
        ))}
      </div>

      {draftLines.length > 0 && (
        <div className="fixed inset-x-0 bottom-0 z-30 border-t border-zinc-200 bg-white p-3 shadow-[0_-4px_12px_rgba(0,0,0,0.06)]">
          <div className="mx-auto flex max-w-3xl items-center gap-3">
            <button
              onClick={onOpenDraft}
              className="flex flex-1 flex-col text-left"
            >
              <span className="text-xs text-zinc-500">
                Sepet · {draftCount} ürün
              </span>
              <span className="text-lg font-bold tabular-nums text-zinc-900">
                {formatTRY(draftTotal)}
              </span>
            </button>
            <button
              onClick={onSend}
              disabled={sending}
              className="rounded-full bg-amber-700 px-6 py-3 text-base font-semibold text-white shadow-sm active:bg-amber-800 disabled:opacity-50"
            >
              {sending ? "Gönderiliyor..." : "Mutfağa Gönder"}
            </button>
          </div>
        </div>
      )}

      {picking && (
        <QtyModal item={picking} onClose={onClosePick} onAdd={onAdd} />
      )}

      {showDraft && (
        <DraftSheet
          lines={draftLines}
          total={draftTotal}
          sending={sending}
          onClose={onCloseDraft}
          onQty={onQty}
          onSend={onSend}
        />
      )}
    </main>
  );
}

// ---- Quantity picker (stepper, no system keyboard) -----------------------

function QtyModal({
  item,
  onClose,
  onAdd,
}: {
  item: MenuItem;
  onClose: () => void;
  onAdd: (qty: number, note?: string) => void;
}) {
  const [qty, setQty] = useState(1);
  const [note, setNote] = useState("");

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 sm:items-center"
      onClick={onClose}
    >
      <div
        className="w-full max-w-md rounded-t-3xl bg-white p-5 sm:rounded-3xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-baseline justify-between">
          <h3 className="text-lg font-semibold text-zinc-900">{item.name}</h3>
          <span className="text-base font-semibold tabular-nums text-zinc-900">
            {formatTRY(item.price)}
          </span>
        </div>

        <div className="mt-6 flex items-center justify-center gap-6">
          <button
            onClick={() => setQty((q) => Math.max(1, q - 1))}
            className="flex h-16 w-16 items-center justify-center rounded-2xl bg-zinc-100 text-3xl font-bold text-zinc-700 active:bg-zinc-200"
            aria-label="Azalt"
          >
            −
          </button>
          <span className="w-16 text-center text-4xl font-bold tabular-nums text-zinc-900">
            {qty}
          </span>
          <button
            onClick={() => setQty((q) => q + 1)}
            className="flex h-16 w-16 items-center justify-center rounded-2xl bg-amber-700 text-3xl font-bold text-white active:bg-amber-800"
            aria-label="Arttır"
          >
            +
          </button>
        </div>

        <input
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder="Not (ops. — az pişmiş, sogansiz...)"
          className="mt-6 w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
        />

        <div className="mt-5 flex gap-3">
          <button
            onClick={onClose}
            className="flex-1 rounded-xl bg-zinc-200 py-3.5 text-base font-semibold text-zinc-700 active:bg-zinc-300"
          >
            İptal
          </button>
          <button
            onClick={() => onAdd(qty, note.trim() || undefined)}
            className="flex-[2] rounded-xl bg-amber-700 py-3.5 text-base font-semibold text-white active:bg-amber-800"
          >
            Sepete Ekle · {formatTRY(item.price * qty)}
          </button>
        </div>
      </div>
    </div>
  );
}

// ---- Draft review sheet --------------------------------------------------

function DraftSheet({
  lines,
  total,
  sending,
  onClose,
  onQty,
  onSend,
}: {
  lines: DraftLine[];
  total: number;
  sending: boolean;
  onClose: () => void;
  onQty: (itemId: string, qty: number) => void;
  onSend: () => void;
}) {
  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-white">
      <header className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
        <button
          onClick={onClose}
          className="rounded-full px-3 py-2 text-sm font-medium text-zinc-600 active:bg-zinc-100"
        >
          Kapat
        </button>
        <h2 className="text-base font-semibold text-zinc-900">Sepet</h2>
        <span className="w-16" />
      </header>

      <div className="flex flex-1 flex-col gap-2 overflow-y-auto p-4">
        {lines.map((l) => (
          <div
            key={l.item.id}
            className="flex items-center gap-3 rounded-2xl border border-zinc-200 bg-white p-3 shadow-sm"
          >
            <div className="flex min-w-0 flex-1 flex-col">
              <span className="truncate text-base font-medium text-zinc-900">
                {l.item.name}
              </span>
              {l.note && (
                <span className="truncate text-xs text-amber-700">{l.note}</span>
              )}
              <span className="text-xs text-zinc-500">
                {formatTRY(l.item.price)}
              </span>
            </div>
            <div className="flex items-center gap-3">
              <button
                onClick={() => onQty(l.item.id, l.qty - 1)}
                className="flex h-10 w-10 items-center justify-center rounded-xl bg-zinc-100 text-2xl font-bold text-zinc-700 active:bg-zinc-200"
                aria-label="Azalt"
              >
                −
              </button>
              <span className="w-8 text-center text-lg font-bold tabular-nums">
                {l.qty}
              </span>
              <button
                onClick={() => onQty(l.item.id, l.qty + 1)}
                className="flex h-10 w-10 items-center justify-center rounded-xl bg-zinc-100 text-2xl font-bold text-zinc-700 active:bg-zinc-200"
                aria-label="Arttır"
              >
                +
              </button>
            </div>
            <span className="w-20 shrink-0 text-right text-base font-semibold tabular-nums text-zinc-900">
              {formatTRY(l.item.price * l.qty)}
            </span>
          </div>
        ))}
      </div>

      <div className="border-t border-zinc-200 p-4">
        <div className="mx-auto flex max-w-3xl items-center gap-3">
          <div className="flex flex-1 flex-col">
            <span className="text-xs text-zinc-500">Sepet toplamı</span>
            <span className="text-xl font-bold tabular-nums text-zinc-900">
              {formatTRY(total)}
            </span>
          </div>
          <button
            onClick={onSend}
            disabled={sending}
            className="rounded-full bg-amber-700 px-6 py-3 text-base font-semibold text-white shadow-sm active:bg-amber-800 disabled:opacity-50"
          >
            {sending ? "Gönderiliyor..." : "Mutfağa Gönder"}
          </button>
        </div>
      </div>
    </div>
  );
}
