// Tiny i18n shim. Per CLAUDE.global.md frontend rules, every visible
// string must go through the dictionary. We keep a single English
// dictionary in MVP; adding a second language is a matter of adding
// another file and a language selector — no library is required.

import { en } from "./en";

export type Dict = typeof en;

const dict: Dict = en;

export function t(
  key: keyof Dict,
  params?: Record<string, string | number>,
): string {
  const raw = dict[key] ?? (key as string);
  if (!params) return raw;
  return raw.replace(/\{(\w+)\}/g, (_, name: string) =>
    params[name] !== undefined ? String(params[name]) : `{${name}}`,
  );
}

// Hook-shaped variant for components that prefer the React naming
// convention. Returns the same `t` function so callers can write
// `const t = useT(); t("nav.sources")`.
export function useT() {
  return t;
}
