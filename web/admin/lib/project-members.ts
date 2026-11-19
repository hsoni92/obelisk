/** Stable exposure gateway refs (match pkg/obelisk exposure_members.go). */
export const EXPOSURE_OPENROUTER = "exposure/openrouter";
export const EXPOSURE_MAIL = "exposure/mail";

export const EXPOSURE_MEMBER_REFS = [EXPOSURE_OPENROUTER, EXPOSURE_MAIL] as const;

export function exposureMemberLabel(ref: string): string {
  switch (ref) {
    case EXPOSURE_OPENROUTER:
      return "OpenRouter gateway";
    case EXPOSURE_MAIL:
      return "Mail gateway";
    default:
      return ref;
  }
}

export function isExposureMemberRef(ref: string): boolean {
  return (EXPOSURE_MEMBER_REFS as readonly string[]).includes(ref);
}