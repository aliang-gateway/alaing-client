<template>
  <div
    id="dashboard-page"
    class="page-container content-section active flex flex-row flex-1 min-w-0 h-full overflow-hidden"
  >
    <aside
      class="w-80 lg:w-96 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800 flex flex-col h-full overflow-y-auto custom-scrollbar"
    >
      <div class="p-8 border-b border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/20">
        <div class="flex items-center gap-4">
          <div class="relative group">
            <div
              class="size-14 ring-2 ring-primary ring-offset-2 dark:ring-offset-slate-900 rounded-xl overflow-hidden shadow-sm transition-transform duration-300 group-hover:scale-105"
            >
              <img
                alt="User avatar"
                class="w-full h-full object-cover"
                src="https://lh3.googleusercontent.com/aida-public/AB6AXuBe4dmnKyGk_XawBt0uNW9E5uEFaIM3EsR5Fc_1I1RXhICJBndC8O_xmynJ0wCM6F7c2J8VdktjImaCqgRPBNrHLCHbvPsfHSyfuhmPaR6UJsJ9Mbj-M0g5dRpiXfW3ZoP3w2xuOq6hCD4sTfDTbJqMK9FHsJ3DU3DsBKF0SMuQITir97QPEC2hviK9f26s-g9gxmmITj4LSyILlnLedqYEwhvESK6n_tNiLuhY94eW6ZQkVkw4YzzgyFb18ZVhgN-8vEcW7oqsIeI"
              />
            </div>
            <div
              class="absolute -bottom-1 -right-1 size-4 bg-primary border-2 border-white dark:border-slate-900 rounded-full"
            ></div>
          </div>
          <div>
            <div class="flex items-center gap-1.5">
              <h2 class="font-bold text-slate-900 dark:text-white tracking-tight">Alex Rivera</h2>
            </div>
            <div class="flex items-center gap-2 mt-1">
              <span
                class="px-2 py-0.5 bg-primary/10 text-primary text-[10px] font-bold rounded uppercase tracking-wider"
              >
                Pro Developer
              </span>
            </div>
            <p class="text-slate-400 text-[11px] mt-1.5 font-medium">Valid until Dec 31, 2025</p>
          </div>
        </div>
      </div>
      <div class="p-8 flex flex-col items-center justify-center gap-6">
        <div class="relative">
          <div class="absolute -inset-4 rounded-full bg-primary/20 scale-110"></div>
          <button
            type="button"
            class="relative size-24 bg-primary text-white rounded-full flex items-center justify-center shadow-lg shadow-primary/40 hover:bg-primary/90 transition-all"
          >
            <span class="material-symbols-outlined text-4xl font-bold">power_settings_new</span>
          </button>
        </div>
        <div class="text-center">
          <p class="text-sm font-semibold text-slate-600 dark:text-slate-400">System Proxy Active</p>
          <p class="text-xs text-slate-400 mt-1">Protecting 12 applications</p>
        </div>
      </div>
      <div class="mx-6 p-4 bg-slate-50 dark:bg-slate-800/50 rounded border border-slate-100 dark:border-slate-700">
        <div class="flex justify-between items-start mb-3">
          <p class="text-xs font-bold text-slate-500 uppercase">Network Status</p>
          <span class="material-symbols-outlined text-primary text-sm">verified</span>
        </div>
        <div class="space-y-2">
          <div class="flex justify-between text-sm">
            <span class="text-slate-500">Mode:</span>
            <span class="font-medium text-slate-700 dark:text-slate-200">TUN/HTTP</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-slate-500">Protocol:</span>
            <span class="font-medium text-slate-700 dark:text-slate-200">SOCKS5</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-slate-500">Certificate:</span>
            <span class="font-medium text-primary">Installed</span>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-2 mt-4">
          <button
            type="button"
            id="sidebarCertDetailsBtn"
            class="px-3 py-1.5 text-xs font-bold border border-slate-200 dark:border-slate-600 rounded hover:bg-white dark:hover:bg-slate-700 transition-colors"
          >
            Details
          </button>
          <button
            type="button"
            id="sidebarCertReinstallBtn"
            class="px-3 py-1.5 text-xs font-bold bg-primary/10 text-primary rounded hover:bg-primary/20 transition-colors"
          >
            Re-install
          </button>
        </div>
      </div>
      <div class="mt-8 px-6 space-y-3">
        <p class="text-[10px] font-bold text-slate-400 uppercase tracking-widest px-1">Quick Tools</p>
        <div class="group relative">
          <button
            type="button"
            @click="emit('openQuickSetup')"
            class="w-full flex items-center justify-between px-4 py-2.5 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded text-sm font-medium hover:border-primary transition-colors"
          >
            <div class="flex items-center gap-3">
              <span class="material-symbols-outlined text-slate-400 text-lg">bolt</span>
              Quick Setup
            </div>
            <span class="material-symbols-outlined text-slate-400 text-sm">arrow_forward_ios</span>
          </button>
        </div>
        <button
          type="button"
          @click="openQuickChat"
          class="w-full flex items-center gap-3 px-4 py-2.5 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded text-sm font-medium hover:border-primary transition-colors"
        >
          <span class="material-symbols-outlined text-slate-400 text-lg">chat_bubble</span>
          Quick Chat
        </button>
        <button
          type="button"
          id="goToSettingsBtn"
          class="w-full flex items-center gap-3 px-4 py-2.5 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded text-sm font-medium hover:border-primary transition-colors"
        >
          <span class="material-symbols-outlined text-slate-400 text-lg">settings</span>
          More Settings
        </button>
      </div>
      <div class="mt-auto p-6 border-t border-slate-100 dark:border-slate-800 bg-slate-50/30 dark:bg-slate-800/10">
        <div class="flex justify-between items-center mb-3">
          <span class="text-xs font-bold text-slate-500 uppercase tracking-wider">Account Balance</span>
          <span class="text-lg font-bold text-slate-900 dark:text-white">$42.50</span>
        </div>
        <button
          type="button"
          class="w-full py-2 bg-slate-900 dark:bg-primary text-white text-xs font-bold rounded hover:opacity-90 transition-opacity"
        >
          Top Up Funds
        </button>
        <p class="text-[10px] text-slate-400 mt-2 text-center italic">Auto-renew enabled</p>
      </div>
    </aside>
    <main class="flex-1 min-w-0 flex flex-col h-full bg-background-light dark:bg-background-dark overflow-hidden">
      <header
        class="h-16 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between px-8 shrink-0"
      >
        <div class="flex items-center gap-4">
          <div class="size-8 bg-primary rounded-lg flex items-center justify-center text-white shadow-sm">
            <span class="material-symbols-outlined">api</span>
          </div>
          <h1 class="font-bold text-xl tracking-tight">
            ALiang
            <span class="text-primary font-medium">Gateway</span>
          </h1>
        </div>
        <div class="flex items-center gap-4">
          <div class="flex items-center gap-1">
            <button
              type="button"
              id="headerSettingsBtn"
              class="size-10 flex items-center justify-center rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 transition-colors"
            >
              <span class="material-symbols-outlined">settings</span>
            </button>
            <button
              type="button"
              class="size-10 flex items-center justify-center rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 transition-colors"
            >
              <span class="material-symbols-outlined">help_outline</span>
            </button>
          </div>
          <div class="h-8 w-px bg-slate-200 dark:bg-slate-800 mx-2"></div>
          <button
            type="button"
            class="px-4 py-1.5 bg-primary/10 text-primary text-xs font-bold rounded-lg hover:bg-primary/20 transition-colors"
          >
            Refresh Dashboard
          </button>
        </div>
      </header>
      <div class="flex-1 overflow-y-auto p-8 custom-scrollbar">
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
          <div
            class="bg-white dark:bg-slate-900 p-6 rounded border border-slate-200 dark:border-slate-800 shadow-sm"
          >
            <div class="flex justify-between items-center mb-6">
              <h3 class="font-bold text-slate-700 dark:text-slate-300">Traffic by Domain</h3>
              <button type="button" class="text-slate-400 hover:text-primary">
                <span class="material-symbols-outlined">more_horiz</span>
              </button>
            </div>
            <div class="flex items-center gap-8">
              <div class="relative size-40 flex items-center justify-center">
                <svg class="size-full transform -rotate-90" viewBox="0 0 36 36">
                  <title>Traffic distribution donut chart</title>
                  <circle cx="18" cy="18" fill="transparent" r="15.9" stroke="#e2e8f0" stroke-width="3"></circle>
                  <circle
                    cx="18"
                    cy="18"
                    fill="transparent"
                    r="15.9"
                    stroke="#21c45d"
                    stroke-dasharray="40 100"
                    stroke-dashoffset="0"
                    stroke-width="3"
                  ></circle>
                  <circle
                    cx="18"
                    cy="18"
                    fill="transparent"
                    r="15.9"
                    stroke="#10b981"
                    stroke-dasharray="25 100"
                    stroke-dashoffset="-40"
                    stroke-width="3"
                  ></circle>
                  <circle
                    cx="18"
                    cy="18"
                    fill="transparent"
                    r="15.9"
                    stroke="#34d399"
                    stroke-dasharray="15 100"
                    stroke-dashoffset="-65"
                    stroke-width="3"
                  ></circle>
                  <circle
                    cx="18"
                    cy="18"
                    fill="transparent"
                    r="15.9"
                    stroke="#6ee7b7"
                    stroke-dasharray="10 100"
                    stroke-dashoffset="-80"
                    stroke-width="3"
                  ></circle>
                  <circle
                    cx="18"
                    cy="18"
                    fill="transparent"
                    r="15.9"
                    stroke="#a7f3d0"
                    stroke-dasharray="10 100"
                    stroke-dashoffset="-90"
                    stroke-width="3"
                  ></circle>
                </svg>
                <div class="absolute inset-0 flex flex-col items-center justify-center">
                  <span class="text-2xl font-bold">100%</span>
                  <span class="text-[10px] text-slate-400 uppercase">Active</span>
                </div>
              </div>
              <div class="flex-1 space-y-2">
                <div class="flex items-center justify-between text-xs">
                  <div class="flex items-center gap-2">
                    <span class="size-2 bg-[#21c45d] rounded-full"></span>
                    Cursor
                  </div>
                  <span class="font-bold">40%</span>
                </div>
                <div class="flex items-center justify-between text-xs">
                  <div class="flex items-center gap-2">
                    <span class="size-2 bg-[#10b981] rounded-full"></span>
                    OpenAI
                  </div>
                  <span class="font-bold">25%</span>
                </div>
                <div class="flex items-center justify-between text-xs">
                  <div class="flex items-center gap-2">
                    <span class="size-2 bg-[#34d399] rounded-full"></span>
                    Claude
                  </div>
                  <span class="font-bold">15%</span>
                </div>
                <div class="flex items-center justify-between text-xs">
                  <div class="flex items-center gap-2">
                    <span class="size-2 bg-[#6ee7b7] rounded-full"></span>
                    ChatGPT
                  </div>
                  <span class="font-bold">10%</span>
                </div>
                <div class="flex items-center justify-between text-xs">
                  <div class="flex items-center gap-2">
                    <span class="size-2 bg-[#a7f3d0] rounded-full"></span>
                    Copilot
                  </div>
                  <span class="font-bold">10%</span>
                </div>
              </div>
            </div>
          </div>
          <div
            class="bg-white dark:bg-slate-900 p-6 rounded border border-slate-200 dark:border-slate-800 shadow-sm"
          >
            <div class="flex justify-between items-center mb-6">
              <h3 class="font-bold text-slate-700 dark:text-slate-300">Traffic Throughput (15s)</h3>
              <div class="flex gap-2">
                <span class="px-2 py-0.5 bg-primary/10 text-primary text-[10px] font-bold rounded">LIVE</span>
                <span
                  class="px-2 py-0.5 bg-slate-100 dark:bg-slate-800 text-slate-500 text-[10px] font-bold rounded"
                >
                  KB/s
                </span>
              </div>
            </div>
            <div class="h-40 w-full relative">
              <svg class="w-full h-full" preserveAspectRatio="none" viewBox="0 0 100 40">
                <title>Traffic throughput line chart</title>
                <path
                  d="M0 35 Q 10 32, 20 30 T 40 25 T 60 35 T 80 15 T 100 20"
                  fill="none"
                  stroke="#21c45d"
                  stroke-width="1.5"
                ></path>
                <path
                  d="M0 35 Q 10 32, 20 30 T 40 25 T 60 35 T 80 15 T 100 20 L 100 40 L 0 40 Z"
                  fill="url(#grad1)"
                  stroke="none"
                ></path>
                <defs>
                  <linearGradient id="grad1" x1="0%" x2="0%" y1="0%" y2="100%">
                    <stop offset="0%" style="stop-color: rgba(33, 196, 93, 0.2); stop-opacity: 1"></stop>
                    <stop offset="100%" style="stop-color: rgba(33, 196, 93, 0); stop-opacity: 1"></stop>
                  </linearGradient>
                </defs>
                <line
                  stroke="#e2e8f0"
                  stroke-dasharray="1"
                  stroke-width="0.1"
                  x1="0"
                  x2="100"
                  y1="35"
                  y2="35"
                ></line>
                <line
                  stroke="#e2e8f0"
                  stroke-dasharray="1"
                  stroke-width="0.1"
                  x1="0"
                  x2="100"
                  y1="25"
                  y2="25"
                ></line>
                <line
                  stroke="#e2e8f0"
                  stroke-dasharray="1"
                  stroke-width="0.1"
                  x1="0"
                  x2="100"
                  y1="15"
                  y2="15"
                ></line>
              </svg>
              <div class="absolute bottom-0 w-full flex justify-between text-[10px] text-slate-400 pt-2">
                <span>15s ago</span>
                <span>10s ago</span>
                <span>5s ago</span>
                <span>Now</span>
              </div>
            </div>
          </div>
        </div>
        <div class="bg-white dark:bg-slate-900 rounded border border-slate-200 dark:border-slate-800 shadow-sm">
          <div class="p-6 border-b border-slate-100 dark:border-slate-800 flex justify-between items-center">
            <div class="flex items-center gap-3">
              <h3 class="font-bold text-slate-700 dark:text-slate-300">Recent API Requests</h3>
              <span class="bg-slate-100 dark:bg-slate-800 text-slate-500 px-2 py-0.5 rounded text-xs font-medium">
                Last 50
              </span>
            </div>
            <button type="button" class="text-xs font-bold text-primary flex items-center gap-1 hover:underline">
              <span class="material-symbols-outlined text-sm">download</span>
              Export Logs
            </button>
          </div>
          <div class="px-6 py-3 border-b border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/30">
            <div class="flex flex-wrap items-center gap-2">
              <select
                v-model="requestFilter"
                class="h-9 rounded border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 px-3 text-xs text-slate-600 dark:text-slate-300"
              >
                <option value="all">过滤：全部</option>
                <option value="success">过滤：成功(2xx)</option>
                <option value="error">过滤：异常(非2xx)</option>
              </select>
              <input
                v-model="pathSearch"
                type="text"
                placeholder="按 path 搜索，例如 /v1/messages"
                class="h-9 min-w-[260px] flex-1 rounded border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 px-3 text-xs text-slate-700 dark:text-slate-300 placeholder:text-slate-400"
              />
            </div>
          </div>
          <div class="overflow-x-auto">
            <table class="w-full text-left">
              <thead
                class="bg-slate-50 dark:bg-slate-800/50 text-slate-400 text-[10px] font-bold uppercase tracking-wider"
              >
                <tr>
                  <th class="px-6 py-3">Method</th>
                  <th class="px-6 py-3">Status</th>
                  <th class="px-6 py-3">Domain</th>
                  <th class="px-6 py-3">Path</th>
                  <th class="px-6 py-3 text-right">上传流量</th>
                  <th class="px-6 py-3 text-right">下载流量</th>
                  <th class="px-6 py-3 text-right">上传Token</th>
                  <th class="px-6 py-3 text-right">下载Token</th>
                  <th class="px-6 py-3 text-right">首次响应时长</th>
                  <th class="px-6 py-3 text-right">全局响应时长</th>
                  <th class="px-6 py-3 text-right">Timestamp</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100 dark:divide-slate-800 text-sm">
                <tr
                  v-for="item in filteredRequestRows"
                  :key="`${item.timestamp}-${item.domain}-${item.path}`"
                  class="hover:bg-slate-50 dark:hover:bg-slate-800/30"
                >
                  <td class="px-6 py-4 font-bold text-slate-700 dark:text-slate-300">{{ item.method }}</td>
                  <td class="px-6 py-4">
                    <span class="inline-flex items-center gap-1 font-medium" :class="item.statusClass">
                      <span class="size-1.5 rounded-full" :class="item.statusDotClass"></span>
                      {{ item.status }}
                    </span>
                  </td>
                  <td class="px-6 py-4 text-slate-500">{{ item.domain }}</td>
                  <td class="px-6 py-4 text-slate-500"><code>{{ item.path }}</code></td>
                  <td class="px-6 py-4 text-right text-slate-500 tabular-nums">{{ formatBytes(item.uploadBytes) }}</td>
                  <td class="px-6 py-4 text-right text-slate-500 tabular-nums">{{ formatBytes(item.downloadBytes) }}</td>
                  <td class="px-6 py-4 text-right text-slate-500 tabular-nums">{{ item.uploadTokens }}</td>
                  <td class="px-6 py-4 text-right text-slate-500 tabular-nums">{{ item.downloadTokens }}</td>
                  <td class="px-6 py-4 text-right text-slate-500 tabular-nums">{{ formatDuration(item.firstResponseMs) }}</td>
                  <td class="px-6 py-4 text-right text-slate-500 tabular-nums">{{ formatDuration(item.globalResponseMs) }}</td>
                  <td class="px-6 py-4 text-right text-slate-400 tabular-nums">{{ item.timestamp }}</td>
                </tr>
                <tr v-if="filteredRequestRows.length === 0">
                  <td colspan="11" class="px-6 py-6 text-center text-xs text-slate-400">没有匹配的请求记录</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div
            class="p-4 bg-slate-50 dark:bg-slate-800/50 flex justify-center border-t border-slate-100 dark:border-slate-800"
          >
            <button
              type="button"
              class="text-xs font-bold text-slate-500 hover:text-slate-700 uppercase tracking-widest"
            >
              Load More Entries
            </button>
          </div>
        </div>
      </div>
    </main>

    <div
      v-if="isQuickChatOpen"
      class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/50 p-4"
      @click.self="closeQuickChat"
    >
      <div class="w-full max-w-2xl overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl dark:border-slate-700 dark:bg-slate-900">
        <div class="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-slate-700">
          <h3 class="text-base font-semibold text-slate-900 dark:text-slate-100">Quick Chat</h3>
          <button
            type="button"
            class="rounded p-1 text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-slate-800 dark:hover:text-slate-200"
            @click="closeQuickChat"
          >
            <span class="material-symbols-outlined text-lg">close</span>
          </button>
        </div>

        <div class="h-[420px] overflow-y-auto bg-slate-50 p-4 dark:bg-slate-800/40">
          <div v-if="quickChatMessages.length === 0" class="text-center text-sm text-slate-400">开始和 AI 对话吧</div>
          <div v-for="(item, index) in quickChatMessages" :key="`${item.role}-${index}`" class="mb-3">
            <div class="mb-1 text-xs text-slate-400">{{ item.role === 'user' ? '我' : 'AI' }}</div>
            <div
              class="inline-block max-w-[90%] rounded-lg px-3 py-2 text-sm"
              :class="item.role === 'user'
                ? 'ml-auto block bg-primary text-white'
                : 'bg-white text-slate-700 dark:bg-slate-700 dark:text-slate-100'"
            >
              {{ item.content }}
            </div>
          </div>
        </div>

        <div class="border-t border-slate-200 p-4 dark:border-slate-700">
          <div class="flex items-center gap-2">
            <input
              v-model="quickChatInput"
              type="text"
              placeholder="输入消息..."
              class="h-10 flex-1 rounded border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none transition focus:border-primary dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
              :disabled="isQuickChatSending"
              @keydown.enter.prevent="sendQuickChat"
            />
            <button
              type="button"
              class="h-10 rounded bg-primary px-4 text-sm font-semibold text-white transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="isQuickChatSending"
              @click="sendQuickChat"
            >
              {{ isQuickChatSending ? '发送中...' : '发送' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue';

const emit = defineEmits(['openQuickSetup']);

const requestFilter = ref('all');
const pathSearch = ref('');
const isQuickChatOpen = ref(false);
const quickChatInput = ref('');
const isQuickChatSending = ref(false);
const quickChatMessages = ref([]);

function openQuickChat() {
  isQuickChatOpen.value = true;
}

function closeQuickChat() {
  isQuickChatOpen.value = false;
}

async function sendQuickChat() {
  const text = quickChatInput.value.trim();
  if (!text || isQuickChatSending.value) {
    return;
  }

  quickChatMessages.value.push({ role: 'user', content: text });
  quickChatInput.value = '';
  isQuickChatSending.value = true;

  try {
    const messagePayload = quickChatMessages.value.slice(-20).map(item => ({
      role: item.role,
      content: item.content
    }));

    const response = await fetch('/api/chat/completions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        message: text,
        history: messagePayload
      })
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const data = await response.json();
    const assistantReply = data?.data?.reply;

    if (!assistantReply || !String(assistantReply).trim()) {
      throw new Error('Empty AI response');
    }

    quickChatMessages.value.push({ role: 'assistant', content: String(assistantReply).trim() });
  } catch (error) {
    quickChatMessages.value.push({
      role: 'assistant',
      content: '暂时无法连接 AI 服务，请稍后重试。'
    });
  } finally {
    isQuickChatSending.value = false;
  }
}

const requestRows = [
  {
    method: 'POST',
    status: '200 OK',
    domain: 'api.openai.com',
    path: '/v1/chat/completions',
    uploadBytes: 16832,
    downloadBytes: 48210,
    uploadTokens: 1321,
    downloadTokens: 2640,
    firstResponseMs: 186,
    globalResponseMs: 742,
    timestamp: '14:20:45.002',
    statusClass: 'text-primary',
    statusDotClass: 'bg-primary'
  },
  {
    method: 'GET',
    status: '200 OK',
    domain: 'api.anthropic.com',
    path: '/v1/messages',
    uploadBytes: 5240,
    downloadBytes: 21904,
    uploadTokens: 804,
    downloadTokens: 1530,
    firstResponseMs: 142,
    globalResponseMs: 605,
    timestamp: '14:20:41.285',
    statusClass: 'text-primary',
    statusDotClass: 'bg-primary'
  },
  {
    method: 'POST',
    status: '429 Limit',
    domain: 'api.cursor.sh',
    path: '/v1/streaming',
    uploadBytes: 12280,
    downloadBytes: 4020,
    uploadTokens: 1160,
    downloadTokens: 200,
    firstResponseMs: 420,
    globalResponseMs: 980,
    timestamp: '14:20:38.910',
    statusClass: 'text-amber-500',
    statusDotClass: 'bg-amber-500'
  },
  {
    method: 'CONNECT',
    status: '503 Timeout',
    domain: 'tunnel.socks5.service',
    path: '/connect',
    uploadBytes: 820,
    downloadBytes: 260,
    uploadTokens: 0,
    downloadTokens: 0,
    firstResponseMs: 1200,
    globalResponseMs: 3000,
    timestamp: '14:20:25.111',
    statusClass: 'text-rose-500',
    statusDotClass: 'bg-rose-500'
  }
];

const filteredRequestRows = computed(() => {
  const pathKeyword = pathSearch.value.trim().toLowerCase();
  return requestRows.filter(item => {
    const isSuccess = isSuccessStatus(item.status);
    if (requestFilter.value === 'success' && !isSuccess) {
      return false;
    }
    if (requestFilter.value === 'error' && isSuccess) {
      return false;
    }
    if (!pathKeyword) {
      return true;
    }
    return item.path.toLowerCase().includes(pathKeyword);
  });
});

function isSuccessStatus(statusText) {
  const match = String(statusText || '').match(/(\d{3})/);
  if (!match) {
    return false;
  }
  const code = Number(match[1]);
  return code >= 200 && code < 300;
}

function formatBytes(bytes) {
  const value = Number(bytes || 0);
  if (value < 1024) {
    return `${value} B`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(2)} KB`;
  }
  return `${(value / (1024 * 1024)).toFixed(2)} MB`;
}

function formatDuration(ms) {
  return `${Number(ms || 0)} ms`;
}
</script>
