<script lang="ts">
  import { replace } from 'svelte-spa-router';
  import { login } from '$lib/stores/auth';
  import { loginApi, registerApi } from '$lib/api/auth';
  import { ApiError } from '$lib/api/client';

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
      error = 'Password must be at least 8 characters long';
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
      error = e instanceof ApiError ? e.message : 'Registration failed. Please try again.';
    } finally {
      submitting = false;
    }
  }
</script>

<main class="mx-auto flex min-h-screen max-w-md flex-col px-4 py-10">
  <header class="mb-8 text-center">
    <h1 class="text-3xl font-bold text-slate-900">Splitwiser</h1>
    <p class="mt-1 text-slate-600">Split bills fairly with friends</p>
  </header>

  <div class="mb-6 grid grid-cols-2 gap-2 rounded-lg bg-white p-1 shadow-sm ring-1 ring-slate-200">
    <button
      type="button"
      disabled={submitting}
      class="rounded-md py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:cursor-not-allowed disabled:opacity-60"
      class:bg-brand-600={activeTab === 'login'}
      class:text-white={activeTab === 'login'}
      class:text-slate-600={activeTab !== 'login'}
      onclick={() => switchTab('login')}
    >
      Login
    </button>
    <button
      type="button"
      disabled={submitting}
      class="rounded-md py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:cursor-not-allowed disabled:opacity-60"
      class:bg-brand-600={activeTab === 'register'}
      class:text-white={activeTab === 'register'}
      class:text-slate-600={activeTab !== 'register'}
      onclick={() => switchTab('register')}
    >
      Register
    </button>
  </div>

  {#if error}
    <div
      role="alert"
      class="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700"
    >
      {error}
    </div>
  {/if}

  <div class="rounded-lg bg-white p-6 shadow-sm ring-1 ring-slate-200">
    {#if activeTab === 'login'}
      <h2 class="mb-4 text-xl font-semibold">Welcome Back</h2>
      <form onsubmit={handleLogin} class="flex flex-col gap-4">
        <label class="flex flex-col gap-1 text-sm">
          <span class="font-medium text-slate-700">Email</span>
          <input
            type="email"
            bind:value={loginEmail}
            oninput={clearError}
            required
            autocomplete="email"
            placeholder="your@email.com"
            class="rounded-md border border-slate-300 px-3 py-2 outline-none focus:border-brand-500 focus:ring-2 focus:ring-brand-100"
          />
        </label>
        <label class="flex flex-col gap-1 text-sm">
          <span class="font-medium text-slate-700">Password</span>
          <input
            type="password"
            bind:value={loginPassword}
            oninput={clearError}
            required
            autocomplete="current-password"
            placeholder="Enter your password"
            class="rounded-md border border-slate-300 px-3 py-2 outline-none focus:border-brand-500 focus:ring-2 focus:ring-brand-100"
          />
        </label>
        <button
          type="submit"
          disabled={submitting}
          class="mt-2 rounded-md bg-brand-600 px-4 py-2 font-medium text-white transition hover:bg-brand-700 disabled:opacity-60"
        >
          {submitting ? 'Logging in…' : 'Login'}
        </button>
      </form>
    {:else}
      <h2 class="mb-4 text-xl font-semibold">Create Account</h2>
      <form onsubmit={handleRegister} class="flex flex-col gap-4">
        <label class="flex flex-col gap-1 text-sm">
          <span class="font-medium text-slate-700">Display Name</span>
          <input
            type="text"
            bind:value={registerName}
            oninput={clearError}
            required
            autocomplete="name"
            placeholder="Your Name"
            class="rounded-md border border-slate-300 px-3 py-2 outline-none focus:border-brand-500 focus:ring-2 focus:ring-brand-100"
          />
          <small class="text-slate-500">This is how you'll appear to other users.</small>
        </label>
        <label class="flex flex-col gap-1 text-sm">
          <span class="font-medium text-slate-700">Email</span>
          <input
            type="email"
            bind:value={registerEmail}
            oninput={clearError}
            required
            autocomplete="email"
            placeholder="your@email.com"
            class="rounded-md border border-slate-300 px-3 py-2 outline-none focus:border-brand-500 focus:ring-2 focus:ring-brand-100"
          />
        </label>
        <label class="flex flex-col gap-1 text-sm">
          <span class="font-medium text-slate-700">Password</span>
          <input
            type="password"
            bind:value={registerPassword}
            oninput={clearError}
            required
            autocomplete="new-password"
            placeholder="At least 8 characters"
            class="rounded-md border border-slate-300 px-3 py-2 outline-none focus:border-brand-500 focus:ring-2 focus:ring-brand-100"
          />
          <small class="text-slate-500">Must be at least 8 characters long.</small>
        </label>
        <button
          type="submit"
          disabled={submitting}
          class="mt-2 rounded-md bg-brand-600 px-4 py-2 font-medium text-white transition hover:bg-brand-700 disabled:opacity-60"
        >
          {submitting ? 'Creating account…' : 'Create Account'}
        </button>
      </form>
    {/if}
  </div>
</main>
