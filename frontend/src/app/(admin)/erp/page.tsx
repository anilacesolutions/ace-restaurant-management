import Link from "next/link";

// Simple dependency-free line icons (24×24, currentColor). Colour comes from the
// tinted square each one sits in.
const iconProps = {
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.75,
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
  className: "h-6 w-6",
};

const Icons = {
  masalar: (
    <svg {...iconProps}>
      <rect x="3" y="3" width="7" height="7" rx="1.5" />
      <rect x="14" y="3" width="7" height="7" rx="1.5" />
      <rect x="3" y="14" width="7" height="7" rx="1.5" />
      <rect x="14" y="14" width="7" height="7" rx="1.5" />
    </svg>
  ),
  garsonlar: (
    <svg {...iconProps}>
      <circle cx="12" cy="8" r="4" />
      <path d="M4 20c0-4 3.6-6 8-6s8 2 8 6" />
    </svg>
  ),
  menu: (
    <svg {...iconProps}>
      <rect x="4" y="3" width="16" height="18" rx="2" />
      <path d="M8 8h8M8 12h8M8 16h5" />
    </svg>
  ),
  cari: (
    <svg {...iconProps}>
      <rect x="3" y="6" width="18" height="13" rx="2" />
      <path d="M3 10h18" />
      <circle cx="16.5" cy="14.5" r="1.1" fill="currentColor" stroke="none" />
    </svg>
  ),
  raporlar: (
    <svg {...iconProps}>
      <path d="M6 3h8l5 5v12a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1z" />
      <path d="M14 3v5h5" />
      <path d="M9 13h6M9 17h4" />
    </svg>
  ),
  grafikler: (
    <svg {...iconProps}>
      <path d="M4 20h16" />
      <path d="M7 20v-6" strokeWidth={2.5} />
      <path d="M12 20V6" strokeWidth={2.5} />
      <path d="M17 20v-9" strokeWidth={2.5} />
    </svg>
  ),
} as const;

const sections = [
  {
    href: "/erp/masalar",
    title: "Masalar",
    desc: "Masa ekle, kaldir",
    icon: Icons.masalar,
    tint: "bg-amber-100 text-amber-700",
  },
  {
    href: "/erp/garsonlar",
    title: "Garsonlar",
    desc: "Garson ekle, duzenle, sil",
    icon: Icons.garsonlar,
    tint: "bg-sky-100 text-sky-700",
  },
  {
    href: "/erp/menu",
    title: "Menu",
    desc: "Kategori ve urunler",
    icon: Icons.menu,
    tint: "bg-emerald-100 text-emerald-700",
  },
  {
    href: "/erp/cari",
    title: "Cari",
    desc: "Giderler, alacaklar, borç-alacak takibi",
    icon: Icons.cari,
    tint: "bg-violet-100 text-violet-700",
  },
  {
    href: "/erp/raporlar",
    title: "Raporlar",
    desc: "Gunluk, haftalik satis",
    icon: Icons.raporlar,
    tint: "bg-rose-100 text-rose-700",
  },
  {
    href: "/erp/grafikler",
    title: "Grafikler",
    desc: "Gelir-gider trendi, kişi sayısı",
    icon: Icons.grafikler,
    tint: "bg-teal-100 text-teal-700",
  },
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
            className="flex min-h-[88px] items-center gap-4 rounded-2xl border border-zinc-200 bg-white px-5 py-4 shadow-sm active:bg-zinc-50"
          >
            <span
              className={`flex h-12 w-12 shrink-0 items-center justify-center rounded-xl ${s.tint}`}
            >
              {s.icon}
            </span>
            <span className="flex min-w-0 flex-col">
              <span className="text-base font-semibold text-zinc-900">
                {s.title}
              </span>
              <span className="mt-0.5 text-sm text-zinc-600">{s.desc}</span>
            </span>
          </Link>
        ))}
      </div>
    </main>
  );
}
