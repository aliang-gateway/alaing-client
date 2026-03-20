<template>
  <div class="settings-content hidden" data-content="system">
    <div class="settings-card">
      <h3 class="settings-card-title">系统设置</h3>
      <div class="space-y-6">
        <div>
          <h4 class="text-sm font-semibold mb-2">运行模式</h4>
          <div class="flex gap-4">
            <label class="flex items-center gap-2 px-3 py-2 rounded border border-slate-200 dark:border-slate-700">
              <input type="radio" name="runMode" id="runModeTun" value="tun" />
              <span>TUN</span>
            </label>
            <label class="flex items-center gap-2 px-3 py-2 rounded border border-slate-200 dark:border-slate-700">
              <input type="radio" name="runMode" id="runModeHttp" value="http" />
              <span>HTTP</span>
            </label>
          </div>
          <div class="mt-3 flex flex-wrap gap-2">
            <button type="button" id="runStartBtn" class="settings-btn-primary">启动</button>
            <button type="button" id="runStopBtn" class="settings-btn-secondary">停止</button>
            <button type="button" id="runModeBtn" class="settings-btn-outline">切换模式</button>
          </div>
          <div class="mt-3 text-sm text-slate-500">
            <span>当前模式:</span>
            <span id="runCurrentMode">-</span>
          </div>
          <div id="runServiceStatus" class="text-sm text-slate-500"></div>
          <div id="runAvailableModes" class="text-sm text-slate-500"></div>
          <div id="runStatusInfo" class="text-sm text-slate-500"></div>
        </div>
        <div>
          <h4 class="text-sm font-semibold mb-2">证书管理</h4>
          <div class="p-4 rounded border border-slate-200 dark:border-slate-700 bg-slate-50/70 dark:bg-slate-800/30">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div class="text-sm font-semibold text-slate-700 dark:text-slate-200">Certificate Management Modal</div>
                <div class="text-xs text-slate-500 dark:text-slate-400 mt-1">查看状态、导出、下载、安装与移除证书</div>
              </div>
              <button type="button" id="openCertManagementModalBtn" class="settings-btn-primary">打开证书弹窗</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'SystemSettings'
}
</script>