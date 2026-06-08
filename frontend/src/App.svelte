<script lang="ts">
  import Router, { replace, type RouteDetailLoaded } from 'svelte-spa-router';
  import { isLoggedIn } from '$lib/stores/auth';
  import Nav from '$lib/components/Nav.svelte';
  import Toast from '$lib/components/Toast.svelte';
  import Login from './routes/Login.svelte';
  import Home from './routes/Home.svelte';
  import Bill from './routes/Bill.svelte';
  import Groups from './routes/Groups.svelte';
  import Group from './routes/Group.svelte';
  import Friends from './routes/Friends.svelte';
  import StyleGuide from './routes/StyleGuide.svelte';

  const routes = {
    '/': Home,
    '/login': Login,
    '/bill/:id': Bill,
    '/groups': Groups,
    '/group/:id': Group,
    '/friends': Friends,
    '/_styles': StyleGuide,
    '*': Login,
  };

  const PUBLIC_PATHS = new Set(['/login', '/_styles']);

  function routeLoaded(event: { detail: RouteDetailLoaded }) {
    const path = event.detail.location;
    const authed = isLoggedIn();
    if (path === '/login' && authed) {
      replace('/');
    } else if (!PUBLIC_PATHS.has(path) && !authed) {
      replace('/login');
    }
  }
</script>

<Nav />
<Router {routes} on:routeLoaded={routeLoaded} />
<Toast />
