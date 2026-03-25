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

            <div
              :class="[
                'space-y-3 custom-scrollbar pr-1',
                editorExpanded
                  ? 'overflow-visible'
                  : 'max-h-[300px] overflow-y-auto',
              ]"
            >
              <div v-for="item in configs" :key="item.uuid" class="space-y-3">
                <div
                  role="button"
                  tabindex="0"
                  :class="[
                    'w-full p-4 rounded-xl flex items-center justify-between transition-all text-left',
                    selectedConfig?.uuid === item.uuid
                      ? 'border border-primary/20 bg-primary/5'
                      : 'border border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/20',
                  ]"
                  @click="toggleConfigEditor(item)"
                  @keydown.enter.prevent="toggleConfigEditor(item)"
                >
                  <div>
                    <p class="text-sm font-bold text-slate-800 dark:text-white">{{ item.name }}</p>
                    <p class="text-xs text-slate-500 mt-1">Version: {{ item.version || 'v1' }}</p>
                    <p class="text-[11px] text-slate-400 mt-0.5 truncate max-w-[340px]" :title="item.file_path">{{ item.file_path }}</p>
                  </div>
                  <div class="flex items-center gap-2">
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
                    <button
                      type="button"
                      class="px-2 py-0.5 text-[10px] font-bold rounded uppercase bg-blue-100 text-blue-700 hover:bg-blue-200"
                      @click.stop="applyConfigItem(item)"
                    >
                      应用
                    </button>
                    <button
                      type="button"
                      class="px-2 py-0.5 text-[10px] font-bold rounded uppercase bg-emerald-100 text-emerald-700 hover:bg-emerald-200"
                      @click.stop="editConfigItem(item)"
                    >
                      编辑
                    </button>
                  </div>
                </div>

                <transition name="slide-up-panel">
                  <section
                    v-if="editorExpanded && selectedConfig?.uuid === item.uuid"
                    class="border-t border-slate-100 dark:border-slate-800 pt-6 px-1"
                  >
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
                      <div class="space-y-1.5 col-span-1">
                        <label class="text-xs font-bold text-slate-500 ml-1" for="quickSetupFormat">Format</label>
                        <select
                          id="quickSetupFormat"
                          v-model="form.format"
                          class="w-full px-4 py-2.5 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg text-sm focus:ring-primary focus:border-primary"
                        >
                          <option value="json">json</option>
                          <option value="yaml">yaml</option>
                        </select>
                      </div>
                      <div class="space-y-1.5 col-span-2">
                        <label class="text-xs font-bold text-slate-500 ml-1" for="quickSetupFilePath">Local Disk Path</label>
                        <input
                          id="quickSetupFilePath"
                          v-model="form.filePath"
                          class="w-full px-4 py-2.5 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg text-sm focus:ring-primary focus:border-primary"
                          type="text"
                        />
                      </div>
                      <div class="space-y-1.5 col-span-2">
                        <label class="text-xs font-bold text-slate-500 ml-1" for="quickSetupContent">Content</label>
                        <div class="border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden">
                          <CodeMirror
                            id="quickSetupContent"
                            v-model="form.content"
                            class="text-sm"
                            :extensions="editorExtensions"
                            :style="{ minHeight: '220px' }"
                            :basic-setup="basicSetup"
                          />
                        </div>
                      </div>
                    </div>

                    <div class="flex items-center justify-between mt-6">
                      <p class="text-xs text-slate-500">{{ statusMessage }}</p>
                    <div class="flex justify-end gap-3">
                        <button
                          type="button"
                          class="px-4 py-2 text-sm font-bold text-slate-500 hover:text-slate-700 transition-colors"
                          @click="collapseEditor"
                        >
                          取消编辑
                        </button>
                        <button
                          type="button"
                          class="px-4 py-2 text-sm font-bold text-slate-600 hover:text-slate-800 transition-colors"
                          @click="saveConfig"
                        >
                          保存
                        </button>
                        <button
                          type="button"
                          class="px-6 py-2 bg-primary text-white text-sm font-bold rounded-lg hover:bg-primary/90 shadow-lg shadow-primary/20 transition-all disabled:opacity-60"
                          :disabled="applying"
                          @click="applyConfig"
                        >
                          {{ applying ? '应用中...' : '应用' }}
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
                </transition>
              </div>

              <div
                v-if="!configs.length"
                class="p-4 rounded-xl border border-dashed border-slate-200 dark:border-slate-700 text-sm text-slate-500"
              >
                No configuration yet for {{ softwareLabel(selectedSoftware) }}.
              </div>
            </div>
          </section>

          <section v-show="!editorExpanded" class="border-t border-slate-100 dark:border-slate-800 pt-6">
            <div class="rounded-lg border border-dashed border-slate-200 dark:border-slate-700 p-4 text-xs text-slate-500">
              点击配置项上的“编辑”按钮，从底部展开编辑窗口。
            </div>
          </section>

        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue';
import { Codemirror as CodeMirror } from 'vue-codemirror';
import { json } from '@codemirror/lang-json';
import { yaml } from '@codemirror/lang-yaml';
import { oneDark } from '@codemirror/theme-one-dark';
import YAML from 'yaml';

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
const editorExpanded = ref(false);
const showAddSoftware = ref(false);
const newSoftwareName = ref('');
const cloud = reactive({
  cloudUrl: '',
  authToken: '',
});

const form = reactive({
  uuid: '',
  name: '',
  filePath: '',
  version: 'v1',
  content: '{}',
  format: 'json',
});

const basicSetup = {
  lineNumbers: true,
  foldGutter: true,
  highlightActiveLineGutter: true,
  highlightActiveLine: true,
};

const editorExtensions = computed(() => {
  const modeExtension = form.format === 'yaml' ? yaml() : json();
  return [modeExtension, oneDark];
});

const hasValidContent = computed(() => {
  const content = form.content || '';
  if (form.format === 'yaml') {
    try {
      YAML.parse(content || '{}');
      return true;
    } catch {
      return false;
    }
  }

  try {
    JSON.parse(content || '{}');
    return true;
  } catch {
    return false;
  }
});

function prettyFormatContent(content, format) {
  const source = (content || '').trim();
  if (!source) {
    return format === 'yaml' ? '' : '{}';
  }

  if (format === 'yaml') {
    try {
      const parsed = YAML.parse(source);
      return YAML.stringify(parsed ?? {}).trim();
    } catch {
      return source;
    }
  }

  try {
    return JSON.stringify(JSON.parse(source), null, 2);
  } catch {
    return source;
  }
}

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
  const itemFormat = item?.format === 'yaml' ? 'yaml' : 'json';
  form.uuid = item?.uuid || '';
  form.name = item?.name || '';
  form.filePath = item?.file_path || defaultPathForSoftware(item?.software || selectedSoftware.value);
  form.version = item?.version || 'v1';
  form.format = itemFormat;
  form.content = prettyFormatContent(item?.content || '{}', itemFormat);
}

function defaultPathForSoftware(software) {
  return `~/.config/${software}/config.json`;
}

function resetFormForSoftware(software) {
  form.uuid = '';
  form.name = `${softwareLabel(software)} Default`;
  form.filePath = defaultPathForSoftware(software);
  form.version = 'v1';
  form.content = '{}';
  form.format = 'json';
}

function createNewConfig() {
  selectedConfig.value = null;
  form.uuid = '';
  form.name = `${softwareLabel(selectedSoftware.value)} Config ${configs.value.length + 1}`;
  form.filePath = defaultPathForSoftware(selectedSoftware.value);
  form.version = 'v1';
  form.content = '{}';
  form.format = 'json';
  editorExpanded.value = false;
  statusMessage.value = '新配置项已创建，请点击“编辑”展开编辑窗口。';
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

function collapseEditor() {
  editorExpanded.value = false;
  statusMessage.value = '编辑已收起。';
}

function toggleConfigEditor(item) {
  const isSameItem = selectedConfig.value?.uuid === item?.uuid;
  if (editorExpanded.value && isSameItem) {
    collapseEditor();
    return;
  }
  selectConfig(item);
}

function editConfigItem(item) {
  selectedConfig.value = item;
  applyConfigToForm(item);
  editorExpanded.value = true;
}

async function saveConfig() {
  if (!form.name || !form.filePath) {
    statusMessage.value = 'Config name and local disk path are required.';
    return;
  }
  if (!hasValidContent.value) {
    statusMessage.value = `Content must be valid ${form.format.toUpperCase()}.`;
    return;
  }

  try {
    const data = await apiCall('/software-config/save', {
      method: 'POST',
      body: JSON.stringify({
        uuid: form.uuid,
        software: selectedSoftware.value,
        name: form.name,
        file_path: form.filePath,
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
  if (!form.name || !form.filePath) {
    statusMessage.value = 'Config name and local disk path are required.';
    return;
  }
  if (!hasValidContent.value) {
    statusMessage.value = `Content must be valid ${form.format.toUpperCase()}.`;
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
        file_path: form.filePath,
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

async function applyConfigItem(item) {
  applying.value = true;
  try {
    await apiCall('/software-config/activate', {
      method: 'POST',
      body: JSON.stringify({
        uuid: item.uuid,
        software: item.software,
        name: item.name,
        file_path: item.file_path,
        version: item.version,
        format: item.format,
        content: item.content,
      }),
    });
    statusMessage.value = `${item.name} applied.`;
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
      editorExpanded.value = false;
      return;
    }
    try {
      await loadConfigs();
      editorExpanded.value = false;
      statusMessage.value = 'Ready.';
    } catch (error) {
      statusMessage.value = `Load failed: ${error.message}`;
      resetFormForSoftware(selectedSoftware.value);
      editorExpanded.value = false;
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
    editorExpanded.value = false;
    statusMessage.value = 'Ready.';
  } catch (error) {
    statusMessage.value = `Load failed: ${error.message}`;
    resetFormForSoftware(selectedSoftware.value);
    editorExpanded.value = false;
  }
});
</script>

<style scoped>
.slide-up-panel-enter-active,
.slide-up-panel-leave-active {
  transition: all 0.25s ease;
}

.slide-up-panel-enter-from,
.slide-up-panel-leave-to {
  opacity: 0;
  transform: translateY(16px);
  max-height: 0;
}

.slide-up-panel-enter-to,
.slide-up-panel-leave-from {
  opacity: 1;
  transform: translateY(0);
  max-height: 1400px;
}
</style>
