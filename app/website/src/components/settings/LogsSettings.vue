<template>
  <div class="settings-pane" data-pane="logs">
    <div class="mt-4 flex flex-col gap-4">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div class="flex items-center gap-2">
          <h2 class="text-xl font-bold text-slate-900 dark:text-white">Log Monitoring</h2>
          <span class="rounded-full bg-primary/20 px-2 py-0.5 text-[10px] font-bold uppercase text-primary">
            {{ isPolling ? 'Live' : 'Paused' }}
          </span>
          <span class="text-xs text-slate-400">{{ filteredEntries.length }} entries</span>
        </div>
        <div class="flex flex-wrap items-center gap-2 rounded-lg border border-slate-200 bg-slate-50 p-2 dark:border-slate-700 dark:bg-slate-800/60">
          <select
            id="logLevelSelect"
            v-model="selectedLevel"
            class="rounded-md border-0 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-600 shadow-sm ring-1 ring-slate-200 focus:outline-none focus:ring-2 focus:ring-primary/40 dark:bg-slate-900 dark:text-slate-200 dark:ring-slate-600"
          >
            <option value="all">Level: All</option>
            <option value="trace">Level: Trace</option>
            <option value="debug">Level: Debug</option>
            <option value="info">Level: Info</option>
            <option value="warn">Level: Warning</option>
            <option value="error">Level: Error</option>
          </select>
          <label class="inline-flex items-center gap-2 rounded-md bg-white px-2.5 py-1.5 text-xs font-medium text-slate-600 shadow-sm ring-1 ring-slate-200 dark:bg-slate-900 dark:text-slate-200 dark:ring-slate-600">
            <input id="logAutoScroll" v-model="autoScroll" type="checkbox" class="rounded border-slate-300 text-primary focus:ring-primary/40" />
            Auto-scroll
          </label>
          <button
            id="logsClearBtn"
            type="button"
            class="rounded-md px-2.5 py-1.5 text-xs font-medium text-slate-500 transition hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-slate-700 dark:hover:text-slate-100"
            @click="clearLogs"
          >
            Clear
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
        class="overflow-y-auto rounded-xl border border-slate-200 bg-slate-950 p-4 font-mono text-xs text-slate-300 dark:border-slate-800"
        style="height: 800px;"
        ref="logsContainer"
      >
        <div v-if="errorMessage" class="mb-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-200">
          {{ errorMessage }}
        </div>
        <div v-if="!filteredEntries.length" class="rounded-lg border border-dashed border-slate-700 bg-slate-900/60 px-4 py-6 text-sm text-slate-500">
          No log data is available yet.
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
      <button id="logConfigSaveBtn" type="button"></button>
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

const entries = ref([]);
const selectedLevel = ref('all');
const autoScroll = ref(true);
const isLoading = ref(false);
const isPolling = ref(true);
const errorMessage = ref('');
const logsContainer = ref(null);

let pollTimer = null;

const filteredEntries = computed(() => {
  if (selectedLevel.value === 'all') {
    return entries.value;
  }
  const expectedLevel = selectedLevel.value.toUpperCase();
  return entries.value.filter((entry) => String(entry.level || '').toUpperCase() === expectedLevel);
});

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
    errorMessage.value = error instanceof Error ? error.message : 'Failed to load logs.';
  } finally {
    isLoading.value = false;
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
    errorMessage.value = error instanceof Error ? error.message : 'Failed to clear logs.';
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

watch(selectedLevel, () => {
  loadLogs();
});

onMounted(() => {
  loadLogs();
  startPolling();
});

onBeforeUnmount(() => {
  stopPolling();
});
</script>
