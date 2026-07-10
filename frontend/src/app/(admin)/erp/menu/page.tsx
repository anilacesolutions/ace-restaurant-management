"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api, ApiError } from "@/lib/api";
import { formatTRY, fromTL, toTL } from "@/lib/money";
import type { CategoryWithItems, MenuItem, MenuResponse } from "@/lib/types";

const KDV_ORANLARI = [1, 10, 20] as const;
const POS_KODLARI = ["A", "B", "C", "D", "E", "F", "G", "H"] as const;

export default function ErpMenuPage() {
  const [cats, setCats] = useState<CategoryWithItems[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<MenuItem | "new" | null>(null);

  async function load() {
    try {
      const r = await api<MenuResponse>("/api/v1/menu/admin");
      setCats(r.categories);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    load();
  }, []);

  const categoryNames = useMemo(
    () => (cats ?? []).map((c) => c.name),
    [cats],
  );

  const isEmpty =
    cats !== null && cats.every((c) => (c.items ?? []).length === 0);

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
          <h1 className="text-2xl font-semibold text-zinc-900">Menu Yonetimi</h1>
        </div>
        <button
          onClick={() => setEditing("new")}
          className="rounded-full bg-amber-700 px-5 py-2.5 text-sm font-semibold text-white shadow-sm active:bg-amber-800"
        >
          + Yeni Urun
        </button>
      </header>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {cats === null ? (
        <p className="text-sm text-zinc-500">Yukleniyor...</p>
      ) : isEmpty ? (
        <p className="text-sm text-zinc-500">
          Henuz urun yok. + Yeni Urun ile baslayin.
        </p>
      ) : (
        <div className="flex flex-col gap-6">
          {cats
            .filter((c) => (c.items ?? []).length > 0)
            .map((c) => (
              <section key={c.id} className="flex flex-col gap-2">
                <h2 className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-zinc-500">
                  {c.name}
                  {!c.active && (
                    <span className="rounded-full bg-zinc-200 px-2 py-0.5 text-[10px] font-medium text-zinc-600">
                      pasif kategori
                    </span>
                  )}
                </h2>
                <ul className="flex flex-col gap-2">
                  {c.items.map((it) => (
                    <li key={it.id}>
                      <button
                        onClick={() => setEditing(it)}
                        className="flex w-full items-center gap-3 rounded-2xl border border-zinc-200 bg-white p-3 text-left shadow-sm active:bg-zinc-50"
                      >
                        <Thumb url={it.imageUrl} name={it.name} />
                        <div className="flex min-w-0 flex-1 flex-col">
                          <span className="flex items-center gap-2 truncate text-base font-semibold text-zinc-900">
                            {it.name}
                            {!it.available && (
                              <span className="shrink-0 rounded-full bg-zinc-200 px-2 py-0.5 text-[10px] font-medium text-zinc-600">
                                pasif
                              </span>
                            )}
                          </span>
                          <span className="mt-0.5 flex flex-wrap items-center gap-1.5 text-xs text-zinc-500">
                            <span>KDV %{it.kdvOrani}</span>
                            {it.otvVar && <Badge>OTV</Badge>}
                            {it.posDepartmanKodu && (
                              <Badge>POS {it.posDepartmanKodu}</Badge>
                            )}
                            {it.kitchenPrint && <Badge>Mutfak</Badge>}
                          </span>
                        </div>
                        <span className="shrink-0 text-base font-semibold tabular-nums text-zinc-900">
                          {formatTRY(it.price)}
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>
              </section>
            ))}
        </div>
      )}

      {editing !== null && (
        <ItemFormSheet
          item={editing === "new" ? null : editing}
          categoryNames={categoryNames}
          categories={(cats ?? []).map((c) => ({ id: c.id, name: c.name }))}
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

function Badge({ children }: { children: React.ReactNode }) {
  return (
    <span className="rounded-full bg-amber-100 px-2 py-0.5 font-medium text-amber-800">
      {children}
    </span>
  );
}

function Thumb({ url, name }: { url?: string; name: string }) {
  if (!url) {
    return (
      <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-xl bg-zinc-100 text-lg font-semibold text-zinc-400">
        {name.slice(0, 1).toUpperCase()}
      </div>
    );
  }
  return (
    // S3-hosted user uploads with arbitrary hosts — next/image would need
    // remotePatterns per bucket/CDN; a plain img is the pragmatic choice here.
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={url}
      alt={name}
      className="h-14 w-14 shrink-0 rounded-xl object-cover"
    />
  );
}

// ---- Form sheet ----------------------------------------------------------

interface FixRow {
  categoryId: string;
  count: number;
  perPeople: number; // her kaç kişiye (1 = kişi başı)
}

interface FormState {
  categoryName: string;
  name: string;
  priceTL: string;
  kdvOrani: number;
  otvVar: boolean;
  posDepartmanKodu: string;
  kitchenPrint: boolean;
  available: boolean;
  imageUrl: string;
  isFix: boolean;
  fixIncludes: FixRow[];
}

function ItemFormSheet({
  item,
  categoryNames,
  categories,
  onClose,
  onSaved,
}: {
  item: MenuItem | null;
  categoryNames: string[];
  categories: { id: string; name: string }[];
  onClose: () => void;
  onSaved: () => void | Promise<void>;
}) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [f, setF] = useState<FormState>(() => ({
    categoryName: "",
    name: item?.name ?? "",
    priceTL: item ? String(toTL(item.price)) : "",
    kdvOrani: item?.kdvOrani ?? 10,
    otvVar: item?.otvVar ?? false,
    posDepartmanKodu: item?.posDepartmanKodu ?? "",
    kitchenPrint: item?.kitchenPrint ?? true,
    available: item?.available ?? true,
    imageUrl: item?.imageUrl ?? "",
    isFix: item?.isFix ?? false,
    fixIncludes:
      item?.fixIncludes?.map((c) => ({
        categoryId: c.categoryId,
        count: c.count,
        perPeople: c.perPeople && c.perPeople > 0 ? c.perPeople : 1,
      })) ?? [],
  }));
  const [busy, setBusy] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [imageNote, setImageNote] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  function set<K extends keyof FormState>(k: K, v: FormState[K]) {
    setF((prev) => ({ ...prev, [k]: v }));
  }

  async function handleFile(file: File) {
    setImageNote(null);
    setUploading(true);
    try {
      const ext = file.name.includes(".")
        ? file.name.split(".").pop()!
        : "jpg";
      const presign = await api<{ uploadUrl: string; publicUrl: string }>(
        "/api/v1/menu/images/presign",
        { method: "POST", body: JSON.stringify({ contentType: file.type, ext }) },
      );
      // Upload straight to S3 — a different host, no cookies, raw fetch.
      const put = await fetch(presign.uploadUrl, {
        method: "PUT",
        body: file,
        headers: { "Content-Type": file.type },
      });
      if (!put.ok) throw new Error(`S3 yukleme hatasi (${put.status})`);
      set("imageUrl", presign.publicUrl);
    } catch (e) {
      if (e instanceof ApiError && e.status === 501) {
        setImageNote("Gorsel yukleme kapali (S3 ayarli degil). Urunu gorselsiz kaydedebilirsin.");
      } else {
        setImageNote(e instanceof Error ? e.message : String(e));
      }
    } finally {
      setUploading(false);
    }
  }

  async function remove() {
    if (!item) return;
    setError(null);
    setDeleting(true);
    try {
      await api(`/api/v1/menu/items/${item.id}`, { method: "DELETE" });
      await onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setDeleting(false);
      setConfirmDelete(false);
    }
  }

  async function save() {
    setError(null);
    const tl = Number(f.priceTL.replace(",", "."));
    if (!Number.isFinite(tl) || tl <= 0) {
      setError("Gecerli bir fiyat gir");
      return;
    }
    const fixRows = f.fixIncludes.filter((r) => r.categoryId && r.count > 0);
    if (f.isFix && fixRows.length === 0) {
      setError("Fiks menü için en az bir içerik satırı ekle (kategori + adet)");
      return;
    }
    setBusy(true);
    try {
      const body = JSON.stringify({
        // A fiks menü always lives in its own "Fiks Menü" section, not in a
        // food category — keeps it separate from the meze/salata it includes.
        categoryName: f.isFix ? "Fiks Menü" : f.categoryName,
        name: f.name,
        price: fromTL(tl),
        kdvOrani: f.kdvOrani,
        otvVar: f.otvVar,
        posDepartmanKodu: f.posDepartmanKodu,
        kitchenPrint: f.kitchenPrint,
        available: f.available,
        imageUrl: f.imageUrl,
        isFix: f.isFix,
        fixIncludes: f.isFix ? fixRows : [],
      });
      if (item) {
        await api(`/api/v1/menu/items/${item.id}`, { method: "PATCH", body });
      } else {
        await api("/api/v1/menu/items", { method: "POST", body });
      }
      await onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
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
          {item ? "Urunu Duzenle" : "Yeni Urun"}
        </h2>
        <button
          onClick={save}
          disabled={busy || uploading}
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

        {/* Gorsel */}
        <Field label="Gorsel (opsiyonel)">
          <div className="flex items-center gap-3">
            <Thumb url={f.imageUrl || undefined} name={f.name || "?"} />
            <label className="rounded-xl border border-zinc-300 bg-white px-4 py-2.5 text-sm font-medium text-zinc-700 shadow-sm active:bg-zinc-50">
              {uploading ? "Yukleniyor..." : f.imageUrl ? "Degistir" : "Gorsel sec"}
              <input
                type="file"
                accept="image/*"
                className="hidden"
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) handleFile(file);
                }}
              />
            </label>
            {f.imageUrl && (
              <button
                onClick={() => set("imageUrl", "")}
                className="rounded-xl px-3 py-2 text-sm font-medium text-red-700 active:bg-red-50"
              >
                Kaldir
              </button>
            )}
          </div>
          {imageNote && (
            <p className="mt-2 text-xs text-amber-700">{imageNote}</p>
          )}
        </Field>

        {/* Kategori — fiks menuler otomatik "Fiks Menu" kategorisine gider */}
        {f.isFix ? (
          <div className="rounded-xl bg-amber-50 px-4 py-3 text-sm text-amber-900">
            Bu bir <strong>Fiks menu</strong> — otomatik olarak{" "}
            <strong>&quot;Fiks Menu&quot;</strong> kategorisinde gosterilir. (Icerigini
            asagida tanimla.)
          </div>
        ) : (
          <Field label="Kategori">
            <input
              value={f.categoryName}
              onChange={(e) => set("categoryName", e.target.value)}
              placeholder="orn. Meze, Ana Yemek, Icecek"
              className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
            />
            {categoryNames.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-2">
                {categoryNames.map((n) => (
                  <button
                    key={n}
                    onClick={() => set("categoryName", n)}
                    className={`rounded-full px-3 py-1.5 text-sm font-medium active:bg-amber-100 ${
                      f.categoryName === n
                        ? "bg-amber-700 text-white"
                        : "bg-zinc-100 text-zinc-700"
                    }`}
                  >
                    {n}
                  </button>
                ))}
              </div>
            )}
          </Field>
        )}

        {/* Urun adi */}
        <Field label="Urun adi">
          <input
            value={f.name}
            onChange={(e) => set("name", e.target.value)}
            placeholder="orn. Humus"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        {/* Fiyat */}
        <Field label="Fiyat (TL, KDV dahil)">
          <input
            value={f.priceTL}
            onChange={(e) => set("priceTL", e.target.value)}
            inputMode="decimal"
            placeholder="orn. 120"
            className="w-full rounded-xl border border-zinc-300 bg-white px-4 py-3 text-base tabular-nums text-zinc-900 shadow-sm outline-none focus:border-amber-500"
          />
        </Field>

        {/* KDV */}
        <Field label="KDV orani">
          <Segmented
            options={KDV_ORANLARI.map((k) => ({ value: k, label: `%${k}` }))}
            value={f.kdvOrani}
            onChange={(v) => set("kdvOrani", v)}
          />
        </Field>

        {/* Fiks menü */}
        <Toggle
          label="Fiks menü (set menü)"
          desc="Acarsan bu urun bir set menu olur. Siparis edilince asagidaki icerik kadar urun secilir ve adisyona 0 TL olarak eklenir. Fiyat kisi basidir."
          value={f.isFix}
          onChange={(v) => set("isFix", v)}
        />
        {f.isFix && (
          <Field label="Fiks icerigi (kisi basina)">
            <div className="flex flex-col gap-3">
              {f.fixIncludes.length === 0 && (
                <p className="text-sm text-zinc-400">
                  Henuz icerik yok. Asagidan ekle (orn. Meze x2, Salata x1).
                </p>
              )}
              {f.fixIncludes.map((row, i) => (
                <div
                  key={i}
                  className="rounded-xl border border-zinc-200 bg-zinc-50 p-3"
                >
                  <div className="mb-2 flex items-center justify-between">
                    <span className="text-xs font-medium text-zinc-500">
                      Kategori
                    </span>
                    <button
                      onClick={() =>
                        set(
                          "fixIncludes",
                          f.fixIncludes.filter((_, j) => j !== i),
                        )
                      }
                      className="text-sm font-medium text-red-700 active:opacity-70"
                    >
                      Kaldir
                    </button>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {categories.map((c) => (
                      <button
                        key={c.id}
                        onClick={() =>
                          set(
                            "fixIncludes",
                            f.fixIncludes.map((r, j) =>
                              j === i ? { ...r, categoryId: c.id } : r,
                            ),
                          )
                        }
                        className={`rounded-full px-3 py-1.5 text-sm font-medium ${
                          row.categoryId === c.id
                            ? "bg-amber-700 text-white"
                            : "border border-zinc-300 bg-white text-zinc-700"
                        }`}
                      >
                        {c.name}
                      </button>
                    ))}
                  </div>
                  <div className="mt-3 flex items-center gap-3">
                    <span className="text-sm text-zinc-600">Adet:</span>
                    <button
                      onClick={() =>
                        set(
                          "fixIncludes",
                          f.fixIncludes.map((r, j) =>
                            j === i
                              ? { ...r, count: Math.max(1, r.count - 1) }
                              : r,
                          ),
                        )
                      }
                      className="h-9 w-9 rounded-lg bg-zinc-200 text-xl font-bold text-zinc-700 active:bg-zinc-300"
                    >
                      −
                    </button>
                    <span className="w-6 text-center text-lg font-bold tabular-nums">
                      {row.count}
                    </span>
                    <button
                      onClick={() =>
                        set(
                          "fixIncludes",
                          f.fixIncludes.map((r, j) =>
                            j === i ? { ...r, count: r.count + 1 } : r,
                          ),
                        )
                      }
                      className="h-9 w-9 rounded-lg bg-zinc-200 text-xl font-bold text-zinc-700 active:bg-zinc-300"
                    >
                      +
                    </button>
                  </div>
                  <div className="mt-3 flex items-center gap-3">
                    <span className="text-sm text-zinc-600">Her</span>
                    <button
                      onClick={() =>
                        set(
                          "fixIncludes",
                          f.fixIncludes.map((r, j) =>
                            j === i
                              ? { ...r, perPeople: Math.max(1, r.perPeople - 1) }
                              : r,
                          ),
                        )
                      }
                      className="h-9 w-9 rounded-lg bg-zinc-200 text-xl font-bold text-zinc-700 active:bg-zinc-300"
                    >
                      −
                    </button>
                    <span className="w-6 text-center text-lg font-bold tabular-nums">
                      {row.perPeople}
                    </span>
                    <button
                      onClick={() =>
                        set(
                          "fixIncludes",
                          f.fixIncludes.map((r, j) =>
                            j === i ? { ...r, perPeople: r.perPeople + 1 } : r,
                          ),
                        )
                      }
                      className="h-9 w-9 rounded-lg bg-zinc-200 text-xl font-bold text-zinc-700 active:bg-zinc-300"
                    >
                      +
                    </button>
                    <span className="text-sm text-zinc-600">kişiye</span>
                  </div>
                  <p className="mt-2 text-xs text-amber-800">
                    →{" "}
                    {row.perPeople === 1
                      ? `kişi başı ${row.count} adet`
                      : `her ${row.perPeople} kişiye ${row.count} adet (küsurat yukarı yuvarlanır)`}
                  </p>
                </div>
              ))}
              <button
                onClick={() =>
                  set("fixIncludes", [
                    ...f.fixIncludes,
                    { categoryId: "", count: 1, perPeople: 1 },
                  ])
                }
                className="rounded-xl border border-dashed border-amber-400 py-2.5 text-sm font-semibold text-amber-800 active:bg-amber-50"
              >
                + Kategori ekle
              </button>
            </div>
          </Field>
        )}

        {/* POS departman kodu */}
        <Field label="POS departman kodu (yazar kasa, opsiyonel)">
          <div className="flex flex-wrap gap-2">
            {POS_KODLARI.map((k) => (
              <button
                key={k}
                onClick={() =>
                  set("posDepartmanKodu", f.posDepartmanKodu === k ? "" : k)
                }
                className={`h-12 w-12 rounded-xl text-base font-semibold active:bg-amber-100 ${
                  f.posDepartmanKodu === k
                    ? "bg-amber-700 text-white"
                    : "bg-zinc-100 text-zinc-700"
                }`}
              >
                {k}
              </button>
            ))}
          </div>
        </Field>

        {/* Toggles */}
        <Toggle
          label="Aktif (satista)"
          desc="Kapatirsan urun kasada ve garsonda gorunmez."
          value={f.available}
          onChange={(v) => set("available", v)}
        />
        <Toggle
          label="Mutfak fisi bassin"
          desc="Siparis verildiginde mutfaga fis cikar (yemekler icin acik, sise icecek icin kapat)."
          value={f.kitchenPrint}
          onChange={(v) => set("kitchenPrint", v)}
        />
        <Toggle
          label="OTV var (alkol)"
          desc="Alkollu urunlerde acik. Raporda OTV ayri gosterilir."
          value={f.otvVar}
          onChange={(v) => set("otvVar", v)}
        />

        {/* Silme — sadece mevcut urunde */}
        {item && (
          <div className="mt-4 border-t border-zinc-200 pt-5">
            <button
              onClick={() => setConfirmDelete(true)}
              disabled={busy || deleting}
              className="w-full rounded-xl border border-red-300 bg-red-50 py-3 text-base font-semibold text-red-700 active:bg-red-100 disabled:opacity-50"
            >
              Urunu Sil
            </button>
            <p className="mt-2 text-center text-xs text-zinc-400">
              Urun menuden kalkar. Gecmis adisyonlar etkilenmez.
            </p>
          </div>
        )}
      </div>

      {confirmDelete && item && (
        <div className="fixed inset-0 z-[60] flex items-end justify-center bg-black/40 p-4 sm:items-center">
          <div className="w-full max-w-sm rounded-2xl bg-white p-5 shadow-xl">
            <h3 className="text-lg font-semibold text-zinc-900">Urunu sil?</h3>
            <p className="mt-2 text-sm text-zinc-600">
              <strong>{item.name}</strong> menuden kalicak olarak silinecek. Bu
              islem geri alinamaz.
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

function Segmented<T extends string | number>({
  options,
  value,
  onChange,
}: {
  options: { value: T; label: string }[];
  value: T;
  onChange: (v: T) => void;
}) {
  return (
    <div className="flex gap-2">
      {options.map((o) => (
        <button
          key={String(o.value)}
          onClick={() => onChange(o.value)}
          className={`flex-1 rounded-xl py-3 text-base font-semibold active:bg-amber-100 ${
            value === o.value
              ? "bg-amber-700 text-white"
              : "bg-zinc-100 text-zinc-700"
          }`}
        >
          {o.label}
        </button>
      ))}
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
