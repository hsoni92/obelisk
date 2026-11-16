/** Shared token classes for Radix primitive wrappers. */
export const overlayClass =
  "fixed inset-0 z-50 bg-black/30 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0";

export const popoverContentClass =
  "z-50 overflow-hidden rounded-2xl bg-popover text-popover-foreground shadow-rb-2 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95";

export const menuContentClass = `${popoverContentClass} min-w-[10rem] p-1`;

export const menuItemClass =
  "relative flex cursor-pointer select-none items-center gap-2 rounded-xl px-3 py-2 text-sm outline-none data-[highlighted]:bg-muted data-[disabled]:pointer-events-none data-[disabled]:opacity-50";

export const menuLabelClass = "px-2 py-1.5 text-eyebrow text-muted-foreground";

export const menuSeparatorClass = "my-1 h-px bg-border";

export const focusRingClass =
  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";
