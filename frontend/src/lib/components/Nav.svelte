<script lang="ts">
  import { link } from 'svelte-spa-router';
  import active from 'svelte-spa-router/active';
  import { LogOut } from 'lucide-svelte';
  import { currentUser, logout } from '$lib/stores/auth';

  const links = [
    { href: '/', label: 'Home' },
    { href: '/groups', label: 'Groups' },
    { href: '/friends', label: 'Friends' },
  ];

  const ACTIVE_CLASS = 'bg-brand-50 text-brand-700';
  const INACTIVE_CLASS = 'text-slate-600 hover:bg-slate-100';
</script>

{#if $currentUser}
  <header class="border-b border-slate-200 bg-white">
    <nav
      class="mx-auto flex max-w-5xl items-center justify-between gap-4 px-4 py-3"
      aria-label="Primary"
    >
      <a use:link href="/" class="text-lg font-semibold text-slate-900 hover:text-brand-700">
        Splitwiser
      </a>

      <ul class="flex flex-1 items-center gap-1 sm:gap-2">
        {#each links as item}
          <li>
            <a
              use:link
              use:active={{ path: item.href, className: ACTIVE_CLASS, inactiveClassName: INACTIVE_CLASS }}
              href={item.href}
              class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
            >
              {item.label}
            </a>
          </li>
        {/each}
      </ul>

      <div class="flex items-center gap-3 text-sm">
        <span class="hidden text-slate-600 sm:inline">
          {$currentUser.displayName || $currentUser.email}
        </span>
        <button
          type="button"
          aria-label="Logout"
          onclick={() => logout()}
          class="inline-flex items-center gap-1 rounded-md border border-slate-300 px-2.5 py-1 text-slate-700 hover:bg-slate-100"
        >
          <LogOut size={14} />
          <span class="hidden sm:inline">Logout</span>
        </button>
      </div>
    </nav>
  </header>
{/if}
