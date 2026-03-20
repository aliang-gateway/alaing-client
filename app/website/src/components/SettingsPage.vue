<template>
  <div
    id="settings-page"
    class="page-container content-section hidden flex-1 min-w-0 h-full overflow-hidden bg-background-light dark:bg-background-dark"
  >
    <div class="relative flex min-h-screen w-full flex-col overflow-hidden">
      <header class="sticky top-0 z-50 w-full border-b border-primary/10 bg-white/80 backdrop-blur-md dark:bg-background-dark/80">
        <div class="mx-auto flex h-16 w-full max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
          <div class="flex items-center gap-3">
            <div class="flex h-8 w-8 items-center justify-center rounded bg-primary text-white">
              <span class="material-symbols-outlined text-xl">shield_with_heart</span>
            </div>
            <span class="text-lg font-bold tracking-tight">ALiang Gateway</span>
          </div>
          <nav class="hidden items-center gap-8 md:flex">
            <a class="border-b-2 border-primary py-5 text-sm font-medium text-primary" href="javascript:void(0)">Settings</a>
          </nav>
          <div class="flex items-center gap-4">
            <div class="hidden text-right text-xs lg:flex lg:flex-col">
              <span class="font-bold text-slate-900 dark:text-white">John Doe</span>
              <span class="flex items-center gap-1 text-primary">
                <span class="h-1.5 w-1.5 rounded-full bg-primary"></span>
                Pro Member
              </span>
            </div>
            <div
              class="h-10 w-10 rounded border border-primary/20 bg-slate-100 bg-cover bg-center"
              style="background-image: url('https://lh3.googleusercontent.com/aida-public/AB6AXuBTPPt9zzYRYAr26_d0HjwFLfA_naITBgueuE6BCbNI5fKlasMcI9X_y2Dg2HWaF2ahcriYFLMXUEMfgpfeCrzOKQnFERDgabAlU8FNjAK6-W9gGCpNTHzbshzH7rPPAaQlOds8d9PI9iKDIXmUbAdZcjClJQmz6yJfHCl-N07NQowejz85L_OsBGeWZqlpt-AQhf9WqWw0ybuMkfCwDVVWZEQknrAH_-ODbr_ZDOC1aYz0fDcP0-sw3vHEawkqCTXHJHqCzhCx4Rc')"
            ></div>
          </div>
        </div>
      </header>

      <main class="mx-auto w-full max-w-7xl grow p-4 sm:p-6 lg:p-8">
        <div class="grid grid-cols-1 gap-8 lg:grid-cols-12">
          <section class="flex flex-col gap-4 lg:col-span-8">
            <RulesSettings />
            <LogsSettings />
          </section>
          <aside class="flex flex-col gap-6 lg:col-span-4">
            <UserInfoSettings />
            <SystemSettings />
          </aside>
        </div>
      </main>

      <footer class="mt-auto border-t border-slate-100 py-6 dark:border-slate-800">
        <div class="mx-auto flex max-w-7xl items-center justify-between px-4 text-[10px] font-medium text-slate-400 sm:px-6 lg:px-8">
          <div>© 2024 ALiang Gateway. All rights reserved.</div>
          <div class="flex gap-4 uppercase tracking-tighter">
            <a class="hover:text-primary" href="javascript:void(0)">Docs</a>
            <a class="hover:text-primary" href="javascript:void(0)">Privacy</a>
            <a class="hover:text-primary" href="javascript:void(0)">Github</a>
          </div>
        </div>
      </footer>

      <div class="compat-anchors hidden" aria-hidden="true">
        <button type="button" id="backToDashboard"></button>
        <button type="button" class="settings-tab active" data-tab="rules"></button>
        <button type="button" class="settings-tab" data-tab="logs"></button>
        <button type="button" class="settings-tab" data-tab="userinfo"></button>
        <button type="button" class="settings-tab" data-tab="system"></button>
        <div class="settings-content active" data-content="rules"></div>
        <div class="settings-content hidden" data-content="logs"></div>
        <div class="settings-content hidden" data-content="userinfo"></div>
        <div class="settings-content hidden" data-content="system"></div>
      </div>
    </div>
  </div>
</template>

<script>
import RulesSettings from './settings/RulesSettings.vue';
import UserInfoSettings from './settings/UserInfoSettings.vue';
import LogsSettings from './settings/LogsSettings.vue';
import SystemSettings from './settings/SystemSettings.vue';

export default {
  name: 'SettingsPage',
  components: {
    RulesSettings,
    UserInfoSettings,
    LogsSettings,
    SystemSettings
  }
}
</script>
