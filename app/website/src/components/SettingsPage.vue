<template>
  <div
    id="settings-page"
    class="page-container content-section hidden flex-1 min-w-0 h-full overflow-hidden bg-background-light dark:bg-background-dark"
  >
    <div class="flex h-full overflow-hidden">
      <aside
        class="w-80 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800 flex flex-col h-full overflow-y-auto custom-scrollbar"
      >
        <div class="p-8 border-b border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/20">
          <div class="flex items-center gap-4">
            <div class="relative">
              <div
                class="size-14 ring-2 ring-primary ring-offset-2 dark:ring-offset-slate-900 rounded-xl overflow-hidden shadow-sm bg-primary/10 flex items-center justify-center"
              >
                <span class="material-symbols-outlined text-primary text-3xl">settings</span>
              </div>
              <div
                class="absolute -bottom-1 -right-1 size-4 bg-primary border-2 border-white dark:border-slate-900 rounded-full"
              ></div>
            </div>
            <div>
              <h2 class="font-bold text-slate-900 dark:text-white tracking-tight">Settings Center</h2>
              <span
                class="inline-flex mt-1 px-2 py-0.5 bg-primary/10 text-primary text-[10px] font-bold rounded uppercase tracking-wider"
              >
                Control Panel
              </span>
              <p class="text-slate-400 text-[11px] mt-1.5 font-medium">Unified gateway control & diagnostics</p>
            </div>
          </div>
        </div>

        <div class="px-6 pt-6 pb-4 space-y-2">
          <p class="text-[10px] font-bold text-slate-400 uppercase tracking-widest px-2">Settings Modules</p>
          <button
            type="button"
            class="settings-tab active w-full flex items-center gap-3 px-4 py-2.5 border border-primary bg-primary/10 text-primary text-sm font-semibold rounded"
            data-tab="rules"
          >
            <span class="material-symbols-outlined text-lg">tune</span>
            规则设置
          </button>
          <button
            type="button"
            class="settings-tab w-full flex items-center gap-3 px-4 py-2.5 border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 hover:border-primary text-sm font-medium rounded transition-colors"
            data-tab="userinfo"
          >
            <span class="material-symbols-outlined text-lg">person</span>
            用户信息
          </button>
          <button
            type="button"
            class="settings-tab w-full flex items-center gap-3 px-4 py-2.5 border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 hover:border-primary text-sm font-medium rounded transition-colors"
            data-tab="logs"
          >
            <span class="material-symbols-outlined text-lg">terminal</span>
            日志监控
          </button>
          <button
            type="button"
            class="settings-tab w-full flex items-center gap-3 px-4 py-2.5 border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 hover:border-primary text-sm font-medium rounded transition-colors"
            data-tab="system"
          >
            <span class="material-symbols-outlined text-lg">dns</span>
            系统设置
          </button>
        </div>

        <div
          class="mt-auto p-6 border-t border-slate-100 dark:border-slate-800 bg-slate-50/30 dark:bg-slate-800/10"
        >
          <div class="rounded border border-slate-100 dark:border-slate-700 bg-white dark:bg-slate-900 p-4">
            <p class="text-xs font-bold text-slate-500 uppercase tracking-wider mb-2">Status</p>
            <div class="space-y-1.5 text-xs text-slate-500">
              <div class="flex justify-between">
                <span>Runtime</span>
                <span class="text-primary font-semibold">Connected</span>
              </div>
              <div class="flex justify-between">
                <span>Profile</span>
                <span class="font-medium text-slate-700 dark:text-slate-200">Default</span>
              </div>
              <div class="flex justify-between">
                <span>Last Sync</span>
                <span class="font-medium text-slate-700 dark:text-slate-200">Just now</span>
              </div>
            </div>
          </div>
        </div>
      </aside>

      <main class="flex-1 flex flex-col h-full bg-background-light dark:bg-background-dark overflow-hidden">
        <header
          class="h-16 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between px-8 shrink-0"
        >
          <div class="flex items-center gap-4">
            <button
              type="button"
              id="backToDashboard"
              class="inline-flex items-center gap-2 text-slate-600 dark:text-slate-400 hover:text-primary transition-colors"
            >
              <span class="material-symbols-outlined">arrow_back</span>
              <span class="font-medium">Back to Dashboard</span>
            </button>
          </div>
          <div class="flex items-center gap-3">
            <div class="size-8 bg-primary rounded-lg flex items-center justify-center text-white shadow-sm">
              <span class="material-symbols-outlined text-base">settings</span>
            </div>
            <h1 class="font-bold text-xl tracking-tight">
              Gateway
              <span class="text-primary font-medium">Settings</span>
            </h1>
          </div>
          <div class="px-3 py-1 bg-primary/10 text-primary text-xs font-bold rounded">LIVE</div>
        </header>

        <div class="flex-1 overflow-y-auto p-8 custom-scrollbar">
          <div class="settings-content active" data-content="rules">
            <div class="bg-white dark:bg-slate-900 p-6 rounded border border-slate-200 dark:border-slate-800 shadow-sm">
              <div class="flex items-center justify-between mb-4">
                <h3 class="font-bold text-lg text-slate-700 dark:text-slate-300">规则引擎</h3>
                <span class="px-2 py-0.5 bg-primary/10 text-primary text-[10px] font-bold rounded uppercase">
                  Rules
                </span>
              </div>
              <div class="space-y-4">
                <div class="flex flex-wrap gap-3">
                  <button type="button" id="rulesEnableBtn" class="px-4 py-2 bg-primary text-white rounded hover:bg-primary/90">
                    启用
                  </button>
                  <button type="button" id="rulesDisableBtn" class="px-4 py-2 bg-slate-500 text-white rounded hover:bg-slate-600">
                    禁用
                  </button>
                  <button
                    type="button"
                    id="rulesReloadBtn"
                    class="px-4 py-2 border border-slate-300 rounded hover:bg-slate-50 dark:hover:bg-slate-800"
                  >
                    重载
                  </button>
                  <button
                    type="button"
                    id="rulesClearCacheBtn"
                    class="px-4 py-2 border border-slate-300 rounded hover:bg-slate-50 dark:hover:bg-slate-800"
                  >
                    清空缓存
                  </button>
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <label
                    class="flex items-center justify-between px-3 py-2 rounded border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/40"
                  >
                    <span class="text-sm">GeoIP 路由</span>
                    <input type="checkbox" id="geoipEnabledSwitch" class="rounded text-primary" />
                  </label>
                  <label
                    class="flex items-center justify-between px-3 py-2 rounded border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/40"
                  >
                    <span class="text-sm">无车道</span>
                    <input type="checkbox" id="nonelaneEnabledSwitch" class="rounded text-primary" />
                  </label>
                </div>
                <div class="flex gap-2">
                  <input
                    type="text"
                    id="rulesLookupDomain"
                    placeholder="域名查询"
                    class="flex-1 px-3 py-2 border border-slate-300 rounded focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary"
                  />
                  <button type="button" id="rulesLookupBtn" class="px-4 py-2 bg-primary text-white rounded hover:bg-primary/90">
                    查询
                  </button>
                </div>
                <div id="rulesLookupResult" class="text-sm text-slate-600 dark:text-slate-300"></div>
              </div>
            </div>
          </div>

          <div class="settings-content hidden" data-content="userinfo">
            <div class="bg-white dark:bg-slate-900 p-6 rounded border border-slate-200 dark:border-slate-800 shadow-sm">
              <h3 class="font-bold text-lg mb-4 text-slate-700 dark:text-slate-300">用户信息</h3>
              <div id="authUserInfoContainer" class="space-y-3">
                <div class="flex justify-between p-3 rounded bg-slate-50 dark:bg-slate-800/40">
                  <span class="text-slate-500">用户名</span>
                  <span class="font-medium">-</span>
                </div>
                <div class="flex justify-between p-3 rounded bg-slate-50 dark:bg-slate-800/40">
                  <span class="text-slate-500">余额</span>
                  <span id="userBalance" class="font-semibold text-primary">-</span>
                </div>
              </div>
              <div class="mt-6">
                <div class="flex flex-wrap gap-2">
                  <input
                    type="text"
                    id="authTokenInput"
                    placeholder="Token"
                    class="flex-1 min-w-[220px] px-3 py-2 border border-slate-300 rounded focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary"
                  />
                  <button type="button" id="authActivateBtn" class="px-4 py-2 bg-primary text-white rounded hover:bg-primary/90">
                    激活
                  </button>
                  <button type="button" id="authLogoutBtn" class="px-4 py-2 bg-slate-500 text-white rounded hover:bg-slate-600">
                    登出
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div class="settings-content hidden" data-content="logs">
            <div class="bg-white dark:bg-slate-900 p-6 rounded border border-slate-200 dark:border-slate-800 shadow-sm">
              <div class="flex flex-wrap justify-between items-center gap-3 mb-4">
                <h3 class="font-bold text-lg text-slate-700 dark:text-slate-300">日志监控</h3>
                <div class="flex flex-wrap gap-2">
                  <select
                    id="logLevelSelect"
                    class="px-3 py-2 border border-slate-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary"
                  >
                    <option value="all">All</option>
                    <option value="info">Info</option>
                    <option value="warn">Warn</option>
                    <option value="error">Error</option>
                  </select>
                  <select
                    id="logSourceSelect"
                    class="px-3 py-2 border border-slate-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary"
                  >
                    <option value="all">All</option>
                  </select>
                  <button
                    type="button"
                    id="logsRefreshBtn"
                    class="px-3 py-2 border border-slate-300 rounded text-sm hover:bg-slate-50 dark:hover:bg-slate-800"
                  >
                    刷新
                  </button>
                  <button
                    type="button"
                    id="logsClearBtn"
                    class="px-3 py-2 border border-slate-300 rounded text-sm hover:bg-slate-50 dark:hover:bg-slate-800"
                  >
                    清空
                  </button>
                  <button type="button" id="wsConnectBtn" class="px-3 py-2 bg-primary text-white rounded text-sm hover:bg-primary/90">
                    连接
                  </button>
                  <button
                    type="button"
                    id="wsDisconnectBtn"
                    class="px-3 py-2 bg-slate-500 text-white rounded text-sm hover:bg-slate-600"
                  >
                    断开
                  </button>
                </div>
              </div>
              <div
                id="logsOutput"
                class="h-96 overflow-y-auto bg-slate-50 dark:bg-slate-800 p-4 rounded border border-slate-100 dark:border-slate-700 font-mono text-xs"
              ></div>
              <div id="wsConnectionStatus" class="mt-2 text-xs text-slate-500"></div>
            </div>
          </div>

          <div class="settings-content hidden" data-content="system">
            <div class="bg-white dark:bg-slate-900 p-6 rounded border border-slate-200 dark:border-slate-800 shadow-sm">
              <h3 class="font-bold text-lg mb-4 text-slate-700 dark:text-slate-300">系统设置</h3>
              <div class="space-y-6">
                <div>
                  <h4 class="text-sm font-semibold mb-2">运行模式</h4>
                  <div class="flex gap-4">
                    <label class="flex items-center gap-2 px-3 py-2 rounded border border-slate-200 dark:border-slate-700">
                      <input type="radio" name="runMode" id="runModeTun" value="tun" />
                      <span>TUN</span>
                    </label>
                    <label class="flex items-center gap-2 px-3 py-2 rounded border border-slate-200 dark:border-slate-700">
                      <input type="radio" name="runMode" id="runModeHttp" value="http" />
                      <span>HTTP</span>
                    </label>
                  </div>
                  <div class="mt-3 flex flex-wrap gap-2">
                    <button type="button" id="runStartBtn" class="px-4 py-2 bg-primary text-white rounded hover:bg-primary/90">
                      启动
                    </button>
                    <button type="button" id="runStopBtn" class="px-4 py-2 bg-slate-500 text-white rounded hover:bg-slate-600">
                      停止
                    </button>
                    <button
                      type="button"
                      id="runModeBtn"
                      class="px-4 py-2 border border-slate-300 rounded hover:bg-slate-50 dark:hover:bg-slate-800"
                    >
                      切换模式
                    </button>
                  </div>
                  <div class="mt-3 text-sm text-slate-500">
                    <span>当前模式:</span>
                    <span id="runCurrentMode">-</span>
                  </div>
                  <div id="runServiceStatus" class="text-sm text-slate-500"></div>
                  <div id="runAvailableModes" class="text-sm text-slate-500"></div>
                  <div id="runStatusInfo" class="text-sm text-slate-500"></div>
                </div>

                <div>
                  <h4 class="text-sm font-semibold mb-2">证书管理</h4>
                  <div
                    class="p-4 rounded border border-slate-200 dark:border-slate-700 bg-slate-50/70 dark:bg-slate-800/30"
                  >
                    <div class="flex flex-wrap items-center justify-between gap-3">
                      <div>
                        <div class="text-sm font-semibold text-slate-700 dark:text-slate-200">
                          Certificate Management Modal
                        </div>
                        <div class="text-xs text-slate-500 dark:text-slate-400 mt-1">查看状态、导出、下载、安装与移除证书</div>
                      </div>
                      <button
                        type="button"
                        id="openCertManagementModalBtn"
                        class="px-4 py-2 bg-primary text-white rounded text-sm font-semibold hover:bg-primary/90 transition-colors"
                      >
                        打开证书弹窗
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  </div>
</template>
