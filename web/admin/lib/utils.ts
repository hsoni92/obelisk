import { type ClassValue, clsx, cn as cnMerge } from "cn";

/**
 * Our typography utilities (globals.css). The class merger doesn't know them,
 * so it treats `text-eyebrow` as a text *color* and drops it whenever a real
 * color class (e.g. `text-muted-foreground`) is present. We merge without them
 * and re-attach them afterwards.
 */
const TYPE_UTILITY =
  /^(?:[\w-]+:)*text-(?:display|page-title|section-title|stat|eyebrow|body|body-small|metadata|label)$/;
/** An explicit Tailwind font-size in the merged result means the caller overrode ours. */
const EXPLICIT_SIZE = /^(?:[\w-]+:)*text-(?:xs|sm|base|lg|xl|\dxl|\[\d.*)$/;

export function cn(...inputs: ClassValue[]) {
  const tokens = clsx(inputs).split(/\s+/).filter(Boolean);
  const custom = tokens.filter((t) => TYPE_UTILITY.test(t));
  if (custom.length === 0) return cnMerge(inputs);
  const rest = cnMerge(tokens.filter((t) => !TYPE_UTILITY.test(t)));
  const overridden = rest.split(" ").some((t) => EXPLICIT_SIZE.test(t));
  // Last custom type utility wins, mirroring normal merge semantics.
  const keep = overridden ? [] : [custom[custom.length - 1]];
  return [...keep, rest].filter(Boolean).join(" ");
}
