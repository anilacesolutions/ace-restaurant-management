// Mirrors backend/internal/domain — keep in sync.
// When fields change here, change them in Go too.

import type { Kurus } from "./money";

export interface MenuCategory {
  id: string;
  restaurantId: string;
  name: string;
  sortOrder: number;
  active: boolean;
}

export type OrderStatus =
  | "open"
  | "sent"
  | "preparing"
  | "ready"
  | "served"
  | "closed"
  | "cancelled";

export type OrderItemStatus =
  | "new"
  | "sent"
  | "ready"
  | "served"
  | "voided"
  | "refunded";

export interface OrderItem {
  id: string;
  menuItemId: string;
  name: string;
  qty: number;
  unitPrice: Kurus;
  kdvOrani: number;
  otvVar: boolean;
  posDepartmanKodu: string;
  note?: string;
  status: OrderItemStatus;
  addedAt: string;
  addedBy: string;
  isFix?: boolean;
  fixGroupId?: string;
}

export interface Order {
  id: string;
  restaurantId: string;
  tableNumber: number;
  waiterId: string;
  status: OrderStatus;
  items: OrderItem[];
  subtotal: Kurus;
  kdvBreakdown: Record<string, Kurus>;
  otv: Kurus;
  grandTotal: Kurus;
  paymentMethod?: string;
  openedAt: string;
  closedAt?: string;
  updatedAt: string;
}

export interface OrderResponse {
  order: Order | null;
}

export interface ActiveOrdersResponse {
  orders: Order[];
}

export interface FixComponent {
  categoryId: string;
  count: number;
  perPeople?: number; // items per N people; undefined/1 = per person
}

export interface MenuItem {
  id: string;
  restaurantId: string;
  categoryId: string;
  name: string;
  description?: string;
  price: Kurus;
  imageUrl?: string;
  kdvOrani: number;
  otvVar: boolean;
  posDepartmanKodu: string;
  available: boolean;
  sortOrder: number;
  kitchenPrint: boolean;
  isFix?: boolean;
  fixIncludes?: FixComponent[];
}

export interface CategoryWithItems extends MenuCategory {
  items: MenuItem[];
}

export interface MenuResponse {
  categories: CategoryWithItems[];
}

export interface Table {
  id: string;
  restaurantId: string;
  number: number;
  label?: string;
  seats?: number;
  active: boolean;
}

export interface TablesResponse {
  tables: Table[];
}

export interface Waiter {
  id: string;
  restaurantId: string;
  name: string;
  phone?: string;
  active: boolean;
}

export interface WaitersResponse {
  waiters: Waiter[];
}

export interface Payment {
  id: string;
  amount: Kurus;
  paidAt: string; // RFC3339
  note?: string;
}

export interface Expense {
  id: string;
  restaurantId: string;
  category: string;
  amount: Kurus;
  supplier?: string;
  note?: string;
  spentAt: string; // RFC3339
  payments: Payment[] | null;
  createdBy: string;
  createdAt: string;
}

export interface ExpensesResponse {
  expenses: Expense[];
}

export interface User {
  id: string;
  restaurantId: string;
  username: string;
  role: "admin";
  createdAt: string;
  lastLoginAt?: string;
}

export type Me =
  | { kind: "admin"; user: User }
  | { kind: "waiter"; waiter: Waiter };

export interface QRIssueResponse {
  token: string;
  expiresAt: string;
}
