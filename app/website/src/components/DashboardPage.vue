<template>
  <div
    v-if="currentPage === 'dashboard'"
    id="dashboard-page"
    class="page-container content-section active flex flex-row flex-1 min-w-0 h-full overflow-hidden"
  >
    <aside
      class="w-80 lg:w-96 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800 flex flex-col h-full overflow-y-auto custom-scrollbar"
    >
      <div
        class="p-8 border-b border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/20 transition-colors"
        :class="!isAuthenticated ? 'cursor-pointer hover:bg-primary/5 dark:hover:bg-primary/10' : ''"
        :role="!isAuthenticated ? 'button' : undefined"
        :tabindex="!isAuthenticated ? 0 : undefined"
        @click="handleAccountCardClick"
        @keydown.enter.prevent="handleAccountCardClick"
        @keydown.space.prevent="handleAccountCardClick"
      >
        <div class="flex items-center gap-4">
          <div class="relative group">
            <div
              class="size-14 ring-2 ring-primary ring-offset-2 dark:ring-offset-slate-900 rounded-xl overflow-hidden shadow-sm transition-transform duration-300 group-hover:scale-105 flex items-center justify-center bg-gradient-to-br from-primary to-emerald-500 text-white"
            >
              <span class="text-lg font-bold uppercase tracking-[0.2em] ml-[0.2em]">{{ userAvatarText }}</span>
            </div>
            <div
              class="absolute -bottom-1 -right-1 size-4 bg-primary border-2 border-white dark:border-slate-900 rounded-full"
            ></div>
          </div>
          <div>
            <div class="flex items-center gap-1.5">
              <h2 class="font-bold text-slate-900 dark:text-white tracking-tight">{{ userDisplayName }}</h2>
            </div>
            <div class="flex items-center gap-2 mt-1">
              <span
                class="px-2 py-0.5 bg-primary/10 text-primary text-[10px] font-bold rounded uppercase tracking-wider"
              >
                {{ planLabel }}
              </span>
            </div>
            <p class="text-slate-400 text-[11px] mt-1.5 font-medium">{{ accountSubtitle }}</p>
          </div>
        </div>
        <div
          class="mt-4 rounded-lg border px-3 py-2 text-xs"
          :class="isAuthenticated ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-300' : 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-300'"
        >
          <div class="flex items-center justify-between gap-3">
            <span>{{ authNotice }}</span>
            <button
              v-if="!isAuthenticated"
              type="button"
              class="shrink-0 rounded-md bg-white/80 px-2.5 py-1 text-[11px] font-semibold text-amber-700 transition hover:bg-white dark:bg-slate-900/70 dark:text-amber-200 dark:hover:bg-slate-900"
              @click.stop="openLoginModal"
            >
              立即登录
            </button>
          </div>
        </div>
      </div>
      <div class="p-8 flex flex-col items-center justify-center gap-6">
        <div class="relative">
          <div class="absolute -inset-4 rounded-full scale-110 transition-all" :class="powerButtonHaloClass"></div>
          <button
            type="button"
            :disabled="powerButtonDisabled"
            :title="powerButtonTitle"
            class="relative size-24 rounded-full border flex items-center justify-center transition-all disabled:cursor-not-allowed"
            :class="powerButtonClass"
            @click="toggleProxyPower"
          >
            <span
              v-if="runActionLoading"
              class="inline-block size-8 border-4 border-white/30 border-t-white rounded-full animate-spin"
            ></span>
            <span v-else class="material-symbols-outlined text-4xl font-bold">power_settings_new</span>
          </button>
        </div>
        <div class="text-center">
          <p class="text-sm font-semibold text-slate-600 dark:text-slate-400">{{ proxyStatusTitle }}</p>
          <p class="text-xs text-slate-400 mt-1">{{ runActionLoading ? powerButtonBusyText : proxyStatusSubtitle }}</p>
        </div>
      </div>
      <div class="mx-6 p-4 bg-slate-50 dark:bg-slate-800/50 rounded border border-slate-100 dark:border-slate-700">
        <div class="flex justify-between items-start mb-3">
          <p class="text-xs font-bold text-slate-500 uppercase">Network Status</p>
          <span
            v-if="certLoading"
            class="inline-block size-4 border-2 border-slate-200 border-t-slate-400 rounded-full animate-spin"
          ></span>
          <span
            v-else
            class="material-symbols-outlined text-sm"
            :class="networkStatusIconClass"
          >{{ networkStatusIcon }}</span>
        </div>
          <div class="space-y-2">
            <div class="flex justify-between text-sm">
              <span class="text-slate-500">Mode:</span>
              <span class="font-medium text-slate-700 dark:text-slate-200">{{ runModeLabel }}</span>
            </div>
          <div class="flex justify-between text-sm">
            <span class="text-slate-500">Protocol:</span>
            <span class="font-medium text-slate-700 dark:text-slate-200">SOCKS5</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-slate-500">Certificate:</span>
            <span class="font-medium" :class="certBadgeClass">{{ certBadgeText }}</span>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-2 mt-4">
          <button
            type="button"
            class="px-3 py-1.5 text-xs font-bold border border-slate-200 dark:border-slate-600 rounded hover:bg-white dark:hover:bg-slate-700 transition-colors"
            @click="openCertModal"
          >
            Details
          </button>
          <button
            type="button"
            class="px-3 py-1.5 text-xs font-bold bg-primary/10 text-primary rounded hover:bg-primary/20 transition-colors"
            @click="handleReinstall"
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
            :disabled="!isAuthenticated"
            @click="openQuickSetup"
            class="w-full flex items-center gap-3 px-4 py-2.5 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded text-sm font-medium hover:border-primary transition-colors"
            :class="!isAuthenticated ? 'cursor-not-allowed opacity-60 hover:border-slate-200 dark:hover:border-slate-700' : ''"
          >
            <span class="material-symbols-outlined text-slate-400 text-lg">bolt</span>
            Quick Setup
          </button>
        </div>
        <button
          type="button"
          :disabled="!isAuthenticated"
          @click="openQuickChat"
          class="w-full flex items-center gap-3 px-4 py-2.5 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded text-sm font-medium hover:border-primary transition-colors"
          :class="!isAuthenticated ? 'cursor-not-allowed opacity-60 hover:border-slate-200 dark:hover:border-slate-700' : ''"
        >
          <span class="material-symbols-outlined text-slate-400 text-lg">chat_bubble</span>
          Quick Chat
        </button>
        <button
          type="button"
          class="w-full flex items-center gap-3 px-4 py-2.5 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded text-sm font-medium hover:border-primary transition-colors"
          @click="handleShowSettings"
        >
          <span class="material-symbols-outlined text-slate-400 text-lg">settings</span>
          More Settings
        </button>
      </div>
      <div class="mt-auto p-6 border-t border-slate-100 dark:border-slate-800 bg-slate-50/30 dark:bg-slate-800/10">
        <div class="flex justify-between items-center mb-3">
          <span class="text-xs font-bold text-slate-500 uppercase tracking-wider">Account Balance</span>
          <span class="text-lg font-bold text-slate-900 dark:text-white">{{ accountBalanceText }}</span>
        </div>
        <button
          type="button"
          class="w-full py-2 bg-slate-900 dark:bg-primary text-white text-xs font-bold rounded hover:opacity-90 transition-opacity"
          @click="handleTopUp"
        >
          Top Up Funds
        </button>
        <p class="text-[10px] text-slate-400 mt-2 text-center italic">{{ accountBalanceHint }}</p>
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
              class="size-10 flex items-center justify-center rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 transition-colors"
            >
              <span class="material-symbols-outlined">help_outline</span>
            </button>
          </div>
          <div class="h-8 w-px bg-slate-200 dark:bg-slate-800 mx-2"></div>
          <button
            type="button"
            class="px-4 py-1.5 bg-primary/10 text-primary text-xs font-bold rounded-lg hover:bg-primary/20 transition-colors"
            @click="loadDashboardUsageData"
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
              <h3 class="font-bold text-slate-700 dark:text-slate-300">Model Usage Distribution</h3>
              <button type="button" class="text-slate-400 hover:text-primary">
                <span class="material-symbols-outlined">more_horiz</span>
              </button>
            </div>
            <div v-if="dashboardError" class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300">
              {{ dashboardError }}
            </div>
            <div v-else-if="dashboardLoading" class="rounded-lg border border-dashed border-slate-300 bg-slate-50 px-4 py-10 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-400">
              Loading model usage distribution...
            </div>
            <div v-else-if="!modelDistribution.length" class="rounded-lg border border-dashed border-slate-300 bg-slate-50 px-4 py-10 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-400">
              No model usage data is available yet.
            </div>
            <div v-else class="flex items-center gap-8">
              <div class="relative size-40 flex items-center justify-center">
                <svg class="size-full transform -rotate-90" viewBox="0 0 36 36">
                  <title>Model usage distribution donut chart</title>
                  <circle cx="18" cy="18" fill="transparent" r="15.9" stroke="#e2e8f0" stroke-width="3"></circle>
                  <circle
                    v-for="item in modelDistribution"
                    :key="item.label"
                    cx="18"
                    cy="18"
                    fill="transparent"
                    r="15.9"
                    :stroke="item.color"
                    :stroke-dasharray="item.dashArray"
                    :stroke-dashoffset="item.dashOffset"
                    stroke-width="3"
                  ></circle>
                </svg>
                <div class="absolute inset-0 flex flex-col items-center justify-center">
                  <span class="text-2xl font-bold">{{ formatCount(totalRequestCount) }}</span>
                  <span class="text-[10px] text-slate-400 uppercase">Requests</span>
                </div>
              </div>
              <div class="flex-1 space-y-2">
                <div
                  v-for="item in modelDistribution"
                  :key="`${item.label}-legend`"
                  class="flex items-center justify-between text-xs"
                >
                  <div class="flex items-center gap-2">
                    <span class="size-2 rounded-full" :style="{ backgroundColor: item.color }"></span>
                    <span class="truncate">{{ item.label }}</span>
                  </div>
                  <span class="font-bold">{{ item.percent.toFixed(0) }}%</span>
                </div>
              </div>
            </div>
          </div>
          <div
            class="bg-white dark:bg-slate-900 p-6 rounded border border-slate-200 dark:border-slate-800 shadow-sm"
          >
            <div class="flex justify-between items-center mb-6">
              <h3 class="font-bold text-slate-700 dark:text-slate-300">Usage Trend</h3>
              <div class="flex gap-2">
                <span class="px-2 py-0.5 bg-primary/10 text-primary text-[10px] font-bold rounded">LIVE</span>
                <span
                  class="px-2 py-0.5 bg-slate-100 dark:bg-slate-800 text-slate-500 text-[10px] font-bold rounded"
                >
                  Requests
                </span>
              </div>
            </div>
            <div v-if="dashboardLoading" class="rounded-lg border border-dashed border-slate-300 bg-slate-50 px-4 py-10 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-400">
              Loading usage trend...
            </div>
            <div v-else-if="!trendPoints.length" class="rounded-lg border border-dashed border-slate-300 bg-slate-50 px-4 py-10 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-400">
              No trend data is available yet.
            </div>
            <div v-else class="h-40 w-full relative">
              <svg class="w-full h-full" preserveAspectRatio="none" viewBox="0 0 100 40">
                <title>Usage trend line chart</title>
                <path
                  :d="trendPath"
                  fill="none"
                  stroke="#21c45d"
                  stroke-width="1.5"
                ></path>
                <path
                  :d="trendAreaPath"
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
                <span v-for="point in trendPoints" :key="point.label">{{ point.label }}</span>
              </div>
              <div class="absolute left-0 top-0 rounded bg-white/80 px-2 py-1 text-[10px] font-semibold text-slate-500 dark:bg-slate-900/80 dark:text-slate-300">
                {{ trendSummaryText }}
              </div>
            </div>
          </div>
        </div>
        <div class="bg-white dark:bg-slate-900 rounded border border-slate-200 dark:border-slate-800 shadow-sm">
          <div class="p-6 border-b border-slate-100 dark:border-slate-800 flex justify-between items-center">
            <div class="flex items-center gap-3">
              <h3 class="font-bold text-slate-700 dark:text-slate-300">Recent Usage Records</h3>
              <span class="bg-slate-100 dark:bg-slate-800 text-slate-500 px-2 py-0.5 rounded text-xs font-medium">
                10 / page, max 20 pages
              </span>
            </div>
            <div class="flex items-center gap-4">
              <button
                type="button"
                class="text-xs font-bold text-slate-500 flex items-center gap-1 hover:text-slate-700 dark:text-slate-300 dark:hover:text-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                :disabled="dashboardLoading"
                @click="refreshUsageRecords"
              >
                <span class="material-symbols-outlined text-sm">refresh</span>
                Refresh
              </button>
              <button type="button" class="text-xs font-bold text-primary flex items-center gap-1 hover:underline" @click="exportUsageRecords">
                <span class="material-symbols-outlined text-sm">download</span>
                Export Data
              </button>
            </div>
          </div>
          <div class="px-6 py-3 border-b border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/30">
            <div class="flex flex-wrap items-center gap-2">
              <select
                v-model="requestFilter"
                class="h-9 rounded border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 px-3 text-xs text-slate-600 dark:text-slate-300"
                @change="applyUsageFilters"
              >
                <option value="all">过滤：全部</option>
                <option value="chat">过滤：Chat</option>
                <option value="stream">过滤：流式</option>
                <option value="image">过滤：Image</option>
              </select>
              <input
                v-model="pathSearch"
                type="text"
                placeholder="按 endpoint / model / key 搜索"
                class="h-9 min-w-[260px] flex-1 rounded border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 px-3 text-xs text-slate-700 dark:text-slate-300 placeholder:text-slate-400"
                @keydown.enter.prevent="applyUsageFilters"
              />
              <button
                type="button"
                class="h-9 rounded bg-slate-900 px-3 text-xs font-semibold text-white transition hover:bg-slate-700 dark:bg-slate-100 dark:text-slate-900 dark:hover:bg-slate-200"
                :disabled="dashboardLoading"
                @click="applyUsageFilters"
              >
                Apply Filters
              </button>
              <button
                type="button"
                class="h-9 rounded border border-slate-200 bg-white px-3 text-xs font-semibold text-slate-600 transition hover:border-slate-300 hover:text-slate-800 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300 dark:hover:border-slate-600 dark:hover:text-slate-100"
                :disabled="dashboardLoading || (requestFilter === 'all' && !pathSearch.trim())"
                @click="resetUsageFilters"
              >
                Reset
              </button>
            </div>
          </div>
          <div class="overflow-x-auto">
            <table class="w-full text-left">
              <thead
                class="bg-slate-50 dark:bg-slate-800/50 text-slate-400 text-[10px] font-bold uppercase tracking-wider"
              >
                <tr>
                  <th class="px-6 py-3">Type</th>
                  <th class="px-6 py-3">Model</th>
                  <th class="px-6 py-3">Endpoint</th>
                  <th class="px-6 py-3">API Key</th>
                  <th class="px-6 py-3">Group</th>
                  <th class="px-6 py-3 text-right">Input Tokens</th>
                  <th class="px-6 py-3 text-right">Output Tokens</th>
                  <th class="px-6 py-3 text-right">Total Tokens</th>
                  <th class="px-6 py-3 text-right">Actual Cost</th>
                  <th class="px-6 py-3 text-right">Duration</th>
                  <th class="px-6 py-3 text-right">Timestamp</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100 dark:divide-slate-800 text-sm">
                <tr
                  v-for="item in filteredRequestRows"
                  :key="`${item.id}-${item.createdAt}-${item.endpoint}`"
                  class="hover:bg-slate-50 dark:hover:bg-slate-800/30"
                >
                  <td class="px-6 py-4 font-bold text-slate-700 dark:text-slate-300">{{ item.requestType }}</td>
                  <td class="px-6 py-4 text-slate-500">{{ item.model }}</td>
                  <td class="px-6 py-4 text-slate-500"><code>{{ item.endpoint }}</code></td>
                  <td class="px-6 py-4 text-slate-500">{{ item.apiKeyName }}</td>
                  <td class="px-6 py-4 text-slate-500">{{ item.groupName }}</td>
                  <td class="px-6 py-4 text-right text-slate-500 tabular-nums">{{ formatCount(item.inputTokens) }}</td>
                  <td class="px-6 py-4 text-right text-slate-500 tabular-nums">{{ formatCount(item.outputTokens) }}</td>
                  <td class="px-6 py-4 text-right text-slate-500 tabular-nums">{{ formatCount(item.totalTokens) }}</td>
                  <td class="px-6 py-4 text-right text-slate-500 tabular-nums">{{ formatCurrency(item.actualCost) }}</td>
                  <td class="px-6 py-4 text-right text-slate-500 tabular-nums">{{ formatDuration(item.durationMs) }}</td>
                  <td class="px-6 py-4 text-right text-slate-400 tabular-nums">{{ formatDateTime(item.createdAt) }}</td>
                </tr>
                <tr v-if="filteredRequestRows.length === 0">
                  <td colspan="11" class="px-6 py-6 text-center text-xs text-slate-400">没有匹配的使用记录</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div
            class="p-4 bg-slate-50 dark:bg-slate-800/50 flex flex-col gap-3 border-t border-slate-100 dark:border-slate-800 sm:flex-row sm:items-center sm:justify-between"
          >
            <p class="text-xs text-slate-500 dark:text-slate-400">
              Showing {{ filteredRequestRows.length }} of {{ formatCount(usageTotal) }} records on page {{ usagePage }}.
            </p>
            <div class="flex items-center justify-end gap-2">
              <button
                type="button"
                class="h-9 rounded border border-slate-200 bg-white px-3 text-xs font-semibold text-slate-600 transition hover:border-slate-300 hover:text-slate-800 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300 dark:hover:border-slate-600 dark:hover:text-slate-100"
                :disabled="dashboardLoading || usagePage <= 1"
                @click="changeUsagePage(usagePage - 1)"
              >
                Prev
              </button>
              <div class="min-w-[120px] text-center text-xs font-semibold text-slate-500 dark:text-slate-300">
                {{ usagePage }} / {{ usageTotalPages }}
              </div>
              <button
                type="button"
                class="h-9 rounded border border-slate-200 bg-white px-3 text-xs font-semibold text-slate-600 transition hover:border-slate-300 hover:text-slate-800 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300 dark:hover:border-slate-600 dark:hover:text-slate-100"
                :disabled="dashboardLoading || usagePage >= usageTotalPages"
                @click="changeUsagePage(usagePage + 1)"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      </div>
    </main>

    <div
      v-if="isLoginModalOpen"
      class="fixed inset-0 z-[1000] flex items-center justify-center bg-slate-950/60 p-4 backdrop-blur-sm"
      @click.self="closeLoginModal"
    >
      <div class="w-full max-w-md overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl dark:border-slate-700 dark:bg-slate-900">
        <div class="border-b border-slate-200 bg-slate-50/80 px-5 py-4 dark:border-slate-700 dark:bg-slate-800/60">
          <div class="flex items-start justify-between gap-4">
            <div>
              <p class="text-xs font-bold uppercase tracking-[0.2em] text-primary">Account Access</p>
              <h3 class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">登录后继续使用 Dashboard</h3>
              <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">
                登录后可解锁代理控制、Quick Setup、Quick Chat 与账户相关功能。
              </p>
            </div>
            <button
              type="button"
              class="rounded-lg p-1.5 text-slate-500 transition hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-slate-800 dark:hover:text-slate-200"
              :disabled="loginPending"
              @click="closeLoginModal"
            >
              <span class="material-symbols-outlined text-lg">close</span>
            </button>
          </div>
        </div>

        <form class="space-y-4 p-5" @submit.prevent="submitLogin">
          <div class="rounded-xl border border-dashed border-amber-200 bg-amber-50/70 p-4 text-xs text-amber-700 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
            当前未登录。输入账户信息后，弹窗会在登录成功后自动关闭。
          </div>

          <div>
            <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-500">Email</label>
            <input
              v-model.trim="loginEmail"
              type="email"
              autocomplete="username"
              class="h-11 w-full rounded-lg border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none transition focus:border-primary dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
              placeholder="name@example.com"
              :disabled="loginPending"
            />
          </div>

          <div>
            <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-500">Password</label>
            <input
              v-model="loginPassword"
              type="password"
              autocomplete="current-password"
              class="h-11 w-full rounded-lg border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none transition focus:border-primary dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
              placeholder="Enter your password"
              :disabled="loginPending"
            />
          </div>

          <p v-if="loginError" class="text-xs text-rose-500">{{ loginError }}</p>

          <div class="flex gap-3 pt-1">
            <button
              type="button"
              class="inline-flex h-11 flex-1 items-center justify-center rounded-lg border border-slate-200 px-4 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:text-slate-100 dark:hover:bg-slate-800"
              :disabled="loginPending"
              @click="closeLoginModal"
            >
              取消
            </button>
            <button
              type="submit"
              class="inline-flex h-11 flex-1 items-center justify-center rounded-lg bg-primary px-4 text-sm font-semibold text-white transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="loginPending"
            >
              {{ loginPending ? '登录中...' : '登录' }}
            </button>
          </div>
        </form>
      </div>
    </div>

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
import { computed, ref, onMounted, onUnmounted, watch } from 'vue';
import { useCertStatus } from '../composables/useCertStatus';
import { useNavigation } from '../composables/useNavigation';
import { useAuthStore } from '../stores/auth';
import { getUserCenterProfile } from '../services/userCenterApi';
import { extractUsagePagination, extractUsageRecords } from '../utils/dashboardData';
import {
  getDashboardModels,
  getDashboardStats,
  getDashboardTrend,
  getDashboardUsageRecords
} from '../services/dashboardApi';

const { certStatus, loading: certLoading, startPolling, stopPolling } = useCertStatus();
const { currentPage, showSettings } = useNavigation();
const { isAuthenticated, user, userDisplayName, planLabel, authNotice, loginPending, loginError, loginWithPassword } = useAuthStore();

const emit = defineEmits(['openQuickSetup', 'openCertModal', 'startCertReinstall']);

const requestFilter = ref('all');
const pathSearch = ref('');
const isLoginModalOpen = ref(false);
const loginEmail = ref('');
const loginPassword = ref('');
const isQuickChatOpen = ref(false);
const quickChatInput = ref('');
const isQuickChatSending = ref(false);
const quickChatMessages = ref([]);
const runMode = ref('unknown');
const runIsRunning = ref(false);
const runStatus = ref('');
const runDescription = ref('');
const runSyncError = ref('');
const runActionLoading = ref(false);
const runActionMessage = ref('');
const startupStatus = ref('UNKNOWN');
const accountBalance = ref(null);
const dashboardLoading = ref(false);
const dashboardError = ref('');
const dashboardStats = ref({});
const dashboardModels = ref([]);
const dashboardTrend = ref([]);
const dashboardUsageRecords = ref([]);
const usagePage = ref(1);
const usagePageSize = ref(10);
const usageTotal = ref(0);
const usageTotalPages = ref(1);
const usageMaxPages = 20;
const appliedRequestFilter = ref('all');
let runStatusTimer = null;

const accountSubtitle = computed(() => {
  if (!isAuthenticated.value) {
    return 'Log in to unlock proxy controls and account-linked usage.';
  }
  if (user.value?.endTime) {
    return `Valid until ${user.value.endTime}`;
  }
  return 'Authenticated session active';
});

const userAvatarText = computed(() => {
  const emailPrefix = typeof user.value?.email === 'string' ? user.value.email.trim().slice(0, 2) : '';
  if (emailPrefix) {
    return emailPrefix;
  }

  const usernamePrefix = typeof user.value?.username === 'string' ? user.value.username.trim().slice(0, 2) : '';
  if (usernamePrefix) {
    return usernamePrefix;
  }

  return 'GU';
});

const accountBalanceText = computed(() => {
  const value = Number(accountBalance.value);
  if (!Number.isFinite(value)) {
    return isAuthenticated.value ? '--' : '$0.00';
  }
  return `$${value.toFixed(2)}`;
});

const accountBalanceHint = computed(() => {
  if (!isAuthenticated.value) {
    return 'Log in to sync your current balance.';
  }
  return accountBalance.value === null ? 'Syncing balance...' : 'Click to top up your account.';
});

const totalRequestCount = computed(() => Number(dashboardStats.value?.total_requests || 0));

const modelDistribution = computed(() => {
  const items = Array.isArray(dashboardModels.value) ? dashboardModels.value : [];
  const total = items.reduce((sum, item) => sum + Number(item?.requests || 0), 0);
  if (total <= 0) {
    return [];
  }

  const palette = ['#21c45d', '#10b981', '#34d399', '#6ee7b7', '#a7f3d0'];
  const topItems = items
    .slice()
    .sort((left, right) => Number(right?.requests || 0) - Number(left?.requests || 0))
    .slice(0, 5)
    .map((item, index) => ({
      label: item?.model || `Model ${index + 1}`,
      requests: Number(item?.requests || 0),
      percent: (Number(item?.requests || 0) / total) * 100,
      color: palette[index] || palette[palette.length - 1]
    }));

  let cumulative = 0;
  return topItems.map((item) => {
    const segment = {
      ...item,
      dashArray: `${item.percent} 100`,
      dashOffset: -cumulative
    };
    cumulative += item.percent;
    return segment;
  });
});

const trendPoints = computed(() => {
  const items = Array.isArray(dashboardTrend.value) ? dashboardTrend.value : [];
  if (!items.length) {
    return [];
  }
  const maxRequests = Math.max(...items.map((item) => Number(item?.requests || 0)), 1);
  const step = items.length > 1 ? 100 / (items.length - 1) : 100;
  return items.map((item, index) => ({
    x: Number((index * step).toFixed(2)),
    y: Number((35 - ((Number(item?.requests || 0) / maxRequests) * 25)).toFixed(2)),
    label: formatTrendLabel(item?.date),
    requests: Number(item?.requests || 0),
    actualCost: Number(item?.actual_cost || item?.cost || 0)
  }));
});

const trendPath = computed(() => {
  if (!trendPoints.value.length) {
    return '';
  }
  return trendPoints.value
    .map((point, index) => `${index === 0 ? 'M' : 'L'}${point.x} ${point.y}`)
    .join(' ');
});

const trendAreaPath = computed(() => {
  if (!trendPoints.value.length) {
    return '';
  }
  const linePath = trendPath.value;
  return `${linePath} L 100 40 L 0 40 Z`;
});

const trendSummaryText = computed(() => {
  if (!trendPoints.value.length) {
    return 'No recent request trend yet.';
  }
  const latestPoint = trendPoints.value[trendPoints.value.length - 1];
  return `${latestPoint.requests} requests in latest bucket`;
});

async function syncAccountBalance() {
  if (!isAuthenticated.value) {
    accountBalance.value = null;
    return;
  }

  try {
    const envelope = await getUserCenterProfile();
    if (envelope?.status === 'success') {
      const nextBalance = Number(envelope?.data?.balance);
      accountBalance.value = Number.isFinite(nextBalance) ? nextBalance : 0;
      return;
    }
    accountBalance.value = null;
  } catch {
    accountBalance.value = null;
  }
}

function handleTopUp() {
  window.open('https://www.aliang.one', '_blank', 'noopener,noreferrer');
}

async function loadDashboardUsageData() {
  if (!isAuthenticated.value) {
    dashboardStats.value = {};
    dashboardModels.value = [];
    dashboardTrend.value = [];
    dashboardUsageRecords.value = [];
    usagePage.value = 1;
    usageTotal.value = 0;
    usageTotalPages.value = 1;
    dashboardError.value = '';
    return;
  }

  dashboardLoading.value = true;
  dashboardError.value = '';

  try {
    const [statsEnvelope, trendEnvelope, modelsEnvelope, usageEnvelope] = await Promise.all([
      getDashboardStats(),
      getDashboardTrend(),
      getDashboardModels(),
      getDashboardUsageRecords({
        page: usagePage.value,
        perPage: usagePageSize.value,
        requestType: appliedRequestFilter.value
      })
    ]);

    dashboardStats.value = asObject(statsEnvelope?.data);
    dashboardTrend.value = Array.isArray(trendEnvelope?.data?.trend) ? trendEnvelope.data.trend : [];
    dashboardModels.value = Array.isArray(modelsEnvelope?.data?.models) ? modelsEnvelope.data.models : [];
    dashboardUsageRecords.value = extractUsageRecords(usageEnvelope?.data)
      .slice(0, usagePageSize.value)
      .map(normalizeUsageRecord);
    const pagination = extractUsagePagination(usageEnvelope?.data, usagePage.value, usagePageSize.value);
    usagePage.value = pagination.page;
    usageTotal.value = pagination.total;
    usageTotalPages.value = Math.min(Math.max(pagination.totalPages, 1), usageMaxPages);
  } catch (error) {
    dashboardError.value = error instanceof Error ? error.message : 'Failed to load dashboard usage data.';
    dashboardStats.value = {};
    dashboardModels.value = [];
    dashboardTrend.value = [];
    dashboardUsageRecords.value = [];
    usageTotal.value = 0;
    usageTotalPages.value = 1;
  } finally {
    dashboardLoading.value = false;
  }
}

function refreshUsageRecords() {
  loadDashboardUsageData();
}

function applyUsageFilters() {
  usagePage.value = 1;
  appliedRequestFilter.value = requestFilter.value;
  loadDashboardUsageData();
}

function resetUsageFilters() {
  requestFilter.value = 'all';
  pathSearch.value = '';
  appliedRequestFilter.value = 'all';
  usagePage.value = 1;
  loadDashboardUsageData();
}

function changeUsagePage(nextPage) {
  const targetPage = Number(nextPage || 1);
  if (!Number.isFinite(targetPage)) {
    return;
  }
  const boundedPage = Math.min(Math.max(1, targetPage), Math.min(Math.max(usageTotalPages.value, 1), usageMaxPages));
  if (boundedPage === usagePage.value) {
    return;
  }
  usagePage.value = boundedPage;
  loadDashboardUsageData();
}

function normalizeUsageRecord(item) {
  const raw = item && typeof item === 'object' ? item : {};
  return {
    id: raw.id || '',
    model: typeof raw.model === 'string' ? raw.model : '-',
    endpoint: typeof raw.inbound_endpoint === 'string' ? raw.inbound_endpoint : '-',
    apiKeyName: typeof raw?.api_key?.name === 'string' ? raw.api_key.name : '-',
    groupName: typeof raw?.group?.name === 'string' ? raw.group.name : '-',
    requestType: typeof raw.request_type === 'string' ? raw.request_type : '-',
    stream: Boolean(raw.stream),
    inputTokens: Number(raw.input_tokens || 0),
    outputTokens: Number(raw.output_tokens || 0),
    totalTokens: Number(raw.input_tokens || 0) + Number(raw.output_tokens || 0) + Number(raw.cache_creation_tokens || 0) + Number(raw.cache_read_tokens || 0),
    actualCost: Number(raw.actual_cost || 0),
    durationMs: Number(raw.duration_ms || 0),
    firstTokenMs: Number(raw.first_token_ms || 0),
    createdAt: typeof raw.created_at === 'string' ? raw.created_at : ''
  };
}

function asObject(value) {
  return value && typeof value === 'object' && !Array.isArray(value) ? value : {};
}

function formatTrendLabel(value) {
  if (typeof value !== 'string' || !value) {
    return '';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

function formatCount(value) {
  const numeric = Number(value || 0);
  if (!Number.isFinite(numeric)) {
    return '0';
  }
  return numeric.toLocaleString();
}

function formatCurrency(value) {
  const numeric = Number(value || 0);
  if (!Number.isFinite(numeric)) {
    return '$0.00';
  }
  return `$${numeric.toFixed(2)}`;
}

function formatDateTime(value) {
  if (typeof value !== 'string' || !value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function exportUsageRecords() {
  const payload = JSON.stringify(dashboardUsageRecords.value, null, 2);
  const blob = new Blob([payload], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = 'dashboard-usage-records.json';
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

function openQuickSetup() {
  if (!isAuthenticated.value) {
    return;
  }
  emit('openQuickSetup');
}

function handleAccountCardClick() {
  if (isAuthenticated.value) {
    return;
  }
  openLoginModal();
}

function openLoginModal() {
  if (isAuthenticated.value) {
    return;
  }
  isLoginModalOpen.value = true;
}

function closeLoginModal() {
  if (loginPending.value) {
    return;
  }
  isLoginModalOpen.value = false;
  loginPassword.value = '';
}

async function submitLogin() {
  const success = await loginWithPassword({
    email: loginEmail.value,
    password: loginPassword.value
  });

  if (success) {
    closeLoginModal();
  }
}

function openQuickChat() {
  if (!isAuthenticated.value) {
    return;
  }
  isQuickChatOpen.value = true;
}

function closeQuickChat() {
  isQuickChatOpen.value = false;
}

function openCertModal() {
  emit('openCertModal');
}

function handleReinstall() {
  if (!confirm('确定要重新安装证书吗？\n将执行：移除旧证书 → 重新生成 → 安装到系统')) {
    return;
  }
  emit('openCertModal');
  setTimeout(() => emit('startCertReinstall'), 300);
}

function handleShowSettings() {
  showSettings();
}

const runModeLabel = computed(() => {
  if (runMode.value === 'tun') return 'TUN';
  if (runMode.value === 'http') return 'HTTP';
  return 'Unknown';
});

const proxyStatusTitle = computed(() => {
  if (runSyncError.value) return 'System Proxy Status Unknown';
  if (runIsRunning.value) return 'System Proxy Active';
  if (!canStartProxy.value) return 'System Proxy Not Ready';
  return 'System Proxy Inactive';
});

const proxyStatusSubtitle = computed(() => {
  if (runActionMessage.value) return runActionMessage.value;
  if (runSyncError.value) return `Sync failed: ${runSyncError.value}`;
  if (!isAuthenticated.value) {
    return 'Disabled: login required before proxy operations can start.';
  }
  if (!canStartProxy.value) {
    switch (startupStatus.value) {
      case 'UNCONFIGURED':
        return 'Disabled: no backend user session was found. Please log in or restore local user info first.';
      case 'CONFIGURING':
        return 'Disabled: account activation is still in progress. Please wait until the backend is ready.';
      default:
        return `Disabled: backend startup status is ${startupStatus.value}. Proxy start is blocked until it becomes ready.`;
    }
  }
  if (runDescription.value) return runDescription.value;
  if (runStatus.value) return runStatus.value;
  return runIsRunning.value ? 'Service is running' : 'Service is stopped';
});

const powerButtonBusyText = computed(() => (runIsRunning.value ? 'Stopping proxy...' : 'Starting proxy...'));

const canStartProxy = computed(() => {
  if (!isAuthenticated.value) {
    return false;
  }
  return startupStatus.value === 'READY' || startupStatus.value === 'CONFIGURED';
});

const powerButtonTitle = computed(() => {
  if (runActionLoading.value) return powerButtonBusyText.value;
  if (powerButtonDisabled.value && !runIsRunning.value) return proxyStatusSubtitle.value;
  return runIsRunning.value ? 'Stop proxy' : 'Start proxy';
});

const powerButtonHaloClass = computed(() => {
  if (powerButtonDisabled.value && !runIsRunning.value) {
    return 'bg-slate-300/50 dark:bg-slate-700/50';
  }
  if (runIsRunning.value) {
    return 'bg-rose-500/20';
  }
  return 'bg-primary/20';
});

const powerButtonClass = computed(() => {
  if (powerButtonDisabled.value && !runIsRunning.value) {
    return 'border-slate-400 bg-slate-300 text-slate-500 shadow-none dark:border-slate-600 dark:bg-slate-700 dark:text-slate-400';
  }
  if (runIsRunning.value) {
    return 'border-rose-400 bg-rose-500 text-white shadow-lg shadow-rose-500/40 hover:bg-rose-500/90';
  }
  return 'border-primary/70 bg-primary text-white shadow-lg shadow-primary/40 hover:bg-primary/90';
});

const powerButtonDisabled = computed(() => {
  if (runActionLoading.value) return true;
  if (runIsRunning.value) return false;
  return !canStartProxy.value;
});

async function syncRunStatus() {
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
    const normalizedMode = String(data?.current_mode || '').toLowerCase();
    runMode.value = normalizedMode === 'http' ? 'http' : normalizedMode === 'tun' ? 'tun' : 'unknown';
    runIsRunning.value = Boolean(data?.is_running);
    runStatus.value = typeof data?.status === 'string' ? data.status : '';
    runDescription.value = typeof data?.description === 'string' ? data.description : '';
    runSyncError.value = '';
  } catch (error) {
    runSyncError.value = error instanceof Error ? error.message : 'Unknown error';
  }
}

async function syncStartupStatus() {
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
  } catch (error) {
    startupStatus.value = 'UNKNOWN';
  }
}

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

async function toggleProxyPower() {
  if (runActionLoading.value) return;
  runActionLoading.value = true;
  runActionMessage.value = '';
  runSyncError.value = '';
  try {
    await syncStartupStatus();
    if (!runIsRunning.value && !canStartProxy.value) {
      throw new Error(proxyStatusSubtitle.value);
    }
    if (runMode.value === 'unknown') {
      await syncRunStatus();
    }
    const endpoint = runIsRunning.value ? '/api/run/stop' : '/api/run/start';
    const actionText = runIsRunning.value ? 'stop' : 'start';
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
      const detail = payload?.msg || payload?.message || payload?.data?.status || `HTTP ${response.status}`;
      throw new Error(`Failed to ${actionText} proxy: ${detail}`);
    }
    if (payload?.code !== 0) {
      throw new Error(payload?.msg || payload?.message || `Failed to ${actionText} proxy`);
    }
    const data = payload?.data || {};
    if (data?.status === 'failed') {
      throw new Error(data?.msg || `Failed to ${actionText} proxy`);
    }

    runActionMessage.value = runIsRunning.value ? 'Proxy stopped successfully.' : 'Proxy start request sent successfully.';
    await syncStartupStatus();
    await syncRunStatus();
  } catch (error) {
    runSyncError.value = error instanceof Error ? error.message : 'Proxy action failed';
  } finally {
    runActionLoading.value = false;
    if (runActionMessage.value) {
      window.setTimeout(() => {
        runActionMessage.value = '';
      }, 2500);
    }
  }
}

const certOverallState = computed(() => {
  const s = certStatus.value;
  if (!s) return 'unknown';
  if (s.is_trusted) return 'trusted';
  if (s.is_installed) return 'installed';
  if (s.is_exported) return 'exported';
  return 'not_found';
});

const networkStatusIcon = computed(() => {
  switch (certOverallState.value) {
    case 'trusted': return 'verified';
    case 'installed': return 'shield';
    case 'exported': return 'upload_file';
    case 'not_found': return 'error';
    default: return 'help';
  }
});

const networkStatusIconClass = computed(() => {
  switch (certOverallState.value) {
    case 'trusted': return 'text-emerald-500';
    case 'installed': return 'text-blue-500';
    case 'exported': return 'text-amber-500';
    case 'not_found': return 'text-red-500';
    default: return 'text-slate-400';
  }
});

const certBadgeText = computed(() => {
  switch (certOverallState.value) {
    case 'trusted': return 'Trusted';
    case 'installed': return 'Installed';
    case 'exported': return 'Generated';
    case 'not_found': return 'Not Found';
    default: return 'Loading...';
  }
});

const certBadgeClass = computed(() => {
  switch (certOverallState.value) {
    case 'trusted': return 'text-emerald-500';
    case 'installed': return 'text-blue-500';
    case 'exported': return 'text-amber-500';
    case 'not_found': return 'text-red-500';
    default: return 'text-slate-400';
  }
});

onMounted(() => {
  startPolling();
  syncStartupStatus();
  syncRunStatus();
  syncAccountBalance();
  loadDashboardUsageData();
  runStatusTimer = window.setInterval(syncRunStatus, 10000);
});

onUnmounted(() => {
  stopPolling();
  if (runStatusTimer !== null) {
    window.clearInterval(runStatusTimer);
    runStatusTimer = null;
  }
});

watch(isAuthenticated, (authenticated) => {
  if (authenticated) {
    isLoginModalOpen.value = false;
    loginPassword.value = '';
    syncAccountBalance();
    loadDashboardUsageData();
    return;
  }
  accountBalance.value = null;
  loadDashboardUsageData();
});

async function sendQuickChat() {
  if (!isAuthenticated.value) {
    quickChatMessages.value.push({
      role: 'assistant',
      content: '请先登录，随后才能使用 Quick Chat。'
    });
    return;
  }

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

const filteredRequestRows = computed(() => {
  const pathKeyword = pathSearch.value.trim().toLowerCase();
  return dashboardUsageRecords.value.filter(item => {
    if (appliedRequestFilter.value === 'chat' && item.requestType !== 'chat') {
      return false;
    }
    if (appliedRequestFilter.value === 'stream' && !item.stream) {
      return false;
    }
    if (appliedRequestFilter.value === 'image' && item.requestType !== 'image') {
      return false;
    }
    if (!pathKeyword) {
      return true;
    }
    return [
      item.endpoint,
      item.model,
      item.apiKeyName,
      item.groupName
    ].some((value) => String(value || '').toLowerCase().includes(pathKeyword));
  });
});

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
