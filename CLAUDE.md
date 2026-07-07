# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A restaurant operations system for a single restaurant (designed to scale to multi-tenant later). Three user surfaces, all web-based:

- **Waiter app** — runs on the waiter's own phone browser. Order entry at the table.
- **Cashier / main terminal** — touchscreen kiosk at the register. Order management, payment, and the waiter login QR.
- **ERP** — inventory, purchases, daily/weekly sales analytics.

Plus two non-user services:

- **Kitchen printer bridge** — small service on a mini-PC/Pi in the restaurant. Drives the thermal printer over USB/network.
- **POS bridge** (Phase 2) — same pattern, drives the fiscal device (yazar kasa) and bank card POS.

There is no mobile app. Everything is web.

## Repos in this directory

Three independent projects, each with its own deploy:

- `backend/` — Go. REST API + MQTT integration + business logic. Talks to MongoDB.
- `frontend/` — Next.js (App Router). All three user surfaces in one app, routed by role.
- `bridge/` — Go. Runs inside the restaurant. Subscribes to MQTT, drives the thermal printer. Will gain POS later.

Each has its own README/CLAUDE.md once initialized — this file is the cross-cutting picture.

## Stack and key decisions

| Layer | Choice | Why |
|---|---|---|
| Frontend | Next.js + TypeScript + Tailwind + shadcn/ui + Tremor | Touch-first UI, charts for ERP, single codebase for all three surfaces |
| Backend | Go | Cheap goroutines hold many idle MQTT/WSS connections well; single-binary deploy |
| Realtime | **MQTT** (not raw WSS) | Topic pub/sub maps the flow naturally; QoS 1/2 gives order-loss guarantees without us writing a queue; LWT gives presence/offline detection for free |
| Broker | AWS IoT Core (managed) | Already on company AWS; no broker to operate |
| Database | MongoDB | One DB for orders, menu, and ERP. Aggregation pipeline handles sales reports |
| Deploy | AWS (company account) | Backend + frontend in AWS; bridge runs on-prem in the restaurant |

## Realtime flow (the load-bearing part)

The whole point of the system is that an order placed at a table appears in the kitchen and at the cashier **instantly**, and status changes (preparing / ready / cancelled) flow back to the waiter's phone. This is MQTT, not REST.

Topic layout (subject to refinement in the topics task):

```
restaurant/{restaurantId}/table/{tableNo}/orders         # waiter publishes; cashier + kitchen subscribe
restaurant/{restaurantId}/kitchen/print                  # backend publishes; printer bridge subscribes (QoS 1+)
restaurant/{restaurantId}/order/{orderId}/status         # cashier publishes; waiter subscribes
restaurant/{restaurantId}/presence/waiter/{waiterId}     # LWT — auto-published when client disconnects
```

Rules:
- Order-bearing topics use **QoS 1 minimum** (QoS 2 for kitchen/print) — orders must not be lost.
- Every MQTT-publishing client (waiter phone, cashier, bridge) registers an **LWT** so we can mark them offline immediately on disconnect.
- Client-side **offline queue + reconnect**: a waiter losing signal for 2 seconds mid-order must not lose the order — buffer locally, flush on reconnect.

REST is for everything non-realtime: menu CRUD, ERP reports, login/QR issue, stock, purchases.

## Auth model

Waiters do **not** have user/password. They log in by scanning a QR code on the cashier terminal:

- The cashier terminal displays a rotating QR that encodes a short-lived login token.
- The waiter opens the web app on their phone and scans → backend exchanges token for a session.
- Waiter sessions are bound to the current shift; no long-lived credentials on the phones.

Cashier and ERP users do have credentials (standard auth).

## UI rules — non-negotiable

Both the waiter phone and the cashier terminal are touch-only. **No mouse.** This shapes everything:

- Min tap target 44px, prefer 60px+.
- **No hover states.** All info must be visible without hover. No tooltips.
- Replace `<select>` and dropdowns with full-screen modal/sheet pickers.
- Custom on-screen numeric keypad for quantities/prices/table numbers — do not summon the system keyboard for numbers.
- No right-click / context menus. Long-press is okay but never required.
- Destructive actions (cancel order, close table) need an explicit confirm — there is no undo.
- Long lists virtualize. The cashier panel can hold 50+ active orders.

ERP reports may also be viewed from a desktop browser — mouse is fine there, but build for touch first.

**Light theme only.** Do not add a dark mode, do not honor `prefers-color-scheme`. The cashier touchscreen is always-on in a lit restaurant and the warm light palette is intentional. The CSS pins `color-scheme: light` and pages must not use Tailwind `dark:` variants.

## Language and printing

- All UI text is **Turkish, hardcoded**. No i18n framework yet — add it only when a second language is needed.
- **Thermal printer outputs are ASCII, ALL CAPS** by design. `KOFTE`, `AYRAN`, `KOLA` — Turkish character handling on the printer is intentionally avoided. Conversion happens in the bridge before sending ESC/POS.

## KDV / VAT — design this in from day one

Turkish restaurant VAT is heterogeneous. The schema and the cashier flow must carry VAT info per line item from the start, even though POS integration is Phase 2.

Every menu item carries:

- `kdvOrani` — typically 10 (food, non-alcoholic) or 20 (alcohol)
- `otvVar: boolean` — alcohol has ÖTV (excise) included in price but reported separately
- `posDepartmanKodu` — the yazar kasa department code (A/B/C/...) the item maps to; needed when POS integration arrives

ERP reports must break down revenue by VAT rate (daily total at 10%, daily total at 20%, ÖTV separately). Build the aggregations for this from the start.

## POS — phased

MVP ships **without** POS integration. The restaurant operates with a manual fiscal device until Phase 2.

Phase 2 design notes (do not implement until brand/model is chosen):
- One "bağlı POS" (linked POS) flow: when the cashier closes a table, the entire bill (items, per-line VAT, totals) goes to the fiscal device in a single transaction. The operator does not re-enter items.
- Combined YN ÖKC POS devices (fiscal + card in one) are strongly preferred over separate devices.
- Vendor SDKs are almost always Windows-only C++/.NET — the POS bridge will run on a Windows machine at the register, similar pattern to the printer bridge.
- Before starting Phase 2, get: (a) which yazar kasa brand/model, (b) which bank for card POS.

## ERP scope (MVP)

- **Purchases / stok girişi**: simple form ("bugün 10 kg marul, X TL"), one row per purchase, linked to a supplier (optional).
- **Stock movements** auto-decrement as items sell (recipe-style mapping comes later; for MVP a flat product list is fine).
- **Reports**: best-selling meze / drink / main, daily and weekly views; revenue with VAT breakdown.

## What is NOT in scope (yet)

- Mobile app
- Multi-restaurant tenancy (schema is ready for it, but no UI/auth flow)
- POS integration (Phase 2)
- Recipe-level inventory (purchase → finished dish mapping)
- Reservation system
- Customer-facing features (online ordering, loyalty, etc.)

## Project-specific gotchas to remember

- The bridge is **on-prem**, not in AWS. It reaches MQTT outbound only — no inbound ports. Auth via per-restaurant client certificates (IoT Core).
- MongoDB connection string will be provided by the user — do not invent one or run a local Mongo without asking.
- When designing schemas or APIs, assume **single-restaurant for the UI**, but key documents by `restaurantId` already — easy migration later.
- All currency is **TRY**, store as integer kuruş (no floats), display formatted.
