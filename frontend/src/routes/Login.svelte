<script lang="ts">
  import { replace } from 'svelte-spa-router';
  import { fly } from 'svelte/transition';
  import { login } from '$lib/stores/auth';
  import { loginApi, registerApi } from '$lib/api/auth';
  import { ApiError } from '$lib/api/client';
  import { rise } from '$lib/motion';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Card from '$lib/components/ui/Card.svelte';

  let activeTab: 'login' | 'register' = $state('login');
  let error = $state('');
  let submitting = $state(false);

  let loginEmail = $state('');
  let loginPassword = $state('');

  let registerName = $state('');
  let registerEmail = $state('');
  let registerPassword = $state('');

  function switchTab(tab: 'login' | 'register') {
    if (submitting) return;
    activeTab = tab;
    error = '';
    // Prevent the opposite tab's typed password from being submitted
    // by accident if the user toggles between the forms.
    loginPassword = '';
    registerPassword = '';
  }

  function clearError() {
    if (error) error = '';
  }

  async function handleLogin(event: SubmitEvent) {
    event.preventDefault();
    error = '';
    submitting = true;
    try {
      const data = await loginApi(loginEmail.trim(), loginPassword);
      login(data.token, data.user);
      replace('/');
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Invalid email or password.';
    } finally {
      submitting = false;
    }
  }

  async function handleRegister(event: SubmitEvent) {
    event.preventDefault();
    error = '';
    if (registerPassword.length < 8) {
      error = 'Password needs at least 8 characters.';
      return;
    }
    submitting = true;
    try {
      const data = await registerApi(
        registerEmail.trim(),
        registerPassword,
        registerName.trim(),
      );
      login(data.token, data.user);
      replace('/');
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not create the account. Try again.';
    } finally {
      submitting = false;
    }
  }
</script>

<main class="mx-auto flex min-h-screen max-w-md flex-col justify-center px-5 py-10">
  <header class="mb-8 text-center">
    <h1 class="display-wonk text-4xl font-semibold text-text" style="letter-spacing: -0.02em;">
      Splitwiser
    </h1>
    <p class="mt-2 text-[0.9375rem] text-text-muted">
      Split bills fairly. Settle up without the awkwardness.
    </p>
  </header>

  <div
    class="mb-5 grid grid-cols-2 gap-1 rounded-input bg-surface-sunken p-1 border border-border"
    role="tablist"
    aria-label="Account"
  >
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'login'}
      disabled={submitting}
      class={[
        'rounded-[4px] py-1.5 text-[0.8125rem] font-medium transition-colors',
        'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
        'disabled:cursor-not-allowed disabled:opacity-55',
        activeTab === 'login'
          ? 'bg-surface-elevated text-text shadow-pop'
          : 'text-text-muted hover:text-text',
      ]}
      onclick={() => switchTab('login')}
    >
      Log in
    </button>
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'register'}
      disabled={submitting}
      class={[
        'rounded-[4px] py-1.5 text-[0.8125rem] font-medium transition-colors',
        'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
        'disabled:cursor-not-allowed disabled:opacity-55',
        activeTab === 'register'
          ? 'bg-surface-elevated text-text shadow-pop'
          : 'text-text-muted hover:text-text',
      ]}
      onclick={() => switchTab('register')}
    >
      Sign up
    </button>
  </div>

  {#if error}
    <div
      role="alert"
      transition:fly={rise}
      class="mb-4 rounded-input border border-danger/30 bg-danger-soft px-3 py-2 text-[0.875rem] text-danger"
    >
      {error}
    </div>
  {/if}

  <Card padding="lg">
    {#if activeTab === 'login'}
      <div in:fly={rise}>
        <h2 class="mb-5 text-xl">Welcome back</h2>
        <form onsubmit={handleLogin} class="flex flex-col gap-4">
          <Input
            label="Email"
            type="email"
            bind:value={loginEmail}
            oninput={clearError}
            required
            autocomplete="email"
            placeholder="you@example.com"
          />
          <Input
            label="Password"
            type="password"
            bind:value={loginPassword}
            oninput={clearError}
            required
            autocomplete="current-password"
            placeholder="Your password"
          />
          <div class="mt-1">
            <Button type="submit" loading={submitting} fullWidth>
              {submitting ? 'Logging in…' : 'Log in'}
            </Button>
          </div>
        </form>
      </div>
    {:else}
      <div in:fly={rise}>
        <h2 class="mb-5 text-xl">Make an account</h2>
        <form onsubmit={handleRegister} class="flex flex-col gap-4">
          <Input
            label="Display name"
            hint="How friends will see you."
            type="text"
            bind:value={registerName}
            oninput={clearError}
            required
            autocomplete="name"
            placeholder="Your name"
          />
          <Input
            label="Email"
            type="email"
            bind:value={registerEmail}
            oninput={clearError}
            required
            autocomplete="email"
            placeholder="you@example.com"
          />
          <Input
            label="Password"
            hint="At least 8 characters."
            type="password"
            bind:value={registerPassword}
            oninput={clearError}
            required
            autocomplete="new-password"
            placeholder="Pick something strong"
          />
          <div class="mt-1">
            <Button type="submit" loading={submitting} fullWidth>
              {submitting ? 'Creating account…' : 'Create account'}
            </Button>
          </div>
        </form>
      </div>
    {/if}
  </Card>

  <p class="mt-6 text-center text-[0.75rem] text-text-subtle">
    Free, open source, and a little bit warmer than the rest.
  </p>
</main>
