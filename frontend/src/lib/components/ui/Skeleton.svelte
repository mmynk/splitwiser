<script lang="ts">
  interface Props {
    height?: string;
    width?: string;
    rounded?: 'input' | 'card' | 'pill';
    class?: string;
  }

  let {
    height = 'h-4',
    width = 'w-full',
    rounded = 'input',
    class: extra = '',
  }: Props = $props();

  const RADIUS = {
    input: 'rounded-input',
    card: 'rounded-card',
    pill: 'rounded-pill',
  } as const;
</script>

<div
  aria-hidden="true"
  class={[
    'sw-skeleton',
    RADIUS[rounded],
    height,
    width,
    extra,
  ]}
></div>

<style>
  .sw-skeleton {
    position: relative;
    overflow: hidden;
    background: var(--color-surface-sunken);
    border: 1px solid var(--color-border);
  }
  .sw-skeleton::after {
    content: '';
    position: absolute;
    inset: 0;
    transform: translateX(-100%);
    background: linear-gradient(
      90deg,
      transparent 0%,
      var(--color-primary-soft) 50%,
      transparent 100%
    );
    animation: sw-shimmer 1.6s ease-in-out infinite;
  }
  @keyframes sw-shimmer {
    0% { transform: translateX(-100%); }
    60% { transform: translateX(100%); }
    100% { transform: translateX(100%); }
  }
  @media (prefers-reduced-motion: reduce) {
    .sw-skeleton::after { animation: none; opacity: 0.6; }
  }
</style>
