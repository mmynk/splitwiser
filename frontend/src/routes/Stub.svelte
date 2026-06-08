<script lang="ts">
  interface Props {
    title: string;
    message: string;
  }

  let { title, message }: Props = $props();
</script>

<main class="mx-auto max-w-3xl px-4 py-10">
  <h1 class="mb-6 text-2xl font-bold">{title}</h1>
  <div class="rounded-lg bg-surface-elevated p-6 shadow-sm ring-1 ring-border">
    <p class="text-text">{message}</p>
  </div>
</main>
