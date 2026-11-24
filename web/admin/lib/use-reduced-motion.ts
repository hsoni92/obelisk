import { useSyncExternalStore } from "react";

const QUERY = "(prefers-reduced-motion: reduce)";

function subscribe(onChange: () => void) {
  const mq = window.matchMedia(QUERY);
  mq.addEventListener("change", onChange);
  return () => mq.removeEventListener("change", onChange);
}

/**
 * Whether the user prefers reduced motion. Unlike framer-motion's
 * useReducedMotion it reports `false` during server rendering and hydration and
 * switches afterwards, so server-rendered markup never mismatches. Use it in
 * components that render on the server.
 */
export function useSafeReducedMotion(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => window.matchMedia(QUERY).matches,
    () => false,
  );
}
