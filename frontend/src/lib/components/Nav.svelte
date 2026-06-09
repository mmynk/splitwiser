<script lang="ts">
  import { link } from 'svelte-spa-router';
  import active from 'svelte-spa-router/active';
  import { LogOut, Sun, Moon, Monitor } from 'lucide-svelte';
  import { currentUser, logout } from '$lib/stores/auth';
  import { theme, setTheme, nextTheme, type Theme } from '$lib/stores/theme';
  import IconButton from '$lib/components/ui/IconButton.svelte';

  const links = [
    { href: '/', label: 'Home' },
    { href: '/groups', label: 'Groups' },
    { href: '/friends', label: 'Friends' },
  ];

  const ACTIVE_CLASS = 'bg-primary-soft text-primary';
  const INACTIVE_CLASS = 'text-text-muted hover:bg-surface-sunken hover:text-text';

  const THEME_ICON = { light: Sun, dark: Moon, system: Monitor } as const;
  const THEME_NOUN: Record<Theme, string> = {
    light: 'light',
    dark: 'dark',
    system: 'system',
  };
</script>

{#if $currentUser}
  <header class="border-b border-border bg-surface-elevated">
    <nav
      class="mx-auto flex max-w-5xl items-center justify-between gap-2 px-4 py-3 sm:gap-4 sm:px-6"
      aria-label="Primary"
    >
      <a
        use:link
        href="/"
        class="display-wonk text-base font-semibold text-text hover:text-primary sm:text-xl"
      >
        Splitwiser
      </a>

      <ul class="flex flex-1 items-center gap-1 sm:gap-2">
        {#each links as item}
          <li>
            <a
              use:link
              use:active={{ path: item.href, className: ACTIVE_CLASS, inactiveClassName: INACTIVE_CLASS }}
              href={item.href}
              class="rounded-input px-3 py-1.5 text-[0.875rem] font-medium transition-colors"
            >
              {item.label}
            </a>
          </li>
        {/each}
      </ul>

      <div class="flex items-center gap-2 text-[0.875rem]">
        <span class="hidden text-text-muted sm:inline">
          {$currentUser.displayName || $currentUser.email}
        </span>

        {#key $theme}
          {@const Icon = THEME_ICON[$theme]}
          <IconButton
            ariaLabel="Switch theme (currently {THEME_NOUN[$theme]})"
            title="Theme: {THEME_NOUN[$theme]}"
            variant="ghost"
            size="sm"
            onclick={() => setTheme(nextTheme($theme))}
          >
            <Icon size={15} strokeWidth={1.75} aria-hidden="true" />
          </IconButton>
        {/key}

        <button
          type="button"
          aria-label="Logout"
          onclick={() => logout()}
          class="inline-flex items-center gap-1.5 rounded-input border border-border px-2.5 py-1 text-text-muted hover:bg-surface-sunken hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
        >
          <LogOut size={14} strokeWidth={1.75} aria-hidden="true" />
          <span class="hidden sm:inline">Logout</span>
        </button>
      </div>
    </nav>
  </header>
{/if}
