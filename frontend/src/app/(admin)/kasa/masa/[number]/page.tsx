"use client";

import { useParams } from "next/navigation";
import { TableOrderEntry } from "@/components/TableOrderEntry";

export default function KasaTablePage() {
  const params = useParams<{ number: string }>();
  return (
    <TableOrderEntry
      tableNumber={Number(params.number)}
      backHref="/kasa"
      unauthorizedHref="/login"
      cashier
    />
  );
}
