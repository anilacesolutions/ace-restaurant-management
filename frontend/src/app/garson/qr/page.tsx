"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { api, ApiError } from "@/lib/api";

type State =
  | { kind: "exchanging" }
  | { kind: "success" }
  | { kind: "error"; message: string };

function ExchangeFlow() {
  const router = useRouter();
  const params = useSearchParams();
  const token = params.get("token") ?? "";
  const [state, setState] = useState<State>({ kind: "exchanging" });

  useEffect(() => {
    if (!token) {
      setState({ kind: "error", message: "QR token bulunamadi." });
      return;
    }
    api("/api/v1/auth/qr/exchange", {
      method: "POST",
      body: JSON.stringify({ token }),
    })
      .then(() => {
        setState({ kind: "success" });
        router.replace("/garson");
      })
      .catch((e) => {
        setState({
          kind: "error",
          message: e instanceof ApiError ? e.message : String(e),
        });
      });
  }, [token, router]);

  if (state.kind === "exchanging") {
    return <p className="text-zinc-500">Giris yapiliyor...</p>;
  }
  if (state.kind === "success") {
    return <p className="text-zinc-500">Yonlendiriliyor...</p>;
  }
  return (
    <>
      <h1 className="text-xl font-semibold text-zinc-900">QR Gecersiz</h1>
      <p className="max-w-sm text-sm text-zinc-600">{state.message}</p>
      <p className="max-w-sm text-sm text-zinc-500">
        Kasadaki yoneticiden yeni bir QR isteyin.
      </p>
    </>
  );
}

export default function GarsonQRExchangePage() {
  return (
    <main className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center">
      <Suspense fallback={<p className="text-zinc-500">Yukleniyor...</p>}>
        <ExchangeFlow />
      </Suspense>
    </main>
  );
}
