<script lang="ts">
  import { link } from 'svelte-spa-router';
  import Button from '$lib/components/ui/Button.svelte';
</script>

<main class="mx-auto flex min-h-[60vh] max-w-md flex-col items-center justify-center gap-5 px-5 py-10 text-center">
  <div class="display-wonk text-[5rem] leading-none text-primary" aria-hidden="true">404</div>
  <h1 class="font-serif text-2xl font-semibold text-text">This page got lost.</h1>
  <p class="text-[0.9375rem] text-text-muted">
    Probably owes you for lunch. Head back home and we'll forget about it.
  </p>
  <a use:link href="/">
    <Button variant="primary">Back home</Button>
  </a>
</main>
