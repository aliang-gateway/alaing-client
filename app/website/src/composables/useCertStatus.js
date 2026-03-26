import { ref } from 'vue';

const API_BASE = '/api';
const CERT_TYPE = 'mitm-ca';

const certStatus = ref(null);
const loading = ref(false);
const error = ref(null);

let pollTimer = null;
let refCount = 0;

const POLL_INTERVAL = 3000;

function statusKey(data) {
  return JSON.stringify([
    data.is_exported,
    data.is_installed,
    data.is_trusted,
    data.trust_status
  ]);
}

let lastKey = '';

async function fetchStatus() {
  const isFirstLoad = !certStatus.value;
  if (isFirstLoad) loading.value = true;

  try {
    const res = await fetch(`${API_BASE}/cert/status?cert_type=${encodeURIComponent(CERT_TYPE)}`);
    if (!res.ok) {
      const data = await res.json().catch(() => null);
      throw new Error(data?.data?.error_msg || data?.msg || data?.message || `HTTP ${res.status}`);
    }
    const data = (await res.json()).data || {};
    const key = statusKey(data);
    if (key !== lastKey) {
      lastKey = key;
      certStatus.value = data;
    }
    if (error.value) error.value = null;
  } catch (err) {
    if (!certStatus.value) {
      const fallback = { is_exported: false, is_installed: false, is_trusted: false, trust_status: 'not_found' };
      lastKey = statusKey(fallback);
      certStatus.value = fallback;
    }
    if (!error.value) error.value = err.message;
  } finally {
    if (isFirstLoad) loading.value = false;
  }
}

function startPolling() {
  refCount++;
  if (refCount === 1) {
    fetchStatus();
    pollTimer = setInterval(fetchStatus, POLL_INTERVAL);
  }
}

function stopPolling() {
  refCount--;
  if (refCount <= 0) {
    refCount = 0;
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  }
}

function invalidateCache() {
  lastKey = '';
}

export function useCertStatus() {
  return {
    certStatus,
    loading,
    error,
    fetchStatus,
    startPolling,
    stopPolling,
    invalidateCache
  };
}
