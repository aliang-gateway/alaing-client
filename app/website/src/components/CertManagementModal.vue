<template>
  <div id="certManagementModal" class="fixed inset-0 z-[120] hidden items-center justify-center p-4">
    <div id="certManagementModalBackdrop" class="absolute inset-0 bg-slate-900/45 backdrop-blur-sm"></div>
    <div
      class="relative z-10 w-full max-w-2xl bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 shadow-2xl overflow-hidden"
    >
      <div class="px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
        <div>
          <h3 class="text-lg font-bold text-slate-900 dark:text-slate-100">Certificate Management</h3>
          <p class="text-xs text-slate-500 dark:text-slate-400 mt-1">参考 Networking Dashboard 证书弹窗样式</p>
        </div>
        <button
          type="button"
          id="certModalCloseBtn"
          aria-label="Close certificate modal"
          class="size-9 rounded-full flex items-center justify-center text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800"
        >
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>

      <div class="p-6 space-y-5">
        <div class="flex items-center gap-3 flex-wrap">
          <select
            id="cert-type-select"
            class="px-3 py-2 border border-slate-300 dark:border-slate-600 rounded text-sm focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary bg-white dark:bg-slate-800"
          >
            <option value="mitm-ca">MITM CA</option>
            <option value="root-ca">Root CA</option>
            <option value="mtls-cert">mTLS Certificate</option>
          </select>
          <button
            type="button"
            id="btn-check-cert"
            class="px-3 py-2 border border-slate-300 dark:border-slate-600 rounded text-sm hover:bg-slate-50 dark:hover:bg-slate-800"
          >
            检查状态
          </button>
        </div>

        <div
          id="cert-status-container"
          class="p-4 rounded-lg bg-slate-50 dark:bg-slate-800/40 border border-slate-200 dark:border-slate-700"
        >
          <div id="cert-status-content" class="text-sm text-slate-500"></div>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <button
            type="button"
            id="btn-export-cert"
            class="flex items-center gap-3 px-4 py-3 border border-slate-200 dark:border-slate-700 rounded-lg text-sm hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
          >
            <span class="material-symbols-outlined text-slate-500">upload_file</span>
            <span>导出证书</span>
          </button>
          <button
            type="button"
            id="btn-install-cert"
            class="flex items-center gap-3 px-4 py-3 border border-slate-200 dark:border-slate-700 rounded-lg text-sm hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
          >
            <span class="material-symbols-outlined text-slate-500">check_circle</span>
            <span>安装到系统</span>
          </button>
          <button
            type="button"
            id="btn-download-cert"
            class="flex items-center gap-3 px-4 py-3 border border-slate-200 dark:border-slate-700 rounded-lg text-sm hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
          >
            <span class="material-symbols-outlined text-slate-500">download</span>
            <span>下载 PEM</span>
          </button>
          <button
            type="button"
            id="btn-remove-cert"
            class="flex items-center gap-3 px-4 py-3 border border-red-200 dark:border-red-500/40 rounded-lg text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
          >
            <span class="material-symbols-outlined text-red-500">delete</span>
            <span>移除证书</span>
          </button>
        </div>
      </div>

      <div class="px-6 pb-5">
        <div class="text-center text-[11px] text-slate-400" id="certModalLastRefreshed">Last refreshed: -</div>
      </div>
    </div>
  </div>
</template>
