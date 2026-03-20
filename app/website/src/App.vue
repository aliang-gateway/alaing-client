<template>
  <div class="flex h-screen overflow-hidden">
    <DashboardPage />
    <SettingsPage />
    <CertManagementModal />
    <CompatibilityAnchors />
  </div>
</template>

<script setup>
import DashboardPage from './components/DashboardPage.vue';
import SettingsPage from './components/SettingsPage.vue';
import CertManagementModal from './components/CertManagementModal.vue';
import CompatibilityAnchors from './components/CompatibilityAnchors.vue';
</script>

<style>
@tailwind base;
@tailwind components;
@tailwind utilities;
</style>
