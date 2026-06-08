<script lang="ts">
  import { Plus, Trash2, Search, Sun, Moon, Monitor, ArrowRight, Check } from 'lucide-svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import IconButton from '$lib/components/ui/IconButton.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Amount from '$lib/components/ui/Amount.svelte';
  import { theme, setTheme, type Theme } from '$lib/stores/theme';

  let inputValue = $state('');
  let inputError = $state('');

  const THEMES: { id: Theme; label: string; icon: typeof Sun }[] = [
    { id: 'light', label: 'Light mode', icon: Sun },
    { id: 'dark', label: 'Dark mode', icon: Moon },
    { id: 'system', label: 'Follow system', icon: Monitor },
  ];

  const swatches = [
    { name: 'primary', cls: 'bg-primary text-primary-foreground' },
    { name: 'primary-hover', cls: 'bg-primary-hover text-primary-foreground' },
    { name: 'primary-soft', cls: 'bg-primary-soft text-text' },
    { name: 'surface', cls: 'bg-surface text-text border border-border' },
    { name: 'surface-elevated', cls: 'bg-surface-elevated text-text border border-border' },
    { name: 'surface-sunken', cls: 'bg-surface-sunken text-text border border-border' },
    { name: 'success', cls: 'bg-success text-white' },
    { name: 'success-soft', cls: 'bg-success-soft text-success' },
    { name: 'danger', cls: 'bg-danger text-white' },
    { name: 'danger-soft', cls: 'bg-danger-soft text-danger' },
    { name: 'warning', cls: 'bg-warning text-white' },
    { name: 'warning-soft', cls: 'bg-warning-soft text-warning' },
  ];
</script>

<main class="mx-auto max-w-4xl px-5 py-8 sm:px-8">
  <header class="mb-8 flex items-end justify-between gap-4">
    <div>
      <h1 class="display-wonk text-3xl">Style guide</h1>
      <p class="mt-1 text-text-muted">Every primitive, every state. The reference.</p>
    </div>
    <div class="flex gap-1 rounded-input bg-surface-sunken p-1 border border-border" role="group" aria-label="Theme">
      {#each THEMES as opt}
        {@const Icon = opt.icon}
        <IconButton
          ariaLabel={opt.label}
          title={opt.label}
          variant={$theme === opt.id ? 'primary' : 'ghost'}
          size="sm"
          onclick={() => setTheme(opt.id)}
        >
          <Icon size={14} strokeWidth={1.75} aria-hidden="true" />
        </IconButton>
      {/each}
    </div>
  </header>

  <section class="mb-10">
    <h2 class="mb-4 text-xl">Color</h2>
    <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
      {#each swatches as s}
        <div class="rounded-card border border-border overflow-hidden">
          <div class="{s.cls} flex h-16 items-center justify-center text-[0.75rem] font-medium">
            {s.name}
          </div>
        </div>
      {/each}
    </div>
  </section>

  <section class="mb-10">
    <h2 class="mb-4 text-xl">Typography</h2>
    <Card>
      <div class="flex flex-col gap-4">
        <div>
          <div class="text-[0.75rem] uppercase tracking-wider text-text-subtle mb-1">Display — Fraunces WONK</div>
          <div class="display-wonk text-4xl">Settle up, settle down.</div>
        </div>
        <div>
          <div class="text-[0.75rem] uppercase tracking-wider text-text-subtle mb-1">h1 — Fraunces</div>
          <h1>Brunch with Alex</h1>
        </div>
        <div>
          <div class="text-[0.75rem] uppercase tracking-wider text-text-subtle mb-1">h2 — Fraunces</div>
          <h2>Recent bills</h2>
        </div>
        <div>
          <div class="text-[0.75rem] uppercase tracking-wider text-text-subtle mb-1">body — Inter Tight</div>
          <p>The quick brown fox jumps over the lazy dog. 0123456789</p>
        </div>
        <div>
          <div class="text-[0.75rem] uppercase tracking-wider text-text-subtle mb-1">money — Fraunces tabular</div>
          <div class="flex flex-col gap-1">
            <Amount value={1234.56} size="display" />
            <Amount value={42.5} signed showPlus size="lg" />
            <Amount value={-18.75} signed size="lg" />
            <Amount value={0} signed size="lg" />
            <Amount value={12345.67} abbreviate size="lg" />
          </div>
        </div>
      </div>
    </Card>
  </section>

  <section class="mb-10">
    <h2 class="mb-4 text-xl">Buttons</h2>
    <Card>
      <div class="flex flex-col gap-6">
        <div class="flex flex-wrap items-center gap-3">
          <Button variant="primary">Primary</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="danger">Delete</Button>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <Button size="sm">Small</Button>
          <Button size="md">Medium</Button>
          <Button size="lg">Large</Button>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <Button loading>Loading</Button>
          <Button disabled>Disabled</Button>
          <Button>
            <Plus size={16} strokeWidth={1.75} />
            With icon
          </Button>
          <Button variant="secondary">
            Continue
            <ArrowRight size={16} strokeWidth={1.75} />
          </Button>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <IconButton ariaLabel="Add"><Plus size={16} strokeWidth={1.75} /></IconButton>
          <IconButton ariaLabel="Delete" variant="danger"><Trash2 size={16} strokeWidth={1.75} /></IconButton>
          <IconButton ariaLabel="Search" variant="secondary"><Search size={16} strokeWidth={1.75} /></IconButton>
          <IconButton ariaLabel="Confirm" variant="primary"><Check size={16} strokeWidth={1.75} /></IconButton>
        </div>
      </div>
    </Card>
  </section>

  <section class="mb-10">
    <h2 class="mb-4 text-xl">Inputs</h2>
    <Card>
      <div class="flex flex-col gap-5 max-w-md">
        <Input label="Email" placeholder="you@example.com" type="email" bind:value={inputValue} />
        <Input
          label="Amount"
          placeholder="0.00"
          hint="Use dot for decimals."
          type="number"
          inputmode="decimal"
        >
          {#snippet prefix()}
            <span class="text-text-muted">$</span>
          {/snippet}
        </Input>
        <Input
          label="Bill title"
          placeholder="Pizza night"
          error={inputError}
        />
        <Button variant="ghost" size="sm" onclick={() => (inputError = inputError ? '' : 'Title is required.')}>
          {inputError ? 'Clear error' : 'Trigger error state'}
        </Button>
      </div>
    </Card>
  </section>

  <section class="mb-10">
    <h2 class="mb-4 text-xl">Cards</h2>
    <div class="grid gap-4 sm:grid-cols-2">
      <Card>
        <h3 class="font-sans font-semibold mb-1">Resting card</h3>
        <p class="text-[0.875rem] text-text-muted">
          Hairline border, no shadow. The default.
        </p>
      </Card>
      <Card elevated>
        <h3 class="font-sans font-semibold mb-1">Elevated card</h3>
        <p class="text-[0.875rem] text-text-muted">
          Warm-tinted pop shadow. Reserve for popovers and modals.
        </p>
      </Card>
    </div>
  </section>

  <section class="mb-10">
    <h2 class="mb-4 text-xl">Status surfaces</h2>
    <div class="flex flex-col gap-2">
      <div class="rounded-input border border-success/30 bg-success-soft px-3 py-2 text-[0.875rem] text-success">
        All square. Nice.
      </div>
      <div class="rounded-input border border-danger/30 bg-danger-soft px-3 py-2 text-[0.875rem] text-danger">
        Couldn't reach the server. Check your connection and try again.
      </div>
      <div class="rounded-input border border-warning/30 bg-warning-soft px-3 py-2 text-[0.875rem] text-warning">
        Subtotal doesn't match the sum of items.
      </div>
    </div>
  </section>

  <section class="mb-10">
    <h2 class="mb-4 text-xl">Motion</h2>
    <p class="text-[0.875rem] text-text-muted">
      Every transition uses <code class="rounded bg-surface-sunken px-1 font-mono text-[0.75rem]">ease = cubicOut, dur = 180ms</code>.
      Enter fades + rises 4px. The toast and login form already use it — open them to see it in flesh.
    </p>
  </section>
</main>
