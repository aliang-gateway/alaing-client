<template>
  <div class="settings-pane" data-pane="logs">
    <div class="mt-4 flex flex-col gap-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <h2 class="text-xl font-bold">Log Monitoring</h2>
          <span class="rounded-full bg-primary/20 px-2 py-0.5 text-[10px] font-bold uppercase text-primary">Live</span>
        </div>
        <div class="flex items-center gap-2">
          <select id="logLevelSelect" class="rounded border border-slate-200 bg-white px-2 py-1 text-xs dark:border-slate-800 dark:bg-slate-900">
            <option value="debug">Level: Debug</option>
            <option value="info">Level: Info</option>
            <option value="warn">Level: Warning</option>
            <option value="error">Level: Error</option>
            <option value="all">Level: All</option>
          </select>
          <button type="button" id="wsDisconnectBtn" class="rounded p-1.5 transition-colors hover:bg-slate-100 dark:hover:bg-slate-800">
            <span class="material-symbols-outlined text-lg">stop_circle</span>
          </button>
          <button type="button" id="logsRefreshBtn" class="rounded p-1.5 transition-colors hover:bg-slate-100 dark:hover:bg-slate-800">
            <span class="material-symbols-outlined text-lg">refresh</span>
          </button>
        </div>
      </div>

      <div class="h-64 overflow-y-auto rounded-xl border border-slate-200 bg-slate-950 p-4 font-mono text-xs text-slate-300 dark:border-slate-800">
        <div id="logsOutput" class="whitespace-pre-wrap break-all"></div>
      </div>
    </div>

    <div class="compat-anchors" aria-hidden="true">
      <button id="logsClearBtn" type="button"></button>
      <select id="logSourceSelect"><option value="all">All</option></select>
      <input id="logAutoScroll" type="checkbox" checked />
      <button id="logConfigSaveBtn" type="button"></button>
      <button id="logFilterBtn" type="button"></button>
      <button id="wsConnectBtn" type="button"></button>
      <button id="logsFullscreenBtn" type="button"><i class="bi bi-fullscreen"></i></button>
      <div id="logs-page"></div>
      <div id="wsConnectionStatus"></div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'LogsSettings'
}
</script>
