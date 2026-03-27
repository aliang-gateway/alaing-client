import { ref } from 'vue';

const runMode = ref('unknown');
const runIsRunning = ref(false);
const runStatus = ref('');
const runDescription = ref('');
const runSyncError = ref('');
const startupStatus = ref('UNKNOWN');
const loading = ref(false);

const POLL_INTERVAL = 5000;

let pollTimer = null;
let refCount = 0;
let hasLoadedOnce = false;

function extractApiErrorMessage(payload, status, fallback) {
  return (
    payload?.msg ||
    payload?.message ||
    payload?.data?.error_msg ||
    payload?.data?.message ||
    payload?.data?.details?.error ||
    payload?.data?.details?.error_msg ||
    `${fallback}: HTTP ${status}`
  );
}

function normalizeMode(mode) {
  const value = String(mode || '').toLowerCase();
  if (value === 'http') {
    return 'http';
  }
  if (value === 'tun') {
    return 'tun';
  }
  return 'unknown';
}

export async function syncRunStatus() {
  try {
    const response = await fetch('/api/run/status');
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(extractApiErrorMessage(payload, response.status, 'Failed to sync run status'));
    }
    if (payload?.code !== 0) {
      throw new Error(payload?.msg || 'Failed to sync run status');
    }

    const data = payload?.data || {};
    runMode.value = normalizeMode(data?.current_mode);
    runIsRunning.value = Boolean(data?.is_running);
    runStatus.value = typeof data?.status === 'string' ? data.status : '';
    runDescription.value = typeof data?.description === 'string' ? data.description : '';
    runSyncError.value = '';
    return data;
  } catch (error) {
    runSyncError.value = error instanceof Error ? error.message : 'Unknown error';
    throw error;
  }
}

export async function syncStartupStatus() {
  try {
    const response = await fetch('/api/startup/status');
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(extractApiErrorMessage(payload, response.status, 'Failed to sync startup status'));
    }
    if (payload?.code !== 0) {
      throw new Error(payload?.msg || 'Failed to sync startup status');
    }

    const data = payload?.data || {};
    startupStatus.value = typeof data?.status === 'string' ? data.status : 'UNKNOWN';
    return data;
  } catch {
    startupStatus.value = 'UNKNOWN';
    return null;
  }
}

export async function refreshRunState() {
  const isFirstLoad = !hasLoadedOnce;
  if (isFirstLoad) {
    loading.value = true;
  }

  const [runResult] = await Promise.allSettled([
    syncRunStatus(),
    syncStartupStatus()
  ]);

  hasLoadedOnce = true;
  if (isFirstLoad) {
    loading.value = false;
  }

  if (runResult.status === 'rejected') {
    throw runResult.reason;
  }

  return {
    mode: runMode.value,
    isRunning: runIsRunning.value,
    status: runStatus.value,
    description: runDescription.value,
    startupStatus: startupStatus.value
  };
}

export function startPolling() {
  refCount += 1;
  if (refCount === 1) {
    refreshRunState().catch(() => {});
    pollTimer = window.setInterval(() => {
      refreshRunState().catch(() => {});
    }, POLL_INTERVAL);
  }
}

export function stopPolling() {
  refCount -= 1;
  if (refCount <= 0) {
    refCount = 0;
    if (pollTimer !== null) {
      window.clearInterval(pollTimer);
      pollTimer = null;
    }
  }
}

export function useRunStatus() {
  return {
    runMode,
    runIsRunning,
    runStatus,
    runDescription,
    runSyncError,
    startupStatus,
    loading,
    syncRunStatus,
    syncStartupStatus,
    refreshRunState,
    startPolling,
    stopPolling
  };
}
