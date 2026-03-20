<template>
  <div class="settings-pane" data-pane="system">
    <div class="rounded-xl border border-slate-200 bg-white p-5 dark:border-slate-800 dark:bg-background-dark">
      <h3 class="mb-4 flex items-center gap-2 font-bold">
        <span class="material-symbols-outlined text-primary">settings</span>
        System Settings
      </h3>

      <div class="space-y-6">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-semibold">Run Mode</p>
            <p class="text-[10px] text-slate-500">Toggle between TUN and HTTP</p>
          </div>
          <div class="flex rounded bg-slate-100 p-1 dark:bg-slate-800">
            <button type="button" class="rounded bg-primary px-3 py-1 text-[10px] font-bold text-white shadow-sm">TUN</button>
            <button type="button" class="px-3 py-1 text-[10px] font-bold text-slate-500">HTTP</button>
          </div>
        </div>

        <div class="space-y-2">
          <p class="text-sm font-semibold">Software Status</p>
          <div class="rounded border-l-4 border-primary bg-slate-50 p-3 dark:bg-slate-800/50">
            <p class="mb-2 text-[10px] leading-relaxed text-slate-600 dark:text-slate-400">
              Register Opencode to macOS LaunchDaemons to ensure background auto-start and system-wide interception.
            </p>
            <button type="button" class="w-full rounded bg-slate-900 py-1.5 text-[11px] font-bold text-white hover:opacity-90 dark:bg-primary">
              Register to System (macOS)
            </button>
          </div>
        </div>

        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <p class="text-sm font-semibold">Certificate CA</p>
            <span class="flex items-center gap-1 text-[10px] font-bold text-primary">
              <span class="material-symbols-outlined text-xs">verified_user</span>
              Trusted
            </span>
          </div>
          <div class="space-y-1 rounded border border-slate-100 p-2 font-mono text-[10px] text-slate-500 dark:border-slate-800">
            <div><span class="text-slate-400">Subject:</span> Opencode Local CA</div>
            <div><span class="text-slate-400">Validity:</span> 2024-2029 (Valid)</div>
            <div class="truncate"><span class="text-slate-400">Finger:</span> 7A:9C:B5:E1:02...</div>
          </div>
          <div class="grid grid-cols-2 gap-2">
            <button type="button" class="flex items-center justify-center gap-1 rounded border border-slate-200 py-1.5 text-[10px] font-bold hover:bg-slate-50 dark:border-slate-800 dark:hover:bg-slate-800">
              <span class="material-symbols-outlined text-sm">download</span>
              Export
            </button>
            <button type="button" class="flex items-center justify-center gap-1 rounded border border-primary py-1.5 text-[10px] font-bold text-primary hover:bg-primary/5">
              <span class="material-symbols-outlined text-sm">install_desktop</span>
              Install
            </button>
          </div>
          <button type="button" class="flex w-full items-center justify-center gap-1 text-[10px] font-medium text-red-500">
            <span class="material-symbols-outlined text-sm">delete</span>
            Remove Root Certificate
          </button>
        </div>

        <div class="compat-anchors" aria-hidden="true">
          <input type="radio" name="runMode" id="runModeTun" value="tun" checked />
          <input type="radio" name="runMode" id="runModeHttp" value="http" />
          <button type="button" id="runStartBtn"></button>
          <button type="button" id="runStopBtn"></button>
          <button type="button" id="runModeBtn"></button>
          <div id="runCurrentMode"></div>
          <div id="runServiceStatus"></div>
          <div id="runAvailableModes"></div>
          <div id="runStatusInfo"></div>
          <button type="button" id="openCertManagementModalBtn"></button>
          <button type="button" id="btn-install-cert"></button>
          <button type="button" id="btn-export-cert"></button>
          <button type="button" id="btn-remove-cert"></button>
          <button type="button" id="btn-download-cert"></button>
          <button type="button" id="btn-check-cert"></button>
          <span id="cert-status"></span>
          <span id="cert-source"></span>
          <span id="cert-validity"></span>
          <span id="cert-fingerprint"></span>
          <span id="cert-subject"></span>
          <span id="cert-issuer"></span>
          <span id="cert-san"></span>
          <span id="cert-not-before"></span>
          <span id="cert-not-after"></span>
          <span id="cert-last-check"></span>
          <span id="cert-status-badge"></span>
          <div id="cert-raw-detail"></div>
          <span id="cert-trust"></span>
          <span id="cert-device"></span>
          <span id="cert-error"></span>
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
