// ============ 全局状态管理 ============
const appState = {
    currentPage: 'dashboard',
    proxies: [],
    currentProxy: null,
    doorMembers: [],
    loading: false,
    // 新增状态属性
    runStatus: {
        isRunning: false,
        currentMode: null,
        lastUpdated: null
    },
    ruleEngineStatus: {
        isEnabled: false,
        lastUpdated: null
    },
    logConfig: {
        level: 'info',
        source: '',
        autoScroll: true
    },
    statusPollingInterval: null
};

// ============ 流量图表数据管理 ============
const chartDataManager = {
    maxPoints: 60,  // 保留60个数据点（90秒，1.5秒/点）

    // 历史数据
    data: {
        timestamps: [],      // 时间戳（ms）
        uploadSpeeds: [],    // Bytes/s
        downloadSpeeds: []   // Bytes/s
    },

    // 上一次的累计值（用于计算速率）
    lastValues: {
        uploadTotal: 0,
        downloadTotal: 0,
        timestamp: 0
    },

    // 添加数据点
    addPoint(uploadTotal, downloadTotal) {
        const now = Date.now();
        const timeDelta = (now - this.lastValues.timestamp) / 1000;  // 秒

        // 计算速率（Bytes/s）
        const uploadSpeed = timeDelta > 0 ?
            Math.max(0, (uploadTotal - this.lastValues.uploadTotal) / timeDelta) : 0;
        const downloadSpeed = timeDelta > 0 ?
            Math.max(0, (downloadTotal - this.lastValues.downloadTotal) / timeDelta) : 0;

        // 追加数据
        this.data.timestamps.push(now);
        this.data.uploadSpeeds.push(uploadSpeed);
        this.data.downloadSpeeds.push(downloadSpeed);

        // 保留最近N个点
        if (this.data.timestamps.length > this.maxPoints) {
            this.data.timestamps.shift();
            this.data.uploadSpeeds.shift();
            this.data.downloadSpeeds.shift();
        }

        // 保存当前值
        this.lastValues.uploadTotal = uploadTotal;
        this.lastValues.downloadTotal = downloadTotal;
        this.lastValues.timestamp = now;
    },

    clear() {
        this.data = { timestamps: [], uploadSpeeds: [], downloadSpeeds: [] };
        this.lastValues = { uploadTotal: 0, downloadTotal: 0, timestamp: 0 };
    }
};

const dashboardRequestLog = {
    maxItems: 50,
    entries: []
};

const chatStore = {
    storageKey: 'alianggate-chat-history',
    maxItems: 200,
    entries: []
};

const domainCategoryMap = {
    cursor: ['cursor.sh', 'api2.cursor.sh', 'cursor.com'],
    openai: ['openai.com', 'api.openai.com'],
    claude: ['anthropic.com', 'claude.ai'],
    chatgpt: ['chatgpt.com'],
    copilot: ['githubcopilot.com', 'copilot.microsoft.com']
};

let activeDomainFilter = 'all';

// 全局图表实例
let charts = {};

function classifyDomainCategory(domain) {
    const normalized = String(domain || '').toLowerCase();
    for (const [category, patterns] of Object.entries(domainCategoryMap)) {
        if (patterns.some(pattern => normalized.includes(pattern))) {
            return category;
        }
    }
    return null;
}

function updateDomainPieChartByLogs() {
    if (!charts.domainPie) {
        return;
    }

    const nameMap = {
        cursor: 'Cursor',
        openai: 'OpenAI',
        claude: 'Claude',
        chatgpt: 'ChatGPT',
        copilot: 'Copilot'
    };

    const bucket = {
        cursor: 0,
        openai: 0,
        claude: 0,
        chatgpt: 0,
        copilot: 0
    };

    dashboardRequestLog.entries.forEach(item => {
        const category = classifyDomainCategory(item.domain);
        if (!category) {
            return;
        }

        if (activeDomainFilter !== 'all' && activeDomainFilter !== category) {
            return;
        }

        bucket[category] += Number(item.bytes || 0);
    });

    const labels = [];
    const values = [];

    Object.entries(bucket).forEach(([key, value]) => {
        if (value > 0) {
            labels.push(nameMap[key]);
            values.push(value);
        }
    });

    charts.domainPie.data.labels = labels.length > 0 ? labels : ['暂无数据'];
    charts.domainPie.data.datasets[0].data = values.length > 0 ? values : [1];
    charts.domainPie.update('none');
}

function deriveLogDomain(entry) {
    const candidates = [entry?.domain, entry?.host, entry?.server_name, entry?.target];
    const firstCandidate = candidates.find(v => typeof v === 'string' && v.trim() !== '');
    if (firstCandidate) {
        return firstCandidate;
    }

    const message = String(entry?.message || '');
    const match = message.match(/([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}/);
    return match ? match[0] : 'unknown';
}

function deriveLogPath(entry) {
    const candidates = [entry?.path, entry?.uri, entry?.url];
    const firstCandidate = candidates.find(v => typeof v === 'string' && v.trim() !== '');
    return firstCandidate || '/';
}

function deriveLogMethod(entry) {
    const method = typeof entry?.method === 'string' ? entry.method.toUpperCase() : '';
    if (method) {
        return method;
    }

    const message = String(entry?.message || '');
    const match = message.match(/\b(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\b/i);
    return match ? match[1].toUpperCase() : 'LOG';
}

function appendDashboardRequestLog(entry) {
    const listEl = document.getElementById('requestLogList');
    const countEl = document.getElementById('logCount');
    if (!listEl) {
        return;
    }

    const timestamp = entry?.timestamp || new Date().toLocaleTimeString();
    const level = (entry?.level || 'INFO').toUpperCase();
    const domain = deriveLogDomain(entry);
    const method = deriveLogMethod(entry);
    const path = deriveLogPath(entry);
    const source = entry?.source || '-';
    const message = entry?.message || '';
    const bytes = Number(entry?.bytes || entry?.size || 0);

    dashboardRequestLog.entries.unshift({ timestamp, level, domain, method, path, source, message, bytes });
    if (dashboardRequestLog.entries.length > dashboardRequestLog.maxItems) {
        dashboardRequestLog.entries = dashboardRequestLog.entries.slice(0, dashboardRequestLog.maxItems);
    }

    const fragment = document.createDocumentFragment();
    dashboardRequestLog.entries.forEach((item, index) => {
        const detailsId = `reqLogDetail-${index}`;
        const row = document.createElement('button');
        row.type = 'button';
        row.className = 'log-item';
        row.setAttribute('aria-expanded', 'false');
        row.innerHTML = `
            <span class="log-time">${item.timestamp}</span>
            <span class="log-domain">${item.domain}</span>
            <span class="log-status">${item.method}</span>
            <div class="log-detail" id="${detailsId}" hidden>
                <div><strong>路径:</strong> <code>${item.path}</code></div>
                <div><strong>级别:</strong> ${item.level}</div>
                <div><strong>来源:</strong> ${item.source}</div>
                <div><strong>流量:</strong> ${formatBytes(item.bytes || 0)}</div>
                <div><strong>消息:</strong> ${item.message}</div>
            </div>
        `;

        row.addEventListener('click', () => {
            const detail = row.querySelector('.log-detail');
            const expanded = row.getAttribute('aria-expanded') === 'true';
            row.setAttribute('aria-expanded', expanded ? 'false' : 'true');
            if (detail) {
                detail.hidden = expanded;
            }
        });

        fragment.appendChild(row);
    });

    listEl.innerHTML = '';
    listEl.appendChild(fragment);
    listEl.scrollTop = 0;

    if (countEl) {
        countEl.textContent = `${dashboardRequestLog.entries.length} 条记录`;
    }

    updateDomainPieChartByLogs();
}

function loadChatHistory() {
    try {
        const saved = window.localStorage.getItem(chatStore.storageKey);
        const parsed = saved ? JSON.parse(saved) : [];
        chatStore.entries = Array.isArray(parsed) ? parsed.slice(0, chatStore.maxItems) : [];
    } catch (error) {
        chatStore.entries = [];
    }
}

function persistChatHistory() {
    try {
        window.localStorage.setItem(chatStore.storageKey, JSON.stringify(chatStore.entries.slice(0, chatStore.maxItems)));
    } catch (error) {
        console.warn('Persist chat history failed:', error);
    }
}

function renderChatMessages(filterKeyword = '') {
    const container = document.getElementById('chatMessages');
    if (!container) {
        return;
    }

    const keyword = String(filterKeyword || '').trim().toLowerCase();
    const list = keyword
        ? chatStore.entries.filter(item => (item.text || '').toLowerCase().includes(keyword))
        : chatStore.entries;

    if (list.length === 0) {
        container.innerHTML = `
            <div class="chat-welcome">
                <i class="bi bi-robot"></i>
                <p>${keyword ? '没有匹配的历史记录' : '所有对话数据都存储在本地'}</p>
            </div>
        `;
        return;
    }

    container.innerHTML = list.map(item => `
        <div class="chat-message ${item.role === 'user' ? 'user' : 'assistant'}">
            <div class="chat-message-meta">${item.role === 'user' ? '我' : '助手'} · ${item.time}</div>
            <div class="chat-message-text">${item.text}</div>
        </div>
    `).join('');

    container.scrollTop = container.scrollHeight;
}

function addChatMessage(role, text) {
    const safeText = String(text || '').trim();
    if (!safeText) {
        return;
    }
    const item = {
        role,
        text: safeText,
        time: new Date().toLocaleString()
    };
    chatStore.entries.push(item);
    if (chatStore.entries.length > chatStore.maxItems) {
        chatStore.entries = chatStore.entries.slice(chatStore.entries.length - chatStore.maxItems);
    }
    persistChatHistory();
    renderChatMessages();
}

function initDomainFilterControls() {
    const filterButtons = document.querySelectorAll('[data-domain-filter]');
    if (filterButtons.length === 0) {
        return;
    }

    filterButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            const value = btn.getAttribute('data-domain-filter') || 'all';
            activeDomainFilter = value;
            filterButtons.forEach(node => {
                if (node === btn) {
                    node.classList.add('active');
                } else {
                    node.classList.remove('active');
                }
            });
            updateDomainPieChartByLogs();
        });
    });
}

function initQuickChat() {
    const quickChatBtn = document.getElementById('quickChatBtn');
    const chatModalEl = document.getElementById('chatModal');
    const chatInput = document.getElementById('chatInput');
    const chatSendBtn = document.getElementById('chatSendBtn');
    const chatSearchInput = document.getElementById('chatSearchInput');
    const chatClearHistoryBtn = document.getElementById('chatClearHistoryBtn');

    if (!chatModalEl || !chatInput || !chatSendBtn) {
        return;
    }

    loadChatHistory();
    renderChatMessages();

    let modalInstance = null;
    if (window.bootstrap && window.bootstrap.Modal) {
        modalInstance = window.bootstrap.Modal.getOrCreateInstance(chatModalEl);
    }

    if (quickChatBtn) {
        quickChatBtn.addEventListener('click', () => {
            if (modalInstance) {
                modalInstance.show();
            }
            renderChatMessages(chatSearchInput ? chatSearchInput.value : '');
        });
    }

    const send = () => {
        const text = chatInput.value.trim();
        if (!text) {
            return;
        }
        addChatMessage('user', text);
        addChatMessage('assistant', `已保存到本地：${text}`);
        chatInput.value = '';
    };

    chatSendBtn.addEventListener('click', send);
    chatInput.addEventListener('keydown', event => {
        if (event.key === 'Enter') {
            event.preventDefault();
            send();
        }
    });

    if (chatSearchInput) {
        chatSearchInput.addEventListener('input', () => {
            renderChatMessages(chatSearchInput.value);
        });
    }

    if (chatClearHistoryBtn) {
        chatClearHistoryBtn.addEventListener('click', () => {
            chatStore.entries = [];
            persistChatHistory();
            renderChatMessages(chatSearchInput ? chatSearchInput.value : '');
        });
    }
}

const API_BASE = 'http://127.0.0.1:56431/api';

async function apiCall(endpoint, options = {}) {
    try {
        const response = await fetch(`${API_BASE}${endpoint}`, {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.msg || data.message || '请求失败');
        }

        return data.data || data;
    } catch (error) {
        console.error(`API call failed [${endpoint}]:`, error);
        throw error;
    }
}

async function apiGet(endpoint) {
    return apiCall(endpoint, { method: 'GET' });
}

async function apiPost(endpoint, data) {
    return apiCall(endpoint, {
        method: 'POST',
        body: JSON.stringify(data)
    });
}

// ============ 全局状态管理 ============
// 每秒更新一次全局状态
async function updateGlobalStatus() {
    try {
        // 并行调用API提高效率
        const [runStatus, rulesStatus] = await Promise.all([
            apiGet('/run/status').catch(() => ({ is_running: false, current_mode: 'unknown' })),
            apiGet('/rules/engine/status').catch(() => ({ engineEnabled: false }))
        ]);

        // 更新全局状态
        const oldRunningState = appState.runStatus.isRunning;
        const oldRuleEngineState = appState.ruleEngineStatus.isEnabled;

        // 判断服务是否运行：使用 is_running 字段
        const isRunning = runStatus.is_running || false;
        
        appState.runStatus = {
            isRunning: isRunning,
            currentMode: runStatus.current_mode || 'unknown',
            lastUpdated: Date.now()
        };

        appState.ruleEngineStatus = {
            isEnabled: rulesStatus.engineEnabled || false,
            lastUpdated: Date.now()
        };

        // 更新UI
        updateStatusIndicator();

        // 只有状态改变时才更新按钮
        if (oldRunningState !== appState.runStatus.isRunning ||
            oldRuleEngineState !== appState.ruleEngineStatus.isEnabled) {
            updateButtonStates();
        }

    } catch (error) {
        console.error('Status update failed:', error);
    }
}

// 更新状态指示器
function updateStatusIndicator() {
    const statusText = document.getElementById('statusText');
    const indicator = document.getElementById('statusIndicator');

    if (statusText) {
        const runningStatus = appState.runStatus.isRunning ? '运行中' : '已停止';
        statusText.textContent = `NonelaneCore - ${runningStatus}`;
    }

    if (indicator) {
        if (appState.runStatus.isRunning) {
            indicator.className = 'status-indicator running';
        } else {
            indicator.className = 'status-indicator stopped';
        }
    }

    // 更新模式选择按钮状态
    updateModeRadioButtons();
}

// 更新模式选择按钮状态
function updateModeRadioButtons() {
    const tunRadio = document.getElementById('runModeTun');
    const httpRadio = document.getElementById('runModeHttp');

    if (tunRadio && httpRadio && appState.runStatus.currentMode) {
        // 根据当前模式设置选中状态
        if (appState.runStatus.currentMode.toLowerCase() === 'tun') {
            tunRadio.checked = true;
            httpRadio.checked = false;
            tunRadio.parentElement.classList.add('active');
            httpRadio.parentElement.classList.remove('active');
        } else if (appState.runStatus.currentMode.toLowerCase() === 'http') {
            tunRadio.checked = false;
            httpRadio.checked = true;
            tunRadio.parentElement.classList.remove('active');
            httpRadio.parentElement.classList.add('active');
        }
    }
}

// 更新所有按钮状态
function updateButtonStates() {
    updateDashboardButtonStates();
    updateRulesButtonStates();
}

// 更新仪表板按钮状态
function updateDashboardButtonStates() {
    const dashStartBtn = document.getElementById('dashStartBtn');
    const dashStopBtn = document.getElementById('dashStopBtn');
    const runStartBtn = document.getElementById('runStartBtn');
    const runStopBtn = document.getElementById('runStopBtn');

    if (appState.runStatus.isRunning) {
        // 运行中：禁用启动按钮，启用停止按钮
        if (dashStartBtn) {
            dashStartBtn.disabled = true;
            dashStartBtn.classList.add('disabled');
        }
        if (dashStopBtn) {
            dashStopBtn.disabled = false;
            dashStopBtn.classList.remove('disabled');
        }
        if (runStartBtn) {
            runStartBtn.disabled = true;
            runStartBtn.classList.add('disabled');
        }
        if (runStopBtn) {
            runStopBtn.disabled = false;
            runStopBtn.classList.remove('disabled');
        }
    } else {
        // 已停止：启用启动按钮，禁用停止按钮
        if (dashStartBtn) {
            dashStartBtn.disabled = false;
            dashStartBtn.classList.remove('disabled');
        }
        if (dashStopBtn) {
            dashStopBtn.disabled = true;
            dashStopBtn.classList.add('disabled');
        }
        if (runStartBtn) {
            runStartBtn.disabled = false;
            runStartBtn.classList.remove('disabled');
        }
        if (runStopBtn) {
            runStopBtn.disabled = true;
            runStopBtn.classList.add('disabled');
        }
    }
}

// 更新规则引擎按钮状态
function updateRulesButtonStates() {
    const rulesEnableBtn = document.getElementById('rulesEnableBtn');
    const rulesDisableBtn = document.getElementById('rulesDisableBtn');

    if (rulesEnableBtn && rulesDisableBtn) {
        if (appState.ruleEngineStatus.isEnabled) {
            // 已启用：禁用启用按钮，启用禁用按钮
            rulesEnableBtn.disabled = true;
            rulesEnableBtn.classList.add('disabled');
            rulesDisableBtn.disabled = false;
            rulesDisableBtn.classList.remove('disabled');
        } else {
            // 已禁用：启用启用按钮，禁用禁用按钮
            rulesEnableBtn.disabled = false;
            rulesEnableBtn.classList.remove('disabled');
            rulesDisableBtn.disabled = true;
            rulesDisableBtn.classList.add('disabled');
        }
    }
}

// ============ 通知功能 ============
function showNotification(message, type = 'success') {
    // 确保通知容器存在
    let container = document.getElementById('notification-container');
    if (!container) {
        container = document.createElement('div');
        container.id = 'notification-container';
        document.body.appendChild(container);
    }

    const alertClass = type === 'success' ? 'alert-success' : 'alert-danger';
    const notificationId = 'notification-' + Date.now();
    const alertHtml = `
        <div class="alert ${alertClass} alert-dismissible fade show notification" role="alert" id="${notificationId}">
            ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        </div>
    `;

    container.insertAdjacentHTML('beforeend', alertHtml);

    // 自动移除通知
    setTimeout(() => {
        const alert = document.getElementById(notificationId);
        if (alert) {
            // 添加淡出动画
            alert.classList.add('removing');
            // 等待动画完成后移除元素
            setTimeout(() => alert.remove(), 300); // 等待淡出动画完成
        }
    }, 5000);
}

function showSuccess(message) {
    showNotification(message, 'success');
}

function showError(message) {
    showNotification(message, 'error');
}

// ============ 加载状态管理 ============
function showLoading(element) {
    element.classList.add('loading');
    element.disabled = true;
}

function hideLoading(element) {
    element.classList.remove('loading');
    element.disabled = false;
}

// ============ 模态加载框 ============
let modalLoadingInstance = null;

/**
 * 显示模态加载框
 * @param {string} message - 提示文字 (默认: "加载中，请稍候...")
 */
function showModalLoading(message = '加载中，请稍候...') {
    // 防止重复创建
    if (modalLoadingInstance) {
        return;
    }

    // 创建DOM结构
    const overlay = document.createElement('div');
    overlay.id = 'modal-loading-overlay';
    overlay.className = 'modal-loading-overlay';
    overlay.innerHTML = `
        <div class="modal-loading-container">
            <div class="modal-loading-spinner"></div>
            <div class="modal-loading-text">${message}</div>
        </div>
    `;

    // 插入到body
    document.body.appendChild(overlay);
    modalLoadingInstance = overlay;

    // 防止背景滚动
    document.body.style.overflow = 'hidden';
}

/**
 * 隐藏模态加载框
 */
function hideModalLoading() {
    if (!modalLoadingInstance) {
        return;
    }

    // 添加淡出动画
    modalLoadingInstance.classList.add('fade-out');

    // 等待动画完成后移除
    setTimeout(() => {
        if (modalLoadingInstance && modalLoadingInstance.parentNode) {
            modalLoadingInstance.parentNode.removeChild(modalLoadingInstance);
            modalLoadingInstance = null;
            document.body.style.overflow = '';
        }
    }, 200); // 与CSS动画时长一致
}

// 格式化日期时间
function formatDateTime(isoString) {
    try {
        const date = new Date(isoString);
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
    } catch (error) {
        return isoString || '-';
    }
}

// 格式化相对时间（从Unix时间戳）
function formatRelativeTime(timestamp) {
    if (!timestamp || timestamp === 0) return '未测试';

    const now = Math.floor(Date.now() / 1000);
    const diff = now - parseInt(timestamp);

    if (diff < 0) return '刚才';
    if (diff < 60) return '刚才';
    if (diff < 3600) return Math.floor(diff / 60) + '分钟前';
    if (diff < 86400) return Math.floor(diff / 3600) + '小时前';
    if (diff < 2592000) return Math.floor(diff / 86400) + '天前';
    return Math.floor(diff / 2592000) + '月前';
}

// ============ 仪表板功能 ============
async function loadDashboard() {
    try {
        const [runStatus, proxyList, doorMembers, rulesStatus] = await Promise.all([
            apiGet('/run/status'),
            apiGet('/proxy/list'),
            apiGet('/proxy/door/members').catch(() => ({ members: [] })),
            apiGet('/rules/engine/status').catch(() => null)
        ]);

        // 获取当前代理，如果失败则为 null
        let currentProxyDisplay = '-';
        try {
            const currentProxy = await apiGet('/proxy/current/get');
            if (currentProxy && currentProxy.name && !currentProxy.error) {
                // 如果有 show_name，使用它，否则使用 name
                currentProxyDisplay = currentProxy.show_name || currentProxy.name;
            }
        } catch (error) {
            console.log('当前代理未设置:', error.message);
        }

        // 判断服务是否运行：使用 is_running 字段
        const isRunning = runStatus.is_running || false;
        
        // 更新全局状态
        appState.runStatus.isRunning = isRunning;
        appState.runStatus.currentMode = runStatus.current_mode || 'unknown';
        appState.runStatus.lastUpdated = Date.now();

        // 更新仪表板
        let runningStatusText = '';
        if (isRunning) {
            if (runStatus.current_mode === 'tun') {
                runningStatusText = 'TUN 运行中';
            } else if (runStatus.current_mode === 'http') {
                runningStatusText = 'HTTP 运行中';
            } else {
                runningStatusText = '运行中';
            }
        } else {
            runningStatusText = runStatus.current_mode === 'tun' ? 'TUN 模式（未启动）' : 'HTTP 模式（未启动）';
        }
        
        document.getElementById('dashRunStatus').textContent = runningStatusText;
        document.getElementById('dashCurrentProxy').textContent = currentProxyDisplay;
        document.getElementById('dashRunMode').textContent = runStatus.current_mode || '-';
        document.getElementById('dashRuleStatus').textContent = rulesStatus?.engineEnabled ? '启用' : '禁用';
        // proxyList 返回格式是 { proxies: {...}, count: ... }
        const proxyCount = proxyList?.proxies ? Object.keys(proxyList.proxies).length : (proxyList?.count || 0);
        document.getElementById('dashProxyCount').textContent = proxyCount;
        document.getElementById('dashDoorCount').textContent = doorMembers?.members?.length || 0;
        document.getElementById('dashLastUpdate').textContent = new Date().toLocaleTimeString();

        // 更新顶部状态指示器
        const indicator = document.getElementById('statusIndicator');
        const statusText = document.getElementById('statusText');
        if (isRunning) {
            indicator.className = 'status-indicator running';
            statusText.textContent = 'NonelaneCore - 运行中';
        } else {
            indicator.className = 'status-indicator stopped';
            statusText.textContent = 'NonelaneCore - 已停止';
        }
        
        // 更新按钮状态
        updateButtonStates();

        // 加载统计数据
        await loadStatsData().catch(error => console.error('Failed to load stats:', error));

        // 初始化图表
        initChart();
    } catch (error) {
        console.error('加载仪表板数据失败:', error);
    }
}

document.getElementById('dashStartBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    apiPost('/run/start')
        .then((result) => {
            // 如果返回成功，立即更新状态
            if (result && (result.status === 'success' || result.status === 'already_running')) {
                // 根据返回结果判断服务已启动
                appState.runStatus.isRunning = true;
                updateStatusIndicator();
                updateButtonStates();
                showSuccess(result.message || '服务启动成功');
            } else {
                showSuccess('服务启动成功');
            }
            // 延迟一点再刷新，确保后端状态已更新
            setTimeout(() => {
                loadDashboard();
                loadRunStatus();
            }, 500);
        })
        .catch(error => {
            showError('启动失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

document.getElementById('dashStopBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    apiPost('/run/stop')
        .then((result) => {
            // 如果返回成功，立即更新状态
            if (result && result.status === 'success') {
                appState.runStatus.isRunning = false;
                updateStatusIndicator();
                updateButtonStates();
                showSuccess(result.message || '服务停止成功');
            } else {
                showSuccess('服务停止成功');
            }
            // 延迟一点再刷新，确保后端状态已更新
            setTimeout(() => {
                loadDashboard();
                loadRunStatus();
            }, 500);
        })
        .catch(error => {
            showError('停止失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

// ============ 延迟测试功能 ============
async function testProxyLatency() {
    const btn = document.getElementById('testLatencyBtn');
    if (!btn) return;

    const originalText = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>测试中...';

    try {
        const result = await apiPost('/proxy/door/test-latency', {});

        // 处理"testing"状态响应
        if (result.status === 'testing') {
            btn.innerHTML = '<i class="bi bi-hourglass-split"></i> 测试进行中';
            btn.classList.add('btn-warning');
            btn.classList.remove('btn-info');
            setTimeout(() => {
                btn.innerHTML = originalText;
                btn.classList.remove('btn-warning');
                btn.classList.add('btn-info');
                btn.disabled = false;
            }, 3000);
            return;
        }

        // 测试完成，刷新代理数据
        showSuccess('延迟测试完成');
        await loadProxyData();

        btn.innerHTML = '<i class="bi bi-check-circle"></i> 测试完成';
        btn.classList.add('btn-danger');
        btn.classList.remove('btn-info');

        setTimeout(() => {
            btn.innerHTML = originalText;
            btn.classList.remove('btn-danger');
            btn.classList.add('btn-info');
            btn.disabled = false;
        }, 2000);

    } catch (error) {
        console.error('Latency test failed:', error);
        showError('延迟测试失败: ' + error.message);
        btn.innerHTML = '<i class="bi bi-exclamation-circle"></i> 测试失败';
        btn.classList.add('btn-danger');
        btn.classList.remove('btn-info');

        setTimeout(() => {
            btn.innerHTML = originalText;
            btn.classList.remove('btn-danger');
            btn.classList.add('btn-info');
            btn.disabled = false;
        }, 3000);
    }
}

// ============ 代理管理功能 ============
async function loadProxyData() {
    try {
        // 从单一数据源获取所有代理（包括door:xxx虚拟成员）+ 当前代理信息
        const proxyListData = await apiGet('/proxy/list');

        // 从响应中提取当前代理信息和自动模式标识
        let currentProxy = null;
        let isDoorAutoMode = false;
        let currentProxyInfo = null;

        if (proxyListData) {
            // 从新的响应结构中获取当前代理信息
            if (proxyListData.current_proxy) {
                currentProxy = proxyListData.current_proxy.name;
                currentProxyInfo = proxyListData.current_proxy;
            }
            // 获取door自动模式标识
            isDoorAutoMode = proxyListData.is_door_auto_mode || false;
        }

        // 处理代理列表数据 - 直接使用 /api/proxy/list 的结果（包含所有代理）
        let allProxies = [];
        if (proxyListData) {
            if (proxyListData.proxies && typeof proxyListData.proxies === 'object') {
                // 如果是对象格式 { name: {...}, ...}，转换为数组
                allProxies = Object.keys(proxyListData.proxies).map(name => ({
                    name: name,
                    ...proxyListData.proxies[name]
                }));
            } else if (Array.isArray(proxyListData.proxies)) {
                // 如果已经是数组
                allProxies = proxyListData.proxies;
            }
        }

        console.log('完整代理列表:', allProxies);
        console.log('当前代理:', currentProxy);
        console.log('Door自动模式:', isDoorAutoMode);

        // 使用当前代理作为selectedProxy
        let selectedProxy = currentProxy;

        // 提取door成员（名称以"door:"开头）用于下拉框显示
        const doorProxies = allProxies.filter(proxy => proxy.name && proxy.name.startsWith('door:'));

        // 更新当前代理选择框 - 只显示door成员
        const select = document.getElementById('proxySelect');
        select.innerHTML = '';
        if (doorProxies.length === 0) {
            select.innerHTML = '<option>暂无 Door 成员</option>';
        } else {
            // 如果是自动模式，添加提示
            const autoModeHint = isDoorAutoMode ? ' (自动模式)' : '';
            doorProxies.forEach(proxy => {
                const option = document.createElement('option');
                option.value = proxy.name;
                // 显示时去掉"door:"前缀
                const displayName = proxy.name.substring(5);
                option.textContent = displayName + autoModeHint;
                if (proxy.name === selectedProxy) {
                    option.selected = true;
                }
                select.appendChild(option);
            });
        }

        // 显示当前代理信息
        const currentProxyDisplay = document.getElementById('currentProxyDisplay');
        if (currentProxyDisplay && currentProxyInfo) {
            const displayName = currentProxyInfo.show_name || currentProxyInfo.name;
            const autoModeLabel = isDoorAutoMode && currentProxyInfo.name.startsWith('door:')
                ? ' <span class="badge bg-info ms-2">自动模式</span>'
                : '';
            currentProxyDisplay.innerHTML = `当前代理: <strong>${displayName}</strong>${autoModeLabel}`;
        }

        // 更新所有代理表格
        const tbody = document.getElementById('proxyTableBody');
        if (allProxies.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">暂无代理</td></tr>';
        } else {
            tbody.innerHTML = allProxies.map(proxy => {
                // 判断是否是当前代理
                const isCurrent = proxy.name === selectedProxy;
                // 显示名称：door成员时隐藏"door:"前缀
                const displayName = proxy.name && proxy.name.startsWith('door:')
                    ? proxy.name.substring(5)
                    : proxy.name;

                // 处理延迟显示
                const latency = proxy.latency;
                let latencyDisplay = '';
                let latencyClass = '';

                if (latency === -1) {
                    latencyClass = 'latency-failed';
                    latencyDisplay = '<span class="' + latencyClass + '">失败</span>';
                } else if (latency === 0 || !latency) {
                    latencyClass = 'latency-unknown';
                    latencyDisplay = '<span class="' + latencyClass + '">未测试</span>';
                } else if (latency < 50) {
                    latencyClass = 'latency-excellent';
                    latencyDisplay = '<span class="' + latencyClass + '">' + latency + 'ms</span>';
                } else if (latency < 150) {
                    latencyClass = 'latency-good';
                    latencyDisplay = '<span class="' + latencyClass + '">' + latency + 'ms</span>';
                } else if (latency < 300) {
                    latencyClass = 'latency-normal';
                    latencyDisplay = '<span class="' + latencyClass + '">' + latency + 'ms</span>';
                } else {
                    latencyClass = 'latency-slow';
                    latencyDisplay = '<span class="' + latencyClass + '">' + latency + 'ms</span>';
                }

                // 最后更新时间
                const lastUpdateTime = formatRelativeTime(proxy.last_update);

                return `
                    <tr>
                        <td>${displayName || '-'}</td>
                        <td><span class="badge bg-primary">${proxy.type || 'Unknown'}</span></td>
                        <td>${proxy.addr || '-'}</td>
                        <td class="text-center">${latencyDisplay} <small class="text-muted">(${lastUpdateTime})</small></td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" onclick="switchProxy('${proxy.name}')">
                                ${isCurrent ? '✓ 当前' : '切换'}
                            </button>
                        </td>
                        <td>
                            <button class="btn btn-sm btn-outline-info" onclick="showProxyDetail('${proxy.name}')">
                                <i class="bi bi-info-circle"></i> 详情
                            </button>
                        </td>
                    </tr>
                `;
            }).join('');
        }

        appState.proxies = allProxies;
        appState.currentProxy = selectedProxy;
    } catch (error) {
        console.error('加载代理数据失败:', error);
        document.getElementById('proxyTableBody').innerHTML = '<tr><td colspan="5" class="text-center text-danger">加载失败: ' + error.message + '</td></tr>';
    }
}

// 显示代理详情
async function showProxyDetail(proxyName) {
    try {
        const response = await apiGet(`/proxy/get?name=${encodeURIComponent(proxyName)}`);

        // 更新Modal标题 - 去掉door:前缀显示
        const displayName = proxyName.startsWith('door:') ? proxyName.substring(5) : proxyName;
        document.getElementById('proxyDetailTitle').textContent = displayName;

        // 格式化JSON并显示
        const jsonStr = JSON.stringify(response, null, 2);
        document.getElementById('proxyDetailJson').textContent = jsonStr;

        // 显示Modal
        const modal = new bootstrap.Modal(document.getElementById('proxyDetailModal'));
        modal.show();
    } catch (error) {
        showError('获取代理详情失败: ' + error.message);
    }
}

// 复制代理详情JSON到剪贴板
document.getElementById('copyProxyDetailBtn')?.addEventListener('click', () => {
    const jsonText = document.getElementById('proxyDetailJson').textContent;
    navigator.clipboard.writeText(jsonText)
        .then(() => {
            showSuccess('JSON已复制到剪贴板');
        })
        .catch(error => {
            showError('复制失败: ' + error.message);
        });
});

async function switchProxy(proxyName) {
    try {
        console.log('正在切换��理到:', proxyName);

        // 所有代理切换都使用 current/set API
        await apiPost('/proxy/current/set', { name: proxyName });
        showSuccess(`已切换到 ${proxyName}`);

        appState.currentProxy = proxyName;
        loadProxyData();
        loadDashboard();
    } catch (error) {
        console.error('切换代理失败:', error);
        showError('切换代理失败');
    }
}

async function switchDoorMember(memberName) {
    try {
        console.log('正在切换 Door 成员到:', memberName);
        // 使用 door:memberName 格式调用 current/set API
        await apiPost('/proxy/current/set', { name: `door:${memberName}` });
        showSuccess(`已切换到 ${memberName}`);
        loadProxyData();
        loadDashboard();
    } catch (error) {
        console.error('切换 Door 成员失败:', error);
        showError('切换 Door 成员失败');
    }
}

document.getElementById('switchProxyBtn').addEventListener('click', () => {
    const proxyName = document.getElementById('proxySelect').value;
    if (proxyName) {
        switchProxy(proxyName);
    }
});

// 延迟测试按钮事件监听
if (document.getElementById('testLatencyBtn')) {
    document.getElementById('testLatencyBtn').addEventListener('click', testProxyLatency);
}

// ============ 运行控制功能 ============
async function loadRunStatus() {
    try {
        const status = await apiGet('/run/status');
        
        // 判断服务是否运行：使用 is_running 字段
        const isRunning = status.is_running || false;
        
        document.getElementById('runCurrentMode').textContent = status.current_mode || '-';
        document.getElementById('runServiceStatus').textContent = isRunning ? '运行中' : '已停止';
        document.getElementById('runAvailableModes').textContent = status.available_modes?.join(' / ') || '-';
        document.getElementById('runStatusInfo').textContent = status.status || status.description || '-';
        
        // 更新全局状态
        appState.runStatus.isRunning = isRunning;
        appState.runStatus.currentMode = status.current_mode || 'unknown';
        
        // 更新按钮状态
        updateButtonStates();
    } catch (error) {
        console.error('加载运行状态失败:', error);
    }
}

document.getElementById('runStartBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    apiPost('/run/start')
        .then(() => {
            showSuccess('服务启动成功');
            loadRunStatus();
            loadDashboard();
        })
        .catch(error => {
            showError('启动失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

document.getElementById('runStopBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    apiPost('/run/stop')
        .then(() => {
            showSuccess('服务停止成功');
            loadRunStatus();
            loadDashboard();
        })
        .catch(error => {
            showError('停止失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

document.getElementById('runModeBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    const currentMode = document.querySelector('input[name="runMode"]:checked').value;

    var mode = currentMode;
    if (currentMode === "http") {
        mode = "tun";
    } else {
        mode = "http";
    }
    // 显示模态加载框（替换原来的 showLoading）
    showModalLoading(`正在切换到 ${mode} 模式，请稍候...`);
    
    apiPost('/run/swift', { mode: mode })
        .then(() => {
            hideModalLoading();
            showSuccess(`已切换到 ${mode} 模式`);
            loadRunStatus();
            loadDashboard();
        })
        .catch(error => {
            hideModalLoading();
            showError('模式切换失败: ' + error.message);
        })
        .finally(() => {
            btn.disabled = false;
        });
});


// ============ 规则引擎功能 ============
async function loadRulesData() {
    try {
        // 1. 获取规则引擎状态（包含缓存统计）
        const statusData = await apiGet('/rules/engine/status');

        // 使用后端返回的字段名 engineEnabled 和 geoipEnabled
        const isEnabled = statusData.engineEnabled || false;
        const geoipEnabled = statusData.geoipEnabled || false;

        // 更新全局状态
        appState.ruleEngineStatus.isEnabled = isEnabled;
        appState.ruleEngineStatus.lastUpdated = Date.now();

        // 更新规则引擎状态显示
        const statusBadge = document.getElementById('rules-status-badge');
        const statusText = document.getElementById('rules-status-text');

        if (statusBadge && statusText) {
            if (isEnabled) {
                statusBadge.className = 'badge bg-danger';
                statusText.textContent = '启用';
            } else {
                statusBadge.className = 'badge bg-danger';
                statusText.textContent = '禁用';
            }
        }

        // GeoIP 状态信息已包含在 statusData 中，可在需要时使用

        // 2. 更新缓存统计信息（优先使用 statusData 中的 cache，如果没有则单独请求）
        const cacheData = statusData.cache || null;
        if (cacheData) {
            updateRulesStatistics(cacheData);
        } else {
            // 如果 statusData 中没有 cache，尝试单独获取
            try {
                const cacheStats = await apiGet('/rules/cache/stats');
                if (cacheStats) {
                    updateRulesStatistics(cacheStats);
                }
            } catch (error) {
                console.warn('获取缓存统计失败:', error);
            }
        }

        // 3. 更新按钮状态
        updateRulesButtonStates();

    } catch (error) {
        console.error('加载规则数据失败:', error);
        showError('加载规则引擎数据失败: ' + error.message);

        // 错误状态下也显示基本信息
        const statusElement = document.getElementById('rulesStatus');
        const statusBadge = document.getElementById('rules-status-badge');
        const statusText = document.getElementById('rules-status-text');

        if (statusElement) {
            statusElement.textContent = '未知';
        }

        if (statusBadge && statusText) {
            statusBadge.className = 'badge bg-secondary';
            statusText.textContent = '未知';
        }
    }
}

// 更新规则引擎统计信息
function updateRulesStatistics(cacheData) {
    // 更新缓存命中率
    const cacheHitRateElement = document.getElementById('cache-hit-rate');
    const totalCacheElement = document.getElementById('total-cache');
    const cacheSizeElement = document.getElementById('rulesCacheSize');
    const cacheCountElement = document.getElementById('rulesCacheCount');
    const lastUpdateElement = document.getElementById('cache-last-update');

    if (cacheData) {
        const cacheHits = cacheData.cache_hits || 0;
        const cacheMisses = cacheData.cache_misses || 0;
        const totalHits = cacheHits + cacheMisses;
        const hitRate = totalHits > 0 ? (cacheHits / totalHits * 100).toFixed(2) : 0;

        if (cacheHitRateElement) {
            cacheHitRateElement.textContent = hitRate + '%';
        }

        if (totalCacheElement) {
            totalCacheElement.textContent = totalHits.toLocaleString();
        }

        // 兼容旧的缓存显示
        if (cacheSizeElement) {
            cacheSizeElement.textContent = cacheData.size || 0;
        }

        if (cacheCountElement) {
            cacheCountElement.textContent = cacheData.count || 0;
        }

        if (lastUpdateElement) {
            lastUpdateElement.textContent = new Date().toLocaleTimeString();
        }
    }
}

// 加载路由配置
async function loadRoutingConfig() {
    try {
        const config = await apiGet('/config/routing');
        if (config) {
            populateRulesUI(config);
        }
    } catch (error) {
        console.error('加载路由配置失败:', error);
        // 如果失败，可能是Nacos未配置或为空，使用默认配置
        populateRulesUI({
            to_door: { rules: [] },
            black_list: { rules: [] },
            none_lane: { rules: [] },
            settings: { geoip_enabled: false, none_lane_enabled: false }
        });
    }
}

// 填充规则UI
function populateRulesUI(config) {
    // 更新全局设置开关
    const geoipSwitch = document.getElementById('geoipEnabledSwitch');
    const nonelaneSwitch = document.getElementById('nonelaneEnabledSwitch');

    if (geoipSwitch) {
        geoipSwitch.checked = config.settings?.geoip_enabled || false;
    }
    if (nonelaneSwitch) {
        nonelaneSwitch.checked = config.settings?.none_lane_enabled || false;
    }

    // 填充To Door规则
    populateRuleTable('toDoorRulesBody', config.to_door?.rules || [], 'to_door');

    // 填充黑名单规则
    populateRuleTable('blacklistRulesBody', config.black_list?.rules || [], 'black_list');

    // 填充NoneLane规则
    populateRuleTable('nonelaneRulesBody', config.none_lane?.rules || [], 'none_lane');

    // 保存配置到appState供后续使用
    appState.routingConfig = config;
}

// 填充规则表格
function populateRuleTable(tableBodyId, rules, ruleSet) {
    const tbody = document.getElementById(tableBodyId);
    if (!tbody) return;

    if (!rules || rules.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="text-center text-muted py-3">暂无规则</td></tr>';
        return;
    }

    tbody.innerHTML = rules.map((rule, index) => {
        const typeText = getTypeText(rule.type);
        const enabledChecked = rule.enabled ? 'checked' : '';

        return `
            <tr data-rule-id="${rule.id}">
                <td><span class="badge bg-info">${typeText}</span></td>
                <td><code>${escapeHtml(rule.condition)}</code></td>
                <td class="text-center">
                    <div class="form-check form-switch d-flex justify-content-center">
                        <input class="form-check-input rule-toggle" type="checkbox" ${enabledChecked}
                               data-rule-id="${rule.id}" data-rule-set="${ruleSet}">
                    </div>
                </td>
                <td class="text-center">
                    <button class="btn btn-sm btn-outline-primary edit-rule-btn"
                            data-rule-id="${rule.id}" data-rule-set="${ruleSet}">
                        <i class="bi bi-pencil"></i>
                    </button>
                    <button class="btn btn-sm btn-outline-danger delete-rule-btn ms-1"
                            data-rule-id="${rule.id}" data-rule-set="${ruleSet}">
                        <i class="bi bi-trash"></i>
                    </button>
                </td>
            </tr>
        `;
    }).join('');

    // 绑定事件监听器
    bindRuleTableEvents(tbody, ruleSet);
}

// 绑定规则表格事件
function bindRuleTableEvents(tbody, ruleSet) {
    // 启用/禁用切换
    tbody.querySelectorAll('.rule-toggle').forEach(toggle => {
        toggle.addEventListener('change', async (e) => {
            const ruleId = e.target.dataset.ruleId;
            const enabled = e.target.checked;
            try {
                await apiCall(`/config/routing/rules/${ruleId}/toggle`, {
                    method: 'PUT',
                    body: JSON.stringify({ enabled })
                });
                showSuccess('规则状态已更新');
            } catch (error) {
                showError('更新规则状态失败: ' + error.message);
                e.target.checked = !enabled; // 回滚
            }
        });
    });

    // 编辑按钮
    tbody.querySelectorAll('.edit-rule-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const ruleId = e.currentTarget.dataset.ruleId;
            const ruleSet = e.currentTarget.dataset.ruleSet;
            editRule(ruleId, ruleSet);
        });
    });

    // 删除按钮
    tbody.querySelectorAll('.delete-rule-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const ruleId = e.currentTarget.dataset.ruleId;
            const ruleSet = e.currentTarget.dataset.ruleSet;
            deleteRule(ruleId, ruleSet);
        });
    });
}

// 获取类型文本
function getTypeText(type) {
    const typeMap = {
        'domain': '域名',
        'ip': 'IP段',
        'geoip': 'GeoIP'
    };
    return typeMap[type] || type;
}

// HTML转义函数
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 生成规则ID
function generateRuleId(type) {
    const timestamp = Math.floor(Date.now() / 1000);
    return `rule_${type}_${timestamp}`;
}

// 添加规则
function addRule(ruleSet) {
    const modal = new bootstrap.Modal(document.getElementById('ruleEditModal'));

    // 清空表单
    document.getElementById('ruleTypeSelect').value = 'domain';
    document.getElementById('ruleConditionInput').value = '';
    document.getElementById('ruleEnabledCheckbox').checked = true;
    document.getElementById('ruleIdInput').value = ''; // 空ID表示新建
    document.getElementById('ruleSetInput').value = ruleSet;

    // 更新模态框标题
    document.getElementById('ruleEditModalTitle').textContent = '添加规则';

    // 显示模态框
    modal.show();
}

// 编辑规则
function editRule(ruleId, ruleSet) {
    // 从配置中找到规则
    const config = appState.routingConfig;
    if (!config) {
        showError('配置未加载');
        return;
    }

    let rule = null;
    const ruleSetMap = {
        'to_door': config.to_door?.rules || [],
        'black_list': config.black_list?.rules || [],
        'none_lane': config.none_lane?.rules || []
    };

    const rules = ruleSetMap[ruleSet];
    if (rules) {
        rule = rules.find(r => r.id === ruleId);
    }

    if (!rule) {
        showError('规则未找到');
        return;
    }

    // 填充表单
    document.getElementById('ruleTypeSelect').value = rule.type;
    document.getElementById('ruleConditionInput').value = rule.condition;
    document.getElementById('ruleEnabledCheckbox').checked = rule.enabled;
    document.getElementById('ruleIdInput').value = rule.id;
    document.getElementById('ruleSetInput').value = ruleSet;

    // 更新模态框标题
    document.getElementById('ruleEditModalTitle').textContent = '编辑规则';

    // 显示模态框
    const modal = new bootstrap.Modal(document.getElementById('ruleEditModal'));
    modal.show();
}

// 删除规则
async function deleteRule(ruleId, ruleSet) {
    if (!confirm('确定要删除此规则吗？')) {
        return;
    }

    try {
        const config = appState.routingConfig;
        if (!config) {
            showError('配置未加载');
            return;
        }

        // 从配置中移除规则
        const ruleSetMap = {
            'to_door': config.to_door,
            'black_list': config.black_list,
            'none_lane': config.none_lane
        };

        const targetSet = ruleSetMap[ruleSet];
        if (targetSet && targetSet.rules) {
            targetSet.rules = targetSet.rules.filter(r => r.id !== ruleId);
        }

        // 保存配置
        await apiPost('/config/routing', config);
        showSuccess('规则已删除');

        // 重新加载配置
        loadRoutingConfig();
    } catch (error) {
        showError('删除规则失败: ' + error.message);
    }
}

// 验证规则
function validateRule(type, condition) {
    if (!condition || condition.trim() === '') {
        return '条件不能为空';
    }

    const trimmed = condition.trim();

    switch (type) {
        case 'domain':
            // 域名格式：允许通配符 *.example.com 或 example.com
            if (!/^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/.test(trimmed)) {
                return '域名格式无效（例: *.google.com 或 example.com）';
            }
            break;

        case 'ip': {
            // CIDR格式：192.168.0.0/16
            const cidrRegex = /^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/;
            if (!cidrRegex.test(trimmed)) {
                return 'IP段格式无效（例: 192.168.0.0/16）';
            }
            // 验证IP范围
            const parts = trimmed.split('/');
            const ipParts = parts[0].split('.');
            if (ipParts.some(p => parseInt(p, 10) > 255)) {
                return 'IP地址范围无效';
            }
            const prefix = parseInt(parts[1], 10);
            if (prefix < 0 || prefix > 32) {
                return '子网掩码范围无效（0-32）';
            }
            break;
        }

        case 'geoip':
            // 国家代码：2位大写字母
            if (!/^[A-Z]{2}$/.test(trimmed)) {
                return 'GeoIP格式无效（例: US, CN）';
            }
            break;

        default:
            return '未知的规则类型';
    }

    return null; // 验证通过
}

// 保存规则（来自模态框）
async function saveRuleFromModal() {
    const ruleId = document.getElementById('ruleIdInput').value;
    const ruleSet = document.getElementById('ruleSetInput').value;
    const type = document.getElementById('ruleTypeSelect').value;
    const condition = document.getElementById('ruleConditionInput').value.trim();
    const enabled = document.getElementById('ruleEnabledCheckbox').checked;

    // 验证输入
    const validationError = validateRule(type, condition);
    if (validationError) {
        const errorDiv = document.getElementById('conditionError');
        errorDiv.textContent = validationError;
        errorDiv.style.display = 'block';
        document.getElementById('ruleConditionInput').classList.add('is-invalid');
        return;
    }

    // 清除验证错误
    document.getElementById('conditionError').style.display = 'none';
    document.getElementById('ruleConditionInput').classList.remove('is-invalid');

    try {
        const config = appState.routingConfig;
        if (!config) {
            showError('配置未加载');
            return;
        }

        // 准备规则对象
        const rule = {
            id: ruleId || generateRuleId(type),
            type: type,
            condition: condition,
            enabled: enabled,
            created_at: new Date().toISOString()
        };

        // 获取目标规则集
        const ruleSetMap = {
            'to_door': config.to_door,
            'black_list': config.black_list,
            'none_lane': config.none_lane
        };

        const targetSet = ruleSetMap[ruleSet];
        if (!targetSet) {
            showError('规则集未找到');
            return;
        }

        if (!targetSet.rules) {
            targetSet.rules = [];
        }

        if (ruleId) {
            // 编辑现有规则
            const index = targetSet.rules.findIndex(r => r.id === ruleId);
            if (index !== -1) {
                targetSet.rules[index] = rule;
            }
        } else {
            // 添加新规则
            targetSet.rules.push(rule);
        }

        // 保存配置到后端
        await apiPost('/config/routing', config);
        showSuccess(ruleId ? '规则已更新' : '规则已添加');

        // 关闭模态框
        const modal = bootstrap.Modal.getInstance(document.getElementById('ruleEditModal'));
        modal.hide();

        // 重新加载配置
        loadRoutingConfig();
    } catch (error) {
        showError('保存规则失败: ' + error.message);
    }
}

// 更新类型选择的帮助文本
function updateRuleTypeHelpText() {
    const type = document.getElementById('ruleTypeSelect').value;
    const helpText = document.getElementById('typeHelpText');

    const helpMap = {
        'domain': '示例: *.google.com 或 example.com',
        'ip': '示例: 192.168.0.0/16 或 10.0.0.0/8',
        'geoip': '示例: US (美国), CN (中国), JP (日本) - 使用ISO 3166-1 alpha-2代码'
    };

    helpText.textContent = helpMap[type] || '';
}

// 保存全局配置
async function saveGlobalRoutingConfig() {
    try {
        const config = appState.routingConfig;
        if (!config) {
            showError('配置未加载');
            return;
        }

        // 更新全局设置
        if (!config.settings) {
            config.settings = {};
        }

        config.settings.geoip_enabled = document.getElementById('geoipEnabledSwitch').checked;
        config.settings.none_lane_enabled = document.getElementById('nonelaneEnabledSwitch').checked;

        // 保存到后端
        await apiPost('/config/routing', config);
        showSuccess('配置已保存');

        // 重新加载配置
        loadRoutingConfig();
    } catch (error) {
        showError('保存配置失败: ' + error.message);
    }
}

// 事件监听器 - 添加规则按钮
document.addEventListener('DOMContentLoaded', () => {
    const addToDoorBtn = document.getElementById('addToDoorRuleBtn');
    const addBlacklistBtn = document.getElementById('addBlacklistRuleBtn');
    const addNonelaneBtn = document.getElementById('addNonelaneRuleBtn');

    if (addToDoorBtn) {
        addToDoorBtn.addEventListener('click', () => addRule('to_door'));
    }
    if (addBlacklistBtn) {
        addBlacklistBtn.addEventListener('click', () => addRule('black_list'));
    }
    if (addNonelaneBtn) {
        addNonelaneBtn.addEventListener('click', () => addRule('none_lane'));
    }

    // 规则编辑模态框保存按钮
    const ruleEditSaveBtn = document.getElementById('ruleEditSaveBtn');
    if (ruleEditSaveBtn) {
        ruleEditSaveBtn.addEventListener('click', saveRuleFromModal);
    }

    // 规则类型选择变化
    const ruleTypeSelect = document.getElementById('ruleTypeSelect');
    if (ruleTypeSelect) {
        ruleTypeSelect.addEventListener('change', updateRuleTypeHelpText);
        // 初始化帮助文本
        updateRuleTypeHelpText();
    }

    // 全局配置保存按钮
    const rulesConfigSaveBtn = document.getElementById('rulesConfigSaveBtn');
    if (rulesConfigSaveBtn) {
        rulesConfigSaveBtn.addEventListener('click', saveGlobalRoutingConfig);
    }
});

document.getElementById('rulesEnableBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    apiPost('/rules/engine/enable')
        .then(() => {
            showSuccess('规则引擎已启用');
            loadRulesData();
        })
        .catch(error => {
            showError('启用失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

document.getElementById('rulesDisableBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    apiPost('/rules/engine/disable')
        .then(() => {
            showSuccess('规则引擎已禁用');
            loadRulesData();
        })
        .catch(error => {
            showError('禁用失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

document.getElementById('rulesReloadBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    // 注意：routes.go 中没有 reload 路由，暂时禁用此功能或提示用户
    showError('重新加载功能暂未实现');
    hideLoading(btn);

    // 如果将来后端添加了 reload 路由，可以使用：
    // apiPost('/api/rules/reload')
    //     .then(() => {
    //         showSuccess('规则已重新加载');
    //         loadRulesData();
    //     })
    //     .catch(error => {
    //         showError('重新加载失败: ' + error.message);
    //     })
    //     .finally(() => {
    //         hideLoading(btn);
    //     });
});

document.getElementById('rulesClearCacheBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    apiPost('/rules/cache/clear')
        .then(() => {
            showSuccess('缓存已清空');
            loadRulesData();
        })
        .catch(error => {
            showError('清空缓存失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

document.getElementById('rulesLookupBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    const ip = document.getElementById('rulesLookupDomain').value.trim();

    if (!ip) {
        showError('请输入IP地址');
        return;
    }

    showLoading(btn);
    // 使用POST请求，发送JSON body
    apiPost('/rules/geoip/lookup', { ip: ip })
        .then(result => {
            const output = document.getElementById('rulesLookupResult');
            // result 已经是 data 字段的内容，直接格式化显示
            output.value = JSON.stringify(result, null, 2);
            showSuccess('查询完成');
        })
        .catch(error => {
            showError('查询失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

// ============ WebSocket 日志流管理 ============
const logWebSocket = {
    ws: null,
    isConnected: false,
    reconnectAttempts: 0,
    maxReconnectAttempts: 5,
    reconnectInterval: 3000,
    isManualDisconnect: false,

    // WebSocket URL - 使用相对路径以自动适应当前协议和主机
    url: function() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        return `${protocol}//${window.location.host}/api/logs/stream`;
    },

    // 连接 WebSocket
    connect: function() {
        if (this.ws && (this.ws.readyState === WebSocket.CONNECTING ||
                       this.ws.readyState === WebSocket.OPEN)) {
            return;
        }

        this.updateConnectionStatus('connecting');

        try {
            this.ws = new WebSocket(this.url());

            this.ws.onopen = () => {
                this.isConnected = true;
                this.reconnectAttempts = 0;
                this.updateConnectionStatus('connected');
                console.log('WebSocket 日志连接已建立');
                showSuccess('实时日志连接已建立');
            };

            this.ws.onmessage = (event) => {
                this.handleLogMessage(event);
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket 错误:', error);
                this.updateConnectionStatus('error');
                showError('WebSocket 连接错误');
            };

            this.ws.onclose = (event) => {
                this.isConnected = false;
                this.updateConnectionStatus('disconnected');

                // 自动重连（除非是手动断开）
                if (!this.isManualDisconnect &&
                    event.code !== 1000 &&
                    this.reconnectAttempts < this.maxReconnectAttempts) {
                    console.log(`WebSocket 连接断开，${this.reconnectInterval/1000}秒后重试 (${this.reconnectAttempts + 1}/${this.maxReconnectAttempts})`);
                    setTimeout(() => {
                        this.reconnectAttempts++;
                        this.connect();
                    }, this.reconnectInterval);
                } else if (this.reconnectAttempts >= this.maxReconnectAttempts) {
                    showError('WebSocket 连接失败，请检查网络连接或手动刷新页面');
                }
            };
        } catch (error) {
            console.error('创建 WebSocket 连接失败:', error);
            this.updateConnectionStatus('error');
            showError('创建 WebSocket 连接失败');
        }
    },

    // 断开连接
    disconnect: function() {
        this.isManualDisconnect = true;
        if (this.ws) {
            this.ws.close(1000, 'Client disconnect');
            this.ws = null;
        }
        this.isConnected = false;
        this.updateConnectionStatus('disconnected');
        console.log('WebSocket 连接已断开');
    },

    // 处理日志消息
    handleLogMessage: function(event) {
        try {
            const logEntry = JSON.parse(event.data);
            this.appendLogToOutput(logEntry);
            appendDashboardRequestLog(logEntry);
        } catch (error) {
            console.error('解析日志消息失败:', error);
        }
    },

    // 将日志添加到输出区域
    appendLogToOutput: function(logEntry) {
        const output = document.getElementById('logsOutput');
        if (!output) return;

        // 格式化日志行
        const timestamp = logEntry.timestamp || new Date().toLocaleString();
        const level = logEntry.level || 'INFO';
        const message = logEntry.message || '';
        const source = logEntry.source || '';

        // 根据日志级别设置颜色
        let levelColor = '#abb2bf'; // 默认颜色
        switch (level.toUpperCase()) {
            case 'ERROR':
            case 'FATAL':
                levelColor = '#ff6b6b';
                break;
            case 'WARN':
            case 'WARNING':
                levelColor = '#feca57';
                break;
            case 'INFO':
                levelColor = '#48cae4';
                break;
            case 'DEBUG':
                levelColor = '#868e96';
                break;
            default:
                levelColor = '#abb2bf';
        }

        const formattedLog = `[${timestamp}] [${level}]`;
        const sourcePart = source ? ` [${source}]` : '';
        const fullLogLine = `${formattedLog}${sourcePart} ${message}\n`;

        // 滚动到底部
        const shouldScroll = output.scrollTop + output.clientHeight >= output.scrollHeight - 10;

        // 添加新日志
        output.value += fullLogLine;

        // 限制日志行数以避免内存问题
        const lines = output.value.split('\n');
        if (lines.length > 1000) {
            output.value = lines.slice(-1000).join('\n');
        }

        // 自动滚动到底部
        if (shouldScroll) {
            output.scrollTop = output.scrollHeight;
        }
    },

    // 更新连接状态显示
    updateConnectionStatus: function(status) {
        const statusElement = document.getElementById('wsConnectionStatus');
        const connectBtn = document.getElementById('wsConnectBtn');
        const disconnectBtn = document.getElementById('wsDisconnectBtn');
        const logOutput = document.getElementById('logsOutput');

        if (!statusElement) return;

        let statusText = '';
        let statusClass = '';

        switch (status) {
            case 'connected':
                statusText = '实时连接';
                statusClass = 'badge bg-danger';
                // 禁用连接按钮，启用断开按钮
                if (connectBtn) {
                    connectBtn.disabled = true;
                    connectBtn.classList.add('disabled');
                }
                if (disconnectBtn) {
                    disconnectBtn.disabled = false;
                    disconnectBtn.classList.remove('disabled');
                }
                // 添加连接状态样式到日志输出区
                if (logOutput) {
                    logOutput.classList.add('ws-connected');
                }
                break;
            case 'connecting':
                statusText = '连接中...';
                statusClass = 'badge bg-warning';
                // 禁用两个按钮
                if (connectBtn) {
                    connectBtn.disabled = true;
                    connectBtn.classList.add('disabled');
                }
                if (disconnectBtn) {
                    disconnectBtn.disabled = true;
                    disconnectBtn.classList.add('disabled');
                }
                break;
            case 'disconnected':
                statusText = '已断开';
                statusClass = 'badge bg-secondary';
                // 启用连接按钮，禁用断开按钮
                if (connectBtn) {
                    connectBtn.disabled = false;
                    connectBtn.classList.remove('disabled');
                }
                if (disconnectBtn) {
                    disconnectBtn.disabled = true;
                    disconnectBtn.classList.add('disabled');
                }
                // 移除连接状态样式
                if (logOutput) {
                    logOutput.classList.remove('ws-connected');
                }
                break;
            case 'error':
                statusText = '连接错误';
                statusClass = 'badge bg-danger';
                // 启用连接按钮，禁用断开按钮
                if (connectBtn) {
                    connectBtn.disabled = false;
                    connectBtn.classList.remove('disabled');
                }
                if (disconnectBtn) {
                    disconnectBtn.disabled = true;
                    disconnectBtn.classList.add('disabled');
                }
                // 移除连接状态样式
                if (logOutput) {
                    logOutput.classList.remove('ws-connected');
                }
                break;
            default:
                statusText = '未知状态';
                statusClass = 'badge bg-dark';
                // 默认状态：启用连接，禁用断开
                if (connectBtn) {
                    connectBtn.disabled = false;
                    connectBtn.classList.remove('disabled');
                }
                if (disconnectBtn) {
                    disconnectBtn.disabled = true;
                    disconnectBtn.classList.add('disabled');
                }
        }

        statusElement.className = statusClass;
        statusElement.textContent = statusText;
    }
};

// ============ 日志功能 ============
async function loadLogs() {
    try {
        // 获取当前过滤条件
        const level = appState.logConfig.level || '';
        const source = appState.logConfig.source || '';
        
        // 构建查询参数
        const params = new URLSearchParams();
        if (level) params.append('level', level);
        if (source) params.append('source', source);
        params.append('limit', '1000'); // 限制返回数量
        
        const queryString = params.toString();
        const endpoint = queryString ? `/logs?${queryString}` : '/logs';
        
        const response = await apiGet(endpoint);
        const output = document.getElementById('logsOutput');
        
        // 格式化日志条目
        if (response.entries && Array.isArray(response.entries)) {
            const logText = response.entries.map(entry => {
                const level = entry.level || 'INFO';
                const timestamp = entry.timestamp || '';
                const source = entry.source ? `[${entry.source}]` : '';
                const message = entry.message || '';
                return `${timestamp} ${level} ${source} ${message}`;
            }).join('\n');
            output.value = logText;
        } else {
            output.value = '';
        }
        
        // 如果启用自动滚动，滚动到底部
        if (appState.logConfig.autoScroll) {
            output.scrollTop = output.scrollHeight;
        }
    } catch (error) {
        console.error('加载日志失败:', error);
        const output = document.getElementById('logsOutput');
        if (output) {
            output.value = `加载日志失败: ${error.message}`;
        }
    }
}

document.getElementById('logsRefreshBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    loadLogs()
        .then(() => {
            showSuccess('日志已刷新');
        })
        .catch(error => {
            showError('刷新失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

document.getElementById('logsClearBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    if (!confirm('确定要清空所有日志吗？')) {
        return;
    }
    showLoading(btn);

    // 先清空当前显示的日志
    const output = document.getElementById('logsOutput');
    if (output) {
        output.value = '';
    }

    apiPost('/logs/clear')
        .then(() => {
            showSuccess('日志已清空');
            // 重新加载日志（应该是空的）
            loadLogs();
        })
        .catch(error => {
            showError('清空失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

// 日志配置功能
async function loadLogConfig() {
    try {
        const config = await apiGet('/logs/config');
        
        // 后端返回的 level 是大写，需要转换为大写匹配
        const level = (config.level || 'INFO').toUpperCase();
        
        // 更新全局状态（只保存后端支持的配置）
        appState.logConfig = {
            level: level,
            source: appState.logConfig?.source || '', // source 是前端过滤，不从后端加载
            autoScroll: appState.logConfig?.autoScroll !== false // autoScroll 是前端功能
        };

        // 更新UI
        const levelSelect = document.getElementById('logLevelSelect');
        if (levelSelect) {
            levelSelect.value = level;
        }
        
        const sourceSelect = document.getElementById('logSourceSelect');
        if (sourceSelect && appState.logConfig.source) {
            sourceSelect.value = appState.logConfig.source;
        }
        
        const autoScroll = document.getElementById('logAutoScroll');
        if (autoScroll) {
            autoScroll.checked = appState.logConfig.autoScroll;
        }
    } catch (error) {
        console.error('加载日志配置失败:', error);
        // 使用默认值
        appState.logConfig = {
            level: 'INFO',
            source: '',
            autoScroll: true
        };
    }
}

// 保存日志级别配置（只保存后端支持的配置项）
async function saveLogConfig() {
    const btn = event.target.closest('button');
    showLoading(btn);

    const level = document.getElementById('logLevelSelect').value.toUpperCase();
    
    // 只发送后端支持的配置项
    const configData = {
        level: level
    };

    try {
        await apiPost('/logs/config', configData);
        appState.logConfig.level = level;
        showSuccess('日志级别已更新');
        // 重新加载日志以应用新级别
        loadLogs();
    } catch (error) {
        showError('保存失败: ' + error.message);
    } finally {
        hideLoading(btn);
    }
}

// 应用日志过滤（不保存到后端，只用于前端显示）
function applyLogFilter() {
    const source = document.getElementById('logSourceSelect').value;
    appState.logConfig.source = source;
    appState.logConfig.autoScroll = document.getElementById('logAutoScroll').checked;
    
    // 重新加载日志以应用过滤
    loadLogs();
    showSuccess('过滤已应用');
}

// 日志配置事件监听
document.getElementById('logConfigSaveBtn').addEventListener('click', saveLogConfig);
document.getElementById('logFilterBtn').addEventListener('click', applyLogFilter);

// WebSocket 连接/断开按钮事件
document.getElementById('wsConnectBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    logWebSocket.connect();
    hideLoading(btn);
});

document.getElementById('wsDisconnectBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    logWebSocket.disconnect();
    showSuccess('实时日志连接已断开');
    hideLoading(btn);
});

// 全屏按钮事件
document.getElementById('logsFullscreenBtn').addEventListener('click', () => {
    const logsPage = document.getElementById('logs-page');
    const fullscreenBtn = document.getElementById('logsFullscreenBtn');
    const btnIcon = fullscreenBtn.querySelector('i');

    if (logsPage.classList.contains('fullscreen-mode')) {
        // 退出全屏
        logsPage.classList.remove('fullscreen-mode');
        if (document.exitFullscreen) {
            document.exitFullscreen().catch(err => {
                console.log('Exit fullscreen failed:', err);
            });
        }
        btnIcon.className = 'bi bi-fullscreen';
        fullscreenBtn.innerHTML = '<i class="bi bi-fullscreen"></i> 全屏';
    } else {
        // 进入全屏
        logsPage.classList.add('fullscreen-mode');
        if (document.documentElement.requestFullscreen) {
            document.documentElement.requestFullscreen().catch(err => {
                console.log('Fullscreen request failed:', err);
            });
        }
        btnIcon.className = 'bi bi-arrows-fullscreen';
        fullscreenBtn.innerHTML = '<i class="bi bi-arrows-fullscreen"></i> 退出全屏';
    }
});

// 监听全屏状态变化（用户按ESC退出全屏时同步UI状态）
document.addEventListener('fullscreenchange', () => {
    const logsPage = document.getElementById('logs-page');
    const fullscreenBtn = document.getElementById('logsFullscreenBtn');

    if (!document.fullscreenElement && logsPage.classList.contains('fullscreen-mode')) {
        // 用户通过ESC退出了全屏，同步UI状态
        logsPage.classList.remove('fullscreen-mode');
        fullscreenBtn.innerHTML = '<i class="bi bi-fullscreen"></i> 全屏';
    }
});


// ============ DNS 缓存管理 ============

// 格式化数字，添加千位符
function formatNumber(num) {
    return num.toString().replace(/\B(?=(\\d{3})+(?!\\d))/g, ',');
}

// 获取绑定源的显示颜色（Bootstrap 颜色）
function getBadgeColor(source) {
    const colors = {
        'sni': 'primary',
        'http_host': 'warning',
        'dns': 'success',
        'connect': 'danger'
    };
    return colors[source] || 'primary';
}

// 格式化字节为可读的大小
function formatBytes(bytes, decimals = 2) {
    if (bytes === 0 || bytes === undefined || bytes === null) return '0 B';

    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

// ============ 图表管理函数 ============

// 销毁所有图表实例（用���完全重置或页面卸载）
function destroyCharts() {
    Object.values(charts).forEach(chart => {
        if (chart && typeof chart.destroy === 'function') {
            chart.destroy();
        }
    });
    charts = {};
}

// 初始化流量趋势图表
function initChart() {
    if (document.getElementById('trafficChart')) {
        // 销毁已存在的图表实例，避免重复创建导致的错误
        if (charts.traffic && typeof charts.traffic.destroy === 'function') {
            charts.traffic.destroy();
            charts.traffic = null;
        }

        const ctx = document.getElementById('trafficChart').getContext('2d');
        charts.traffic = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [
                    {
                        label: '上传 (Bytes/s)',
                        data: [],
                        borderColor: '#ff6b6b',
                        backgroundColor: 'rgba(255, 107, 107, 0.1)',
                        borderWidth: 2,
                        tension: 0.3,
                        fill: true,
                        pointRadius: 0,
                        pointHoverRadius: 5
                    },
                    {
                        label: '下载 (Bytes/s)',
                        data: [],
                        borderColor: '#ff9999',
                        backgroundColor: 'rgba(255, 153, 153, 0.1)',
                        borderWidth: 2,
                        tension: 0.3,
                        fill: true,
                        pointRadius: 0,
                        pointHoverRadius: 5
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: { mode: 'index', intersect: false },
                plugins: {
                    legend: { display: false },  // 使用自定义图例
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                let label = context.dataset.label || '';
                                if (label) {
                                    label += ': ';
                                }
                                label += formatBytes(context.parsed.y) + '/s';
                                return label;
                            }
                        }
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            callback: function(value) {
                                return formatBytes(value) + '/s';
                            }
                        }
                    },
                    x: {
                        ticks: {
                            maxTicksLimit: 10
                        }
                    }
                }
            }
        });
    }

    if (document.getElementById('domainPieChart')) {
        if (charts.domainPie && typeof charts.domainPie.destroy === 'function') {
            charts.domainPie.destroy();
            charts.domainPie = null;
        }

        const pieCtx = document.getElementById('domainPieChart').getContext('2d');
        charts.domainPie = new Chart(pieCtx, {
            type: 'pie',
            data: {
                labels: ['暂无数据'],
                datasets: [{
                    data: [1],
                    backgroundColor: ['rgba(52, 211, 153, 0.72)', 'rgba(22, 163, 74, 0.72)', 'rgba(132, 204, 22, 0.72)', 'rgba(16, 185, 129, 0.72)'],
                    borderColor: 'rgba(5, 20, 12, 0.88)',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'bottom',
                        labels: {
                            color: '#d8fbe8',
                            boxWidth: 10
                        }
                    }
                }
            }
        });
    }
}

// 更新图表数据
function updateChartData(statsData) {
    // 添加新数据点
    chartDataManager.addPoint(
        statsData.uploadTotal || 0,
        statsData.downloadTotal || 0
    );

    // 更新流量图
    if (charts.traffic) {
        const timeLabels = chartDataManager.data.timestamps.map(ts => {
            const date = new Date(ts);
            return date.toLocaleTimeString();
        });

        charts.traffic.data.labels = timeLabels;
        charts.traffic.data.datasets[0].data = chartDataManager.data.uploadSpeeds;
        charts.traffic.data.datasets[1].data = chartDataManager.data.downloadSpeeds;
        charts.traffic.update('none'); // 不使用动画，防止图表卡顿
    }
}

// ============ 统计数据加载 ============

// 加载代理统计数据
async function loadStatsData() {
    try {
        const response = await apiGet('/stats/traffic/current');
        if (!response) {
            console.error('Failed to load stats data');
            return;
        }

        // 更新全局统计卡片
        document.getElementById('statsUploadTotal').textContent = formatBytes(response.uploadTotal || 0);
        document.getElementById('statsDownloadTotal').textContent = formatBytes(response.downloadTotal || 0);

        // 计算总连接数
        let totalConnections = 0;
        if (response.byRoute) {
            Object.values(response.byRoute).forEach(route => {
                totalConnections += route.connectionCount || 0;
            });
        }
        document.getElementById('statsConnectionCount').textContent = formatNumber(totalConnections);

        // 总流量
        const totalTraffic = (response.uploadTotal || 0) + (response.downloadTotal || 0);
        document.getElementById('statsTotalTraffic').textContent = formatBytes(totalTraffic);

        // 更新代理分布表格
        renderStatsTable(response.byRoute);
        updateDomainPieChartByLogs();

        // 更新图表数据
        updateChartData(response);
    } catch (error) {
        console.error('Failed to load stats data:', error);
    }
}

// 渲染代理统计表格
function renderStatsTable(byRoute) {
    const tbody = document.getElementById('statsTableBody');

    if (!byRoute || Object.keys(byRoute).length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">暂无数据</td></tr>';
        return;
    }

    // 代理类型的显示名称映射
    const routeNames = {
        'RouteToCursor': 'MITM (Cursor/Nonelane)',
        'RouteToDoor': '代理 (VLESS/Shadowsocks)',
        'RouteDirect': '直连'
    };

    let html = '';

    // 遍历所有代理类型
    Object.entries(byRoute).forEach(([routeType, stats]) => {
        const displayName = routeNames[routeType] || routeType;
        const connectionCount = stats.connectionCount || 0;
        const uploadTotal = stats.uploadTotal || 0;
        const downloadTotal = stats.downloadTotal || 0;
        const averageUpload = stats.averageUpload || 0;
        const averageDownload = stats.averageDownload || 0;

        html += `<tr>
            <td><strong>${displayName}</strong></td>
            <td class="text-right">${formatNumber(connectionCount)}</td>
            <td class="text-right">${formatBytes(uploadTotal)}</td>
            <td class="text-right">${formatBytes(downloadTotal)}</td>
            <td class="text-right">${formatBytes(averageUpload)}</td>
            <td class="text-right">${formatBytes(averageDownload)}</td>
        </tr>`;
    });

    tbody.innerHTML = html;
}

// 加载DNS缓存数据
async function loadDNSCacheData() {
    try {
        // 并行加载统计数据和热点数据
        const [statsResp, hotspotsResp] = await Promise.all([
            apiGet('/dns/stats').catch(() => null),
            apiGet('/dns/hotspots').catch(() => null)
        ]);

        // 更新统计卡片
        if (statsResp) {
            document.getElementById('dnsTotalEntries').textContent = formatNumber(statsResp.size || 0);
            document.getElementById('dnsUniqueDomains').textContent = formatNumber(statsResp.uniqueDomains || 0);
            document.getElementById('dnsUniqueIPs').textContent = formatNumber(statsResp.uniqueIPs || 0);
            document.getElementById('dnsHitRate').textContent = (statsResp.hitRate || 0).toFixed(1) + '%';
        }

        // 更新热点数据
        if (hotspotsResp) {
            renderDNSHotDomains(hotspotsResp.topDomains);
            renderDNSHotIPs(hotspotsResp.topIPs);
        }
    } catch (error) {
        console.error('Failed to load DNS cache data:', error);
    }
}

// 渲染热点域名表格
function renderDNSHotDomains(domains) {
    const container = document.getElementById('dnsHotDomainsContainer');
    if (!domains || domains.length === 0) {
        container.innerHTML = '<div class="text-center text-muted py-4">暂无热点域名</div>';
        return;
    }

    let html = '<div class="table-responsive"><table class="table table-sm"><thead><tr>';
    html += '<th style="width: 5%">排名</th>';
    html += '<th style="width: 35%">域名</th>';
    html += '<th style="width: 15%">访问次数</th>';
    html += '<th style="width: 20%">IP地址</th>';
    html += '<th style="width: 20%">来源</th>';
    html += '<th style="width: 5%">操作</th>';
    html += '</tr></thead><tbody>';

    domains.slice(0, 20).forEach((domain, idx) => {
        const sources = domain.sources ? domain.sources.map(s =>
            `<span class="badge bg-${getBadgeColor(s)}">${s}</span>`
        ).join(' ') : '-';

        html += `<tr>
            <td>${idx + 1}</td>
            <td><code>${domain.domain}</code></td>
            <td>${formatNumber(domain.hitCount)}</td>
            <td><code>${domain.ip}</code></td>
            <td>${sources}</td>
            <td><button class="btn btn-sm btn-danger" onclick="deleteDNSDomain('${domain.domain}')"><i class="bi bi-trash"></i></button></td>
        </tr>`;
    });

    html += '</tbody></table></div>';
    container.innerHTML = html;
}

// 渲染热点IP表格
function renderDNSHotIPs(ips) {
    const container = document.getElementById('dnsHotIPsContainer');
    if (!ips || ips.length === 0) {
        container.innerHTML = '<div class="text-center text-muted py-4">暂无热点IP</div>';
        return;
    }

    let html = '<div class="table-responsive"><table class="table table-sm"><thead><tr>';
    html += '<th style="width: 5%">排名</th>';
    html += '<th style="width: 25%">IP地址</th>';
    html += '<th style="width: 15%">访问次数</th>';
    html += '<th style="width: 45%">关联域名</th>';
    html += '<th style="width: 10%">来源数</th>';
    html += '</tr></thead><tbody>';

    ips.slice(0, 20).forEach((ip, idx) => {
        const domains = ip.associatedDomains ? ip.associatedDomains.join(', ') : '-';
        const domainsDisplay = domains.length > 60 ? domains.substring(0, 60) + '...' : domains;

        html += `<tr>
            <td>${idx + 1}</td>
            <td><code>${ip.ip}</code></td>
            <td>${formatNumber(ip.hitCount)}</td>
            <td title="${domains}"><small>${domainsDisplay}</small></td>
            <td>${ip.sourceCount}</td>
        </tr>`;
    });

    html += '</tbody></table></div>';
    container.innerHTML = html;
}

// 渲染搜索结果
function renderDNSSearchResults(result) {
    const container = document.getElementById('dnsSearchResultsContainer');
    if (!result) {
        container.innerHTML = '<div class="text-center text-muted py-4">未找到结果</div>';
        return;
    }

    let html = '<div class="table-responsive"><table class="table table-sm"><thead><tr>';
    html += '<th>类型</th><th>值</th><th>详情</th></tr></thead><tbody>';

    if (result.domain) {
        const sources = result.sources ? result.sources.map(s =>
            `<span class="badge bg-${getBadgeColor(s)}">${s}</span>`
        ).join(' ') : '-';

        html += `<tr>
            <td>域名</td>
            <td><code>${result.domain}</code></td>
            <td>
                <strong>IP:</strong> <code>${result.ip}</code><br>
                <strong>路由:</strong> <code>${result.route || '-'}</code><br>
                <strong>访问次数:</strong> ${result.hitCount}<br>
                <strong>过期时间:</strong> ${new Date(result.expiresAt).toLocaleString()}<br>
                <strong>来源:</strong> ${sources}
            </td>
        </tr>`;
    }

    if (result.domains && Array.isArray(result.domains)) {
        result.domains.forEach((domain, idx) => {
            const sources = domain.sources ? domain.sources.map(s =>
                `<span class="badge bg-${getBadgeColor(s)}">${s}</span>`
            ).join(' ') : '-';

            html += `<tr>
                <td>IP-域名${idx + 1}</td>
                <td><code>${domain.domain}</code></td>
                <td>
                    <strong>路由:</strong> <code>${domain.route || '-'}</code><br>
                    <strong>访问次数:</strong> ${domain.hitCount}<br>
                    <strong>来源:</strong> ${sources}
                </td>
            </tr>`;
        });
    }

    html += '</tbody></table></div>';
    container.innerHTML = html;
}

// 执行DNS缓存搜索
async function performDNSSearch() {
    const query = document.getElementById('dnsSearchBox').value.trim();
    if (!query) {
        document.getElementById('dnsSearchResultsContainer').innerHTML =
            '<div class="text-center text-muted py-4">请输入要搜索的域名或IP地址</div>';
        return;
    }

    try {
        // 尝试作为域名查询
        let response = await apiCall(`/dns/cache/query?domain=${encodeURIComponent(query)}`).catch(() => null);

        if (!response) {
            // 尝试作为IP反向查询
            response = await apiCall(`/dns/cache/reverse?ip=${encodeURIComponent(query)}`).catch(() => null);
        }

        if (response) {
            renderDNSSearchResults(response);
        } else {
            document.getElementById('dnsSearchResultsContainer').innerHTML =
                '<div class="text-center text-muted py-4">未找到相关结果</div>';
        }
    } catch (error) {
        console.error('Search error:', error);
        showError('搜索失败: ' + error.message);
    }
}

// 删除DNS缓存条目
async function deleteDNSDomain(domain) {
    if (!confirm(`确定要删除域名 "${domain}" 的缓存吗？`)) {
        return;
    }

    try {
        await apiCall(`/dns/cache/${encodeURIComponent(domain)}`, { method: 'DELETE' });
        showSuccess('已删除缓存');
        loadDNSCacheData();
    } catch (error) {
        showError('删除失败: ' + error.message);
    }
}

// 清空所有DNS缓存
async function confirmClearAllDNS() {
    if (!confirm('确定要清除所有DNS缓存吗？此操作不可撤销！')) {
        return;
    }

    try {
        await apiCall('/dns/cache', { method: 'DELETE' });
        showSuccess('已清除所有缓存');
        loadDNSCacheData();
    } catch (error) {
        showError('清除失败: ' + error.message);
    }
}

// DNS缓存页面事件监听
document.addEventListener('DOMContentLoaded', () => {
    const searchBox = document.getElementById('dnsSearchBox');
    if (searchBox) {
        searchBox.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                performDNSSearch();
            }
        });
    }

    const searchBtn = document.getElementById('dnsSearchBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', performDNSSearch);
    }

    const refreshBtn = document.getElementById('dnsRefreshBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', loadDNSCacheData);
    }

    const clearBtn = document.getElementById('dnsClearAllBtn');
    if (clearBtn) {
        clearBtn.addEventListener('click', confirmClearAllDNS);
    }
});

// ============ 页面导航 ============
function switchPage(page) {
    document.querySelectorAll('.content-section').forEach(el => {
        el.classList.remove('active');
        el.classList.add('hidden');
    });

    const pageEl = document.getElementById(`${page}-page`);
    if (pageEl) {
        pageEl.classList.add('active');
        pageEl.classList.remove('hidden');
    }

    document.querySelectorAll('.tab-item').forEach(link => {
        link.classList.remove('active');
        if (link.dataset.page === page) {
            link.classList.add('active');
        }
    });

    appState.currentPage = page;

    // 触发页面特定的初始化
    if (page === 'dashboard') {
        // 加载仪表板和统计数据
        loadDashboard();
        // 设置仪表板统计数据定时刷新（每1.5秒）
        const dashboardStatsInterval = setInterval(() => {
            if (appState.currentPage === 'dashboard') {
                loadStatsData().catch(error => console.error('Failed to load stats:', error));
            } else {
                clearInterval(dashboardStatsInterval);
            }
        }, 1500);
    } else if (page === 'proxy-control') {
        // 合并的代理管理与运行控制页面
        loadProxyData();
        loadRunStatus();
    } else if (page === 'rules') {
        loadRulesData();
        loadRoutingConfig();
    } else if (page === 'logs') {
        // 连接 WebSocket 进行实时日志
        logWebSocket.connect();
        // 加载历史日志
        loadLogs();
        // 加载日志配置
        loadLogConfig();
    } else if (page === 'userinfo') {
        // 加载用户信息和刷新状态
        loadAuthUserInfo();
        loadRefreshStatus();
    } else if (page === 'stats') {
        loadStatsData();
        // 设置代理统计页面的定时刷新（每1.5秒）
        const statsRefreshInterval = setInterval(() => {
            if (appState.currentPage === 'stats') {
                loadStatsData();
            } else {
                clearInterval(statsRefreshInterval);
            }
        }, 1500);
    } else if (page === 'dnscache') {
        loadDNSCacheData();
        // 设置DNS缓存页面的定时刷新（每5秒）
        const dnsRefreshInterval = setInterval(() => {
            if (appState.currentPage === 'dnscache') {
                loadDNSCacheData();
            } else {
                clearInterval(dnsRefreshInterval);
            }
        }, 5000);
    }

    // 更新当前页面状态
    appState.currentPage = page;
}

// 页面清理函数
function cleanupCurrentPage() {
    // WebSocket 保持持久连接，不再断开
    // 这样可以持续接收日志，即使用户切换到其他页面
}

// ============ 菜单导航事件 ============
document.querySelectorAll('.tab-item').forEach(link => {
    link.addEventListener('click', (e) => {
        e.preventDefault();
        const page = link.dataset.page;

        // 清理当前页面
        cleanupCurrentPage();

        // 切换到新页面
        switchPage(page);
    });
});

// ============ 用户认证管理 (全局函数) ============

// 加载用户信息
async function loadAuthUserInfo() {
    try {
        const response = await apiGet('/auth/userinfo');
        if (response.status === 'no_user') {
            displayNoUserInfo();
            return;
        }

        const userInfo = response.data;
        displayUserInfo(userInfo);
        loadRefreshStatus();
    } catch (error) {
        console.error('Failed to load user info:', error);
        displayNoUserInfo();
    }
}

// 显示用户信息
function displayUserInfo(userInfo) {
    const container = document.getElementById('authUserInfoContainer');
    const balanceEl = document.getElementById('userBalance');
    const trafficPercent = userInfo.traffic_total > 0
        ? Math.round((userInfo.traffic_used / userInfo.traffic_total) * 100)
        : 0;

    const trafficUsedGB = (userInfo.traffic_used / (1024 * 1024 * 1024)).toFixed(2);
    const trafficTotalGB = (userInfo.traffic_total / (1024 * 1024 * 1024)).toFixed(2);

    container.innerHTML = `
        <div class="row g-3">
            <div class="col-md-6">
                <div>
                    <strong>用户名:</strong> ${userInfo.username}
                </div>
            </div>
            <div class="col-md-6">
                <div>
                    <strong>计划名称:</strong> ${userInfo.plan_name}
                </div>
            </div>
            <div class="col-md-6">
                <div>
                    <strong>计划类型:</strong> ${userInfo.plan_type}
                </div>
            </div>
            <div class="col-md-6">
                <div>
                    <strong>有效期:</strong> ${userInfo.start_time} 至 ${userInfo.end_time}
                </div>
            </div>
            <div class="col-12">
                <div>
                    <strong>流量使用:</strong>
                    <div class="progress mt-2" style="height: 20px;">
                        <div class="progress-bar ${trafficPercent > 90 ? 'bg-danger' : trafficPercent > 70 ? 'bg-warning' : 'bg-primary'}"
                             role="progressbar"
                             style="width: ${trafficPercent}%;"
                             aria-valuenow="${trafficPercent}"
                             aria-valuemin="0"
                             aria-valuemax="100">
                            ${trafficUsedGB}GB / ${trafficTotalGB}GB (${trafficPercent}%)
                        </div>
                    </div>
                </div>
            </div>
            <div class="col-md-6">
                <div>
                    <strong>AI 提问:</strong> ${userInfo.ai_ask_used} / ${userInfo.ai_ask_total}
                </div>
            </div>
            <div class="col-md-6">
                <div>
                    <strong>最后更新:</strong> ${formatDateTime(userInfo.updated_at)}
                </div>
            </div>
        </div>
    `;

    // 显示登出按钮
    document.getElementById('authLogoutBtn').style.display = 'inline-block';

    // 显示刷新状态卡片
    document.getElementById('authRefreshStatusCard').style.display = 'block';

    if (balanceEl) {
        const balance = Number(userInfo.balance || userInfo.amount || 0);
        balanceEl.textContent = `${balance.toFixed(2)} CNY`;
    }
}

// 显示无用户信息
function displayNoUserInfo() {
    const container = document.getElementById('authUserInfoContainer');
    const balanceEl = document.getElementById('userBalance');
    container.innerHTML = `
        <div class="text-center text-muted py-4">
            <p>暂无用户信息，请先激活 Token</p>
        </div>
    `;

    // 隐藏登出按钮
    document.getElementById('authLogoutBtn').style.display = 'none';

    // 隐藏刷新状态卡片
    document.getElementById('authRefreshStatusCard').style.display = 'none';

    if (balanceEl) {
        balanceEl.textContent = '--';
    }
}

// 加载刷新状态
async function loadRefreshStatus() {
    try {
        const response = await apiGet('/auth/refresh-status');
        if (response.status === 'success') {
            updateRefreshStatus(response.data);
        }
    } catch (error) {
        console.error('Failed to load refresh status:', error);
    }
}

// 更新刷新状态显示
function updateRefreshStatus(status) {
    const runningBadge = document.getElementById('authRefreshRunning');
    const lastUpdateEl = document.getElementById('authLastUpdate');
    const refreshIntervalEl = document.getElementById('authRefreshInterval');
    const errorEl = document.getElementById('authRefreshError');
    const errorMsgEl = document.getElementById('authRefreshErrorMsg');

    // 更新运行状态
    runningBadge.className = status.is_running ? 'badge bg-success' : 'badge bg-secondary';
    runningBadge.textContent = status.is_running ? '运行中' : '未运行';

    // 更新最后更新时间
    if (status.last_update_time) {
        lastUpdateEl.textContent = formatDateTime(status.last_update_time);
    }

    // 更新刷新间隔
    refreshIntervalEl.textContent = status.refresh_interval || '1 分钟';

    // 更新错误显示
    if (status.last_error) {
        errorEl.style.display = 'block';
        errorMsgEl.textContent = status.last_error;
    } else {
        errorEl.style.display = 'none';
    }
}

// ============ 页面加载完成后初始化 ============
document.addEventListener('DOMContentLoaded', () => {
    // 加载仪表板数据
    loadDashboard();

    // 预加载用户信息（后台加载，用户点击"用户信息"时无需等待）
    loadAuthUserInfo().catch(error => {
        console.debug('Initial user info load failed (this is normal if no user is configured):', error);
    });

    // 自动连接日志 WebSocket（持久连接，不因页面切换而断开）
    console.log('Auto-connecting to log WebSocket on page load...');
    logWebSocket.connect();

    // 初始化全局状态
    updateGlobalStatus();

    // 启动全局状态轮询（每1秒）
    appState.statusPollingInterval = setInterval(updateGlobalStatus, 1000);

    // 设置仪表板数据定时刷新（每5秒）
    setInterval(() => {
        if (appState.currentPage === 'dashboard') {
            loadDashboard();
        }
    }, 5000);

    // 页面卸载时清理轮询
    window.addEventListener('beforeunload', () => {
        if (appState.statusPollingInterval) {
            clearInterval(appState.statusPollingInterval);
            appState.statusPollingInterval = null;
        }
    });

    // ===== 用户认证管理事件监听器 =====

    // 激活Token
    document.getElementById('authActivateBtn').addEventListener('click', async () => {
        const token = document.getElementById('authTokenInput').value.trim();
        if (!token) {
            showError('请输入 Token');
            return;
        }

        const btn = document.getElementById('authActivateBtn');
        showLoading(btn);

        try {
            const response = await apiPost('/auth/activate', { token });
            if (response.status === 'success') {
                showSuccess('Token 激活成功');
                document.getElementById('authTokenInput').value = '';
                loadAuthUserInfo();
            } else {
                showError(response.msg || 'Token 激活失败');
            }
        } catch (error) {
            showError('激活失败: ' + error.message);
        } finally {
            hideLoading(btn);
        }
    });

    // 登出
    document.getElementById('authLogoutBtn').addEventListener('click', async () => {
        if (!confirm('确定要登出吗？')) {
            return;
        }

        const btn = document.getElementById('authLogoutBtn');
        showLoading(btn);

        try {
            await apiPost('/auth/logout', {});
            showSuccess('登出成功');
            displayNoUserInfo();
            document.getElementById('authTokenInput').value = '';
        } catch (error) {
            showError('登出失败: ' + error.message);
        } finally {
            showSuccess(btn);
        }
    });

    // 用户信息将在切换到 userinfo 页面时加载
    // 定时刷新用户信息（每30秒）- 仅在 userinfo 页面时刷新
    setInterval(() => {
        if (appState.currentPage === 'userinfo') {
            loadAuthUserInfo();
            loadRefreshStatus();
        }
    }, 30000);

    // ===== Certificate Management =====

    // 获取证书状态
    async function loadCertStatus() {
        try {
            const response = await apiGet('/cert/status');
            updateCertStatusDisplay(response);
        } catch (error) {
            console.error('Failed to load cert status:', error);
        }
    }

    // 检查证书安装状态
    async function checkCertInstallation() {
        const certType = document.getElementById('cert-type-select').value;
        const btn = document.getElementById('btn-check-cert');
        showLoading(btn);

        try {
            const result = await apiPost('/cert/status', { cert_type: certType });
            updateCertStatusDisplay(result);
            showSuccess(`证书状态: ${result.is_installed ? '✓ 已安装' : '✗ 未安装'}`);
        } catch (error) {
            showError('检查失败: ' + error.message);
        } finally {
            hideLoading(btn);
        }
    }

    // 导出证书到 ~/.nonelane/
    async function exportCert() {
        const certType = document.getElementById('cert-type-select').value;
        const btn = document.getElementById('btn-export-cert');
        showLoading(btn);

        try {
            const result = await apiPost('/cert/export', { cert_type: certType });
            showSuccess(`证书已导出到: ${result.export_path}`);
            loadCertStatus(); // 刷新状态
        } catch (error) {
            showError('导出失败: ' + error.message);
        } finally {
            hideLoading(btn);
        }
    }

    // 下载证书文件
    async function downloadCert() {
        const certType = document.getElementById('cert-type-select').value;
        downloadFile(`/cert/download?cert_type=${certType}`, `${certType}.pem`);
    }

    // 安装证书到系统
    async function installCert() {
        const certType = document.getElementById('cert-type-select').value;

        if (!confirm('此操作需要管理员权限。继续吗？')) {
            return;
        }

        const btn = document.getElementById('btn-install-cert');
        showLoading(btn);

        try {
            const result = await apiPost('/cert/install', { cert_type: certType });
            showSuccess('证书安装成功！');
            loadCertStatus(); // 刷新状态
        } catch (error) {
            showError('安装失败: ' + error.message);
        } finally {
            hideLoading(btn);
        }
    }

    // 移除证书
    async function removeCert() {
        const certType = document.getElementById('cert-type-select').value;

        if (!confirm('确定要移除证书吗？')) {
            return;
        }

        const btn = document.getElementById('btn-remove-cert');
        showLoading(btn);

        try {
            const result = await apiPost('/cert/remove', { cert_type: certType });
            showSuccess('证书已移除！');
            loadCertStatus(); // 刷新状态
        } catch (error) {
            showError('移除失败: ' + error.message);
        } finally {
            hideLoading(btn);
        }
    }

    // 更新证书状态显示
    function updateCertStatusDisplay(certStatus) {
        const container = document.getElementById('cert-status-content') || document.getElementById('cert-status-container');
        if (!certStatus || !certStatus.cert_type) {
            container.innerHTML = '<div class="alert alert-warning">暂无证书信息</div>';
            return;
        }

        // 构建状态 badges (NEW)
        const exportedBadge = certStatus.is_exported
            ? '<span class="badge bg-danger"><i class="bi bi-check-circle"></i> 已导出</span>'
            : '<span class="badge bg-danger"><i class="bi bi-x-circle"></i> 未导出</span>';

        const installedBadge = certStatus.is_installed
            ? '<span class="badge bg-danger"><i class="bi bi-check-circle"></i> 已安装</span>'
            : '<span class="badge bg-danger"><i class="bi bi-x-circle"></i> 未安装</span>';

        const trustedBadge = certStatus.is_trusted
            ? '<span class="badge bg-info"><i class="bi bi-shield-check"></i> 已信任</span>'
            : '<span class="badge bg-warning"><i class="bi bi-exclamation-triangle"></i> 未信任</span>';

        // 信任状态描述 (NEW)
        const trustStatusText = getTrustStatusText(certStatus.trust_status);
        const trustStatusBadge = getTrustStatusBadge(certStatus.trust_status);

        // 构建 HTML
        const html = `
            <div class="cert-item p-3 border rounded mb-2">
                <div class="row mb-3">
                    <div class="col-md-12">
                        <h6 class="mb-2">${getCertTypeName(certStatus.cert_type)}</h6>
                        <div class="status-badges">
                            ${exportedBadge}
                            ${installedBadge}
                            ${trustedBadge}
                        </div>
                    </div>
                </div>

                <div class="row mb-2">
                    <div class="col-md-12">
                        <div class="trust-status-info">
                            <small><strong>信任状态:</strong> ${trustStatusBadge}</small>
                        </div>
                    </div>
                </div>

                <div class="row">
                    <div class="col-md-12">
                        <small>
                            <div><strong>主体:</strong> ${certStatus.subject || '-'}</div>
                            <div><strong>颁发者:</strong> ${certStatus.issuer || '-'}</div>
                            <div><strong>有效期:</strong> ${certStatus.not_before || '-'} ~ ${certStatus.not_after || '-'}</div>
                            <div><strong>指纹:</strong> <code style="font-size: 0.75em;">${(certStatus.fingerprint || '-').substring(0, 16)}...</code></div>
                        </small>
                    </div>
                </div>
            </div>
        `;
        container.innerHTML = html;
    }

    // NEW 辅助函数：获取信任状态的中文描述
    function getTrustStatusText(trustStatus) {
        const statusMap = {
            'not_found': '证书不存在',
            'installed_not_trusted': '已安装但未信任',
            'system_trusted': '系统信任',
            'unsupported_platform': '不支持的平台'
        };
        return statusMap[trustStatus] || trustStatus;
    }

    // NEW 辅助函数：获取信任状态的 badge HTML
    function getTrustStatusBadge(trustStatus) {
        const badgeMap = {
            'not_found': '<span class="badge bg-danger">证书不存在</span>',
            'installed_not_trusted': '<span class="badge bg-warning">已安装但未信任</span>',
            'system_trusted': '<span class="badge bg-danger">系统信任</span>',
            'unsupported_platform': '<span class="badge bg-secondary">不支持的平台</span>'
        };
        return badgeMap[trustStatus] || `<span class="badge bg-secondary">${trustStatus}</span>`;
    }

    function getCertTypeName(type) {
        const names = {
            'mitm-ca': 'MITM CA (客户端拦截)',
            'root-ca': 'Root CA (根证书)',
            'mtls-cert': 'mTLS Certificate (后端通信)'
        };
        return names[type] || type;
    }

    // 事件监听器绑定
    if (document.getElementById('btn-check-cert')) {
        document.getElementById('btn-check-cert').addEventListener('click', checkCertInstallation);
        document.getElementById('btn-export-cert').addEventListener('click', exportCert);
        document.getElementById('btn-download-cert').addEventListener('click', downloadCert);
        document.getElementById('btn-install-cert').addEventListener('click', installCert);
        document.getElementById('btn-remove-cert').addEventListener('click', removeCert);

        // 页面加载时获取证书状态
        loadCertStatus();

        // 当页面可见时每10秒刷新一次证书状态
        let certStatusPolling = null;

        // 处理页面可见性变化
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                if (certStatusPolling) clearInterval(certStatusPolling);
            } else {
                loadCertStatus(); // 页面重新显示时立即更新
                if (!certStatusPolling) {
                    certStatusPolling = setInterval(loadCertStatus, 10000);
                }
            }
        });

        // 初始启动轮询
        certStatusPolling = setInterval(loadCertStatus, 10000);
    }

    initQuickChat();
    initDomainFilterControls();
});

// ============ P3: Real-time Traffic Statistics ============

// Current timescale state
let currentTimescale = '1s';
let trafficStatsRefreshInterval = null;

// Refresh traffic statistics from backend
async function refreshTrafficStats() {
    try {
        // Fetch historical stats for chart
        const statsResponse = await fetch(`/api/stats/traffic/${currentTimescale}`);
        if (!statsResponse.ok) {
            console.error('Failed to fetch traffic stats:', statsResponse.status);
            return;
        }

        const statsResult = await statsResponse.json();
        if (statsResult.success && statsResult.data && statsResult.data.stats) {
            updateTrafficChart(statsResult.data.stats);
        }

        // Fetch current stats for active connections display
        const currentResponse = await fetch('/api/stats/traffic/current');
        if (currentResponse.ok) {
            const currentResult = await currentResponse.json();
            if (currentResult.success && currentResult.data) {
                updateConnectionInfo(currentResult.data.active_connections);
                updateCurrentRates(currentResult.data.upload_rate, currentResult.data.download_rate);
            }
        }
    } catch (error) {
        console.error('Error refreshing traffic stats:', error);
    }
}

// Update traffic chart with new data from backend
function updateTrafficChart(statsArray) {
    if (!charts.traffic || !statsArray || statsArray.length === 0) {
        return;
    }

    // Prepare data arrays
    const uploads = [];
    const downloads = [];
    const timestamps = [];

    for (let i = 0; i < statsArray.length; i++) {
        const stat = statsArray[i];
        const date = new Date(stat.timestamp * 1000);
        timestamps.push(formatTime(date));
        uploads.push(stat.upload_bytes);
        downloads.push(stat.download_bytes);
    }

    // Update chart data (use 'none' mode for smooth update without animation)
    charts.traffic.data.labels = timestamps;
    charts.traffic.data.datasets[0].data = uploads;
    charts.traffic.data.datasets[1].data = downloads;
    charts.traffic.update('none');
}

// Format time for chart labels (HH:MM:SS)
function formatTime(date) {
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    const seconds = String(date.getSeconds()).padStart(2, '0');
    return `${hours}:${minutes}:${seconds}`;
}

// Update active connections display
function updateConnectionInfo(connectionCount) {
    const element = document.getElementById('statsConnectionCount');
    if (element) {
        element.textContent = connectionCount || '0';
    }
}

// Update current upload/download rates
function updateCurrentRates(uploadRate, downloadRate) {
    const uploadElement = document.getElementById('statsUploadRate');
    const downloadElement = document.getElementById('statsDownloadRate');

    if (uploadElement) {
        uploadElement.textContent = formatBytes(uploadRate || 0) + '/s';
    }
    if (downloadElement) {
        downloadElement.textContent = formatBytes(downloadRate || 0) + '/s';
    }
}

// Initialize real-time monitoring on dashboard page
function initRealTimeMonitoring() {
    // Stop any existing interval
    if (trafficStatsRefreshInterval) {
        clearInterval(trafficStatsRefreshInterval);
    }

    // Initial refresh
    refreshTrafficStats();

    // Set up interval for continuous refresh (every 1 second)
    trafficStatsRefreshInterval = setInterval(refreshTrafficStats, 1000);

    // Setup timescale buttons
    const timescaleButtons = document.querySelectorAll('[data-timescale]');
    timescaleButtons.forEach(btn => {
        btn.addEventListener('click', function() {
            const newTimescale = this.getAttribute('data-timescale');
            switchTrafficTimescale(newTimescale);
        });
    });
}

// Switch traffic monitoring timescale
function switchTrafficTimescale(newTimescale) {
    if (['1s', '5s', '15s'].includes(newTimescale)) {
        currentTimescale = newTimescale;

        // Update button visual state
        document.querySelectorAll('[data-timescale]').forEach(btn => {
            btn.classList.remove('active');
            if (btn.getAttribute('data-timescale') === newTimescale) {
                btn.classList.add('active');
            }
        });

        // Immediately refresh with new timescale
        refreshTrafficStats();
    }
}

// Stop real-time monitoring (cleanup)
function stopRealTimeMonitoring() {
    if (trafficStatsRefreshInterval) {
        clearInterval(trafficStatsRefreshInterval);
        trafficStatsRefreshInterval = null;
    }
}

// Cleanup on page unload
window.addEventListener('beforeunload', function() {
    stopRealTimeMonitoring();
});

// Start monitoring when dashboard page becomes active
document.addEventListener('DOMContentLoaded', function() {
    // Check if we're on the dashboard page
    const dashboardPage = document.getElementById('dashboard-page');
    if (dashboardPage && dashboardPage.classList.contains('active')) {
        initRealTimeMonitoring();
    }

    // Listen for page navigation to dashboard
    const navLinks = document.querySelectorAll('.nav-link');
    navLinks.forEach(link => {
        link.addEventListener('click', function() {
            const targetPage = this.getAttribute('data-page');
            if (targetPage === 'dashboard') {
                // Delay to allow page transition
                setTimeout(initRealTimeMonitoring, 100);
            } else {
                // Stop monitoring when leaving dashboard
                stopRealTimeMonitoring();
            }
        });
    });
});

// ============ 仪表板/设置页切换（新UI） ============
document.addEventListener('DOMContentLoaded', () => {
    const dashboardPage = document.getElementById('dashboard-page');
    const settingsPage = document.getElementById('settings-page');
    const goToSettingsBtn = document.getElementById('goToSettingsBtn');
    const headerSettingsBtn = document.getElementById('headerSettingsBtn');
    const backToDashboard = document.getElementById('backToDashboard');

    const settingsTabs = document.querySelectorAll('.settings-tab');
    const settingsContents = document.querySelectorAll('.settings-content');

    const showSettingsPage = () => {
        if (!dashboardPage || !settingsPage) {
            return;
        }
        appState.currentPage = 'settings';
        dashboardPage.classList.remove('active');
        dashboardPage.classList.add('hidden');
        settingsPage.classList.remove('hidden');
        settingsPage.classList.add('active');
    };

    const showDashboardPage = () => {
        if (!dashboardPage || !settingsPage) {
            return;
        }
        appState.currentPage = 'dashboard';
        settingsPage.classList.remove('active');
        settingsPage.classList.add('hidden');
        dashboardPage.classList.remove('hidden');
        dashboardPage.classList.add('active');
    };

    if (dashboardPage && settingsPage) {
        appState.currentPage = 'dashboard';
        dashboardPage.classList.add('active');
        dashboardPage.classList.remove('hidden');
        settingsPage.classList.add('hidden');
    }

    if (goToSettingsBtn) {
        goToSettingsBtn.addEventListener('click', showSettingsPage);
    }

    if (headerSettingsBtn) {
        headerSettingsBtn.addEventListener('click', showSettingsPage);
    }

    if (backToDashboard) {
        backToDashboard.addEventListener('click', showDashboardPage);
    }

    if (settingsTabs.length > 0 && settingsContents.length > 0) {
        settingsTabs.forEach(tab => {
            tab.addEventListener('click', () => {
                const targetTab = tab.getAttribute('data-tab');
                if (!targetTab) {
                    return;
                }

                settingsTabs.forEach(btn => {
                    btn.classList.remove('active');
                });
                tab.classList.add('active');

                settingsContents.forEach(content => {
                    if (content.getAttribute('data-content') === targetTab) {
                        content.classList.add('active');
                        content.classList.remove('hidden');
                    } else {
                        content.classList.remove('active');
                        content.classList.add('hidden');
                    }
                });
            });
        });
    }

    const dashboardCheckBtn = document.getElementById('dashBtnCheckCert');
    if (dashboardCheckBtn) {
        dashboardCheckBtn.addEventListener('click', () => {
            showSettingsPage();
            const tab = document.querySelector('.settings-tab[data-tab="system"]');
            if (tab instanceof HTMLElement) {
                tab.click();
            }
        });
    }

    const dashboardInstallBtn = document.getElementById('dashBtnInstallCert');
    if (dashboardInstallBtn) {
        dashboardInstallBtn.addEventListener('click', () => {
            showSettingsPage();
            const tab = document.querySelector('.settings-tab[data-tab="system"]');
            if (tab instanceof HTMLElement) {
                tab.click();
            }
            const installBtn = document.getElementById('btn-install-cert');
            if (installBtn instanceof HTMLElement) {
                installBtn.click();
            }
        });
    }
});
