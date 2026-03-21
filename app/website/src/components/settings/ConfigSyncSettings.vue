<template>
  <div class="settings-content hidden" data-content="config-sync">
    <div class="settings-card">
      <div class="flex items-center justify-between mb-4 gap-3 flex-wrap">
        <h3 class="settings-card-title !mb-0">配置同步中心</h3>
        <div class="flex items-center gap-2">
          <button type="button" class="settings-btn-outline" @click="loadConfigs">刷新</button>
          <button type="button" class="settings-btn-primary" :disabled="pushing" @click="pushSelectedToCloud">
            {{ pushing ? '推送中...' : '一键推送选中到云端' }}
          </button>
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-3 mb-4">
        <input
          v-model="filters.software"
          class="px-3 py-2 border border-slate-300 rounded"
          type="text"
          placeholder="软件名（可选）"
          @keydown.enter.prevent="loadConfigs"
        />
        <div class="flex gap-2">
          <input
            v-model="cloud.cloudUrl"
            class="flex-1 px-3 py-2 border border-slate-300 rounded"
            type="text"
            placeholder="云端 URL"
          />
          <button type="button" class="settings-btn-outline" :disabled="comparing" @click="compareWithCloud">
            {{ comparing ? '比较中...' : '比较新旧' }}
          </button>
        </div>
        <input
          v-model="cloud.authToken"
          class="px-3 py-2 border border-slate-300 rounded md:col-span-2"
          type="text"
          placeholder="云端 Token（可选）"
        />
      </div>

      <div class="overflow-auto border border-slate-200 rounded-lg">
        <table class="w-full text-sm">
          <thead class="bg-slate-50">
            <tr>
              <th class="text-left p-2">选择</th>
              <th class="text-left p-2">软件名</th>
              <th class="text-left p-2">配置名</th>
              <th class="text-left p-2">路径</th>
              <th class="text-left p-2">版本</th>
              <th class="text-left p-2">更新时间</th>
              <th class="text-left p-2">对比</th>
              <th class="text-left p-2">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in items" :key="item.uuid" class="border-t border-slate-100">
              <td class="p-2">
                <input
                  type="checkbox"
                  :checked="!!item.selected"
                  @change="toggleSelect(item, $event.target.checked)"
                />
              </td>
              <td class="p-2">{{ item.software }}</td>
              <td class="p-2">{{ item.name }}</td>
              <td class="p-2 truncate max-w-[240px]" :title="item.file_path">{{ item.file_path }}</td>
              <td class="p-2">{{ item.version || '-' }}</td>
              <td class="p-2">{{ item.updated_at || '-' }}</td>
              <td class="p-2">
                <span
                  class="text-xs px-2 py-0.5 rounded"
                  :class="freshnessClass(item.freshness_status)"
                >
                  {{ freshnessLabel(item.freshness_status) }}
                </span>
              </td>
              <td class="p-2">
                <div class="flex gap-2 flex-wrap">
                  <button type="button" class="settings-btn-outline !py-1 !px-2" @click="applyItem(item)">应用</button>
                  <button type="button" class="settings-btn-outline !py-1 !px-2" @click="copyContent(item)">复制</button>
                  <button type="button" class="settings-btn-outline !py-1 !px-2" @click="removeConfig(item)">删除</button>
                </div>
              </td>
            </tr>
            <tr v-if="!items.length">
              <td class="p-4 text-slate-500" colspan="8">暂无配置数据</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 grid grid-cols-1 md:grid-cols-2 gap-3">
        <input v-model="editor.software" class="px-3 py-2 border border-slate-300 rounded" placeholder="软件名" type="text" />
        <input v-model="editor.name" class="px-3 py-2 border border-slate-300 rounded" placeholder="配置名" type="text" />
        <input v-model="editor.filePath" class="px-3 py-2 border border-slate-300 rounded md:col-span-2" placeholder="配置路径" type="text" />
        <input v-model="editor.version" class="px-3 py-2 border border-slate-300 rounded" placeholder="版本号" type="text" />
        <select v-model="editor.format" class="px-3 py-2 border border-slate-300 rounded">
          <option value="json">json</option>
          <option value="yaml">yaml</option>
        </select>
        <div class="md:col-span-2 border border-slate-300 rounded overflow-hidden">
          <CodeMirror
            v-model="editor.content"
            class="text-sm"
            :extensions="editorExtensions"
            :style="{ minHeight: '220px' }"
            :basic-setup="basicSetup"
          />
        </div>
      </div>

      <div class="mt-3 flex justify-end gap-2">
        <button type="button" class="settings-btn-secondary" @click="resetEditor">重置</button>
        <button type="button" class="settings-btn-primary" :disabled="saving" @click="saveConfig">
          {{ saving ? '保存中...' : '保存配置' }}
        </button>
      </div>

      <div class="mt-3 text-sm text-slate-500">{{ status }}</div>
    </div>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue';
import { Codemirror as CodeMirror } from 'vue-codemirror';
import { json } from '@codemirror/lang-json';
import { yaml } from '@codemirror/lang-yaml';
import { oneDark } from '@codemirror/theme-one-dark';

const items = ref([]);
const status = ref('就绪');
const pushing = ref(false);
const saving = ref(false);
const comparing = ref(false);

const filters = reactive({ software: '' });
const cloud = reactive({ cloudUrl: '', authToken: '' });
const editor = reactive({
  uuid: '',
  software: '',
  name: '',
  filePath: '',
  version: 'v1',
  format: 'json',
  content: '{}',
});

const basicSetup = {
  lineNumbers: true,
  foldGutter: true,
  highlightActiveLineGutter: true,
  highlightActiveLine: true,
};

const editorExtensions = computed(() => {
  const modeExtension = editor.format === 'yaml' ? yaml() : json();
  return [modeExtension, oneDark];
});

function normalizeApi(payload) {
  return payload?.data ?? payload;
}

async function request(path, options = {}) {
  const resp = await fetch(`/api${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
    ...options,
  });
  const payload = await resp.json();
  if (!resp.ok || payload.code !== 0) {
    throw new Error(payload.msg || '请求失败');
  }
  return normalizeApi(payload);
}

function resetEditor() {
  editor.uuid = '';
  editor.software = filters.software || '';
  editor.name = '';
  editor.filePath = '';
  editor.version = 'v1';
  editor.format = 'json';
  editor.content = '{}';
}

function freshnessClass(statusValue) {
  switch (statusValue) {
    case 'local_newer':
      return 'bg-emerald-100 text-emerald-700';
    case 'cloud_newer':
      return 'bg-amber-100 text-amber-700';
    case 'same':
      return 'bg-slate-100 text-slate-700';
    case 'local_only':
      return 'bg-blue-100 text-blue-700';
    case 'cloud_only':
      return 'bg-purple-100 text-purple-700';
    default:
      return 'bg-slate-100 text-slate-500';
  }
}

function freshnessLabel(statusValue) {
  switch (statusValue) {
    case 'local_newer':
      return '本地更新';
    case 'cloud_newer':
      return '云端更新';
    case 'same':
      return '一致';
    case 'local_only':
      return '仅本地';
    case 'cloud_only':
      return '仅云端';
    default:
      return '-';
  }
}

async function loadConfigs() {
  const query = filters.software.trim()
    ? `?software=${encodeURIComponent(filters.software.trim())}`
    : '';
  const data = await request(`/software-config/list${query}`, { method: 'GET' });
  items.value = (data.items || []).map((item) => ({ ...item, freshness_status: item.freshness_status || '' }));
  status.value = `已加载 ${items.value.length} 条配置`; 
}

function fillEditor(item) {
  editor.uuid = item.uuid || '';
  editor.software = item.software || '';
  editor.name = item.name || '';
  editor.filePath = item.file_path || '';
  editor.version = item.version || 'v1';
  editor.format = item.format || 'json';
  editor.content = item.content || '{}';
}

async function saveConfig() {
  if (!editor.software.trim() || !editor.name.trim() || !editor.filePath.trim() || !editor.content.trim()) {
    status.value = '软件名、配置名、路径、内容不能为空';
    return;
  }

  saving.value = true;
  try {
    const data = await request('/software-config/save', {
      method: 'POST',
      body: JSON.stringify({
        uuid: editor.uuid,
        software: editor.software.trim(),
        name: editor.name.trim(),
        file_path: editor.filePath.trim(),
        version: editor.version.trim(),
        format: editor.format,
        content: editor.content,
      }),
    });
    editor.uuid = data.uuid;
    await request('/software-config/log', {
      method: 'POST',
      body: JSON.stringify({
        action: 'frontend_save',
        software: data.software,
        config_uuid: data.uuid,
        config_name: data.name,
        detail: 'saved from config sync settings',
      }),
    });
    await loadConfigs();
    status.value = '配置已保存';
  } catch (error) {
    status.value = `保存失败: ${error.message}`;
  } finally {
    saving.value = false;
  }
}

async function toggleSelect(item, selected) {
  try {
    await request('/software-config/select', {
      method: 'POST',
      body: JSON.stringify({ uuid: item.uuid, selected }),
    });
    item.selected = selected;
    await request('/software-config/log', {
      method: 'POST',
      body: JSON.stringify({
        action: 'frontend_select',
        software: item.software,
        config_uuid: item.uuid,
        config_name: item.name,
        detail: `selected=${selected}`,
      }),
    });
  } catch (error) {
    status.value = `选择失败: ${error.message}`;
  }
}

async function copyContent(item) {
  try {
    await navigator.clipboard.writeText(item.content || '');
    await request('/software-config/log', {
      method: 'POST',
      body: JSON.stringify({
        action: 'frontend_copy',
        software: item.software,
        config_uuid: item.uuid,
        config_name: item.name,
        detail: 'copied config content',
      }),
    });
    fillEditor(item);
    status.value = `已复制：${item.name}`;
  } catch (error) {
    status.value = `复制失败: ${error.message}`;
  }
}

async function applyItem(item) {
  try {
    await request('/software-config/activate', {
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
    await request('/software-config/log', {
      method: 'POST',
      body: JSON.stringify({
        action: 'frontend_apply',
        software: item.software,
        config_uuid: item.uuid,
        config_name: item.name,
        detail: `applied to ${item.file_path}`,
      }),
    });
    status.value = `已应用：${item.name}`;
    await loadConfigs();
  } catch (error) {
    status.value = `应用失败: ${error.message}`;
  }
}

async function removeConfig(item) {
  try {
    await request('/software-config/delete', {
      method: 'POST',
      body: JSON.stringify({ uuid: item.uuid }),
    });
    await loadConfigs();
    status.value = `已删除：${item.name}`;
  } catch (error) {
    status.value = `删除失败: ${error.message}`;
  }
}

async function compareWithCloud() {
  if (!cloud.cloudUrl.trim()) {
    status.value = '请先填写云端 URL';
    return;
  }
  comparing.value = true;
  try {
    const data = await request('/software-config/compare', {
      method: 'POST',
      body: JSON.stringify({
        cloud_url: cloud.cloudUrl.trim(),
        auth_token: cloud.authToken.trim(),
      }),
    });
    const map = new Map();
    (data.items || []).forEach((it) => {
      map.set(it.uuid, it.status);
    });
    items.value = items.value.map((it) => ({
      ...it,
      freshness_status: map.get(it.uuid) || 'local_only',
    }));
    status.value = `比较完成，共 ${data.items?.length || 0} 条`;
  } catch (error) {
    status.value = `比较失败: ${error.message}`;
  } finally {
    comparing.value = false;
  }
}

async function pushSelectedToCloud() {
  if (!cloud.cloudUrl.trim()) {
    status.value = '请先填写云端 URL';
    return;
  }
  pushing.value = true;
  try {
    const data = await request('/software-config/cloud/push-selected', {
      method: 'POST',
      body: JSON.stringify({
        cloud_url: cloud.cloudUrl.trim(),
        auth_token: cloud.authToken.trim(),
      }),
    });
    status.value = `已推送 ${data.synced_count || 0} 条到云端（${data.last_synced_at || ''}）`;
  } catch (error) {
    status.value = `推送失败: ${error.message}`;
  } finally {
    pushing.value = false;
  }
}

loadConfigs().catch((error) => {
  status.value = `初始化失败: ${error.message}`;
});
</script>
