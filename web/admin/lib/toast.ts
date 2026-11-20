"use client";

export type ToastVariant = "default" | "success" | "error" | "info" | "warning";

export type ToastRecord = {
  id: string;
  title: string;
  description?: string;
  variant: ToastVariant;
  open: boolean;
  duration: number;
};

export type ToastInput = {
  title: string;
  description?: string;
  variant?: ToastVariant;
  duration?: number;
};

const TOAST_LIMIT = 5;
const DEFAULT_DURATION = 5000;
const ERROR_DURATION = 8000;

let memoryState: ToastRecord[] = [];
const listeners = new Set<(state: ToastRecord[]) => void>();

function dispatch(next: ToastRecord[]) {
  memoryState = next;
  listeners.forEach((listener) => listener(memoryState));
}

function genId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

const serverToastsSnapshot: ToastRecord[] = [];

export function getToasts() {
  return memoryState;
}

/** Stable empty snapshot for useSyncExternalStore during SSR. */
export function getToastsServerSnapshot() {
  return serverToastsSnapshot;
}

export function subscribe(listener: (state: ToastRecord[]) => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function dismiss(id: string) {
  dispatch(
    memoryState.map((t) => (t.id === id ? { ...t, open: false } : t)),
  );
}

export function remove(id: string) {
  dispatch(memoryState.filter((t) => t.id !== id));
}

function durationFor(input: ToastInput) {
  if (input.duration != null) return input.duration;
  return input.variant === "error" ? ERROR_DURATION : DEFAULT_DURATION;
}

export function push(input: ToastInput) {
  const id = genId();
  const record: ToastRecord = {
    id,
    title: input.title,
    description: input.description,
    variant: input.variant ?? "default",
    open: true,
    duration: durationFor(input),
  };
  dispatch([record, ...memoryState].slice(0, TOAST_LIMIT));
}

export const toast = {
  show: (input: ToastInput) => push(input),
  success: (title: string, description?: string) =>
    push({ title, description, variant: "success" }),
  error: (title: string, description?: string) =>
    push({ title, description, variant: "error" }),
  info: (title: string, description?: string) =>
    push({ title, description, variant: "info" }),
  warning: (title: string, description?: string) =>
    push({ title, description, variant: "warning" }),
};