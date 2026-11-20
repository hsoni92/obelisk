/** Copy text — works on VPS http://<ip> (no secure context) via textarea fallback. */
export async function copyToClipboard(value: string): Promise<void> {
  if (typeof window === "undefined") {
    throw new Error("copy unavailable");
  }
  if (window.isSecureContext && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return;
    } catch {
      /* permission denied — try legacy path */
    }
  }
  if (!copyWithTextarea(value)) {
    throw new Error("copy failed");
  }
}

function copyWithTextarea(value: string): boolean {
  const area = document.createElement("textarea");
  area.value = value;
  area.setAttribute("readonly", "");
  area.style.position = "fixed";
  area.style.top = "0";
  area.style.left = "0";
  area.style.width = "1px";
  area.style.height = "1px";
  area.style.padding = "0";
  area.style.border = "none";
  area.style.outline = "none";
  area.style.boxShadow = "none";
  area.style.background = "transparent";
  const anchor =
    document.activeElement instanceof HTMLElement
      ? document.activeElement
      : document.body;
  anchor.insertAdjacentElement("afterend", area);
  area.focus();
  area.select();
  area.setSelectionRange(0, value.length);
  let ok = false;
  try {
    ok = document.execCommand("copy");
  } finally {
    area.remove();
  }
  return ok;
}