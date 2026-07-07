"use client";

import { useParams } from "next/navigation";
import { TableOrderEntry } from "@/components/TableOrderEntry";

export default function GarsonTablePage() {
  const params = useParams<{ number: string }>();
  return (
    <TableOrderEntry
      tableNumber={Number(params.number)}
      backHref="/garson"
      unauthorizedHref="/garson/oturum-bitti"
    />
  );
}
