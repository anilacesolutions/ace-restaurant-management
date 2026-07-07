import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Without this, `next dev` rejects HMR WebSocket connections from non-
  // localhost origins, which leaves React client chunks half-loaded — submit
  // handlers don't bind and forms fall back to HTML GET. This lets us scan
  // QR codes from a phone on the same LAN. Pure dev concern; ignored in prod.
  allowedDevOrigins: [
    "192.168.3.30",
    "192.168.*.*",
    "10.*.*.*",
    "172.16.*.*",
  ],
};

export default nextConfig;
