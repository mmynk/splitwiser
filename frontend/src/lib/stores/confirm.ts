import { writable } from 'svelte/store';

export interface ConfirmRequest {
  title: string;
  body?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: 'default' | 'danger';
}

interface OpenState extends ConfirmRequest {
  id: number;
  resolve: (ok: boolean) => void;
}

const state = writable<OpenState | null>(null);
export const confirmState = state;

let nextId = 1;

export function confirmAction(req: ConfirmRequest): Promise<boolean> {
  return new Promise((resolve) => {
    state.update((current) => {
      // Resolve a pre-existing pending dialog before overwriting; otherwise its
      // awaiter hangs forever.
      current?.resolve(false);
      return {
        id: nextId++,
        title: req.title,
        body: req.body,
        confirmLabel: req.confirmLabel ?? 'Confirm',
        cancelLabel: req.cancelLabel ?? 'Cancel',
        tone: req.tone ?? 'default',
        resolve,
      };
    });
  });
}

export function resolveConfirm(ok: boolean): void {
  state.update((current) => {
    current?.resolve(ok);
    return null;
  });
}
