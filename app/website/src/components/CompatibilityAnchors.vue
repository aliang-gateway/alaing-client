<template>
  <div class="compat-anchors hidden" style="display: none !important; visibility: hidden; height: 0; overflow: hidden">
    <div id="authRefreshError" class="hidden"></div>
    <div id="authRefreshErrorMsg" class="hidden"></div>
    <div id="authRefreshInterval" class="hidden"></div>
    <div id="authRefreshRunning" class="hidden"></div>
    <div id="authLastUpdate" class="hidden"></div>
    <div id="authRefreshStatusCard" class="hidden"></div>
    <div id="ruleEditModal" class="hidden modal"></div>
    <select id="ruleTypeSelect" class="hidden"><option value="">-</option></select>
    <div id="ruleEditModalTitle" class="hidden modal"></div>
    <input type="checkbox" id="ruleEnabledCheckbox" class="hidden" />
    <input type="text" id="ruleIdInput" class="hidden" />
    <input type="text" id="ruleConditionInput" class="hidden" />
    <input type="text" id="ruleSetInput" class="hidden" />
    <button type="button" id="ruleEditSaveBtn" class="hidden"></button>
    <button type="button" id="rulesConfigSaveBtn" class="hidden"></button>
    <div id="rulesCacheCount" class="hidden"></div>
    <div id="rulesCacheSize" class="hidden"></div>
    <div id="rulesStatus" class="hidden"></div>
    <div id="rules-status-badge" class="hidden"></div>
    <div id="rules-status-text" class="hidden"></div>
    <table><tbody id="toDoorRulesBody" class="hidden"></tbody></table>
    <table><tbody id="blacklistRulesBody" class="hidden"></tbody></table>
    <table><tbody id="nonelaneRulesBody" class="hidden"></tbody></table>
    <button type="button" id="addBlacklistRuleBtn" class="hidden"></button>
    <button type="button" id="addNonelaneRuleBtn" class="hidden"></button>
    <button type="button" id="addToDoorRuleBtn" class="hidden"></button>
    <button type="button" id="logsFullscreenBtn" class="hidden"></button>
    <div id="logCount" class="hidden"></div>
    <button type="button" id="logFilterBtn" class="hidden"></button>
    <div id="logAutoScroll" class="hidden"></div>
    <button type="button" id="logConfigSaveBtn" class="hidden"></button>
    <div id="statsUploadTotal" class="hidden"></div>
    <div id="statsDownloadTotal" class="hidden"></div>
    <div id="statsConnectionCount" class="hidden"></div>
    <div id="statsTotalTraffic" class="hidden"></div>
    <table><tbody id="statsTableBody" class="hidden"></tbody></table>
    <div id="statsUploadRate" class="hidden"></div>
    <div id="statsDownloadRate" class="hidden"></div>
    <div id="dnsTotalEntries" class="hidden"></div>
    <div id="dnsUniqueDomains" class="hidden"></div>
    <div id="dnsUniqueIPs" class="hidden"></div>
    <div id="dnsHitRate" class="hidden"></div>
    <div id="dnsSearchBox" class="hidden"></div>
    <button type="button" id="dnsSearchBtn" class="hidden"></button>
    <div id="dnsSearchResultsContainer" class="hidden"></div>
    <div id="dnsHotDomainsContainer" class="hidden"></div>
    <div id="dnsHotIPsContainer" class="hidden"></div>
    <button type="button" id="dnsRefreshBtn" class="hidden"></button>
    <button type="button" id="dnsClearAllBtn" class="hidden"></button>
    <select id="proxySelect" class="hidden"><option value="">-</option></select>
    <button type="button" id="switchProxyBtn" class="hidden"></button>
    <button type="button" id="testLatencyBtn" class="hidden"></button>
    <table><tbody id="proxyTableBody" class="hidden"></tbody></table>
    <div id="currentProxyDisplay" class="hidden"></div>
    <div id="proxyDetailModal" class="hidden modal"></div>
    <div id="proxyDetailTitle" class="hidden"></div>
    <div id="proxyDetailJson" class="hidden"></div>
    <button type="button" id="copyProxyDetailBtn" class="hidden"></button>
    <div id="statusIndicator" class="hidden"></div>
    <div id="statusText" class="hidden"></div>
    <button type="button" id="dashStartBtn" class="hidden"></button>
    <button type="button" id="dashStopBtn" class="hidden"></button>
    <div id="dashRunStatus" class="hidden"></div>
    <div id="dashCurrentProxy" class="hidden"></div>
    <div id="dashRunMode" class="hidden"></div>
    <div id="dashRuleStatus" class="hidden"></div>
    <div id="dashProxyCount" class="hidden"></div>
    <div id="dashDoorCount" class="hidden"></div>
    <div id="dashLastUpdate" class="hidden"></div>
    <canvas id="trafficChart" class="hidden"></canvas>
    <canvas id="domainPieChart" class="hidden"></canvas>
    <div id="requestLogList" class="hidden"></div>
    <button type="button" id="dashBtnCheckCert" class="hidden"></button>
    <button type="button" id="dashBtnInstallCert" class="hidden"></button>
    <button type="button" id="quickChatBtn" class="hidden"></button>
    <div id="chatModal" class="hidden modal"></div>
    <div id="chatMessages" class="hidden"></div>
    <input type="text" id="chatInput" class="hidden" />
    <button type="button" id="chatSendBtn" class="hidden"></button>
    <button type="button" id="chatClearHistoryBtn" class="hidden"></button>
    <input type="text" id="chatSearchInput" class="hidden" />
    <button type="button" id="goToSettingsBtnCompat" class="hidden"></button>
    <div id="logs-page" class="hidden"></div>
    <div id="rules-page" class="hidden"></div>
    <div id="proxy-control-page" class="hidden"></div>
    <div id="userinfo-page" class="hidden"></div>
    <div id="dnscache-page" class="hidden"></div>
    <div id="conditionError" class="hidden"></div>
    <div id="typeHelpText" class="hidden"></div>
    <div id="notification-container" class="hidden"></div>
    <div id="cache-hit-rate" class="hidden"></div>
    <div id="total-cache" class="hidden"></div>
    <div id="cache-last-update" class="hidden"></div>
  </div>
</template>
