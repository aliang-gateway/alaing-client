<template>
  <div
    v-if="open"
    class="fixed inset-0 z-[130] flex items-center justify-center p-4"
    role="dialog"
    aria-modal="true"
    aria-label="Quick Setup"
  >
    <div class="absolute inset-0 bg-slate-900/40 backdrop-blur-sm" @click="emit('close')"></div>

    <div
      class="relative z-10 w-full max-w-5xl h-[680px] bg-white dark:bg-slate-900 shadow-2xl rounded-xl flex overflow-hidden border border-slate-200 dark:border-slate-800"
    >
      <aside class="w-64 border-r border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/30 flex flex-col">
        <div class="p-6 border-b border-slate-100 dark:border-slate-800">
          <h3 class="text-sm font-bold text-slate-400 uppercase tracking-widest">Software</h3>
        </div>

        <nav class="flex-1 px-4 py-3 space-y-2 overflow-y-auto custom-scrollbar">
          <button
            v-for="software in softwares"
            :key="software"
            type="button"
            :class="[
              'w-full flex items-center gap-3 px-4 py-3 rounded-lg text-sm transition-colors border',
              software === selectedSoftware
                ? 'bg-white dark:bg-slate-800 border-primary/20 text-primary font-bold shadow-sm'
                : 'border-transparent text-slate-600 dark:text-slate-400 hover:bg-white dark:hover:bg-slate-800 font-medium',
            ]"
            @click="selectSoftware(software)"
          >
            <span
              :class="[
                'size-2 rounded-full',
                software === selectedSoftware ? 'bg-primary' : 'bg-slate-300 dark:bg-slate-600',
              ]"
            ></span>
            {{ softwareLabel(software) }}
          </button>

          <button
            type="button"
            class="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg text-sm border border-dashed border-slate-300 dark:border-slate-600 text-slate-600 dark:text-slate-300 hover:bg-white dark:hover:bg-slate-800"
            @click="showAddSoftware = !showAddSoftware"
          >
            <span class="material-symbols-outlined text-base">add</span>
            新增软件
          </button>

          <div v-if="showAddSoftware" class="p-3 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 space-y-2">
            <input
              v-model="newSoftwareName"
              class="w-full px-3 py-2 text-sm bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded"
              type="text"
              placeholder="输入软件名"
              @keydown.enter.prevent="addSoftware"
            />
            <button
              type="button"
              class="w-full py-2 text-xs font-bold bg-primary text-white rounded hover:bg-primary/90"
              @click="addSoftware"
            >
              添加
            </button>
          </div>
        </nav>

        <div class="p-6 mt-auto border-t border-slate-100 dark:border-slate-800">
          <button
            type="button"
            class="w-full py-2.5 bg-primary text-white text-xs font-bold rounded-lg hover:bg-primary/90 transition-all flex items-center justify-center gap-2 shadow-lg shadow-primary/20 disabled:opacity-60"
            :disabled="syncing"
            @click="syncAllToCloud"
          >
            <span class="material-symbols-outlined text-sm">cloud_upload</span>
            {{ syncing ? 'Syncing...' : 'Sync All to Cloud' }}
          </button>
        </div>
      </aside>

      <div class="flex-1 flex flex-col bg-white dark:bg-slate-900 overflow-hidden">
        <header class="h-16 border-b border-slate-100 dark:border-slate-800 px-8 flex items-center justify-between shrink-0">
          <div class="flex items-center gap-3">
            <span class="text-slate-900 dark:text-white font-bold">{{ softwareLabel(selectedSoftware) }} Configurations</span>
            <span class="px-2 py-0.5 bg-primary/10 text-primary text-[10px] font-bold rounded uppercase">
              {{ configs.length }} items
            </span>
          </div>
          <button
            type="button"
            aria-label="Close quick setup"
            class="size-8 flex items-center justify-center text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
            @click="emit('close')"
          >
            <span class="material-symbols-outlined">close</span>
          </button>
        </header>

        <div class="flex-1 overflow-y-auto p-8 custom-scrollbar space-y-8">
          <section>
            <div class="flex items-center justify-between mb-4">
              <h4 class="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Config Items</h4>
              <div class="flex items-center gap-2">
                <button
                  type="button"
                  class="px-3 py-1.5 text-xs font-bold border border-slate-200 dark:border-slate-700 rounded hover:bg-white dark:hover:bg-slate-800 transition-colors"
                  @click="createNewConfig"
                >
                  新增配置项
                </button>
                <button
                  type="button"
                  class="px-3 py-1.5 text-xs font-bold border border-slate-200 dark:border-slate-700 rounded hover:bg-white dark:hover:bg-slate-800 transition-colors"
                  @click="loadConfigs"
                >
                  Refresh
                </button>
              </div>
            </div>

            <div class="space-y-3">
              <button
                v-for="item in configs"
                :key="item.uuid"
                type="button"
                :class="[
                  'w-full p-4 rounded-xl flex items-center justify-between transition-all text-left',
                  selectedConfig?.uuid === item.uuid
                    ? 'border border-primary/20 bg-primary/5'
                    : 'border border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/20',
                ]"
                @click="selectConfig(item)"
              >
                <div>
                  <p class="text-sm font-bold text-slate-800 dark:text-white">{{ item.name }}</p>
                  <p class="text-xs text-slate-500 mt-1">Version: {{ item.version || 'v1' }}</p>
                </div>
                <span
                  :class="[
                    'px-2 py-0.5 text-[10px] font-bold rounded uppercase',
                    item.in_use
                      ? 'bg-primary/10 text-primary'
                      : 'bg-slate-200 text-slate-500 dark:bg-slate-700 dark:text-slate-300',
                  ]"
                >
                  {{ item.in_use ? 'In Use' : 'Saved' }}
                </span>
              </button>

              <div
                v-if="!configs.length"
                class="p-4 rounded-xl border border-dashed border-slate-200 dark:border-slate-700 text-sm text-slate-500"
              >
                No configuration yet for {{ softwareLabel(selectedSoftware) }}.
              </div>
            </div>
          </section>

          <section class="border-t border-slate-100 dark:border-slate-800 pt-8">
            <div class="flex items-center justify-between mb-6">
              <h4 class="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Selected Config</h4>
              <span class="text-[10px] text-slate-400">{{ selectedConfig?.uuid || 'N/A' }}</span>
            </div>

            <div class="grid grid-cols-2 gap-6">
              <div class="space-y-1.5 col-span-1">
                <label class="text-xs font-bold text-slate-500 ml-1" for="quickSetupConfigName">Config Name</label>
                <input
                  id="quickSetupConfigName"
                  v-model="form.name"
                  class="w-full px-4 py-2.5 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg text-sm focus:ring-primary focus:border-primary"
                  type="text"
                />
              </div>
              <div class="space-y-1.5 col-span-1">
                <label class="text-xs font-bold text-slate-500 ml-1" for="quickSetupVersion">Version</label>
                <input
                  id="quickSetupVersion"
                  v-model="form.version"
                  class="w-full px-4 py-2.5 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg text-sm focus:ring-primary focus:border-primary"
                  type="text"
                />
              </div>
              <div class="space-y-1.5 col-span-2">
                <label class="text-xs font-bold text-slate-500 ml-1" for="quickSetupFilePath">Local Disk Path</label>
                <input
                  id="quickSetupFilePath"
                  v-model="softwarePath"
                  class="w-full px-4 py-2.5 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg text-sm focus:ring-primary focus:border-primary"
                  type="text"
                />
              </div>
              <div class="space-y-1.5 col-span-2">
                <label class="text-xs font-bold text-slate-500 ml-1" for="quickSetupContent">Content</label>
                <textarea
                  id="quickSetupContent"
                  v-model="form.content"
                  class="w-full px-4 py-3 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg text-sm font-mono focus:ring-primary focus:border-primary"
                  rows="6"
                ></textarea>
              </div>
            </div>

            <div class="flex items-center justify-between mt-6">
              <p class="text-xs text-slate-500">{{ statusMessage }}</p>
              <div class="flex justify-end gap-3">
                <button
                  type="button"
                  class="px-4 py-2 text-sm font-bold text-slate-600 hover:text-slate-800 transition-colors"
                  @click="saveConfig"
                >
                  Save
                </button>
                <button
                  type="button"
                  class="px-6 py-2 bg-primary text-white text-sm font-bold rounded-lg hover:bg-primary/90 shadow-lg shadow-primary/20 transition-all disabled:opacity-60"
                  :disabled="applying"
                  @click="applyConfig"
                >
                  {{ applying ? 'Applying...' : 'Apply' }}
                </button>
              </div>
            </div>

            <div class="mt-5 grid grid-cols-1 md:grid-cols-2 gap-3">
              <input
                v-model="cloud.cloudUrl"
                class="px-4 py-2.5 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg text-sm focus:ring-primary focus:border-primary"
                type="text"
                placeholder="Cloud sync URL (e.g. https://example.com/configs)"
              />
              <input
                v-model="cloud.authToken"
                class="px-4 py-2.5 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg text-sm focus:ring-primary focus:border-primary"
                type="text"
                placeholder="Cloud auth token (optional)"
              />
            </div>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue';

const props = defineProps({
  open: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['close']);

const softwares = ref(['opencode', 'claude', 'cursor', 'openai']);
const selectedSoftware = ref('opencode');
const configs = ref([]);
const selectedConfig = ref(null);
const statusMessage = ref('Select software and configuration item.');
const applying = ref(false);
const syncing = ref(false);
const showAddSoftware = ref(false);
const newSoftwareName = ref('');
const softwarePaths = reactive({
  opencode: defaultPathForSoftware('opencode'),
  claude: defaultPathForSoftware('claude'),
  cursor: defaultPathForSoftware('cursor'),
  openai: defaultPathForSoftware('openai'),
});

const cloud = reactive({
  cloudUrl: '',
  authToken: '',
});

const form = reactive({
  uuid: '',
  name: '',
  version: 'v1',
  content: '{}',
  format: 'json',
});

const softwarePath = computed({
  get() {
    return softwarePaths[selectedSoftware.value] || defaultPathForSoftware(selectedSoftware.value);
  },
  set(value) {
    softwarePaths[selectedSoftware.value] = value;
  },
});

const hasValidJson = computed(() => {
  try {
    JSON.parse(form.content || '{}');
    return true;
  } catch {
    return false;
  }
});

function softwareLabel(software) {
  return software.charAt(0).toUpperCase() + software.slice(1);
}

async function apiCall(endpoint, options = {}) {
  const response = await fetch(`/api${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
    ...options,
  });
  const payload = await response.json();
  if (!response.ok || payload.code !== 0) {
    throw new Error(payload?.msg || 'Request failed');
  }
  return payload.data;
}

function applyConfigToForm(item) {
  form.uuid = item?.uuid || '';
  form.name = item?.name || '';
  form.version = item?.version || 'v1';
  form.content = item?.content || '{}';
  form.format = item?.format || 'json';
}

function defaultPathForSoftware(software) {
  return `~/.config/${software}/config.json`;
}

function resetFormForSoftware(software) {
  form.uuid = '';
  form.name = `${softwareLabel(software)} Default`;
  form.version = 'v1';
  form.content = '{}';
  form.format = 'json';
}

function createNewConfig() {
  selectedConfig.value = null;
  form.uuid = '';
  form.name = `${softwareLabel(selectedSoftware.value)} Config ${configs.value.length + 1}`;
  form.version = 'v1';
  form.content = '{}';
  form.format = 'json';
  statusMessage.value = '新配置项已创建，请填写后保存。';
}

function addSoftware() {
  const normalized = newSoftwareName.value.trim().toLowerCase();
  if (!normalized) {
    statusMessage.value = '请输入软件名。';
    return;
  }
  if (!softwares.value.includes(normalized)) {
    softwares.value.push(normalized);
  }
  if (!softwarePaths[normalized]) {
    softwarePaths[normalized] = defaultPathForSoftware(normalized);
  }
  newSoftwareName.value = '';
  showAddSoftware.value = false;
  selectSoftware(normalized);
  statusMessage.value = `已新增软件：${softwareLabel(normalized)}`;
}

async function loadConfigs() {
  const data = await apiCall(`/software-config/list?software=${encodeURIComponent(selectedSoftware.value)}`, {
    method: 'GET',
  });
  configs.value = data.items || [];
  if (configs.value.length > 0) {
    const normalizedPath = configs.value[0].file_path || softwarePath.value || defaultPathForSoftware(selectedSoftware.value);
    softwarePath.value = normalizedPath;
    configs.value = configs.value.map((item) => ({
      ...item,
      file_path: normalizedPath,
    }));
    selectedConfig.value = configs.value[0];
    applyConfigToForm(configs.value[0]);
  } else {
    selectedConfig.value = null;
    resetFormForSoftware(selectedSoftware.value);
  }
}

function selectSoftware(software) {
  selectedSoftware.value = software;
}

function selectConfig(item) {
  selectedConfig.value = item;
  applyConfigToForm(item);
}

async function saveConfig() {
  if (!form.name || !softwarePath.value) {
    statusMessage.value = 'Config name and local disk path are required.';
    return;
  }
  if (!hasValidJson.value) {
    statusMessage.value = 'Content must be valid JSON.';
    return;
  }

  try {
    const data = await apiCall('/software-config/save', {
      method: 'POST',
      body: JSON.stringify({
        uuid: form.uuid,
        software: selectedSoftware.value,
        name: form.name,
        file_path: softwarePath.value,
        version: form.version,
        in_use: !!selectedConfig.value?.in_use,
        format: form.format,
        content: form.content,
      }),
    });
    form.uuid = data.uuid;
    statusMessage.value = 'Config saved.';
    await loadConfigs();
  } catch (error) {
    statusMessage.value = `Save failed: ${error.message}`;
  }
}

async function applyConfig() {
  if (!form.name || !softwarePath.value) {
    statusMessage.value = 'Config name and local disk path are required.';
    return;
  }
  if (!hasValidJson.value) {
    statusMessage.value = 'Content must be valid JSON.';
    return;
  }

  applying.value = true;
  try {
    await apiCall('/software-config/activate', {
      method: 'POST',
      body: JSON.stringify({
        uuid: form.uuid,
        software: selectedSoftware.value,
        name: form.name,
        file_path: softwarePath.value,
        version: form.version,
        format: form.format,
        content: form.content,
      }),
    });
    statusMessage.value = 'Config applied.';
    await loadConfigs();
  } catch (error) {
    statusMessage.value = `Apply failed: ${error.message}`;
  } finally {
    applying.value = false;
  }
}

async function syncAllToCloud() {
  if (!cloud.cloudUrl.trim()) {
    statusMessage.value = 'Cloud URL is required for sync.';
    return;
  }
  syncing.value = true;
  try {
    const data = await apiCall('/software-config/cloud/push', {
      method: 'POST',
      body: JSON.stringify({
        cloud_url: cloud.cloudUrl,
        auth_token: cloud.authToken,
      }),
    });
    statusMessage.value = `Synced ${data.synced_count} configs at ${data.last_synced_at}.`;
  } catch (error) {
    statusMessage.value = `Cloud sync failed: ${error.message}`;
  } finally {
    syncing.value = false;
  }
}

watch(
  () => props.open,
  async (value) => {
    if (!value) {
      return;
    }
    try {
      await loadConfigs();
      statusMessage.value = 'Ready.';
    } catch (error) {
      statusMessage.value = `Load failed: ${error.message}`;
      resetFormForSoftware(selectedSoftware.value);
    }
  },
  { immediate: true },
);

watch(selectedSoftware, async () => {
  if (!props.open) {
    return;
  }
  try {
    await loadConfigs();
    statusMessage.value = 'Ready.';
  } catch (error) {
    statusMessage.value = `Load failed: ${error.message}`;
    resetFormForSoftware(selectedSoftware.value);
  }
});
</script>
