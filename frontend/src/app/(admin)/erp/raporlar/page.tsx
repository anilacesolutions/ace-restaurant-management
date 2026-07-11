"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatTRY } from "@/lib/money";
import type { SalesReport } from "@/lib/types";

const TZ = "Europe/Istanbul";

function todayISO(): string {
  return new Date().toLocaleDateString("en-CA", { timeZone: TZ });
}

// Date math on YYYY-MM-DD via UTC-noon to dodge DST edges.
function parseYMD(s: string): Date {
  const [y, m, d] = s.split("-").map(Number);
  return new Date(Date.UTC(y, m - 1, d, 12));
}
function fmtYMD(d: Date): string {
  return d.toISOString().slice(0, 10);
}
function addDays(s: string, n: number): string {
  const d = parseYMD(s);
  d.setUTCDate(d.getUTCDate() + n);
  return fmtYMD(d);
}
function mondayOf(s: string): string {
  const d = parseYMD(s);
  const dow = d.getUTCDay(); // 0=Sun..6=Sat
  d.setUTCDate(d.getUTCDate() + (dow === 0 ? -6 : 1 - dow));
  return fmtYMD(d);
}
function firstOfMonth(s: string): string {
  const [y, m] = s.split("-");
  return `${y}-${m}-01`;
}
function firstOfNextMonth(s: string): string {
  const [y, m] = s.split("-").map(Number);
  return m === 12
    ? `${y + 1}-01-01`
    : `${y}-${String(m + 1).padStart(2, "0")}-01`;
}

type TabKey = "today" | "week" | "month" | "custom";

const TABS: { key: TabKey; label: string }[] = [
  { key: "today", label: "Bugün" },
  { key: "week", label: "Bu Hafta" },
  { key: "month", label: "Bu Ay" },
  { key: "custom", label: "Tarih Seç" },
];

export default function ErpRaporlarPage() {
  const [tab, setTab] = useState<TabKey>("today");
  const [customFrom, setCustomFrom] = useState(todayISO);
  const [customTo, setCustomTo] = useState(todayISO);
  const [report, setReport] = useState<SalesReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const range = useMemo(() => {
    const today = todayISO();
    switch (tab) {
      case "today":
        return { from: today, to: addDays(today, 1) };
      case "week": {
        const mon = mondayOf(today);
        return { from: mon, to: addDays(mon, 7) };
      }
      case "month":
        return { from: firstOfMonth(today), to: firstOfNextMonth(today) };
      case "custom":
        // inclusive end day → exclusive upper bound
        return { from: customFrom, to: addDays(customTo, 1) };
    }
  }, [tab, customFrom, customTo]);

  async function load(r: { from: string; to: string }) {
    setLoading(true);
    try {
      const rep = await api<SalesReport>(
        `/api/v1/reports/sales?from=${r.from}&to=${r.to}`,
      );
      setReport(rep);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load(range);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [range.from, range.to]);

  const avgTicket =
    report && report.orderCount > 0
      ? Math.round(report.revenue / report.orderCount)
      : 0;

  const nakit = report?.payment?.["nakit"] ?? 0;
  const kart = report?.payment?.["kart"] ?? 0;
  const otherPay = report
    ? report.revenue - nakit - kart
    : 0;

  const kdvLines = Object.entries(report?.kdv ?? {})
    .filter(([, v]) => v > 0)
    .sort((a, b) => Number(a[0]) - Number(b[0]));

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
        <h1 className="text-2xl font-semibold text-zinc-900">Raporlar</h1>
      </header>

      {/* Tabs */}
      <div className="flex gap-2 overflow-x-auto pb-1">
        {TABS.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`shrink-0 rounded-full px-4 py-2 text-sm font-semibold active:opacity-90 ${
              tab === t.key
                ? "bg-amber-700 text-white"
                : "bg-zinc-100 text-zinc-700"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === "custom" && (
        <div className="flex items-center gap-2">
          <input
            type="date"
            value={customFrom}
            onChange={(e) => setCustomFrom(e.target.value)}
            className="flex-1 rounded-xl border border-zinc-300 bg-white px-3 py-2.5 text-sm text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
          <span className="text-zinc-400">—</span>
          <input
            type="date"
            value={customTo}
            onChange={(e) => setCustomTo(e.target.value)}
            className="flex-1 rounded-xl border border-zinc-300 bg-white px-3 py-2.5 text-sm text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </div>
      )}

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {loading && report === null ? (
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      ) : report === null ? null : (
        <div className="flex flex-col gap-5">
          {/* Ciro */}
          <div className="rounded-2xl border border-amber-200 bg-amber-50 p-5">
            <span className="text-sm font-medium text-amber-900">Ciro</span>
            <p className="mt-1 text-4xl font-bold tabular-nums text-amber-900">
              {formatTRY(report.revenue)}
            </p>
            <div className="mt-3 flex gap-6 border-t border-amber-200 pt-3 text-sm">
              <div className="flex flex-col">
                <span className="text-amber-900/70">Hesap</span>
                <span className="font-semibold tabular-nums text-amber-900">
                  {report.orderCount}
                </span>
              </div>
              <div className="flex flex-col">
                <span className="text-amber-900/70">Ort. adisyon</span>
                <span className="font-semibold tabular-nums text-amber-900">
                  {formatTRY(avgTicket)}
                </span>
              </div>
              <div className="flex flex-col">
                <span className="text-amber-900/70">Matrah (net)</span>
                <span className="font-semibold tabular-nums text-amber-900">
                  {formatTRY(report.net)}
                </span>
              </div>
            </div>
          </div>

          {report.orderCount === 0 ? (
            <p className="text-sm text-zinc-500">
              Bu aralıkta kapatılmış hesap yok.
            </p>
          ) : (
            <>
              {/* Nakit / Kart */}
              <Section title="Ödeme">
                <div className="grid grid-cols-2 gap-3">
                  <PayCell label="Nakit" value={nakit} tone="green" />
                  <PayCell label="Kart" value={kart} tone="sky" />
                </div>
                {otherPay > 0 && (
                  <p className="mt-2 text-xs text-zinc-500">
                    Belirtilmemiş: {formatTRY(otherPay)}
                  </p>
                )}
              </Section>

              {/* KDV */}
              <Section title="KDV Kırılımı">
                <div className="flex flex-col gap-1.5">
                  {kdvLines.length === 0 ? (
                    <p className="text-sm text-zinc-400">KDV verisi yok.</p>
                  ) : (
                    kdvLines.map(([rate, amt]) => (
                      <Row key={rate} label={`KDV %${rate}`} value={amt} />
                    ))
                  )}
                  {report.otv > 0 && <Row label="ÖTV" value={report.otv} />}
                </div>
              </Section>

              {/* En çok satan */}
              <Section title="En Çok Satan">
                <ul className="flex flex-col gap-2">
                  {report.topItems.map((it, i) => (
                    <li
                      key={it.name}
                      className="flex items-center gap-3 rounded-xl border border-zinc-100 bg-white p-3 shadow-sm"
                    >
                      <span className="w-5 shrink-0 text-center text-sm font-bold text-zinc-400">
                        {i + 1}
                      </span>
                      <span className="min-w-0 flex-1 truncate text-base text-zinc-900">
                        {it.name}
                      </span>
                      <span className="shrink-0 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-bold tabular-nums text-amber-800">
                        {it.qty} adet
                      </span>
                      <span className="w-24 shrink-0 text-right text-sm font-semibold tabular-nums text-zinc-700">
                        {formatTRY(it.revenue)}
                      </span>
                    </li>
                  ))}
                </ul>
              </Section>
            </>
          )}
        </div>
      )}
    </main>
  );
}

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-2">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
        {title}
      </h2>
      {children}
    </div>
  );
}

function PayCell({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: "green" | "sky";
}) {
  const color = tone === "green" ? "text-green-700" : "text-sky-700";
  return (
    <div className="rounded-2xl border border-zinc-200 bg-white p-4 shadow-sm">
      <span className="text-sm text-zinc-500">{label}</span>
      <p className={`mt-1 text-xl font-bold tabular-nums ${color}`}>
        {formatTRY(value)}
      </p>
    </div>
  );
}

function Row({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex items-baseline justify-between rounded-xl border border-zinc-100 bg-white px-4 py-2.5 shadow-sm">
      <span className="text-sm text-zinc-600">{label}</span>
      <span className="text-base font-semibold tabular-nums text-zinc-900">
        {formatTRY(value)}
      </span>
    </div>
  );
}
