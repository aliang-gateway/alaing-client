import { createApp } from 'vue';
import App from './App.vue';
import '../assets/styles.css';
import { restoreAuthSession } from './stores/auth';
import { initializeTheme } from './composables/useTheme';

async function bootstrap() {
  initializeTheme();
  await restoreAuthSession();
  createApp(App).mount('#app');
}

bootstrap();
