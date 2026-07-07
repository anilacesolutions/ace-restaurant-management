// All monetary values are stored as integer kurus.
// Use these helpers everywhere — never use floats for money.

export type Kurus = number;

export function fromTL(tl: number): Kurus {
  return Math.round(tl * 100);
}

export function toTL(kurus: Kurus): number {
  return kurus / 100;
}

const formatter = new Intl.NumberFormat("tr-TR", {
  style: "currency",
  currency: "TRY",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

export function formatTRY(kurus: Kurus): string {
  return formatter.format(toTL(kurus));
}
