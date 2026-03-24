<template>
  <div class="settings-pane flex min-h-[calc(100vh-14rem)] flex-1 flex-col" data-pane="rules">
    <div class="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-background-dark">
      <div class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h2 class="text-xl font-bold text-slate-900 dark:text-white">Customer Configuration</h2>
          <p class="text-sm text-slate-500">
            Manage the customer-facing proxy, AI rules, and proxy rules stored by the customer config API.
          </p>
        </div>
        <div class="flex flex-col items-start gap-2 md:items-end">
          <span class="rounded bg-primary/10 px-3 py-1 text-[11px] font-bold uppercase tracking-wide text-primary">
            {{ loading ? 'Loading' : 'Customer only' }}
          </span>
          <span v-if="version" class="text-[11px] text-slate-400">Version {{ version }}</span>
        </div>
      </div>

      <div
        v-if="error"
        class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200"
        role="alert"
      >
        {{ error }}
      </div>

      <div v-if="loading" class="rounded-lg border border-dashed border-slate-300 bg-slate-50 px-4 py-8 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-400">
        Loading customer configuration…
      </div>

      <form v-else class="space-y-6" @submit.prevent="handleSubmit">
        <section class="rounded-xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-900/40">
          <div class="mb-4 flex items-center gap-2">
            <span class="material-symbols-outlined text-primary">vpn_key</span>
            <div>
              <h3 class="font-semibold text-slate-900 dark:text-white">Customer Proxy</h3>
              <p class="text-xs text-slate-500">Only `customer.proxy` fields are editable here.</p>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <label class="space-y-2">
              <span class="text-xs font-semibold uppercase tracking-wide text-slate-500">Proxy type</span>
              <select
                v-model="form.proxy.type"
                class="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-700 shadow-sm transition focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
              >
                <option value="socks">SOCKS</option>
                <option value="http">HTTP</option>
              </select>
            </label>

            <label class="space-y-2">
              <span class="text-xs font-semibold uppercase tracking-wide text-slate-500">Server</span>
              <input
                v-model.trim="form.proxy.server"
                class="w-full rounded-lg border px-3 py-2 text-sm shadow-sm transition focus:outline-none focus:ring-2 dark:text-slate-100"
                :class="serverFieldClass"
                placeholder="127.0.0.1:1080"
                type="text"
                @keydown.enter.prevent
              />
              <p v-if="serverError" class="text-xs text-red-500">{{ serverError }}</p>
            </label>
          </div>
        </section>

        <section class="rounded-xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-900/40">
          <div class="mb-4 flex items-center gap-2">
            <span class="material-symbols-outlined text-primary">robot_2</span>
            <div>
              <h3 class="font-semibold text-slate-900 dark:text-white">AI Rules</h3>
              <p class="text-xs text-slate-500">Choose which backend AI-rule providers are enabled and list excluded domains, one per line.</p>
            </div>
          </div>

          <div v-if="!providerOrder.length" class="rounded-lg border border-dashed border-slate-300 bg-white px-4 py-6 text-sm text-slate-500 dark:border-slate-700 dark:bg-background-dark dark:text-slate-400">
            No AI rule providers were returned by the backend yet.
          </div>

          <div v-else class="grid gap-4 xl:grid-cols-2">
            <template v-for="provider in providerOrder" :key="provider">
            <article
              v-if="form.ai_rules[provider]"
              class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-background-dark"
            >
              <div class="mb-3 flex items-center justify-between gap-3">
                <div>
                  <h4 class="font-semibold text-slate-900 dark:text-white">{{ providerLabel(provider, presetProviders) }}</h4>
                  <p class="text-xs text-slate-500">{{ provider }}</p>
                </div>
                <label class="inline-flex cursor-pointer items-center gap-2 text-xs font-semibold text-slate-600 dark:text-slate-200">
                  <input v-model="form.ai_rules[provider].enable" class="peer sr-only" type="checkbox" />
                  <span class="relative h-6 w-11 rounded-full bg-slate-300 transition-colors after:absolute after:left-0.5 after:top-0.5 after:h-5 after:w-5 after:rounded-full after:bg-white after:transition-transform after:content-[''] peer-checked:bg-primary peer-checked:after:translate-x-5 peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-primary dark:bg-slate-700"></span>
                  
                </label>
              </div>

              <label v-if="isDev" class="space-y-2">
                <span class="text-xs font-semibold uppercase tracking-wide text-slate-500">Exclude domains</span>
                <textarea
                  :value="_providerExcludeTexts[provider]"
                  class="min-h-28 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-sm text-slate-700 shadow-sm transition focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
                  placeholder="example.com&#10;api.example.com"
                  @input="_providerExcludeTexts[provider] = $event.target.value"
                ></textarea>
              </label>
            </article>
            </template>
          </div>
        </section>

        <section class="rounded-xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-900/40">
          <div class="mb-4 flex items-center gap-2">
            <span class="material-symbols-outlined text-primary">rule_settings</span>
            <div>
              <h3 class="font-semibold text-slate-900 dark:text-white">Proxy Rules</h3>
              <p class="text-xs text-slate-500">Edit `customer.proxy_rules`, one rule per line.</p>
            </div>
            <label class="ml-auto inline-flex cursor-pointer items-center gap-2 text-xs font-semibold text-slate-600 dark:text-slate-200">
              <input v-model="form.proxy_rules.enabled" class="peer sr-only" type="checkbox" />
              <span class="relative h-6 w-11 rounded-full bg-slate-300 transition-colors after:absolute after:left-0.5 after:top-0.5 after:h-5 after:w-5 after:rounded-full after:bg-white after:transition-transform after:content-[''] peer-checked:bg-primary peer-checked:after:translate-x-5 peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-primary dark:bg-slate-700"></span>
              Enabled
            </label>
          </div>

          <label class="space-y-2">
            <span class="text-xs font-semibold uppercase tracking-wide text-slate-500">Rules list</span>
            <textarea
              :value="_proxyRulesText"
              class="min-h-40 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-sm text-slate-700 shadow-sm transition focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
              placeholder="domain,example.com,proxy&#10;api.example.com"
              @input="_proxyRulesText = $event.target.value"
            ></textarea>
          </label>
        </section>

        <div class="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-background-dark md:flex-row md:items-center md:justify-between">
          <div class="text-xs text-slate-500">
            This form submits a payload shaped as <span class="font-mono text-slate-700 dark:text-slate-200">{ customer: ... }</span> with no core settings.
          </div>
          <button
            id="rulesConfigSaveBtn"
            :disabled="saving || !!serverError"
            class="inline-flex min-h-11 items-center justify-center gap-2 rounded bg-primary px-4 py-2 text-sm font-medium text-white transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
            type="submit"
          >
            <span class="material-symbols-outlined text-sm">save</span>
            {{ saving ? 'Saving…' : 'Save Configuration' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script>
function defaultConfig() {
  return {
    proxy: {
      type: 'socks',
      server: ''
    },
    ai_rules: {},
    proxy_rules: { enabled: true, rules: [] }
  };
}

function normalizeStringList(items) {
  return Array.isArray(items)
    ? items.map((entry) => String(entry).trim()).filter(Boolean)
    : [];
}

function sanitizeLines(value) {
  return value
    .split('\n')
    .map((entry) => entry.trim())
    .filter(Boolean);
}

function normalizeAiRules(aiRules = {}) {
  if (!aiRules || typeof aiRules !== 'object' || Array.isArray(aiRules)) {
    return {};
  }

  return Object.fromEntries(
    Object.entries(aiRules).map(([provider, value]) => [provider, {
      enable: Boolean(value?.enable),
      exclude: normalizeStringList(value?.exclude)
    }])
  );
}

function normalizeProxyRules(raw) {
  if (!raw) {
    return { enabled: true, rules: [] };
  }
  if (Array.isArray(raw)) {
    return { enabled: true, rules: normalizeStringList(raw) };
  }
  if (typeof raw === 'object') {
    return {
      enabled: Boolean(raw.enabled),
      rules: normalizeStringList(raw.rules)
    };
  }
  return { enabled: true, rules: [] };
}

function normalizeConfig(config = {}) {
  const defaults = defaultConfig();
  return {
    proxy: {
      type: config?.proxy?.type === 'http' ? 'http' : defaults.proxy.type,
      server: typeof config?.proxy?.server === 'string' ? config.proxy.server : defaults.proxy.server
    },
    ai_rules: normalizeAiRules(config?.ai_rules),
    proxy_rules: normalizeProxyRules(config?.proxy_rules)
  };
}

function providerKeys(aiRules = {}) {
  return Object.keys(aiRules);
}

function mergeProviderOrder(configuredKeys, presetProviders) {
  const presetMap = {};
  for (const p of presetProviders) {
    presetMap[p.key] = p;
  }
  const ordered = [...configuredKeys];
  for (const p of presetProviders) {
    if (!presetMap[p.key] || configuredKeys.includes(p.key)) continue;
    ordered.push(p.key);
  }
  return ordered;
}

const SERVER_RE = /^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|\[[\da-fA-F:]+\]):(\d{1,5})$/;

function isValidServer(value) {
  if (!value) return '';
  const match = value.match(SERVER_RE);
  if (!match) return '格式必须为 IP:Port，例如 127.0.0.1:1080';
  const port = Number(match[2]);
  if (port < 1 || port > 65535) return '端口号必须在 1-65535 之间';
  const ip = match[1];
  if (ip.startsWith('[')) return '';
  if (ip.split('.').every((o) => Number(o) >= 0 && Number(o) <= 255)) return '';
  return 'IP 地址不合法';
}

export default {
  name: 'RulesSettings',
  props: {
    config: {
      type: Object,
      default: () => defaultConfig()
    },
    presetProviders: {
      type: Array,
      default: () => []
    },
    loading: {
      type: Boolean,
      default: false
    },
    saving: {
      type: Boolean,
      default: false
    },
    error: {
      type: String,
      default: ''
    },
    version: {
      type: String,
      default: ''
    }
  },
  emits: ['save'],
  data() {
    return {
      isDev: import.meta.env.VITE_MODE === 'dev',
      form: normalizeConfig(this.config),
      _proxyRulesText: '',
      _providerExcludeTexts: {}
    };
  },
  created() {
    this.ensureProviders();
    this.syncTextFromForm();
  },
  computed: {
    providerOrder() {
      return mergeProviderOrder(providerKeys(this.form.ai_rules), this.presetProviders);
    },
    serverError() {
      return isValidServer(this.form.proxy.server);
    },
    serverFieldClass() {
      if (!this.form.proxy.server) return 'border-slate-300 bg-white focus:border-primary focus:ring-primary/20 dark:border-slate-700 dark:bg-slate-900';
      return this.serverError
        ? 'border-red-400 bg-red-50 focus:border-red-500 focus:ring-red-500/20 dark:border-red-500/50 dark:bg-red-900/10'
        : 'border-emerald-400 bg-emerald-50/50 focus:border-emerald-500 focus:ring-emerald-500/20 dark:border-emerald-500/50 dark:bg-emerald-900/10';
    }
  },
  watch: {
    config: {
      deep: true,
      handler(nextConfig) {
        this.form = normalizeConfig(nextConfig);
        this.ensureProviders();
        this.syncTextFromForm();
      }
    },
    presetProviders: {
      deep: true,
      handler() {
        this.ensureProviders();
        this.syncTextFromForm();
      }
    }
  },
  methods: {
    ensureProviders() {
      for (const p of this.presetProviders) {
        if (!(p.key in this.form.ai_rules)) {
          this.form.ai_rules[p.key] = { enable: false, exclude: [] };
        }
      }
    },
    providerLabel(key, presetProviders) {
      const preset = presetProviders.find(p => p.key === key);
      return preset ? preset.label : key;
    },
    syncTextFromForm() {
      this._proxyRulesText = (this.form.proxy_rules.rules || []).join('\n');
      for (const key of Object.keys(this.form.ai_rules)) {
        if (!this._providerExcludeTexts[key]) {
          this._providerExcludeTexts[key] = (this.form.ai_rules[key].exclude || []).join('\n');
        }
      }
    },
    async handleSubmit() {
      const normalized = normalizeConfig(this.form);
      normalized.proxy_rules.rules = sanitizeLines(this._proxyRulesText);
      for (const [key, text] of Object.entries(this._providerExcludeTexts)) {
        if (normalized.ai_rules[key]) {
          normalized.ai_rules[key].exclude = sanitizeLines(text);
        }
      }
      this.$emit('save', normalized);
    }
  }
}
</script>
