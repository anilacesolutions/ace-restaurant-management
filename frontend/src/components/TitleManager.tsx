"use client";

// The admin surfaces are client components, so they can't export per-page
// metadata. This keeps the browser tab title in sync with the current route
// (incl. sub-sections and the dynamic masa number) by setting document.title.
import { usePathname } from "next/navigation";
import { useEffect } from "react";

const LABELS: Record<string, string> = {
  "/kasa": "Kasa",
  "/erp": "Yönetim",
  "/erp/cari": "Cari",
  "/erp/menu": "Menü",
  "/erp/grafikler": "Grafikler",
  "/erp/raporlar": "Raporlar",
  "/erp/masalar": "Masalar",
  "/erp/garsonlar": "Garsonlar",
};

function titleFor(path: string): string {
  const masa = path.match(/^\/kasa\/masa\/(\d+)/);
  if (masa) return `Masa ${masa[1]}`;
  // Longest matching prefix wins (so /erp/cari beats /erp).
  for (const k of Object.keys(LABELS).sort((a, b) => b.length - a.length)) {
    if (path === k || path.startsWith(`${k}/`)) return LABELS[k];
  }
  return "Yönetim";
}

export function TitleManager() {
  const path = usePathname();
  useEffect(() => {
    document.title = `Gün Güzelbahçe | ${titleFor(path ?? "")}`;
  }, [path]);
  return null;
}
