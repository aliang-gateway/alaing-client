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

// ============ API 客户端 ============
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
        statusText.textContent = `Nursorgate2 - ${runningStatus}`;
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
    const alertClass = type === 'success' ? 'alert-success' : 'alert-danger';
    const alertHtml = `
        <div class="alert ${alertClass} alert-dismissible fade show notification" role="alert">
            ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        </div>
    `;

    document.body.insertAdjacentHTML('beforeend', alertHtml);

    // 自动移除通知
    setTimeout(() => {
        const alert = document.querySelector('.notification:last-child');
        if (alert) {
            alert.remove();
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
            statusText.textContent = 'Nursorgate2 - 运行中';
        } else {
            indicator.className = 'status-indicator stopped';
            statusText.textContent = 'Nursorgate2 - 已停止';
        }
        
        // 更新按钮状态
        updateButtonStates();
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

// ============ 代理管理功能 ============
async function loadProxyData() {
    try {
        const [proxyListData, doorMembersData] = await Promise.all([
            apiGet('/proxy/list'),
            apiGet('/proxy/door/members').catch(() => ({ members: [] }))
        ]);

        // 获取当前代理，如果失败则为 null
        let currentProxy = null;
        try {
            const currentProxyData = await apiGet('/proxy/current/get');
            // 如果API返回了有效的代理信息且没有错误，提取name字段
            if (currentProxyData && currentProxyData.name && !currentProxyData.error) {
                currentProxy = currentProxyData.name;
            }
        } catch (error) {
            console.log('当前代理未设置:', error.message);
        }

        console.log('原始数据:', { proxyListData, currentProxy, doorMembersData });

        // 处理代理列表数据 - ListProxies 返回 { proxies: {...}, count: ... }
        let allProxies = [];
        if (proxyListData) {
            if (proxyListData.proxies && typeof proxyListData.proxies === 'object') {
                // 如果是对象格式 { name: {...}, ...}，转换为数组
                allProxies = Object.keys(proxyListData.proxies).map(name => ({
                    name: name,
                    ...proxyListData.proxies[name]
                }));
            } else if (Array.isArray(proxyListData)) {
                // 如果已经是数组
                allProxies = proxyListData;
            }
        }

        const members = Array.isArray(doorMembersData?.members) ? doorMembersData.members : [];

        console.log('转换后的代理列表:', allProxies);
        console.log('当前代理值:', currentProxy);

        // 过滤出 door 的虚拟成员（名称格式为 door:xxx）
        const doorMembers = allProxies.filter(proxy =>
            proxy.name && proxy.name.startsWith('door:')
        );

        console.log('真实 Door 代理:', allProxies.find(p => p.name === 'door'));
        console.log('Door 成员:', doorMembers);

        // 如果当前代理为空或未设置，默认设置为 door 代理
        let selectedProxy = currentProxy;
        if (!selectedProxy && doorMembers.length > 0) {
            selectedProxy = doorMembers[0].name;
            console.log('当前代理未设置，自动设置为:', selectedProxy);
            try {
                // 设置当前代理为第一个 door 成员
                await apiPost('/proxy/current/set', { name: selectedProxy });
                console.log('已自动设置当前代理为:', selectedProxy);
            } catch (error) {
                console.error('自动设置当前代理失败:', error);
            }
        }

        // 更新当前代理选择框 - 显示所有 door 成员
        const select = document.getElementById('proxySelect');
        select.innerHTML = '';
        if (doorMembers.length === 0) {
            select.innerHTML = '<option>暂无 Door 成员</option>';
        } else {
            doorMembers.forEach(proxy => {
                const option = document.createElement('option');
                option.value = proxy.name;
                // 显示时只显示成员名称，不显示 "door:" 前缀
                const displayName = proxy.name.substring(5);
                option.textContent = displayName;
                if (proxy.name === selectedProxy) {
                    option.selected = true;
                }
                select.appendChild(option);
            });
        }

        // 更新所有代理表格
        const tbody = document.getElementById('proxyTableBody');
        if (allProxies.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4" class="text-center text-muted">暂无代理</td></tr>';
        } else {
            tbody.innerHTML = allProxies.map(proxy => {
                // 判断是否是当前代理
                const isCurrent = proxy.name === selectedProxy;
                // 显示名称：虚拟的 door 成员时隐藏 "door:" 前缀
                const displayName = proxy.name && proxy.name.startsWith('door:')
                    ? proxy.name.substring(5)
                    : proxy.name;
                return `
                    <tr>
                        <td>${displayName || '-'}</td>
                        <td><span class="badge bg-primary">${proxy.type || 'Unknown'}</span></td>
                        <td>${proxy.addr || '-'}</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" onclick="switchProxy('${proxy.name}')">
                                ${isCurrent ? '✓ 当前' : '切换'}
                            </button>
                        </td>
                    </tr>
                `;
            }).join('');
        }

        // 更新 Door 成员表格 - 只显示 door 代理的成员
        const doorBody = document.getElementById('doorTableBody');
        if (members.length === 0) {
            doorBody.innerHTML = '<tr><td colspan="4" class="text-center text-muted">暂无 Door 成员</td></tr>';
        } else {
            doorBody.innerHTML = members.map(member => `
                <tr>
                    <td>${member.show_name || member.name || '-'}</td>
                    <td>${member.latency || '-'}ms</td>
                    <td><span class="badge bg-secondary">${member.type || 'Unknown'}</span></td>
                    <td>
                        <button class="btn btn-sm btn-outline-primary" onclick="switchDoorMember('${member.show_name || member.name}')">
                            切换
                        </button>
                    </td>
                </tr>
            `).join('');
        }

        appState.proxies = allProxies;
        appState.currentProxy = selectedProxy;
        appState.doorMembers = members;
    } catch (error) {
        console.error('加载代理数据失败:', error);
        document.getElementById('proxyTableBody').innerHTML = '<tr><td colspan="4" class="text-center text-danger">加载失败: ' + error.message + '</td></tr>';
        document.getElementById('doorTableBody').innerHTML = '<tr><td colspan="4" class="text-center text-danger">加载失败: ' + error.message + '</td></tr>';
    }
}

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
    const mode = document.querySelector('input[name="runMode"]:checked').value;
    showLoading(btn);
    apiPost('/run/mode', { mode: mode })
        .then(() => {
            showSuccess(`已切换到 ${mode} 模式`);
            loadRunStatus();
            loadDashboard();
        })
        .catch(error => {
            showError('模式切换失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

document.getElementById('runUserInfoBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    const username = document.getElementById('runUsername').value;
    const password = document.getElementById('runPassword').value;
    const url = document.getElementById('runUrl').value;

    if (!username || !password || !url) {
        showError('请填写完整的用户信息');
        return;
    }

    showLoading(btn);
    apiPost('/run/userinfo', {
        username: username,
        password: password,
        url: url
    })
        .then(() => {
            showSuccess('用户信息更新成功');
        })
        .catch(error => {
            showError('更新失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
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
                statusBadge.className = 'badge bg-success';
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
                statusClass = 'badge bg-success';
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

// ============ Token 管理 ============
async function loadToken() {
    try {
        const token = await apiGet('/token/get');
        document.getElementById('tokenOutput').value = token.token || '';
    } catch (error) {
        console.error('加载 Token 失败:', error);
    }
}

document.getElementById('tokenRefreshBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    loadToken()
        .then(() => {
            showSuccess('Token 已刷新');
        })
        .catch(error => {
            showError('刷新失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

document.getElementById('tokenSaveBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    const token = document.getElementById('tokenOutput').value;

    if (!token) {
        showError('请输入 Token');
        return;
    }

    showLoading(btn);
    apiPost('/token/set', { token: token })
        .then(() => {
            showSuccess('Token 已保存');
        })
        .catch(error => {
            showError('保存失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

document.getElementById('tokenCopyBtn').addEventListener('click', () => {
    const token = document.getElementById('tokenOutput').value;
    if (!token) {
        showError('没有可复制的 Token');
        return;
    }

    navigator.clipboard.writeText(token)
        .then(() => {
            showSuccess('Token 已复制到剪贴板');
        })
        .catch(error => {
            showError('复制失败');
        });
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

// 加载代理统计数据
async function loadStatsData() {
    try {
        const response = await apiGet('/stats');
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
    // 隐藏所有页面
    document.querySelectorAll('.content-section').forEach(el => {
        el.classList.remove('active');
    });

    // 显示选定页面
    const pageEl = document.getElementById(`${page}-page`);
    if (pageEl) {
        pageEl.classList.add('active');
    }

    // 更新菜单活跃状态
    document.querySelectorAll('.sidebar .nav-link').forEach(link => {
        link.classList.remove('active');
        if (link.dataset.page === page) {
            link.classList.add('active');
        }
    });

    appState.currentPage = page;

    // 触发页面特定的初始化
    if (page === 'proxy') {
        loadProxyData();
    } else if (page === 'run') {
        loadRunStatus();
    } else if (page === 'rules') {
        loadRulesData();
    } else if (page === 'logs') {
        // 连接 WebSocket 进行实时日志
        logWebSocket.connect();
        // 加载历史日志
        loadLogs();
        // 加载日志配置
        loadLogConfig();
    } else if (page === 'tokens') {
        loadToken();
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
    if (appState.currentPage === 'logs') {
        // 断开 WebSocket 连接
        logWebSocket.disconnect();
    }
}

// ============ 菜单导航事件 ============
document.querySelectorAll('.sidebar .nav-link').forEach(link => {
    link.addEventListener('click', (e) => {
        e.preventDefault();
        const page = link.dataset.page;

        // 清理当前页面
        cleanupCurrentPage();

        // 切换到新页面
        switchPage(page);
    });
});

// ============ 页面加载完成后初始化 ============
document.addEventListener('DOMContentLoaded', () => {
    // 加载仪表板数据
    loadDashboard();

    // 初始化按钮状态
    logWebSocket.updateConnectionStatus('disconnected');

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
        const container = document.getElementById('cert-status-container');
        if (!certStatus || !certStatus.cert_type) {
            container.innerHTML = '<div class="alert alert-warning">暂无证书信息</div>';
            return;
        }

        // 构建状态 badges (NEW)
        const exportedBadge = certStatus.is_exported
            ? '<span class="badge bg-success"><i class="bi bi-check-circle"></i> 已导出</span>'
            : '<span class="badge bg-danger"><i class="bi bi-x-circle"></i> 未导出</span>';

        const installedBadge = certStatus.is_installed
            ? '<span class="badge bg-success"><i class="bi bi-check-circle"></i> 已安装</span>'
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
            'system_trusted': '<span class="badge bg-success">系统信任</span>',
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
});