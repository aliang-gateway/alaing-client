<template>
  <div class="settings-pane" data-pane="logs">
    <div class="mt-4 flex flex-col gap-4">
      <section class="rounded-xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-slate-900/70">
        <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h3 class="text-base font-semibold text-slate-900 dark:text-white">{{ t('logs_runtimeTitle') }}</h3>
            <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">{{ t('logs_runtimeDesc') }}</p>
          </div>
          <span
            v-if="isProdBuild"
            class="inline-flex items-center rounded-full bg-amber-100 px-3 py-1 text-[11px] font-semibold uppercase tracking-wide text-amber-700 dark:bg-amber-500/15 dark:text-amber-300"
          >
            {{ t('logs_prodGuard') }}
          </span>
        </div>

        <div class="mt-4 flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
          <label class="space-y-2">
            <span class="text-xs font-semibold uppercase tracking-wide text-slate-500">{{ t('logs_runtimeLevel') }}</span>
            <select
              v-model="configLevel"
              class="min-w-[180px] rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-700 shadow-sm transition focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
            >
              <option v-for="option in runtimeLevelOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          </label>

          <div class="flex flex-wrap items-center gap-2">
            <span class="text-xs text-slate-500 dark:text-slate-400">
              {{ t('logs_runtimeCurrent', { level: configLevel.toUpperCase() }) }}
            </span>
            <button
              id="logConfigSaveBtn"
              type="button"
              class="inline-flex min-h-10 items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="isSavingConfig || !configLevel"
              @click="saveLogConfig"
            >
              <span class="material-symbols-outlined text-sm">save</span>
              {{ isSavingConfig ? t('logs_runtimeSaving') : t('logs_runtimeSave') }}
            </button>
          </div>
        </div>

        <p v-if="configError" class="mt-3 text-sm text-red-500">{{ configError }}</p>
        <p v-else-if="configSuccess" class="mt-3 text-sm text-emerald-600 dark:text-emerald-400">{{ configSuccess }}</p>
      </section>

      <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div class="flex items-center gap-2">
          <h2 class="text-xl font-bold text-slate-900 dark:text-white">{{ t('logs_title') }}</h2>
          <span class="rounded-full bg-primary/20 px-2 py-0.5 text-[10px] font-bold uppercase text-primary">
            {{ isPolling ? t('logs_live') : t('logs_paused') }}
          </span>
          <span class="text-xs text-slate-400">{{ t('logs_entries', { count: filteredEntries.length }) }}</span>
        </div>
        <div class="flex flex-wrap items-center gap-2 rounded-lg border border-slate-200 bg-slate-50 p-2 dark:border-slate-700 dark:bg-slate-800/60">
          <select
            id="logLevelSelect"
            v-model="selectedLevel"
            class="rounded-md border-0 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-600 shadow-sm ring-1 ring-slate-200 focus:outline-none focus:ring-2 focus:ring-primary/40 dark:bg-slate-900 dark:text-slate-200 dark:ring-slate-600"
          >
            <option v-for="option in filterLevelOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
          <label class="inline-flex items-center gap-2 rounded-md bg-white px-2.5 py-1.5 text-xs font-medium text-slate-600 shadow-sm ring-1 ring-slate-200 dark:bg-slate-900 dark:text-slate-200 dark:ring-slate-600">
            <input id="logAutoScroll" v-model="autoScroll" type="checkbox" class="rounded border-slate-300 text-primary focus:ring-primary/40" />
            {{ t('logs_autoScroll') }}
          </label>
          <button
            id="logsClearBtn"
            type="button"
            class="rounded-md px-2.5 py-1.5 text-xs font-medium text-slate-500 transition hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-slate-700 dark:hover:text-slate-100"
            @click="clearLogs"
          >
            {{ t('logs_clear') }}
          </button>
          <button
            id="wsDisconnectBtn"
            type="button"
            class="flex items-center justify-center rounded-md p-1.5 text-slate-400 transition-all hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-500/10"
            @click="togglePolling"
          >
            <span class="material-symbols-outlined text-lg">{{ isPolling ? 'stop_circle' : 'play_circle' }}</span>
          </button>
          <button
            id="logsRefreshBtn"
            type="button"
            class="flex items-center justify-center rounded-md p-1.5 text-slate-400 transition-all hover:bg-primary/10 hover:text-primary"
            :disabled="isLoading"
            @click="loadLogs"
          >
            <span class="material-symbols-outlined text-lg">{{ isLoading ? 'hourglass_top' : 'refresh' }}</span>
          </button>
        </div>
      </div>

      <div
        ref="logsContainer"
        class="overflow-y-auto rounded-xl border border-slate-200 bg-slate-950 p-4 font-mono text-xs text-slate-300 dark:border-slate-800"
        style="height: 800px;"
      >
        <div v-if="errorMessage" class="mb-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-200">
          {{ errorMessage }}
        </div>
        <div v-if="!filteredEntries.length" class="rounded-lg border border-dashed border-slate-700 bg-slate-900/60 px-4 py-6 text-sm text-slate-500">
          {{ t('logs_noData') }}
        </div>
        <div v-else id="logsOutput" class="space-y-1 whitespace-pre-wrap break-all">
          <div
            v-for="(entry, index) in filteredEntries"
            :key="`${entry.timestamp}-${entry.source}-${index}`"
            class="rounded px-2 py-1.5"
            :class="entryRowClass(entry.level)"
          >
            <span class="text-slate-500">[{{ entry.timestamp }}]</span>
            <span class="ml-2 font-semibold" :class="entryLevelClass(entry.level)">{{ entry.level }}</span>
            <span class="ml-2 text-slate-400">({{ entry.source }})</span>
            <span class="ml-2">{{ entry.message }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="compat-anchors" aria-hidden="true">
      <select id="logSourceSelect"><option value="all">All</option></select>
      <button id="logFilterBtn" type="button"></button>
      <button id="wsConnectBtn" type="button"></button>
      <button id="logsFullscreenBtn" type="button"><i class="bi bi-fullscreen"></i></button>
      <div id="logs-page"></div>
      <div id="wsConnectionStatus">{{ isPolling ? 'connected' : 'paused' }}</div>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useI18n } from '../../i18n';

const { t } = useI18n();

const isProdBuild = import.meta.env.PROD || ['prod', 'production'].includes(String(import.meta.env.MODE || '').toLowerCase());

const LOG_LEVEL_OPTIONS = [
  { value: 'trace', labelKey: 'logs_levelTrace' },
  { value: 'debug', labelKey: 'logs_levelDebug' },
  { value: 'info', labelKey: 'logs_levelInfo' },
  { value: 'warn', labelKey: 'logs_levelWarning' },
  { value: 'error', labelKey: 'logs_levelError' }
];

const entries = ref([]);
const selectedLevel = ref('all');
const configLevel = ref(isProdBuild ? 'info' : 'debug');
const autoScroll = ref(true);
const isLoading = ref(false);
const isPolling = ref(true);
const isSavingConfig = ref(false);
const errorMessage = ref('');
const configError = ref('');
const configSuccess = ref('');
const logsContainer = ref(null);

let pollTimer = null;

const minimumRuntimeLevel = computed(() => (isProdBuild ? 'info' : 'trace'));

const runtimeLevelOptions = computed(() => {
  const minimum = levelRank(minimumRuntimeLevel.value);
  return LOG_LEVEL_OPTIONS
    .filter((option) => levelRank(option.value) >= minimum)
    .map((option) => ({
      value: option.value,
      label: t(option.labelKey)
    }));
});

const filterLevelOptions = computed(() => [
  { value: 'all', label: t('logs_levelAll') },
  ...runtimeLevelOptions.value
]);

const filteredEntries = computed(() => {
  if (selectedLevel.value === 'all') {
    return entries.value;
  }
  const expectedLevel = selectedLevel.value.toUpperCase();
  return entries.value.filter((entry) => String(entry.level || '').toUpperCase() === expectedLevel);
});

function levelRank(level) {
  switch (String(level || '').toLowerCase()) {
    case 'trace':
      return 0;
    case 'debug':
      return 1;
    case 'info':
      return 2;
    case 'warn':
      return 3;
    case 'error':
      return 4;
    default:
      return 2;
  }
}

function clampRuntimeLevel(level) {
  const normalized = String(level || '').toLowerCase();
  const allowed = runtimeLevelOptions.value.map((option) => option.value);
  if (allowed.includes(normalized)) {
    return normalized;
  }
  return minimumRuntimeLevel.value;
}

async function loadLogs() {
  isLoading.value = true;
  errorMessage.value = '';

  try {
    const levelQuery = selectedLevel.value !== 'all' ? `&level=${encodeURIComponent(selectedLevel.value.toUpperCase())}` : '';
    const response = await fetch(`/api/logs?limit=200${levelQuery}`);
    const payload = await response.json().catch(() => ({}));
    if (!response.ok || payload?.code !== 0) {
      throw new Error(payload?.msg || payload?.message || `Request failed (${response.status})`);
    }

    const nextEntries = Array.isArray(payload?.data?.entries) ? payload.data.entries : [];
    entries.value = nextEntries.map(normalizeLogEntry);
    await scrollToBottomIfNeeded();
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('logs_loadFailed');
  } finally {
    isLoading.value = false;
  }
}

async function loadLogConfig() {
  configError.value = '';

  try {
    const response = await fetch('/api/logs/config');
    const payload = await response.json().catch(() => ({}));
    if (!response.ok || payload?.code !== 0) {
      throw new Error(payload?.msg || payload?.message || `Request failed (${response.status})`);
    }

    configLevel.value = clampRuntimeLevel(payload?.data?.level || configLevel.value);
  } catch (error) {
    configError.value = error instanceof Error ? error.message : t('logs_runtimeLoadFailed');
  }
}

async function saveLogConfig() {
  isSavingConfig.value = true;
  configError.value = '';
  configSuccess.value = '';

  try {
    const response = await fetch('/api/logs/level', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ level: clampRuntimeLevel(configLevel.value).toUpperCase() })
    });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok || payload?.code !== 0) {
      throw new Error(payload?.msg || payload?.message || `Request failed (${response.status})`);
    }

    configLevel.value = clampRuntimeLevel(payload?.data?.level || configLevel.value);
    configSuccess.value = t('logs_runtimeSaved');
    await loadLogs();
  } catch (error) {
    configError.value = error instanceof Error ? error.message : t('logs_runtimeSaveFailed');
  } finally {
    isSavingConfig.value = false;
  }
}

async function clearLogs() {
  errorMessage.value = '';
  try {
    const response = await fetch('/api/logs/clear', { method: 'POST' });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok || payload?.code !== 0) {
      throw new Error(payload?.msg || payload?.message || `Request failed (${response.status})`);
    }
    entries.value = [];
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('logs_clearFailed');
  }
}

function normalizeLogEntry(entry) {
  const raw = entry && typeof entry === 'object' ? entry : {};
  return {
    level: typeof raw.level === 'string' ? raw.level.toUpperCase() : 'INFO',
    timestamp: typeof raw.timestamp === 'string' ? raw.timestamp : '',
    message: typeof raw.message === 'string' ? raw.message : '',
    source: typeof raw.source === 'string' ? raw.source : 'main'
  };
}

function entryLevelClass(level) {
  switch (String(level || '').toUpperCase()) {
    case 'ERROR':
    case 'FATAL':
    case 'PANIC':
      return 'text-red-300';
    case 'WARN':
      return 'text-amber-300';
    case 'DEBUG':
      return 'text-sky-300';
    case 'TRACE':
      return 'text-violet-300';
    default:
      return 'text-emerald-300';
  }
}

function entryRowClass(level) {
  switch (String(level || '').toUpperCase()) {
    case 'ERROR':
    case 'FATAL':
    case 'PANIC':
      return 'bg-red-500/5';
    case 'WARN':
      return 'bg-amber-500/5';
    case 'DEBUG':
      return 'bg-sky-500/5';
    case 'TRACE':
      return 'bg-violet-500/5';
    default:
      return 'bg-emerald-500/5';
  }
}

async function scrollToBottomIfNeeded() {
  if (!autoScroll.value) {
    return;
  }
  await nextTick();
  const element = logsContainer.value;
  if (element instanceof HTMLElement) {
    element.scrollTop = element.scrollHeight;
  }
}

function startPolling() {
  stopPolling();
  pollTimer = window.setInterval(() => {
    loadLogs();
  }, 5000);
}

function stopPolling() {
  if (pollTimer) {
    window.clearInterval(pollTimer);
    pollTimer = null;
  }
}

function togglePolling() {
  isPolling.value = !isPolling.value;
  if (isPolling.value) {
    startPolling();
    loadLogs();
    return;
  }
  stopPolling();
}

watch(selectedLevel, (nextLevel) => {
  if (nextLevel !== 'all' && levelRank(nextLevel) < levelRank(minimumRuntimeLevel.value)) {
    selectedLevel.value = 'all';
    return;
  }
  loadLogs();
});

watch(runtimeLevelOptions, () => {
  configLevel.value = clampRuntimeLevel(configLevel.value);
  if (selectedLevel.value !== 'all' && levelRank(selectedLevel.value) < levelRank(minimumRuntimeLevel.value)) {
    selectedLevel.value = 'all';
  }
});

onMounted(async () => {
  await Promise.all([
    loadLogConfig(),
    loadLogs()
  ]);
  startPolling();
});

onBeforeUnmount(() => {
  stopPolling();
});
</script>
