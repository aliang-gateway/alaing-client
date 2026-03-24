import { ref } from 'vue';

const currentPage = ref('dashboard');

export function useNavigation() {
  function showSettings() {
    currentPage.value = 'settings';
  }

  function showDashboard() {
    currentPage.value = 'dashboard';
  }

  return {
    currentPage,
    showSettings,
    showDashboard
  };
}
