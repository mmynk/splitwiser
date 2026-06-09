<script lang="ts">
  import type { ComponentType, Snippet } from 'svelte';

  interface Props {
    icon?: ComponentType;
    title: string;
    hint?: string;
    children?: Snippet;
  }

  let { icon: Icon, title, hint, children }: Props = $props();
</script>

<div
  class="flex flex-col items-center gap-3 rounded-card border border-border bg-surface-elevated px-6 py-10 text-center"
>
  {#if Icon}
    <span class="text-text-subtle">
      <Icon size={28} strokeWidth={1.5} />
    </span>
  {/if}
  <p class="font-serif text-[1.0625rem] font-medium text-text">{title}</p>
  {#if hint}
    <p class="-mt-1 text-[0.875rem] text-text-muted">{hint}</p>
  {/if}
  {#if children}
    <div class="mt-1">{@render children()}</div>
  {/if}
</div>
