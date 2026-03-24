<template>
  <div class="flex h-screen overflow-hidden">
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
  </div>
</template>

<script setup>
import { ref } from 'vue';
import DashboardPage from './components/DashboardPage.vue';
import SettingsPage from './components/SettingsPage.vue';
import CertManagementModal from './components/CertManagementModal.vue';
import QuickSetupModal from './components/QuickSetupModal.vue';

const isQuickSetupOpen = ref(false);
const isCertModalOpen = ref(false);
const certModalRef = ref(null);
</script>

<style>
@tailwind base;
@tailwind components;
@tailwind utilities;
</style>
