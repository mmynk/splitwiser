import './app.css';
import { mount } from 'svelte';
import App from './App.svelte';
import './lib/stores/theme'; // initialize theme subscription (applies stored preference)

const target = document.getElementById('app');
if (!target) {
  throw new Error('Missing #app mount point');
}

mount(App, { target });
