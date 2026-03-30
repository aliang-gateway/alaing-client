<template>
  <div class="flex h-screen overflow-hidden">
    <div
      v-if="!isReady"
      class="flex flex-1 items-center justify-center bg-slate-50 text-slate-600 dark:bg-slate-950 dark:text-slate-200"
    >
      <div class="rounded-xl border border-slate-200 bg-white px-6 py-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        Restoring saved session...
      </div>
    </div>
    <template v-else>
    <DashboardPage
      @open-quick-setup="isQuickSetupOpen = true"
      @open-cert-modal="isCertModalOpen = true"
      @start-cert-reinstall="certModalRef?.startReinstall()"
    />
    <SettingsPage />
    <CertManagementModal
      ref="certModalRef"
      v-model="isCertModalOpen"
    />
    <QuickSetupModal :open="isQuickSetupOpen" @close="isQuickSetupOpen = false" />
    </template>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue';
import DashboardPage from './components/DashboardPage.vue';
import SettingsPage from './components/SettingsPage.vue';
import CertManagementModal from './components/CertManagementModal.vue';
import QuickSetupModal from './components/QuickSetupModal.vue';
import { useAuthStore } from './stores/auth';
import { getUserCenterProfile } from './services/userCenterApi';
import { clearChatIdentityProfileCache } from './utils/chatIdentityCache';

const isQuickSetupOpen = ref(false);
const isCertModalOpen = ref(false);
const certModalRef = ref(null);
const { isReady, isAuthenticated } = useAuthStore();

async function syncChatIdentityProfile() {
  if (!isReady.value) {
    return;
  }

  if (!isAuthenticated.value) {
    clearChatIdentityProfileCache();
    return;
  }

  try {
    await getUserCenterProfile();
  } catch (_) {
    // Keep the last successful browser cache available for chat fallback.
  }
}

watch([isReady, isAuthenticated], () => {
  void syncChatIdentityProfile();
}, { immediate: true });
</script>

<style>
@tailwind base;
@tailwind components;
@tailwind utilities;
</style>
