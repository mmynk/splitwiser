import { writable } from 'svelte/store';

export type ToastKind = 'info' | 'success' | 'error';

export interface Toast {
  id: number;
  kind: ToastKind;
  message: string;
}

const DEFAULT_DURATION_MS = 3000;
let nextId = 1;
const timers = new Map<number, ReturnType<typeof setTimeout>>();

function createToastStore() {
  const { subscribe, update } = writable<Toast[]>([]);

  function dismiss(id: number): void {
    const handle = timers.get(id);
    if (handle !== undefined) {
      clearTimeout(handle);
      timers.delete(id);
    }
    update((toasts) => toasts.filter((t) => t.id !== id));
  }

  function push(message: string, kind: ToastKind = 'info', durationMs = DEFAULT_DURATION_MS): number {
    const id = nextId++;
    update((toasts) => [...toasts, { id, kind, message }]);
    if (durationMs > 0) {
      timers.set(id, setTimeout(() => dismiss(id), durationMs));
    }
    return id;
  }

  return {
    subscribe,
    push,
    dismiss,
    info: (m: string, d?: number) => push(m, 'info', d),
    success: (m: string, d?: number) => push(m, 'success', d),
    error: (m: string, d?: number) => push(m, 'error', d),
  };
}

export const toasts = createToastStore();
