// Public QR menu — served at qr.gunguzelbahce.online (view-only, no auth).
// Server component: fetches the public menu and renders it, cached briefly.
import { formatTRY } from "@/lib/money";
import { ADISYON_LOGO } from "@/lib/adisyonLogo";
import type { CategoryWithItems, MenuResponse } from "@/lib/types";

// Render on demand (never bake a static page at build time — the backend may be
// unreachable during the build, which would freeze a "yüklenemedi" fallback into
// the CDN). The menu data itself is still cached 60s so we don't hammer the API.
export const dynamic = "force-dynamic";

async function getMenu(): Promise<CategoryWithItems[] | null> {
  const base = process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080";
  try {
    const res = await fetch(`${base}/api/v1/public/menu`, {
      next: { revalidate: 60 },
    });
    if (!res.ok) return null;
    const data = (await res.json()) as MenuResponse;
    return data.categories;
  } catch {
    return null;
  }
}

export default async function PublicMenuPage() {
  const categories = await getMenu();
  const cats = (categories ?? []).filter((c) => (c.items ?? []).length > 0);

  // categoryId -> name, for describing fiks menü composition
  const catName = new Map(cats.map((c) => [c.id, c.name]));

  return (
    <main className="mx-auto flex min-h-full w-full max-w-2xl flex-col bg-[#f7f7f5]">
      {/* Header */}
      <header className="flex flex-col items-center gap-2 px-4 pb-4 pt-8">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={ADISYON_LOGO}
          alt="Gün Güzelbahçe"
          className="h-24 w-auto"
        />
        <p className="text-xs font-medium uppercase tracking-[0.2em] text-amber-800/70">
          Menü
        </p>
      </header>

      {categories === null ? (
        <p className="px-4 py-16 text-center text-sm text-zinc-500">
          Menü şu an yüklenemedi. Lütfen biraz sonra tekrar deneyin.
        </p>
      ) : cats.length === 0 ? (
        <p className="px-4 py-16 text-center text-sm text-zinc-500">
          Menü hazırlanıyor.
        </p>
      ) : (
        <>
          {/* Category quick-nav (sticky) */}
          <nav className="sticky top-0 z-10 flex gap-2 overflow-x-auto border-b border-amber-100 bg-[#f7f7f5]/95 px-4 py-3 backdrop-blur">
            {cats.map((c) => (
              <a
                key={c.id}
                href={`#cat-${c.id}`}
                className="shrink-0 rounded-full border border-amber-200 bg-white px-4 py-1.5 text-sm font-medium text-amber-900 shadow-sm"
              >
                {c.name}
              </a>
            ))}
          </nav>

          <div className="flex flex-col gap-8 px-4 py-6">
            {cats.map((c) => (
              <section
                key={c.id}
                id={`cat-${c.id}`}
                className="flex scroll-mt-16 flex-col gap-3"
              >
                <h2 className="text-lg font-bold text-zinc-900">
                  {c.name}
                </h2>
                <ul className="flex flex-col gap-3">
                  {(c.items ?? []).map((it) => {
                    const fixLine =
                      it.isFix && it.fixIncludes?.length
                        ? it.fixIncludes
                            .map(
                              (f) =>
                                `${f.count} ${catName.get(f.categoryId) ?? ""}`,
                            )
                            .join(" + ")
                        : "";
                    return (
                      <li
                        key={it.id}
                        className="flex gap-3 rounded-2xl border border-zinc-100 bg-white p-3 shadow-sm"
                      >
                        {it.imageUrl && (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img
                            src={it.imageUrl}
                            alt={it.name}
                            className="h-20 w-20 shrink-0 rounded-xl object-cover"
                          />
                        )}
                        <div className="flex min-w-0 flex-1 flex-col">
                          <div className="flex items-start justify-between gap-2">
                            <span className="flex items-center gap-2 text-base font-semibold text-zinc-900">
                              {it.name}
                              {it.isFix && (
                                <span className="rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-bold uppercase text-amber-800">
                                  Fiks
                                </span>
                              )}
                            </span>
                            <span className="shrink-0 whitespace-nowrap text-base font-bold tabular-nums text-amber-800">
                              {formatTRY(it.price)}
                            </span>
                          </div>
                          {it.description && (
                            <p className="mt-0.5 text-sm text-zinc-500">
                              {it.description}
                            </p>
                          )}
                          {fixLine && (
                            <p className="mt-1 text-xs text-amber-700">
                              {fixLine} · kişi başı
                            </p>
                          )}
                        </div>
                      </li>
                    );
                  })}
                </ul>
              </section>
            ))}
          </div>
        </>
      )}

      <footer className="mt-auto px-4 py-8 text-center">
        <p className="text-sm font-medium text-amber-900">Afiyet olsun</p>
        <p className="mt-1 text-xs text-zinc-400">Gün Güzelbahçe</p>
      </footer>
    </main>
  );
}
