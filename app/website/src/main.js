import { createApp } from 'vue';
import App from './App.vue';
import '../assets/styles.css';
import { restoreAuthSession } from './stores/auth';

async function bootstrap() {
  await restoreAuthSession();
  createApp(App).mount('#app');
}

bootstrap();
