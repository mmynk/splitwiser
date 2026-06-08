<script lang="ts">
  import { currentUser, logout } from '$lib/stores/auth';

  interface Props {
    title: string;
    message: string;
  }

  let { title, message }: Props = $props();
</script>

<main class="mx-auto max-w-3xl px-4 py-10">
  <div class="mb-6 flex items-center justify-between">
    <h1 class="text-2xl font-bold">{title}</h1>
    {#if $currentUser}
      <div class="flex items-center gap-3 text-sm">
        <span class="text-slate-600">👤 {$currentUser.displayName || $currentUser.email}</span>
        <button
          type="button"
          onclick={() => logout()}
          class="rounded-md border border-slate-300 px-3 py-1 hover:bg-slate-100"
        >
          Logout
        </button>
      </div>
    {/if}
  </div>

  <div class="rounded-lg bg-white p-6 shadow-sm ring-1 ring-slate-200">
    <p class="text-slate-700">{message}</p>
  </div>
</main>
