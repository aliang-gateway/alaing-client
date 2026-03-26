<template>
  <div
    v-if="modelValue"
    class="fixed inset-0 z-[120] flex items-center justify-center p-4"
  >
    <div class="absolute inset-0 bg-slate-900/45 backdrop-blur-sm" @click="close"></div>
    <div
      class="relative z-10 w-full max-w-2xl bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 shadow-2xl overflow-hidden"
    >
      <div class="px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
        <div>
          <h3 class="text-lg font-bold text-slate-900 dark:text-slate-100">Certificate Management</h3>
          <p class="text-xs text-slate-500 dark:text-slate-400 mt-1">管理代理证书的安装、下载与状态</p>
        </div>
        <button
          type="button"
          aria-label="Close certificate modal"
          class="size-9 rounded-full flex items-center justify-center text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800"
          @click="close"
        >
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>

      <div class="p-6 space-y-5">
        <div class="flex items-center gap-2 flex-wrap">
          <span class="px-2.5 py-1.5 bg-slate-100 dark:bg-slate-800 rounded-md text-xs font-medium text-slate-700 dark:text-slate-300">
            本地证书
          </span>
          <span class="text-[11px] text-slate-400">自动刷新中</span>
          <span
            class="inline-block size-3.5 border-2 border-slate-200 border-t-primary rounded-full animate-spin"
          ></span>
        </div>

        <!-- Status Display -->
        <div
          class="p-4 rounded-lg bg-slate-50 dark:bg-slate-800/40 border border-slate-200 dark:border-slate-700"
        >
          <div v-if="statusError" class="text-sm text-red-500">{{ statusError }}</div>
          <div v-else-if="certStatus" class="space-y-2">
            <div class="flex flex-wrap gap-2">
              <span
                class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium"
                :class="certStatus.is_exported
                  ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300'
                  : 'bg-slate-100 text-slate-500 dark:bg-slate-500/20 dark:text-slate-400'"
              >
                <span class="material-symbols-outlined text-[12px]">{{ certStatus.is_exported ? 'check_circle' : 'cancel' }}</span>
                {{ certStatus.is_exported ? '已导出' : '未导出' }}
              </span>
              <span
                class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium"
                :class="certStatus.is_installed
                  ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300'
                  : 'bg-slate-100 text-slate-500 dark:bg-slate-500/20 dark:text-slate-400'"
              >
                <span class="material-symbols-outlined text-[12px]">{{ certStatus.is_installed ? 'check_circle' : 'cancel' }}</span>
                {{ certStatus.is_installed ? '已安装' : '未安装' }}
              </span>
              <span
                class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium"
                :class="certStatus.is_trusted
                  ? 'bg-blue-100 text-blue-700 dark:bg-blue-500/20 dark:text-blue-300'
                  : 'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-300'"
              >
                <span class="material-symbols-outlined text-[12px]">{{ certStatus.is_trusted ? 'shield' : 'warning' }}</span>
                {{ certStatus.is_trusted ? '已信任' : '未信任' }}
              </span>
            </div>
            <div class="text-xs text-slate-500 dark:text-slate-400 space-y-0.5 mt-2">
              <div><strong>主体:</strong> {{ certStatus.subject || '-' }}</div>
              <div><strong>颁发者:</strong> {{ certStatus.issuer || '-' }}</div>
              <div><strong>有效期:</strong> {{ certStatus.not_before || '-' }} ~ {{ certStatus.not_after || '-' }}</div>
              <div><strong>指纹:</strong> <code class="break-all">{{ certStatus.fingerprint || '-' }}</code></div>
              <div v-if="certStatus.install_path"><strong>安装路径:</strong> <code class="break-all">{{ certStatus.install_path }}</code></div>
            </div>
          </div>
          <div v-else class="text-sm text-slate-400">正在加载证书信息...</div>
        </div>

        <!-- Action Buttons -->
        <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-2">
          <button
            type="button"
            :disabled="busy"
            class="min-h-9 flex items-center justify-center gap-1.5 px-2.5 py-1.5 border border-slate-200 dark:border-slate-700 rounded-md text-xs hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            @click="installCert"
          >
            <span class="material-symbols-outlined text-[16px] leading-none text-slate-500">check_circle</span>
            <span>安装到系统</span>
          </button>
          <button
            type="button"
            :disabled="downloading"
            class="min-h-9 flex items-center justify-center gap-1.5 px-2.5 py-1.5 border border-slate-200 dark:border-slate-700 rounded-md text-xs hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            @click="downloadCertFile"
          >
            <span
              v-if="downloading"
              class="inline-block size-3.5 border-2 border-slate-300 border-t-primary rounded-full animate-spin"
            ></span>
            <span class="material-symbols-outlined text-[16px] leading-none text-slate-500">download</span>
            <span>下载 PEM</span>
          </button>
          <button
            type="button"
            :disabled="busy"
            class="min-h-9 flex items-center justify-center gap-1.5 px-2.5 py-1.5 border border-red-200 dark:border-red-500/40 rounded-md text-xs text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            @click="removeCert"
          >
            <span class="material-symbols-outlined text-[16px] leading-none text-red-500">delete</span>
            <span>移除证书</span>
          </button>
          <button
            type="button"
            :disabled="busy"
            class="min-h-9 flex items-center justify-center gap-1.5 px-2.5 py-1.5 border border-amber-200 dark:border-amber-500/40 rounded-md text-xs text-amber-700 dark:text-amber-300 hover:bg-amber-50 dark:hover:bg-amber-900/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            @click="generateCert"
          >
            <span class="material-symbols-outlined text-[16px] leading-none text-amber-500">autorenew</span>
            <span>重新生成证书</span>
          </button>
        </div>

        <!-- Generate Result -->
        <div
          v-if="generateResult"
          class="p-4 rounded-lg bg-amber-50/70 dark:bg-amber-900/10 border border-amber-200 dark:border-amber-700/40"
        >
          <div class="text-sm font-semibold text-amber-700 dark:text-amber-300 mb-2">重新生成结果</div>
          <div class="text-xs text-slate-700 dark:text-slate-300 space-y-1">
            <div v-if="generateResult.cn"><strong>CN:</strong> {{ generateResult.cn }}</div>
            <div v-if="generateResult.issuer"><strong>Issuer:</strong> {{ generateResult.issuer }}</div>
            <div><strong>Valid Years:</strong> {{ generateResult.valid_years ?? '-' }}</div>
            <div v-if="generateResult.cert_path"><strong>Cert Path:</strong> <code class="break-all">{{ generateResult.cert_path }}</code></div>
            <div v-if="generateResult.key_path"><strong>Key Path:</strong> <code class="break-all">{{ generateResult.key_path }}</code></div>
          </div>
        </div>

        <!-- Reinstall Section -->
        <div class="p-4 rounded-lg bg-sky-50/70 dark:bg-sky-900/10 border border-sky-200 dark:border-sky-700/40 space-y-3">
          <div class="flex items-center justify-between gap-3">
            <div>
              <div class="text-sm font-semibold text-sky-700 dark:text-sky-300">重新安装</div>
              <div class="text-xs text-slate-600 dark:text-slate-400">移除旧证书 → 重新生成 → 安装到系统</div>
            </div>
            <button
              type="button"
              :disabled="busy"
              class="min-h-9 flex items-center justify-center gap-1.5 px-2.5 py-1.5 border border-sky-300 dark:border-sky-600 rounded-md text-xs text-sky-700 dark:text-sky-300 hover:bg-sky-100/70 dark:hover:bg-sky-900/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              @click="startReinstall"
            >
              <span class="material-symbols-outlined text-[16px] leading-none text-sky-500">restart_alt</span>
              <span>重新安装</span>
            </button>
          </div>
        </div>

        <!-- Operation Audit -->
        <div class="p-4 rounded-lg bg-slate-50/70 dark:bg-slate-800/40 border border-slate-200 dark:border-slate-700">
          <div class="text-xs font-semibold text-slate-600 dark:text-slate-300 mb-2">最近一次证书操作</div>
          <div v-if="lastAudit" class="text-xs text-slate-500 dark:text-slate-400 space-y-0.5">
            <div><strong>操作:</strong> {{ lastAudit.operation }}</div>
            <div><strong>结果:</strong> {{ lastAudit.ok ? '成功' : '失败' }}</div>
            <div><strong>信息:</strong> {{ lastAudit.message }}</div>
            <div><strong>时间:</strong> {{ lastAudit.time }}</div>
          </div>
          <div v-else class="text-xs text-slate-500 dark:text-slate-400">暂无记录</div>
        </div>

        <!-- Feedback Message -->
        <div
          v-if="feedback"
          class="p-3 rounded border text-xs"
          :class="feedback.type === 'error'
            ? 'border-red-200 dark:border-red-700/40 bg-red-50/60 dark:bg-red-900/10 text-red-600 dark:text-red-300'
            : 'border-emerald-200 dark:border-emerald-700/40 bg-emerald-50/60 dark:bg-emerald-900/10 text-emerald-700 dark:text-emerald-300'"
        >
          {{ feedback.message }}
        </div>
      </div>

      <div class="px-6 pb-5">
        <div class="text-center text-[11px] text-slate-400">Last refreshed: {{ lastRefreshed }}</div>
      </div>
    </div>

    <!-- Reinstall Progress Overlay -->
    <div
      v-if="progress.visible"
      class="absolute inset-0 z-20 flex items-center justify-center bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm rounded-xl"
    >
      <div class="w-full max-w-sm p-6 text-center space-y-4">
        <div class="inline-flex size-14 items-center justify-center rounded-full bg-sky-100 dark:bg-sky-900/30">
          <span
            class="size-7 border-[3px] border-sky-200 border-t-sky-600 rounded-full animate-spin"
          ></span>
        </div>
        <div>
          <div class="text-sm font-bold text-slate-700 dark:text-slate-200">{{ progress.title }}</div>
          <div class="text-xs text-slate-400 mt-1">{{ progress.detail }}</div>
        </div>
        <div class="space-y-2 text-left">
          <div
            v-for="(step, idx) in progress.steps"
            :key="idx"
            class="flex items-center gap-2.5 px-3 py-2 rounded-lg border text-xs"
            :class="stepStatusClass(step)"
          >
            <span
              v-if="step.state === 'running'"
              class="inline-block size-3.5 border-2 border-slate-300 border-t-primary rounded-full animate-spin shrink-0"
            ></span>
            <span v-else-if="step.state === 'done'" class="material-symbols-outlined text-[16px] text-emerald-500 shrink-0">check_circle</span>
            <span v-else-if="step.state === 'error'" class="material-symbols-outlined text-[16px] text-red-500 shrink-0">cancel</span>
            <span v-else class="inline-block size-3.5 border border-slate-200 rounded-full shrink-0"></span>
            <span :class="{ 'text-slate-400': step.state === 'pending' }">{{ step.label }}</span>
            <span v-if="step.message" class="ml-auto text-slate-400 truncate max-w-[140px]">{{ step.message }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted, onUnmounted, watch } from 'vue';
import { useCertStatus } from '../composables/useCertStatus';

const API_BASE = '/api';
const CERT_TYPE = 'mitm-ca';
const AUDIT_KEY = 'cert-operation-audit-v1';

const props = defineProps({
  modelValue: { type: Boolean, default: false }
});
const emit = defineEmits(['update:modelValue']);

const { certStatus, loading: certLoading, error: statusError, fetchStatus, startPolling, stopPolling, invalidateCache } = useCertStatus();

const busy = ref(false);
const downloading = ref(false);
const generateResult = ref(null);
const feedback = ref(null);
const lastRefreshed = ref('-');
const lastAudit = ref(null);

const progress = reactive({
  visible: false,
  title: '',
  detail: '',
  steps: []
});

let feedbackTimer = null;
let reinstallPollTimer = null;

function loadAudit() {
  try {
    const raw = localStorage.getItem(AUDIT_KEY);
    if (raw) lastAudit.value = JSON.parse(raw);
  } catch (_) {}
}

function saveAudit(operation, ok, message) {
  const entry = { operation, ok, message, time: new Date().toLocaleString() };
  lastAudit.value = entry;
  try { localStorage.setItem(AUDIT_KEY, JSON.stringify(entry)); } catch (_) {}
}

function showFeedback(message, type = 'success') {
  if (feedbackTimer) clearTimeout(feedbackTimer);
  feedback.value = { message, type };
  feedbackTimer = setTimeout(() => { feedback.value = null; }, 6000);
}

function updateRefreshed() {
  lastRefreshed.value = new Date().toLocaleString();
}

function clearResults() {
  generateResult.value = null;
}

async function apiCall(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch(`${API_BASE}${path}`, opts);
  if (!res.ok) {
    let msg = `请求失败 (${res.status})`;
    try {
      const data = await res.json();
      msg = data?.data?.details?.error || data?.data?.error_msg || data?.msg || data?.message || msg;
    } catch (_) {}
    throw new Error(msg);
  }
  if (method === 'GET' && path.includes('/download')) return res;
  const json = await res.json();
  return json.data || json;
}

async function checkStatus() {
  await fetchStatus();
  updateRefreshed();
}

async function downloadCertFile() {
  downloading.value = true;
  try {
    const res = await fetch(`${API_BASE}/cert/download?cert_type=${encodeURIComponent(CERT_TYPE)}`);
    if (!res.ok) {
      let msg = '下载失败';
      const ct = res.headers.get('content-type') || '';
      if (ct.includes('application/json')) {
        const err = await res.json();
        msg = err?.msg || err?.message || msg;
      }
      throw new Error(msg);
    }
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${CERT_TYPE}.pem`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
    showFeedback('证书下载成功');
    saveAudit('下载证书', true, `${CERT_TYPE}.pem`);
  } catch (err) {
    showFeedback('下载失败: ' + err.message, 'error');
    saveAudit('下载证书', false, err.message);
  } finally {
    downloading.value = false;
  }
}

async function installCert() {
  busy.value = true;
  try {
    await apiCall('POST', '/cert/install', { cert_type: CERT_TYPE });
    showFeedback('证书安装成功');
    saveAudit('安装到系统', true, CERT_TYPE);
    await checkStatus();
  } catch (err) {
    showFeedback('安装失败: ' + err.message, 'error');
    saveAudit('安装到系统', false, err.message);
  } finally {
    busy.value = false;
  }
}

async function removeCert() {
  if (!confirm('确定要移除证书吗？')) return;
  busy.value = true;
  try {
    await apiCall('POST', '/cert/remove', { cert_type: CERT_TYPE });
    showFeedback('证书已移除');
    saveAudit('移除证书', true, CERT_TYPE);
    await checkStatus();
  } catch (err) {
    showFeedback('移除失败: ' + err.message, 'error');
    saveAudit('移除证书', false, err.message);
  } finally {
    busy.value = false;
  }
}

async function generateCert() {
  if (!confirm('重新生成将覆盖当前证书文件，确定继续吗？')) return;
  busy.value = true;
  clearResults();
  try {
    const data = await apiCall('POST', '/cert/generate', { cert_type: CERT_TYPE });
    generateResult.value = data;
    showFeedback('证书生成成功');
    saveAudit('重新生成', true, CERT_TYPE);
    await checkStatus();
  } catch (err) {
    showFeedback('生成失败: ' + err.message, 'error');
    saveAudit('重新生成', false, err.message);
  } finally {
    busy.value = false;
  }
}

function stepStatusClass(step) {
  switch (step.state) {
    case 'running':
      return 'border-primary/40 bg-primary/5 dark:border-primary/30 dark:bg-primary/5';
    case 'done':
      return 'border-emerald-200 bg-emerald-50 dark:border-emerald-700/30 dark:bg-emerald-900/10';
    case 'error':
      return 'border-red-200 bg-red-50 dark:border-red-700/30 dark:bg-red-900/10';
    default:
      return 'border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-800/30';
  }
}

function setStepState(steps, index, state, message) {
  if (steps[index]) {
    steps[index].state = state;
    steps[index].message = message || '';
  }
}

function silentStatusPoll() {
  return fetch(`${API_BASE}/cert/status?cert_type=${encodeURIComponent(CERT_TYPE)}`)
    .then(r => r.ok ? r.json().then(j => j.data || j) : null)
    .catch(() => null);
}

function stopReinstallPoll() {
  if (reinstallPollTimer) { clearInterval(reinstallPollTimer); reinstallPollTimer = null; }
}

async function startReinstall() {
  busy.value = true;
  stopReinstallPoll();

  const steps = [
    { label: '移除旧证书', state: 'pending', message: '' },
    { label: '重新生成证书', state: 'pending', message: '' },
    { label: '安装到系统', state: 'pending', message: '' },
    { label: '验证安装结果', state: 'pending', message: '' }
  ];

  progress.visible = true;
  progress.title = '正在重新安装证书';
  progress.detail = '本地证书';
  progress.steps = steps;

  const finalStatus = { success: false };

  // Start background polling (separate from composable polling)
  reinstallPollTimer = setInterval(async () => {
    const status = await silentStatusPoll();
    if (status) {
      certStatus.value = status;
      invalidateCache();
    }
  }, 1500);

  // Step 1: Remove old cert (ignore errors - cert may not exist)
  setStepState(steps, 0, 'running', '正在移除...');
  try {
    await apiCall('POST', '/cert/remove', { cert_type: CERT_TYPE });
    setStepState(steps, 0, 'done', '已移除或不存在');
  } catch (_) {
    setStepState(steps, 0, 'done', '旧证书不存在，跳过');
  }

  // Step 2: Generate new cert
  setStepState(steps, 1, 'running', '正在生成...');
  try {
    await apiCall('POST', '/cert/generate', { cert_type: CERT_TYPE });
    setStepState(steps, 1, 'done', '生成成功');
  } catch (err) {
    setStepState(steps, 1, 'error', err.message);
    finalStatus.success = false;
    finishReinstall(steps, false, '生成证书失败');
    return;
  }

  // Step 3: Install to system
  setStepState(steps, 2, 'running', '正在安装...');
  try {
    await apiCall('POST', '/cert/install', { cert_type: CERT_TYPE });
    setStepState(steps, 2, 'done', '安装成功');
  } catch (err) {
    setStepState(steps, 2, 'error', err.message);
    finalStatus.success = false;
    finishReinstall(steps, false, '安装证书失败');
    return;
  }

  // Step 4: Verify
  setStepState(steps, 3, 'running', '正在验证...');
  try {
    const status = await apiCall('GET', `/cert/status?cert_type=${encodeURIComponent(CERT_TYPE)}`);
    certStatus.value = status;
    invalidateCache();
    updateRefreshed();
    if (status.is_installed) {
      setStepState(steps, 3, 'done', status.is_trusted ? '已安装并受信任' : '已安装');
      finishReinstall(steps, true, '重新安装成功');
    } else {
      setStepState(steps, 3, 'error', '证书未被检测为已安装');
      finishReinstall(steps, false, '安装验证未通过');
    }
  } catch (err) {
    setStepState(steps, 3, 'error', err.message);
    finishReinstall(steps, false, '验证失败: ' + err.message);
  }
}

function finishReinstall(steps, success, message) {
  stopReinstallPoll();

  if (success) {
    setTimeout(() => {
      progress.visible = false;
      busy.value = false;
      showFeedback('重新安装完成：' + message);
      saveAudit('重新安装', true, message);
      checkStatus();
    }, 800);
  } else {
    for (let i = 0; i < steps.length; i++) {
      if (steps[i].state === 'running') {
        steps[i].state = 'pending';
        steps[i].message = '';
      }
    }
    progress.title = '重新安装失败';
    progress.detail = '本地证书';

    setTimeout(() => {
      progress.visible = false;
      busy.value = false;
      showFeedback('重新安装失败：' + message, 'error');
      saveAudit('重新安装', false, message);
      checkStatus();
    }, 2500);
  }
}

function open() {
  loadAudit();
  startPolling();
  emit('update:modelValue', true);
}

function close() {
  if (busy.value) return;
  stopPolling();
  emit('update:modelValue', false);
}

function onKeydown(e) {
  if (e.key === 'Escape' && props.modelValue && !busy.value) {
    close();
  }
}

watch(() => props.modelValue, (val) => {
  if (val) {
    loadAudit();
    startPolling();
  } else {
    if (!busy.value) stopPolling();
  }
});

onMounted(() => {
  document.addEventListener('keydown', onKeydown);
});

onUnmounted(() => {
  stopReinstallPoll();
  if (feedbackTimer) clearTimeout(feedbackTimer);
  document.removeEventListener('keydown', onKeydown);
});

defineExpose({ open, close, startReinstall });
</script>
