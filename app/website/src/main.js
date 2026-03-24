import { createApp } from 'vue';
import * as bootstrap from 'bootstrap';
import Chart from 'chart.js/auto';
import App from './App.vue';
import { loadLegacyScripts } from './legacy/loadLegacyScripts';
import '../assets/styles.css';

window.bootstrap = bootstrap;
window.Chart = Chart;

async function bootstrapApp() {
  await loadLegacyScripts();
  createApp(App).mount('#app');
  window.dispatchEvent(new Event('app:mounted'));
}

bootstrapApp();

