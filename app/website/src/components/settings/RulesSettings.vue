<template>
  <div class="settings-content active" data-content="rules">
    <div class="settings-card">
      <div class="flex items-center justify-between mb-4">
        <h3 class="settings-card-title !mb-0">规则引擎</h3>
        <span class="px-2 py-0.5 bg-primary/10 text-primary text-[10px] font-bold rounded uppercase">
          Rules
        </span>
      </div>
      <div class="space-y-4">
        <div class="flex flex-wrap gap-3">
          <button type="button" id="rulesEnableBtn" class="settings-btn-primary">
            启用
          </button>
          <button type="button" id="rulesDisableBtn" class="settings-btn-secondary">
            禁用
          </button>
          <button type="button" id="rulesReloadBtn" class="settings-btn-outline">
            重载
          </button>
          <button type="button" id="rulesClearCacheBtn" class="settings-btn-outline">
            清空缓存
          </button>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <label class="flex items-center justify-between px-3 py-2 rounded border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/40">
            <span class="text-sm">GeoIP 路由</span>
            <input type="checkbox" id="geoipEnabledSwitch" class="rounded text-primary" />
          </label>
          <label class="flex items-center justify-between px-3 py-2 rounded border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/40">
            <span class="text-sm">无车道</span>
            <input type="checkbox" id="nonelaneEnabledSwitch" class="rounded text-primary" />
          </label>
        </div>
        <div class="flex gap-2">
          <input type="text" id="rulesLookupDomain" placeholder="域名查询" class="flex-1 px-3 py-2 border border-slate-300 rounded focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary" />
          <button type="button" id="rulesLookupBtn" class="settings-btn-primary">
            查询
          </button>
        </div>
        <div id="rulesLookupResult" class="text-sm text-slate-600 dark:text-slate-300"></div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'RulesSettings'
}
</script>