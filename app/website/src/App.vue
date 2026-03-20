<template>
  <div class="flex h-screen overflow-hidden">
    <DashboardPage @open-quick-setup="isQuickSetupOpen = true" />
    <SettingsPage />
    <CertManagementModal />
    <QuickSetupModal :open="isQuickSetupOpen" @close="isQuickSetupOpen = false" />
    <CompatibilityAnchors />
  </div>
</template>

<script setup>
import { ref } from 'vue';
import DashboardPage from './components/DashboardPage.vue';
import SettingsPage from './components/SettingsPage.vue';
import CertManagementModal from './components/CertManagementModal.vue';
import QuickSetupModal from './components/QuickSetupModal.vue';
import CompatibilityAnchors from './components/CompatibilityAnchors.vue';

const isQuickSetupOpen = ref(false);
</script>

<style>
@tailwind base;
@tailwind components;
@tailwind utilities;
</style>
