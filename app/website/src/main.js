import { createApp } from 'vue';
import * as bootstrap from 'bootstrap';
import Chart from 'chart.js/auto';
import App from './App.vue';
import { loadLegacyScripts } from './legacy/loadLegacyScripts';
import 'bootstrap/dist/css/bootstrap.min.css';
import '../assets/styles.css';

window.bootstrap = bootstrap;
window.Chart = Chart;

createApp(App).mount('#app');

loadLegacyScripts();
