// ============ 全局状态管理 ============
const appState = {
    currentPage: 'dashboard',
    proxies: [],
    currentProxy: null,
    doorMembers: [],
    loading: false
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

        // 更新仪表板
        const runningStatus = runStatus.tun_running ? 'TUN 运行中' : 'HTTP 模式';
        document.getElementById('dashRunStatus').textContent = runningStatus;
        document.getElementById('dashCurrentProxy').textContent = currentProxyDisplay;
        document.getElementById('dashRunMode').textContent = runStatus.current_mode || '-';
        document.getElementById('dashRuleStatus').textContent = rulesStatus?.status === 'enabled' ? '启用' : '禁用';
        // proxyList 返回格式是 { proxies: {...}, count: ... }
        const proxyCount = proxyList?.proxies ? Object.keys(proxyList.proxies).length : (proxyList?.count || 0);
        document.getElementById('dashProxyCount').textContent = proxyCount;
        document.getElementById('dashDoorCount').textContent = doorMembers?.members?.length || 0;
        document.getElementById('dashLastUpdate').textContent = new Date().toLocaleTimeString();

        // 更新顶部状态
        const indicator = document.getElementById('statusIndicator');
        const statusText = document.getElementById('statusText');
        if (runStatus.tun_running) {
            indicator.className = 'status-indicator running';
            statusText.textContent = '运行中';
        } else {
            indicator.className = 'status-indicator stopped';
            statusText.textContent = '已停止';
        }
    } catch (error) {
        console.error('加载仪表板数据失败:', error);
    }
}

document.getElementById('dashStartBtn').addEventListener('click', () => {
    const btn = event.target.closest('button');
    showLoading(btn);
    apiPost('/run/start')
        .then(() => {
            showSuccess('服务启动成功');
            loadDashboard();
            loadRunStatus();
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
        .then(() => {
            showSuccess('服务停止成功');
            loadDashboard();
            loadRunStatus();
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
        document.getElementById('runCurrentMode').textContent = status.current_mode || '-';
        document.getElementById('runServiceStatus').textContent = status.tun_running ? '运行中' : '已停止';
        document.getElementById('runAvailableModes').textContent = status.available_modes?.join(' / ') || '-';
        document.getElementById('runStatusInfo').textContent = status.status || '-';
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
        const [status, cacheInfo] = await Promise.all([
            apiGet('/rules/engine/status'),
            apiGet('/rules/cache/info').catch(() => null)
        ]);

        document.getElementById('rulesStatus').textContent = status.status === 'enabled' ? '启用' : '禁用';
        document.getElementById('rulesLastUpdate').textContent = status.last_update || '-';
        document.getElementById('rulesGeoipStatus').textContent = status.geoip_enabled ? '已加载' : '未加载';

        if (cacheInfo) {
            document.getElementById('rulesCacheSize').textContent = cacheInfo.size || 0;
            document.getElementById('rulesCacheCount').textContent = cacheInfo.count || 0;
        }
    } catch (error) {
        console.error('加载规则数据失败:', error);
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
    apiPost('/rules/rules/reload')
        .then(() => {
            showSuccess('规则已重新加载');
            loadRulesData();
        })
        .catch(error => {
            showError('重新加载失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
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
    const domain = document.getElementById('rulesLookupDomain').value;

    if (!domain) {
        showError('请输入域名');
        return;
    }

    showLoading(btn);
    apiPost('/rules/geoip/lookup', { domain: domain })
        .then(result => {
            const output = document.getElementById('rulesLookupResult');
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
        const logs = await apiGet('/logs/get');
        const output = document.getElementById('logsOutput');
        output.value = logs.logs || '';
        output.scrollTop = output.scrollHeight;
    } catch (error) {
        console.error('加载日志失败:', error);
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

    // 设置定时刷新
    setInterval(() => {
        if (appState.currentPage === 'dashboard') {
            loadDashboard();
        }
    }, 5000);
});