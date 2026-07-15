"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatTRY } from "@/lib/money";
import type { SalesReport, TimeSeriesReport, BucketPoint } from "@/lib/types";

const TZ = "Europe/Istanbul";
type Bucket = "hour" | "day" | "month";
type Tab = "bugun" | "hafta" | "ay" | "yil";

const TABS: { key: Tab; label: string }[] = [
  { key: "bugun", label: "Bugün" },
  { key: "hafta", label: "Bu Hafta" },
  { key: "ay", label: "Bu Ay" },
  { key: "yil", label: "Bu Yıl" },
];

function todayISO(): string {
  return new Date().toLocaleDateString("en-CA", { timeZone: TZ });
}

// Treat the calendar date as UTC midnight so day math never drifts by timezone.
function isoAdd(iso: string, days: number): string {
  const d = new Date(`${iso}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString().slice(0, 10);
}

function rangeFor(tab: Tab): { from: string; to: string; bucket: Bucket } {
  const today = todayISO();
  if (tab === "bugun") return { from: today, to: isoAdd(today, 1), bucket: "hour" };
  if (tab === "hafta") {
    const dow = (new Date(`${today}T00:00:00Z`).getUTCDay() + 6) % 7; // Mon=0
    const from = isoAdd(today, -dow);
    return { from, to: isoAdd(from, 7), bucket: "day" };
  }
  if (tab === "ay") {
    const from = `${today.slice(0, 7)}-01`;
    const [y, m] = from.split("-").map(Number);
    const to = m === 12 ? `${y + 1}-01-01` : `${y}-${String(m + 1).padStart(2, "0")}-01`;
    return { from, to, bucket: "day" };
  }
  const y = Number(today.slice(0, 4));
  return { from: `${y}-01-01`, to: `${y + 1}-01-01`, bucket: "month" };
}

function axisLabel(iso: string, bucket: Bucket): string {
  const d = new Date(iso);
  if (bucket === "hour")
    return d.toLocaleTimeString("tr-TR", { timeZone: TZ, hour: "2-digit" });
  if (bucket === "month")
    return d.toLocaleDateString("tr-TR", { timeZone: TZ, month: "short" });
  return d.toLocaleDateString("tr-TR", { timeZone: TZ, day: "2-digit" });
}

function fullLabel(iso: string, bucket: Bucket): string {
  const d = new Date(iso);
  if (bucket === "hour")
    return d.toLocaleTimeString("tr-TR", {
      timeZone: TZ,
      hour: "2-digit",
      minute: "2-digit",
    });
  if (bucket === "month")
    return d.toLocaleDateString("tr-TR", { timeZone: TZ, month: "long" });
  return d.toLocaleDateString("tr-TR", {
    timeZone: TZ,
    day: "2-digit",
    month: "short",
  });
}

export default function ErpGrafiklerPage() {
  const [tab, setTab] = useState<Tab>("bugun");
  const [sales, setSales] = useState<SalesReport | null>(null);
  const [series, setSeries] = useState<TimeSeriesReport | null>(null);
  const [error, setError] = useState<string | null>(null);

  const rng = useMemo(() => rangeFor(tab), [tab]);

  useEffect(() => {
    setSales(null);
    setSeries(null);
    setError(null);
    const qs = `from=${rng.from}&to=${rng.to}`;
    api<SalesReport>(`/api/v1/reports/sales?${qs}`)
      .then(setSales)
      .catch((e) => setError(e instanceof Error ? e.message : String(e)));
    api<TimeSeriesReport>(`/api/v1/reports/timeseries?${qs}&bucket=${rng.bucket}`)
      .then(setSeries)
      .catch((e) => setError(e instanceof Error ? e.message : String(e)));
  }, [rng.from, rng.to, rng.bucket]);

  const loading = sales === null || series === null;

  return (
    <main className="flex flex-1 flex-col gap-5 p-4 pb-24">
      <header className="flex items-center gap-3">
        <Link
          href="/erp"
          className="flex h-10 w-10 items-center justify-center rounded-full border border-zinc-200 bg-white text-zinc-700 shadow-sm active:bg-zinc-50"
          aria-label="Geri"
        >
          ←
        </Link>
        <h1 className="text-2xl font-semibold text-zinc-900">Grafikler</h1>
      </header>

      {/* Tabs */}
      <div className="grid grid-cols-4 gap-1 rounded-full border border-zinc-200 bg-white p-1 shadow-sm">
        {TABS.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`rounded-full py-2.5 text-sm font-semibold transition-colors ${
              tab === t.key
                ? "bg-amber-700 text-white"
                : "text-zinc-600 active:bg-zinc-50"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {loading ? (
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      ) : (
        <>
          {/* Hero stats */}
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <Stat label="Ciro" value={formatTRY(sales.revenue)} tone="green" />
            <Stat label="Gider" value={formatTRY(sales.expense)} tone="red" />
            <Stat
              label="Net kâr"
              value={formatTRY(sales.profit)}
              tone={sales.profit >= 0 ? "green" : "red"}
            />
            <Stat label="Kişi" value={String(sales.guests)} tone="sky" />
          </div>

          <TrendChart points={series.points} bucket={series.bucket} />
          <GuestsChart
            points={series.points}
            bucket={series.bucket}
            total={series.guests}
          />
          <PaymentChart payment={sales.payment} />
          <TopItemsChart items={sales.topItems} />
        </>
      )}
    </main>
  );
}

function Stat({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone: "green" | "red" | "sky";
}) {
  const color =
    tone === "green"
      ? "text-green-700"
      : tone === "red"
        ? "text-red-700"
        : "text-sky-700";
  return (
    <div className="flex flex-col rounded-2xl border border-zinc-200 bg-white px-4 py-3 shadow-sm">
      <span className="text-xs font-medium text-zinc-500">{label}</span>
      <span className={`text-xl font-bold tabular-nums ${color}`}>{value}</span>
    </div>
  );
}

function Card({
  title,
  legend,
  children,
}: {
  title: string;
  legend?: [string, string][];
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-2xl border border-zinc-200 bg-white p-4 shadow-sm">
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-zinc-700">{title}</h2>
        {legend && (
          <div className="flex items-center gap-3">
            {legend.map(([name, cls]) => (
              <span key={name} className="flex items-center gap-1.5">
                <span className={`h-2.5 w-2.5 rounded-sm ${cls}`} />
                <span className="text-xs text-zinc-500">{name}</span>
              </span>
            ))}
          </div>
        )}
      </div>
      {children}
    </div>
  );
}

function Bar({
  value,
  max,
  className,
}: {
  value: number;
  max: number;
  className: string;
}) {
  const h = value <= 0 ? 0 : Math.max(3, (value / max) * 100);
  return (
    <div
      className={`w-2 rounded-t ${className}`}
      style={{ height: `${h}%` }}
    />
  );
}

function TrendChart({ points, bucket }: { points: BucketPoint[]; bucket: Bucket }) {
  const max = Math.max(1, ...points.map((p) => Math.max(p.revenue, p.expense)));
  const peak = points.reduce<BucketPoint | null>(
    (a, p) => (a && a.revenue >= p.revenue ? a : p),
    null,
  );
  const empty = points.every((p) => p.revenue === 0 && p.expense === 0);

  return (
    <Card
      title="Gelir – Gider"
      legend={[
        ["Gelir", "bg-green-600"],
        ["Gider", "bg-red-500"],
      ]}
    >
      {empty ? (
        <p className="py-6 text-center text-sm text-zinc-400">Veri yok.</p>
      ) : (
        <>
          <div className="flex h-40 items-end gap-1 overflow-x-auto pb-1">
            {points.map((p, i) => (
              <div
                key={i}
                className="flex min-w-[14px] flex-1 flex-col items-center gap-1"
              >
                <div className="flex h-32 w-full items-end justify-center gap-[2px]">
                  <Bar value={p.revenue} max={max} className="bg-green-600" />
                  <Bar value={p.expense} max={max} className="bg-red-500" />
                </div>
                <span className="text-[9px] text-zinc-400">
                  {axisLabel(p.start, bucket)}
                </span>
              </div>
            ))}
          </div>
          {peak && peak.revenue > 0 && (
            <p className="mt-2 text-xs text-zinc-400">
              En yüksek: {formatTRY(peak.revenue)} · {fullLabel(peak.start, bucket)}
            </p>
          )}
        </>
      )}
    </Card>
  );
}

function GuestsChart({
  points,
  bucket,
  total,
}: {
  points: BucketPoint[];
  bucket: Bucket;
  total: number;
}) {
  const max = Math.max(1, ...points.map((p) => p.guests));
  return (
    <Card title={`Kişi sayısı · toplam ${total}`}>
      {total === 0 ? (
        <p className="py-6 text-center text-sm text-zinc-400">
          Bu dönem fiks menü kişisi yok.
        </p>
      ) : (
        <div className="flex h-40 items-end gap-1 overflow-x-auto pb-1">
          {points.map((p, i) => (
            <div
              key={i}
              className="flex min-w-[14px] flex-1 flex-col items-center gap-1"
            >
              <span className="h-3 text-[9px] font-semibold text-sky-700">
                {p.guests > 0 ? p.guests : ""}
              </span>
              <div className="flex h-28 w-full items-end justify-center">
                <div
                  className="w-3 rounded-t bg-sky-500"
                  style={{
                    height:
                      p.guests > 0
                        ? `${Math.max(4, (p.guests / max) * 100)}%`
                        : "0",
                  }}
                />
              </div>
              <span className="text-[9px] text-zinc-400">
                {axisLabel(p.start, bucket)}
              </span>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

const PAYMENT_LABELS: Record<string, string> = {
  nakit: "Nakit",
  kart: "Kart",
  "belirtilmemiş": "Belirtilmemiş",
};

function PaymentChart({ payment }: { payment: Record<string, number> }) {
  const entries = Object.entries(payment).sort((a, b) => b[1] - a[1]);
  const total = entries.reduce((s, [, v]) => s + v, 0);
  const max = Math.max(1, ...entries.map(([, v]) => v));

  return (
    <Card title="Ödeme dağılımı">
      {total === 0 ? (
        <p className="py-6 text-center text-sm text-zinc-400">Veri yok.</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {entries.map(([method, amount]) => (
            <li key={method} className="flex flex-col gap-1">
              <div className="flex items-baseline justify-between text-sm">
                <span className="font-medium text-zinc-700">
                  {PAYMENT_LABELS[method] ?? method}
                </span>
                <span className="tabular-nums text-zinc-900">
                  {formatTRY(amount)}
                  <span className="ml-1.5 text-xs text-zinc-400">
                    %{Math.round((amount / total) * 100)}
                  </span>
                </span>
              </div>
              <div className="h-2.5 w-full overflow-hidden rounded-full bg-zinc-100">
                <div
                  className="h-full rounded-full bg-amber-600"
                  style={{ width: `${(amount / max) * 100}%` }}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}

function TopItemsChart({
  items,
}: {
  items: { name: string; qty: number; revenue: number }[];
}) {
  const top = items.slice(0, 8);
  const max = Math.max(1, ...top.map((i) => i.qty));

  return (
    <Card title="En çok satan">
      {top.length === 0 ? (
        <p className="py-6 text-center text-sm text-zinc-400">Veri yok.</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {top.map((it) => (
            <li key={it.name} className="flex flex-col gap-1">
              <div className="flex items-baseline justify-between text-sm">
                <span className="truncate pr-2 font-medium text-zinc-700">
                  {it.name}
                </span>
                <span className="shrink-0 tabular-nums text-zinc-900">
                  {it.qty} adet
                  <span className="ml-1.5 text-xs text-zinc-400">
                    {formatTRY(it.revenue)}
                  </span>
                </span>
              </div>
              <div className="h-2.5 w-full overflow-hidden rounded-full bg-zinc-100">
                <div
                  className="h-full rounded-full bg-green-600"
                  style={{ width: `${(it.qty / max) * 100}%` }}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
