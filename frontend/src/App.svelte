<script lang="ts">
  import Router, { replace, type RouteDetailLoaded } from 'svelte-spa-router';
  import { fly } from 'svelte/transition';
  import { isLoggedIn } from '$lib/stores/auth';
  import Nav from '$lib/components/Nav.svelte';
  import Toast from '$lib/components/Toast.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import ShortcutsOverlay from '$lib/components/ShortcutsOverlay.svelte';
  import { dur, ease } from '$lib/motion';
  import Login from './routes/Login.svelte';
  import Home from './routes/Home.svelte';
  import Bill from './routes/Bill.svelte';
  import Groups from './routes/Groups.svelte';
  import Group from './routes/Group.svelte';
  import Friends from './routes/Friends.svelte';
  import StyleGuide from './routes/StyleGuide.svelte';
  import NotFound from './routes/NotFound.svelte';

  const routes = {
    '/': Home,
    '/login': Login,
    '/bill/:id': Bill,
    '/groups': Groups,
    '/group/:id': Group,
    '/friends': Friends,
    '/_styles': StyleGuide,
    '*': NotFound,
  };

  const PUBLIC_PATHS = new Set(['/login', '/_styles']);

  let routeKey = $state(0);
  let lastPath = '';

  function routeLoaded(event: { detail: RouteDetailLoaded }) {
    const path = event.detail.location;
    const authed = isLoggedIn();
    if (path === '/login' && authed) {
      replace('/');
      return;
    }
    if (!PUBLIC_PATHS.has(path) && !authed) {
      replace('/login');
      return;
    }
    // Remount the route subtree only when the path actually changes — otherwise
    // svelte-spa-router's same-route navigations would re-fire every onMount,
    // re-fetching data needlessly.
    if (path !== lastPath) {
      lastPath = path;
      routeKey += 1;
    }
  }
</script>

<Nav />
{#key routeKey}
  <div in:fly={{ y: 4, duration: dur, easing: ease }}>
    <Router {routes} on:routeLoaded={routeLoaded} />
  </div>
{/key}
<Toast />
<ConfirmDialog />
<ShortcutsOverlay />
