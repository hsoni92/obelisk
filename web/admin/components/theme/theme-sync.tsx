"use client";

import { applyTheme, readTheme, subscribeTheme } from "@/lib/theme";
import { useEffect } from "react";

/** Keeps the document theme aligned with the saved preference after hydration. */
export function ThemeSync() {
  useEffect(() => {
    applyTheme(readTheme());
    return subscribeTheme(() => applyTheme(readTheme()));
  }, []);
  return null;
}
