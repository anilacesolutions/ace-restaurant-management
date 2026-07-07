"use client";

import { useState } from "react";
import { AdminHeader } from "@/components/AdminHeader";
import { IssueQRModal } from "@/components/IssueQRModal";
import type { User } from "@/lib/types";

export function AdminShell({
  user,
  children,
}: {
  user: User;
  children: React.ReactNode;
}) {
  const [qrOpen, setQrOpen] = useState(false);

  return (
    <div className="flex flex-1 flex-col">
      <AdminHeader user={user} onIssueQR={() => setQrOpen(true)} />
      <div className="flex flex-1 flex-col">{children}</div>
      {qrOpen && <IssueQRModal onClose={() => setQrOpen(false)} />}
    </div>
  );
}
