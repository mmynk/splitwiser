import { writable } from 'svelte/store';

export type Theme = 'light' | 'dark' | 'system';

const STORAGE_KEY = 'theme';

function readStored(): Theme {
  if (typeof localStorage === 'undefined') return 'system';
  const v = localStorage.getItem(STORAGE_KEY);
  return v === 'light' || v === 'dark' ? v : 'system';
}

function apply(t: Theme) {
  if (typeof document === 'undefined') return;
  if (t === 'system') document.documentElement.removeAttribute('data-theme');
  else document.documentElement.setAttribute('data-theme', t);
}

function persist(t: Theme) {
  if (typeof localStorage === 'undefined') return;
  if (t === 'system') localStorage.removeItem(STORAGE_KEY);
  else localStorage.setItem(STORAGE_KEY, t);
}

export const theme = writable<Theme>(readStored());

// Skip the initial subscriber fire — the stored value is already on disk and
// (via the FOUC-prevention script in index.html) already on the documentElement.
let primed = false;
theme.subscribe((v) => {
  if (!primed) { primed = true; return; }
  persist(v);
  apply(v);
});

export function setTheme(next: Theme) {
  theme.set(next);
}

// Single-tap toggle: pick the next state so it always moves *away* from what's
// currently rendered. From 'system' we jump to the explicit opposite of the
// system preference (so tapping in rendered-dark gives you light, not dark).
export function nextTheme(current: Theme): Theme {
  if (current === 'light') return 'dark';
  if (current === 'dark') return 'system';
  const systemDark =
    typeof matchMedia !== 'undefined' &&
    matchMedia('(prefers-color-scheme: dark)').matches;
  return systemDark ? 'light' : 'dark';
}
