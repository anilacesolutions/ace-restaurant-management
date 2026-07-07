"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    // Browsers can autofill the DOM input without firing React's onChange,
    // leaving our state empty even though the user sees their credentials.
    // Read directly from the form so autofill works.
    const form = e.currentTarget as HTMLFormElement;
    const u = ((form.elements.namedItem("username") as HTMLInputElement)?.value || username).trim();
    const p = (form.elements.namedItem("password") as HTMLInputElement)?.value || password;

    if (!u || !p) {
      setError("Kullanici adi ve sifre gerekli.");
      return;
    }

    setBusy(true);
    try {
      await api("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ username: u, password: p }),
      });
      router.replace("/kasa");
    } catch (err) {
      if (err instanceof ApiError) setError(err.message);
      else setError(String(err));
      setBusy(false);
    }
  }

  return (
    <main className="flex flex-1 items-center justify-center p-6">
      <form
        method="post"
        action="javascript:void(0)"
        onSubmit={onSubmit}
        className="flex w-full max-w-sm flex-col gap-5 rounded-2xl border border-zinc-200 bg-white p-8 shadow-sm"
      >
        <header className="flex flex-col items-center gap-1">
          <h1 className="text-2xl font-semibold text-zinc-900">Restoran</h1>
          <p className="text-sm text-zinc-500">Yonetici Girisi</p>
        </header>

        <label className="flex flex-col gap-1.5">
          <span className="text-sm font-medium text-zinc-700">Kullanici Adi</span>
          <input
            name="username"
            type="text"
            autoCapitalize="none"
            autoComplete="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            disabled={busy}
            className="min-h-[48px] rounded-xl border border-zinc-200 bg-zinc-50 px-4 text-base text-zinc-900 outline-none focus:border-amber-700 focus:bg-white disabled:opacity-50"
          />
        </label>

        <label className="flex flex-col gap-1.5">
          <span className="text-sm font-medium text-zinc-700">Sifre</span>
          <input
            name="password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            disabled={busy}
            className="min-h-[48px] rounded-xl border border-zinc-200 bg-zinc-50 px-4 text-base text-zinc-900 outline-none focus:border-amber-700 focus:bg-white disabled:opacity-50"
          />
        </label>

        {error && (
          <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
            {error}
          </div>
        )}

        <button
          type="submit"
          disabled={busy}
          className="min-h-[52px] rounded-xl bg-amber-700 text-base font-semibold text-white shadow-sm active:bg-amber-800 disabled:opacity-50"
        >
          {busy ? "Giris yapiliyor..." : "Giris Yap"}
        </button>
      </form>
    </main>
  );
}
