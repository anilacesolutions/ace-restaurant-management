import Link from "next/link";

const sections = [
  { href: "/erp/masalar", title: "Masalar", desc: "Masa ekle, kaldir" },
  { href: "/erp/menu", title: "Menu", desc: "Kategori ve urunler" },
  { href: "/erp/giderler", title: "Giderler", desc: "Gider defteri, alimlar" },
  { href: "/erp/raporlar", title: "Raporlar", desc: "Gunluk, haftalik satis" },
] as const;

export default function ErpHomePage() {
  return (
    <main className="flex flex-1 flex-col gap-6 p-6">
      <h1 className="text-2xl font-semibold text-zinc-900">Yonetim</h1>
      <div className="grid gap-3 sm:grid-cols-2">
        {sections.map((s) => (
          <Link
            key={s.href}
            href={s.href}
            className="flex min-h-[88px] flex-col justify-center rounded-2xl border border-zinc-200 bg-white px-5 py-4 shadow-sm active:bg-zinc-50"
          >
            <span className="text-base font-semibold text-zinc-900">{s.title}</span>
            <span className="mt-0.5 text-sm text-zinc-600">{s.desc}</span>
          </Link>
        ))}
      </div>
    </main>
  );
}
