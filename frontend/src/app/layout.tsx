import type { Metadata, Viewport } from "next";
import { Inter } from "next/font/google";
import "./globals.css";

const inter = Inter({
  variable: "--font-sans",
  subsets: ["latin", "latin-ext"],
});

export const metadata: Metadata = {
  // Each section sets its own title; the template prefixes the venue name so
  // the tab reads e.g. "Gün Güzelbahçe | Kasa".
  title: {
    default: "Gün Güzelbahçe",
    template: "Gün Güzelbahçe | %s",
  },
  description: "Gün Güzelbahçe sipariş ve yönetim sistemi",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
  themeColor: "#f7f7f5",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="tr" className={`${inter.variable} h-full antialiased`}>
      <body className="min-h-full flex flex-col touch-manipulation select-none">
        {children}
      </body>
    </html>
  );
}
