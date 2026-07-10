"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import type { Waiter, WaitersResponse } from "@/lib/types";

export default function ErpGarsonlarPage() {
  const [waiters, setWaiters] = useState<Waiter[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<Waiter | "new" | null>(null);

  async function load() {
    try {
      const r = await api<WaitersResponse>("/api/v1/waiters/all");
      setWaiters(r.waiters);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    load();
  }, []);

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
          <h1 className="text-2xl font-semibold text-zinc-900">
            Garson Yonetimi
          </h1>
        </div>
        <button
          onClick={() => setEditing("new")}
          className="rounded-full bg-amber-700 px-5 py-2.5 text-sm font-semibold text-white shadow-sm active:bg-amber-800"
        >
          + Yeni Garson
        </button>
      </header>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {waiters === null ? (
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      ) : waiters.length === 0 ? (
        <p className="text-sm text-zinc-500">
          Henuz garson yok. + Yeni Garson ile ekleyin.
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {waiters.map((w) => (
            <li key={w.id}>
              <button
                onClick={() => setEditing(w)}
                className="flex w-full items-center gap-3 rounded-2xl border border-zinc-200 bg-white p-4 text-left shadow-sm active:bg-zinc-50"
              >
                <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-amber-100 text-lg font-semibold text-amber-800">
                  {w.name.slice(0, 1).toUpperCase()}
                </div>
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="flex items-center gap-2 truncate text-base font-semibold text-zinc-900">
                    {w.name}
                    {!w.active && (
                      <span className="shrink-0 rounded-full bg-zinc-200 px-2 py-0.5 text-[10px] font-medium text-zinc-600">
                        pasif
                      </span>
                    )}
                  </span>
                  {w.phone && (
                    <span className="mt-0.5 text-sm tabular-nums text-zinc-500">
                      {w.phone}
                    </span>
                  )}
                </div>
                <span className="shrink-0 text-zinc-300">›</span>
              </button>
            </li>
          ))}
        </ul>
      )}

      {editing !== null && (
        <WaiterFormSheet
          waiter={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
          onSaved={async () => {
            setEditing(null);
            await load();
          }}
        />
      )}
    </main>
  );
}

// ---- Form sheet ----------------------------------------------------------

function WaiterFormSheet({
  waiter,
  onClose,
  onSaved,
}: {
  waiter: Waiter | null;
  onClose: () => void;
  onSaved: () => void | Promise<void>;
}) {
  const [name, setName] = useState(waiter?.name ?? "");
  const [phone, setPhone] = useState(waiter?.phone ?? "");
  const [active, setActive] = useState(waiter?.active ?? true);
  const [busy, setBusy] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function save() {
    setError(null);
    if (!name.trim()) {
      setError("Garson adi gir");
      return;
    }
    setBusy(true);
    try {
      const body = JSON.stringify({ name, phone, active });
      if (waiter) {
        await api(`/api/v1/waiters/${waiter.id}`, { method: "PATCH", body });
      } else {
        await api("/api/v1/waiters", { method: "POST", body });
      }
      await onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (!waiter) return;
    setError(null);
    setDeleting(true);
    try {
      await api(`/api/v1/waiters/${waiter.id}`, { method: "DELETE" });
      await onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setDeleting(false);
      setConfirmDelete(false);
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
        <h2 className="text-base font-semibold text-zinc-900">
          {waiter ? "Garsonu Duzenle" : "Yeni Garson"}
        </h2>
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

        <Field label="Ad Soyad">
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="orn. Ahmet Yilmaz"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        <Field label="Telefon (opsiyonel)">
          <input
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            inputMode="tel"
            placeholder="orn. 0532 000 00 00"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base tabular-nums text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        <Toggle
          label="Aktif (calisiyor)"
          desc="Kapatirsan garson QR listesinde gorunmez ama gecmisi durur. Isten ayrildiginda kullan."
          value={active}
          onChange={setActive}
        />

        {waiter && (
          <div className="mt-4 border-t border-zinc-200 pt-5">
            <button
              onClick={() => setConfirmDelete(true)}
              disabled={busy || deleting}
              className="w-full rounded-xl border border-red-300 bg-red-50 py-3 text-base font-semibold text-red-700 active:bg-red-100 disabled:opacity-50"
            >
              Garsonu Sil
            </button>
            <p className="mt-2 text-center text-xs text-zinc-400">
              Kalici siler. Sadece isten ayrildiysa &quot;Aktif&quot; kapatmak
              yeterli.
            </p>
          </div>
        )}
      </div>

      {confirmDelete && waiter && (
        <div className="fixed inset-0 z-[60] flex items-end justify-center bg-black/40 p-4 sm:items-center">
          <div className="w-full max-w-sm rounded-2xl bg-white p-5 shadow-xl">
            <h3 className="text-lg font-semibold text-zinc-900">Garsonu sil?</h3>
            <p className="mt-2 text-sm text-zinc-600">
              <strong>{waiter.name}</strong> kalici olarak silinecek. Bu islem
              geri alinamaz.
            </p>
            <div className="mt-5 flex gap-3">
              <button
                onClick={() => setConfirmDelete(false)}
                disabled={deleting}
                className="flex-1 rounded-xl border border-zinc-300 bg-white py-3 text-base font-semibold text-zinc-700 active:bg-zinc-50 disabled:opacity-50"
              >
                Vazgec
              </button>
              <button
                onClick={remove}
                disabled={deleting}
                className="flex-1 rounded-xl bg-red-600 py-3 text-base font-semibold text-white active:bg-red-700 disabled:opacity-50"
              >
                {deleting ? "Siliniyor..." : "Sil"}
              </button>
            </div>
          </div>
        </div>
      )}
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

function Toggle({
  label,
  desc,
  value,
  onChange,
}: {
  label: string;
  desc: string;
  value: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <button
      onClick={() => onChange(!value)}
      className="flex items-center justify-between gap-4 rounded-2xl border border-zinc-200 bg-white p-3 text-left shadow-sm active:bg-zinc-50"
    >
      <span className="flex flex-col">
        <span className="text-base font-medium text-zinc-900">{label}</span>
        <span className="mt-0.5 text-xs text-zinc-500">{desc}</span>
      </span>
      <span
        className={`flex h-7 w-12 shrink-0 items-center rounded-full p-0.5 transition-colors ${
          value ? "bg-amber-700" : "bg-zinc-300"
        }`}
      >
        <span
          className={`h-6 w-6 rounded-full bg-white shadow transition-transform ${
            value ? "translate-x-5" : "translate-x-0"
          }`}
        />
      </span>
    </button>
  );
}
