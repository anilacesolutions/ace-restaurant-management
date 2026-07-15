import type { Metadata } from "next";

export const metadata: Metadata = { title: "Kasa" };

export default function KasaLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
