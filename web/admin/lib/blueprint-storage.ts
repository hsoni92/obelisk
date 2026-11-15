const KEY = "obelisk.blueprint.passphrase";

export function readPassphrase(): string | null {
  if (typeof window === "undefined") return null;
  const value = window.localStorage.getItem(KEY);
  return value && value.length > 0 ? value : null;
}

export function writePassphrase(value: string) {
  window.localStorage.setItem(KEY, value);
}

export function forgetPassphrase() {
  window.localStorage.removeItem(KEY);
}
