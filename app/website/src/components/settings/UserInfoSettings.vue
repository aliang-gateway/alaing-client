<template>
  <div class="settings-pane" data-pane="userinfo">
    <div class="rounded-xl border border-slate-200 bg-white p-5 dark:border-slate-800 dark:bg-background-dark">
      <h3 class="mb-4 flex items-center gap-2 font-bold">
        <span class="material-symbols-outlined text-primary">person</span>
        User Information
      </h3>

      <div class="flex items-center justify-between rounded bg-slate-50 p-3 dark:bg-slate-800/50">
        <div>
          <p class="text-xs text-slate-500">Current Plan</p>
          <p class="font-bold uppercase tracking-wider text-slate-900 dark:text-white">Ultra Plus</p>
        </div>
        <div class="text-right">
          <p class="text-xs text-slate-500">Balance</p>
          <p id="userBalance" class="font-bold text-primary">-</p>
        </div>
      </div>

      <div class="mt-4 space-y-2">
        <p class="px-1 text-xs font-semibold uppercase text-slate-400">API Keys</p>
        <div class="space-y-1">
          <div class="group flex items-center justify-between rounded p-2 transition-colors hover:bg-slate-50 dark:hover:bg-slate-800">
            <div class="flex items-center gap-2">
              <span class="material-symbols-outlined text-sm text-slate-400">key</span>
              <span class="text-xs font-medium">OpenAI Key</span>
            </div>
            <span class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-[10px] text-slate-400 dark:bg-slate-700">sk-••••w9X1</span>
          </div>
          <div class="group flex items-center justify-between rounded p-2 transition-colors hover:bg-slate-50 dark:hover:bg-slate-800">
            <div class="flex items-center gap-2">
              <span class="material-symbols-outlined text-sm text-slate-400">key</span>
              <span class="text-xs font-medium">Claude Key</span>
            </div>
            <span class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-[10px] text-slate-400 dark:bg-slate-700">sk-ant-••••2K9L</span>
          </div>
        </div>
      </div>

      <div class="mt-4 border-t border-slate-100 pt-4 dark:border-slate-800">
        <p class="px-1 text-xs font-semibold uppercase text-slate-400">Configurations</p>
        <div class="mt-1 space-y-1">
          <div class="flex items-center justify-between p-2">
            <span class="text-xs font-medium">opencode.json</span>
            <span class="material-symbols-outlined text-sm text-primary">cloud_done</span>
          </div>
          <div class="flex items-center justify-between p-2">
            <span class="text-xs font-medium">Oh-my-opencode.json</span>
            <span class="material-symbols-outlined text-sm text-slate-300">cloud_upload</span>
          </div>
        </div>
      </div>

      <div class="compat-anchors" aria-hidden="true">
        <div id="authUserInfoContainer"></div>
        <button id="authLogoutBtn" type="button"></button>
        <div id="authRefreshStatusCard"></div>
        <div id="authRefreshRunning"></div>
        <div id="authLastUpdate"></div>
        <div id="authRefreshInterval"></div>
        <div id="authRefreshError"></div>
        <div id="authRefreshErrorMsg"></div>
        <button id="authActivateBtn" type="button"></button>
        <input id="authTokenInput" type="text" />
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'UserInfoSettings'
}
</script>
