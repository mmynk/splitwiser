<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLInputAttributes } from 'svelte/elements';

  interface Props extends Omit<HTMLInputAttributes, 'class' | 'prefix'> {
    label?: string;
    hint?: string;
    error?: string;
    value?: string | number;
    prefix?: Snippet;
    suffix?: Snippet;
  }

  let {
    label,
    hint,
    error,
    value = $bindable(''),
    prefix,
    suffix,
    id,
    type = 'text',
    ...rest
  }: Props = $props();

  // Generated once per component instance; only used if the parent didn't pass an id.
  const generatedId = `in-${Math.random().toString(36).slice(2, 8)}`;
  const reactiveId = $derived(id ?? generatedId);
</script>

<div class="flex flex-col gap-1">
  {#if label}
    <label for={reactiveId} class="text-[0.8125rem] font-medium text-text-muted">
      {label}
    </label>
  {/if}

  <div
    class={[
      'flex items-center gap-2 rounded-input bg-surface-elevated px-3',
      'border transition-[border-color,box-shadow] duration-150',
      'focus-within:border-primary focus-within:shadow-[0_0_0_3px_var(--color-primary-soft)]',
      error ? 'border-danger' : 'border-border',
    ]}
  >
    {#if prefix}
      <span class="text-text-subtle">{@render prefix()}</span>
    {/if}
    <input
      id={reactiveId}
      {type}
      bind:value
      class="h-10 flex-1 bg-transparent text-[0.9375rem] text-text outline-none placeholder:text-text-subtle"
      aria-invalid={error ? 'true' : undefined}
      aria-describedby={error ? `${reactiveId}-err` : hint ? `${reactiveId}-hint` : undefined}
      {...rest}
    />
    {#if suffix}
      <span class="text-text-subtle">{@render suffix()}</span>
    {/if}
  </div>

  {#if error}
    <p id="{reactiveId}-err" class="text-[0.75rem] text-danger">{error}</p>
  {:else if hint}
    <p id="{reactiveId}-hint" class="text-[0.75rem] text-text-subtle">{hint}</p>
  {/if}
</div>
