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
            <button
              type="button"
              :disabled="loadingMode || switchingMode || wintunDependency.installing"
              :class="[
                'rounded px-3 py-1 text-[10px] font-bold transition',
                selectedMode === 'tun'
                  ? 'bg-primary text-white shadow-sm'
                  : 'text-slate-500 hover:bg-slate-200 dark:text-slate-300 dark:hover:bg-slate-700/70'
              ]"
              @click="selectMode('tun')"
            >
              TUN
            </button>
            <button
              type="button"
              :disabled="loadingMode || switchingMode || wintunDependency.installing"
              :class="[
                'rounded px-3 py-1 text-[10px] font-bold transition',
                selectedMode === 'http'
                  ? 'bg-primary text-white shadow-sm'
                  : 'text-slate-500 hover:bg-slate-200 dark:text-slate-300 dark:hover:bg-slate-700/70'
              ]"
              @click="selectMode('http')"
            >
              HTTP
            </button>
          </div>
        </div>

        <div class="space-y-3 rounded-lg border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-900/50">
          <div class="flex flex-wrap items-center gap-2 text-[11px] text-slate-600 dark:text-slate-300">
            <span class="rounded bg-slate-200 px-2 py-0.5 font-semibold dark:bg-slate-700">Backend: {{ backendMode.toUpperCase() }}</span>
            <span class="rounded bg-slate-200 px-2 py-0.5 font-semibold dark:bg-slate-700">Selected: {{ selectedMode.toUpperCase() }}</span>
            <span v-if="isRunning !== null" class="rounded bg-slate-200 px-2 py-0.5 font-semibold dark:bg-slate-700">
              {{ isRunning ? 'Running' : 'Stopped' }}
            </span>
          </div>
          <p v-if="modeStatus" class="text-[11px] text-slate-500 dark:text-slate-400">{{ modeStatus }}</p>
          <div
            v-if="showMissingWintunBanner"
            class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-[11px] text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200"
          >
            {{ wintunDependency.message || 'Wintun dependency is missing. We will install it automatically before switching to TUN mode.' }}
          </div>
          <div
            v-if="showInstallingWintunBanner"
            class="rounded-lg border border-sky-200 bg-sky-50 px-3 py-2 text-[11px] text-sky-700 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200"
          >
            {{ wintunDependency.message || 'Installing Wintun dependency in the background...' }}
          </div>
          <p v-if="modeError" class="text-[11px] text-red-500">{{ modeError }}</p>
          <p v-if="modeSuccess" class="text-[11px] text-emerald-600 dark:text-emerald-400">{{ modeSuccess }}</p>

          <div class="grid grid-cols-1 gap-2">
            <button
              type="button"
              class="rounded bg-slate-900 px-3 py-2 text-[11px] font-bold text-white transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-primary"
              :disabled="loadingMode || switchingMode || wintunDependency.installing"
              @click="refreshModeState"
            >
              {{ loadingMode ? 'Refreshing...' : 'Refresh State' }}
            </button>
          </div>
        </div>

        <div
          v-if="showTunSwitchConfirm"
          class="fixed inset-0 z-[1000] flex items-center justify-center bg-slate-950/60 p-4 backdrop-blur-sm"
          @click.self="cancelTunSwitchConfirm"
        >
          <div class="w-full max-w-md overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl dark:border-slate-700 dark:bg-slate-900">
            <div class="border-b border-slate-200 bg-slate-50/80 px-5 py-4 dark:border-slate-700 dark:bg-slate-800/60">
              <div class="flex items-start justify-between gap-4">
                <div>
                  <p class="text-xs font-bold uppercase tracking-[0.2em] text-amber-500">Switch To TUN</p>
                  <h3 class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">Continue switching from HTTP to TUN?</h3>
                  <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">
                    We will return to the dashboard and show the live TUN startup progress dialog while the backend applies the switch.
                  </p>
                </div>
                <button
                  type="button"
                  class="rounded-lg p-1.5 text-slate-500 transition hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-slate-800 dark:hover:text-slate-200"
                  @click="cancelTunSwitchConfirm"
                >
                  <span class="material-symbols-outlined text-lg">close</span>
                </button>
              </div>
            </div>

            <div class="space-y-4 p-5">
              <div class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
                If the system blocks TUN permissions, the startup dialog will keep the error logs visible so the user can decide what to do next.
              </div>

              <div class="flex gap-3">
                <button
                  type="button"
                  class="inline-flex h-11 flex-1 items-center justify-center rounded-lg border border-slate-200 px-4 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 dark:border-slate-700 dark:text-slate-100 dark:hover:bg-slate-800"
                  @click="cancelTunSwitchConfirm"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  class="inline-flex h-11 flex-1 items-center justify-center rounded-lg bg-primary px-4 text-sm font-semibold text-white transition hover:bg-primary/90"
                  @click="confirmTunSwitch"
                >
                  Continue
                </button>
              </div>
            </div>
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
import { nextTick } from 'vue';
import { useNavigation } from '../../composables/useNavigation';
import { useRunStatus } from '../../composables/useRunStatus';

export default {
  name: 'SystemSettings',
  setup() {
    const { showDashboard } = useNavigation();
    const {
      runMode,
      runIsRunning,
      runStatus,
      runWintunDependency,
      runSyncError,
      refreshRunState,
      startPolling,
      stopPolling
    } = useRunStatus();

    return {
      showDashboard,
      sharedRunMode: runMode,
      sharedRunIsRunning: runIsRunning,
      sharedRunStatus: runStatus,
      sharedRunWintunDependency: runWintunDependency,
      sharedRunSyncError: runSyncError,
      refreshSharedRunState: refreshRunState,
      startRunStatePolling: startPolling,
      stopRunStatePolling: stopPolling
    };
  },
  data() {
    return {
      selectedMode: 'tun',
      modeError: '',
      modeSuccess: '',
      loadingMode: false,
      switchingMode: false,
      showTunSwitchConfirm: false,
      pendingTunSwitchAfterInstall: false,
      wintunInstallPending: false,
      wintunInstallMessage: '',
      wintunInstallState: 'idle',
      wintunInstallLockUntilMs: 0,
      wintunInstallPollTimer: null,
      wintunInstallPollInFlight: false
    };
  },
  computed: {
    backendMode() {
      if (this.sharedRunMode === 'http') {
        return 'http';
      }
      if (this.sharedRunMode === 'tun') {
        return 'tun';
      }
      return this.selectedMode;
    },
    isRunning() {
      if (this.sharedRunMode === 'unknown' && !this.sharedRunStatus && !this.sharedRunSyncError) {
        return null;
      }
      return this.sharedRunIsRunning;
    },
    modeStatus() {
      if (this.sharedRunSyncError) {
        return `Sync failed: ${this.sharedRunSyncError}`;
      }
      return typeof this.sharedRunStatus === 'string' ? this.sharedRunStatus : '';
    },
    wintunDependency() {
      const dependency = this.sharedRunWintunDependency;
      if (this.wintunInstallPending) {
        return {
          ...(dependency && typeof dependency === 'object' ? dependency : {}),
          supported: true,
          required: true,
          available: false,
          installing: true,
          state: this.wintunInstallState || 'queued',
          message: this.wintunInstallMessage || 'Installing Wintun dependency in the background...',
          error: ''
        };
      }
      if (dependency && typeof dependency === 'object') {
        return dependency;
      }
      return {
        supported: false,
        required: false,
        available: true,
        installing: false,
        state: 'not_applicable',
        message: '',
        error: ''
      };
    },
    showMissingWintunBanner() {
      return this.wintunDependency.required && !this.wintunDependency.available && !this.wintunDependency.installing;
    },
    showInstallingWintunBanner() {
      return this.wintunDependency.installing;
    }
  },
  mounted() {
    this.startRunStatePolling();
    this.refreshModeState().finally(() => {
      if (this.wintunDependency.installing) {
        this.wintunInstallPending = true;
        this.wintunInstallMessage = this.wintunDependency.message || 'Installing Wintun dependency in the background...';
        this.wintunInstallState = this.wintunDependency.state || 'installing';
        this.ensureWintunInstallProgressModal();
        this.startWintunInstallPolling();
      }
    });
  },
  beforeUnmount() {
    this.stopRunStatePolling();
    this.stopWintunInstallPolling();
  },
  watch: {
    backendMode(nextMode) {
      if (!this.switchingMode && !this.pendingTunSwitchAfterInstall && nextMode) {
        this.selectedMode = nextMode;
      }
    }
  },
  methods: {
    clearMessages() {
      this.modeError = '';
      this.modeSuccess = '';
    },
    async dispatchTunProgressEvent(name, detail = {}) {
      this.showDashboard();
      await nextTick();
      window.dispatchEvent(new CustomEvent(name, { detail }));
    },
    async ensureWintunInstallProgressModal() {
      if (!this.wintunInstallLockUntilMs || this.wintunInstallLockUntilMs < Date.now()) {
        this.wintunInstallLockUntilMs = Date.now() + 30000;
      }
      await this.dispatchTunProgressEvent('aliang:tun-progress-open', {
        phase: 'installing_dependency',
        installState: this.wintunDependency.state || this.wintunInstallState || 'queued',
        lockSeconds: Math.max(1, Math.ceil((this.wintunInstallLockUntilMs - Date.now()) / 1000)),
        title: 'Installing Wintun Dependency',
        detail: 'Windows needs the Wintun driver before TUN mode can start.',
        statusLabel: 'Installing dependency...',
        statusHint: this.wintunDependency.message || this.wintunInstallMessage || 'Preparing the Wintun package and waiting for installation progress.'
      });
    },
    async updateWintunInstallProgressModal() {
      await this.dispatchTunProgressEvent('aliang:tun-progress-update', {
        phase: 'installing_dependency',
        installState: this.wintunDependency.state || this.wintunInstallState || 'installing',
        lockUntilMs: this.wintunInstallLockUntilMs,
        title: 'Installing Wintun Dependency',
        detail: 'Windows needs the Wintun driver before TUN mode can start.',
        statusLabel: 'Installing dependency...',
        statusHint: this.wintunDependency.message || this.wintunInstallMessage || 'Preparing the Wintun package and waiting for installation progress.'
      });
    },
    async selectMode(mode) {
      const normalizedMode = this.normalizeMode(mode);
      if (this.loadingMode || this.switchingMode || this.wintunDependency.installing || this.selectedMode === normalizedMode) {
        return;
      }
      if (this.backendMode === 'http' && normalizedMode === 'tun') {
        this.showTunSwitchConfirm = true;
        return;
      }
      if (normalizedMode === 'tun' && this.wintunDependency.required && !this.wintunDependency.available) {
        this.selectedMode = normalizedMode;
        await this.installWintunDependency({ continueAfterInstall: true });
        return;
      }
      this.selectedMode = normalizedMode;
      await this.switchMode();
    },
    cancelTunSwitchConfirm() {
      this.showTunSwitchConfirm = false;
    },
    async confirmTunSwitch() {
      this.showTunSwitchConfirm = false;
      this.selectedMode = 'tun';
      if (this.wintunDependency.required && !this.wintunDependency.available) {
        await this.installWintunDependency({ continueAfterInstall: true });
        return;
      }
      await this.continueTunSwitchAfterDependencyReady();
    },
    async continueTunSwitchAfterDependencyReady() {
      this.showDashboard();
      await nextTick();
      window.dispatchEvent(new CustomEvent('aliang:tun-progress-open', {
        detail: {
          phase: 'switching_mode',
          title: 'Switching To TUN',
          detail: 'Applying the new run mode and following live TUN startup logs from the dashboard.',
          statusLabel: 'Switching from HTTP to TUN...',
          statusHint: 'The backend is applying the new mode. If TUN startup fails, the error logs will stay visible here.'
        }
      }));
      await this.switchMode({ reportTunProgress: true });
    },
    async requestJSON(url, options = {}) {
      const res = await fetch(url, options);
      const payload = await res.json().catch(() => ({}));
      if (!res.ok || payload?.code !== 0) {
        const msg = payload?.msg || payload?.message || `Request failed (${res.status})`;
        throw new Error(msg);
      }
      return payload?.data ?? payload;
    },
    normalizeMode(mode) {
      return mode === 'http' ? 'http' : 'tun';
    },
    async refreshModeState(options = {}) {
      this.loadingMode = true;
      if (!options.preserveMessages) {
        this.clearMessages();
      }
      try {
        await this.refreshSharedRunState();
        this.selectedMode = this.pendingTunSwitchAfterInstall ? 'tun' : this.backendMode;
      } catch (err) {
        this.modeError = err instanceof Error ? err.message : 'Failed to load run mode status.';
      } finally {
        this.loadingMode = false;
      }
    },
    startWintunInstallPolling() {
      if (this.wintunInstallPollTimer !== null) {
        return;
      }
      this.wintunInstallPollTimer = window.setInterval(() => {
        this.pollWintunInstallState();
      }, 2000);
    },
    stopWintunInstallPolling() {
      if (this.wintunInstallPollTimer !== null) {
        window.clearInterval(this.wintunInstallPollTimer);
        this.wintunInstallPollTimer = null;
      }
    },
    async pollWintunInstallState() {
      if (this.wintunInstallPollInFlight) {
        return;
      }
      this.wintunInstallPollInFlight = true;
      try {
        await this.refreshSharedRunState();
        if (this.wintunDependency.installing) {
          this.wintunInstallPending = true;
          this.wintunInstallState = this.wintunDependency.state || 'installing';
          this.wintunInstallMessage = this.wintunDependency.message || 'Installing Wintun dependency in the background...';
          await this.updateWintunInstallProgressModal();
          return;
        }

        this.stopWintunInstallPolling();
        this.wintunInstallPending = false;
        this.wintunInstallMessage = '';
        this.wintunInstallState = this.wintunDependency.state || 'idle';
        this.wintunInstallLockUntilMs = 0;
        if (this.wintunDependency.available) {
          const shouldContinue = this.pendingTunSwitchAfterInstall;
          this.pendingTunSwitchAfterInstall = false;
          this.modeSuccess = this.wintunDependency.message || 'Wintun dependency is ready.';
          if (shouldContinue) {
            await this.continueTunSwitchAfterDependencyReady();
          } else {
            await this.dispatchTunProgressEvent('aliang:tun-progress-success', {
              title: 'Wintun Ready',
              detail: 'The Windows dependency is installed and TUN mode can now be enabled.',
              statusLabel: 'Dependency ready',
              statusHint: this.modeSuccess,
              message: this.modeSuccess
            });
          }
          return;
        }

        this.pendingTunSwitchAfterInstall = false;
        this.modeError = this.wintunDependency.error || this.wintunDependency.message || 'Failed to install Wintun dependency.';
        await this.dispatchTunProgressEvent('aliang:tun-progress-error', {
          title: 'Wintun Installation Failed',
          detail: 'The Windows dependency could not be installed automatically.',
          statusLabel: 'Dependency install failed',
          statusHint: 'Check the error message below and retry after fixing permissions or network issues.',
          message: this.modeError
        });
      } catch (err) {
        this.modeError = err instanceof Error ? err.message : 'Failed to refresh Wintun installation status.';
        this.wintunInstallPending = false;
        this.wintunInstallMessage = '';
        this.wintunInstallState = 'failed';
        this.wintunInstallLockUntilMs = 0;
        await this.dispatchTunProgressEvent('aliang:tun-progress-error', {
          title: 'Wintun Installation Failed',
          detail: 'The Windows dependency could not be installed automatically.',
          statusLabel: 'Dependency install failed',
          statusHint: 'Check the error message below and retry after fixing permissions or network issues.',
          message: this.modeError
        });
      } finally {
        this.wintunInstallPollInFlight = false;
      }
    },
    async installWintunDependency(options = {}) {
      this.clearMessages();
      this.pendingTunSwitchAfterInstall = Boolean(options.continueAfterInstall);
      this.wintunInstallPending = true;
      this.wintunInstallState = 'queued';
      this.wintunInstallMessage = 'Preparing Wintun dependency installation in the background.';
      this.wintunInstallLockUntilMs = Date.now() + 30000;

      try {
        await this.ensureWintunInstallProgressModal();
        const result = await this.requestJSON('/api/run/wintun/install', {
          method: 'POST'
        });

        if (result?.available) {
          this.wintunInstallPending = false;
          this.wintunInstallMessage = '';
          this.wintunInstallState = 'installed';
          this.wintunInstallLockUntilMs = 0;
          this.modeSuccess = typeof result?.message === 'string' && result.message
            ? result.message
            : 'Wintun dependency is already installed.';
          if (this.pendingTunSwitchAfterInstall) {
            this.pendingTunSwitchAfterInstall = false;
            await this.continueTunSwitchAfterDependencyReady();
          } else {
            await this.dispatchTunProgressEvent('aliang:tun-progress-success', {
              title: 'Wintun Ready',
              detail: 'The Windows dependency is installed and TUN mode can now be enabled.',
              statusLabel: 'Dependency ready',
              statusHint: this.modeSuccess,
              message: this.modeSuccess
            });
          }
          return;
        }

        this.wintunInstallState = typeof result?.state === 'string' && result.state ? result.state : 'queued';
        this.wintunInstallMessage = typeof result?.message === 'string' && result.message
          ? result.message
          : 'Installing Wintun dependency in the background...';
        await this.updateWintunInstallProgressModal();
        this.startWintunInstallPolling();
        await this.pollWintunInstallState();
      } catch (err) {
        this.pendingTunSwitchAfterInstall = false;
        this.wintunInstallPending = false;
        this.wintunInstallMessage = '';
        this.wintunInstallState = 'failed';
        this.wintunInstallLockUntilMs = 0;
        this.modeError = err instanceof Error ? err.message : 'Failed to start Wintun installation.';
        await this.dispatchTunProgressEvent('aliang:tun-progress-error', {
          title: 'Wintun Installation Failed',
          detail: 'The Windows dependency could not be installed automatically.',
          statusLabel: 'Dependency install failed',
          statusHint: 'Check the error message below and retry after fixing permissions or network issues.',
          message: this.modeError
        });
      }
    },
    async switchMode(options = {}) {
      this.switchingMode = true;
      this.clearMessages();
      try {
        const result = await this.requestJSON('/api/run/swift', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ mode: this.selectedMode })
        });
        const status = typeof result?.status === 'string' ? result.status : '';
        if (status === 'failed') {
          throw new Error(result?.msg || 'Mode switch failed');
        }
        this.modeSuccess = typeof result?.message === 'string' && result.message
          ? result.message
          : `Applied ${this.selectedMode.toUpperCase()} mode successfully.`;
        await this.refreshModeState({ preserveMessages: true });
        if (options.reportTunProgress) {
          window.dispatchEvent(new CustomEvent('aliang:tun-progress-success', {
            detail: {
              message: this.modeSuccess || 'Switched to TUN mode successfully.'
            }
          }));
        }
      } catch (err) {
        this.modeError = err instanceof Error ? err.message : 'Failed to switch mode.';
        if (options.reportTunProgress) {
          window.dispatchEvent(new CustomEvent('aliang:tun-progress-error', {
            detail: {
              message: this.modeError
            }
          }));
        }
      } finally {
        this.switchingMode = false;
      }
    }
  }
}
</script>
