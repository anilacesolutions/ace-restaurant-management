import type { Metadata } from "next";

export const metadata: Metadata = { title: "Menü" };

export default function MenuLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
