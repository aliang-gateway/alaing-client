<template>
  <div class="settings-content hidden" data-content="logs">
    <div class="settings-card">
      <div class="flex flex-wrap justify-between items-center gap-3 mb-4">
        <h3 class="settings-card-title !mb-0">日志监控</h3>
        <div class="flex flex-wrap gap-2">
          <select id="logLevelSelect" class="px-3 py-2 border border-slate-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary">
            <option value="all">All</option>
            <option value="info">Info</option>
            <option value="warn">Warn</option>
            <option value="error">Error</option>
          </select>
          <select id="logSourceSelect" class="px-3 py-2 border border-slate-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary">
            <option value="all">All</option>
          </select>
          <button type="button" id="logsRefreshBtn" class="settings-btn-outline">刷新</button>
          <button type="button" id="logsClearBtn" class="settings-btn-outline">清空</button>
          <button type="button" id="wsConnectBtn" class="settings-btn-primary">连接</button>
          <button type="button" id="wsDisconnectBtn" class="settings-btn-secondary">断开</button>
        </div>
      </div>
      <div id="logsOutput" class="h-96 overflow-y-auto bg-slate-50 dark:bg-slate-800 p-4 rounded border border-slate-100 dark:border-slate-700 font-mono text-xs"></div>
      <div id="wsConnectionStatus" class="mt-2 text-xs text-slate-500"></div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'LogsSettings'
}
</script>