<template>
  <div
    id="settings-page"
    class="page-container content-section hidden flex-1 min-w-0 h-full overflow-hidden bg-background-light dark:bg-background-dark"
  >
    <div class="flex h-full overflow-hidden">
      <!-- Left Sidebar/Panel -->
      <aside class="w-80 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800 flex flex-col h-full overflow-y-auto custom-scrollbar">
        <!-- Header Block -->
        <div class="p-8 border-b border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/20">
          <div class="flex items-center gap-4">
            <div class="relative group">
              <div class="size-14 ring-2 ring-primary ring-offset-2 dark:ring-offset-slate-900 rounded-xl overflow-hidden shadow-sm flex items-center justify-center bg-white dark:bg-slate-800">
                <span class="material-symbols-outlined text-primary text-3xl">admin_panel_settings</span>
              </div>
              <div class="absolute -bottom-1 -right-1 size-4 bg-primary border-2 border-white dark:border-slate-900 rounded-full"></div>
            </div>
            <div>
              <div class="flex items-center gap-1.5">
                <h2 class="font-bold text-slate-900 dark:text-white tracking-tight">System Settings</h2>
              </div>
              <div class="flex items-center gap-2 mt-1">
                <span class="px-2 py-0.5 bg-primary/10 text-primary text-[10px] font-bold rounded uppercase tracking-wider">Control Panel</span>
              </div>
              <p class="text-slate-400 text-[11px] mt-1.5 font-medium">Gateway configuration</p>
            </div>
          </div>
        </div>

        <!-- Modules Menu -->
        <div class="px-6 pt-6 pb-4 space-y-3">
          <p class="text-[10px] font-bold text-slate-400 uppercase tracking-widest px-1">Settings Modules</p>
          <button type="button" class="settings-tab active" data-tab="rules">
            <span class="material-symbols-outlined text-lg">tune</span>
            规则设置
          </button>
          <button type="button" class="settings-tab" data-tab="userinfo">
            <span class="material-symbols-outlined text-lg">person</span>
            用户信息
          </button>
          <button type="button" class="settings-tab" data-tab="logs">
            <span class="material-symbols-outlined text-lg">terminal</span>
            日志监控
          </button>
          <button type="button" class="settings-tab" data-tab="system">
            <span class="material-symbols-outlined text-lg">dns</span>
            系统设置
          </button>
        </div>

        <!-- Status -->
        <div class="mt-auto p-6 border-t border-slate-100 dark:border-slate-800 bg-slate-50/30 dark:bg-slate-800/10">
          <div class="flex justify-between items-center mb-3">
            <span class="text-xs font-bold text-slate-500 uppercase tracking-wider">Gateway Status</span>
            <span class="text-xs font-bold text-primary">Connected</span>
          </div>
          <div class="space-y-2">
            <div class="flex justify-between text-xs">
              <span class="text-slate-500">Profile:</span>
              <span class="font-medium text-slate-700 dark:text-slate-200">Default</span>
            </div>
            <div class="flex justify-between text-xs">
              <span class="text-slate-500">Last Sync:</span>
              <span class="font-medium text-slate-700 dark:text-slate-200">Just now</span>
            </div>
          </div>
        </div>
      </aside>

      <!-- Main Content Area -->
      <main class="flex-1 flex flex-col h-full bg-background-light dark:bg-background-dark overflow-hidden">
        <!-- Top Navigation Bar -->
        <header class="h-16 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between px-8 shrink-0">
          <div class="flex items-center gap-4">
            <button type="button" id="backToDashboard" class="size-8 flex items-center justify-center rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 transition-colors">
              <span class="material-symbols-outlined">arrow_back</span>
            </button>
            <div class="h-8 w-px bg-slate-200 dark:bg-slate-800 mx-2"></div>
            <h1 class="font-bold text-xl tracking-tight">Configuration <span class="text-primary font-medium">Center</span></h1>
          </div>
          <div class="flex items-center gap-4">
            <div class="px-3 py-1 bg-primary/10 text-primary text-xs font-bold rounded">LIVE</div>
          </div>
        </header>

        <div class="flex-1 overflow-y-auto p-8 custom-scrollbar">
          <RulesSettings />
          <UserInfoSettings />
          <LogsSettings />
          <SystemSettings />
        </div>
      </main>
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