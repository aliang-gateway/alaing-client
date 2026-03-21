<template>
  <div class="settings-pane flex min-h-[calc(100vh-14rem)] flex-1 flex-col" data-pane="rules">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-xl font-bold">Rule Configuration</h2>
        <p class="text-sm text-slate-500">Define routing domains for LLM providers</p>
      </div>
      <button id="rulesConfigSaveBtn" type="button" class="inline-flex items-center gap-2 rounded bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary/90">
        <span class="material-symbols-outlined text-sm">save</span>
        Save Configuration
      </button>
    </div>

    <div class="mt-4 flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-slate-200 bg-slate-900 dark:border-slate-800">
      <div class="flex items-center gap-2 border-b border-slate-700 bg-slate-800/50 px-4 py-2">
        <div class="flex gap-1.5">
          <div class="h-3 w-3 rounded-full bg-red-500"></div>
          <div class="h-3 w-3 rounded-full bg-yellow-500"></div>
          <div class="h-3 w-3 rounded-full bg-green-500"></div>
        </div>
        <span class="ml-4 font-mono text-xs text-slate-400">rules.json</span>
      </div>
      <div class="flex-1 overflow-auto p-4 font-mono text-sm">
        <pre class="text-primary"><code>{
  "categories": {
    "OpenAI": ["openai.com", "chatgpt.com", "oaistatic.com"],
    "Claude": ["anthropic.com", "claude.ai"],
    "DeepSeek": ["deepseek.com"],
    "Gemini": ["gemini.google.com", "generativelanguage.googleapis.com"],
    "Cursor": ["cursor.sh", "cursor.com"],
    "Grok": ["x.ai", "grok.com"],
    "GLM": ["zhipuai.cn"],
    "Kimi": ["moonshot.cn"]
  },
  "default_outbound": "Proxy",
  "dns_server": "1.1.1.1"
}</code></pre>
      </div>
    </div>

    <div class="compat-anchors" aria-hidden="true">
      <div id="rulesStatus"></div>
      <div id="rules-status-badge"></div>
      <div id="rules-status-text"></div>
      <div id="rulesCacheCount"></div>
      <div id="rulesCacheSize"></div>
      <div id="cache-hit-rate"></div>
      <div id="total-cache"></div>
      <div id="cache-last-update"></div>
      <input id="geoipEnabledSwitch" type="checkbox" />
      <input id="nonelaneEnabledSwitch" type="checkbox" />
      <button id="rulesEnableBtn" type="button"></button>
      <button id="rulesDisableBtn" type="button"></button>
      <button id="rulesReloadBtn" type="button"></button>
      <button id="rulesClearCacheBtn" type="button"></button>
      <input id="rulesLookupDomain" type="text" />
      <button id="rulesLookupBtn" type="button"></button>
      <textarea id="rulesLookupResult"></textarea>
      <table><tbody id="toDoorRulesBody"></tbody></table>
      <table><tbody id="blacklistRulesBody"></tbody></table>
      <table><tbody id="nonelaneRulesBody"></tbody></table>
      <button id="addToDoorRuleBtn" type="button"></button>
      <button id="addBlacklistRuleBtn" type="button"></button>
      <button id="addNonelaneRuleBtn" type="button"></button>
      <div id="ruleEditModal"></div>
      <select id="ruleTypeSelect"><option value="">-</option></select>
      <input id="ruleConditionInput" type="text" />
      <input id="ruleEnabledCheckbox" type="checkbox" />
      <input id="ruleIdInput" type="text" />
      <input id="ruleSetInput" type="text" />
      <div id="ruleEditModalTitle"></div>
      <div id="conditionError"></div>
      <div id="typeHelpText"></div>
      <button id="ruleEditSaveBtn" type="button"></button>
    </div>
  </div>
</template>

<script>
export default {
  name: 'RulesSettings'
}
</script>
