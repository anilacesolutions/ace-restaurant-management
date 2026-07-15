import type { Metadata } from "next";

export const metadata: Metadata = { title: "Garson" };

export default function GarsonLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
