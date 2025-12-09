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
});