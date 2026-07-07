"use client";

import { useEffect, useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { api, ApiError } from "@/lib/api";
import type { QRIssueResponse, Waiter, WaitersResponse } from "@/lib/types";

type Stage =
  | { kind: "picking" }
  | { kind: "issuing"; waiter: Waiter }
  | { kind: "ready"; waiter: Waiter; url: string; expiresAt: Date }
  | { kind: "error"; message: string };

export function IssueQRModal({ onClose }: { onClose: () => void }) {
  const [waiters, setWaiters] = useState<Waiter[] | null>(null);
  const [stage, setStage] = useState<Stage>({ kind: "picking" });
  const [secondsLeft, setSecondsLeft] = useState<number | null>(null);

  useEffect(() => {
    api<WaitersResponse>("/api/v1/waiters")
      .then((r) => setWaiters(r.waiters))
      .catch((e) =>
        setStage({
          kind: "error",
          message: e instanceof ApiError ? e.message : String(e),
        }),
      );
  }, []);

  useEffect(() => {
    if (stage.kind !== "ready") return;
    const id = setInterval(() => {
      const diff = Math.max(0, Math.floor((stage.expiresAt.getTime() - Date.now()) / 1000));
      setSecondsLeft(diff);
    }, 1000);
    return () => clearInterval(id);
  }, [stage]);

  async function issueFor(waiter: Waiter) {
    setStage({ kind: "issuing", waiter });
    try {
      const resp = await api<QRIssueResponse>("/api/v1/auth/qr/issue", {
        method: "POST",
        body: JSON.stringify({ waiterId: waiter.id }),
      });
      const url = `${window.location.origin}/garson/qr?token=${encodeURIComponent(resp.token)}`;
      setStage({
        kind: "ready",
        waiter,
        url,
        expiresAt: new Date(resp.expiresAt),
      });
    } catch (e) {
      setStage({
        kind: "error",
        message: e instanceof ApiError ? e.message : String(e),
      });
    }
  }

  return (
    <div className="fixed inset-0 z-30 flex items-center justify-center bg-black/40 p-4">
      <div className="flex w-full max-w-md flex-col gap-5 rounded-3xl bg-white p-6 shadow-xl">
        <header className="flex items-center justify-between">
          <h2 className="text-xl font-semibold text-zinc-900">Garson QR</h2>
          <button
            onClick={onClose}
            className="flex h-10 w-10 items-center justify-center rounded-full text-zinc-500 active:bg-zinc-100"
            aria-label="Kapat"
          >
            ×
          </button>
        </header>

        {stage.kind === "picking" && (
          <div className="flex flex-col gap-3">
            <p className="text-sm text-zinc-600">
              QR olusturmak istediginiz garsonu secin.
            </p>
            {waiters === null ? (
              <p className="text-sm text-zinc-500">Yukleniyor...</p>
            ) : waiters.length === 0 ? (
              <p className="text-sm text-zinc-500">Aktif garson yok.</p>
            ) : (
              <ul className="flex flex-col gap-2">
                {waiters.map((w) => (
                  <li key={w.id}>
                    <button
                      onClick={() => issueFor(w)}
                      className="flex min-h-[56px] w-full items-center justify-between rounded-2xl border border-zinc-200 bg-white px-4 text-left text-base font-medium text-zinc-900 shadow-sm active:bg-zinc-50"
                    >
                      {w.name}
                      <span className="text-sm text-zinc-400">→</span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}

        {stage.kind === "issuing" && (
          <p className="text-center text-sm text-zinc-500">QR olusturuluyor...</p>
        )}

        {stage.kind === "ready" && (
          <div className="flex flex-col items-center gap-4">
            <p className="text-sm text-zinc-600">
              <span className="font-semibold text-zinc-900">{stage.waiter.name}</span>
              {" "}icin QR — telefonun ile okut.
            </p>
            <div className="rounded-2xl border border-zinc-200 bg-white p-4">
              <QRCodeSVG value={stage.url} size={240} level="M" />
            </div>
            <p className="text-xs text-zinc-500 break-all text-center">{stage.url}</p>
            <p className="text-xs text-zinc-500">
              {secondsLeft === null
                ? ""
                : secondsLeft > 0
                ? `${secondsLeft} sn icinde geçersiz olacak`
                : "QR geçersiz oldu, yeniden olusturun"}
            </p>
            <button
              onClick={() => setStage({ kind: "picking" })}
              className="rounded-full border border-zinc-200 bg-white px-4 py-2 text-sm font-medium text-zinc-700 active:bg-zinc-50"
            >
              Baska garson sec
            </button>
          </div>
        )}

        {stage.kind === "error" && (
          <div className="flex flex-col gap-3">
            <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
              {stage.message}
            </div>
            <button
              onClick={() => setStage({ kind: "picking" })}
              className="rounded-full border border-zinc-200 bg-white px-4 py-2 text-sm font-medium text-zinc-700 active:bg-zinc-50"
            >
              Tekrar dene
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
