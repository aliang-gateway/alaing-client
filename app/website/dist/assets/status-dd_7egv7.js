async function updateGlobalStatus() {
    try {
        const [runStatus, rulesStatus] = await Promise.all([
            apiGet('/run/status').catch(() => ({ is_running: false, current_mode: 'unknown' })),
            apiGet('/rules/engine/status').catch(() => ({ engineEnabled: false }))
        ]);

        const oldRunningState = appState.runStatus.isRunning;
        const oldRuleEngineState = appState.ruleEngineStatus.isEnabled;

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

        updateStatusIndicator();

        if (oldRunningState !== appState.runStatus.isRunning ||
            oldRuleEngineState !== appState.ruleEngineStatus.isEnabled) {
            updateButtonStates();
        }

    } catch (error) {
        console.error('Status update failed:', error);
    }
}

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

    updateModeRadioButtons();
}

function updateModeRadioButtons() {
    const tunRadio = document.getElementById('runModeTun');
    const httpRadio = document.getElementById('runModeHttp');

    if (tunRadio && httpRadio && appState.runStatus.currentMode) {
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

function updateButtonStates() {
    updateDashboardButtonStates();
    updateRulesButtonStates();
}

function updateDashboardButtonStates() {
    const dashStartBtn = document.getElementById('dashStartBtn');
    const dashStopBtn = document.getElementById('dashStopBtn');
    const runStartBtn = document.getElementById('runStartBtn');
    const runStopBtn = document.getElementById('runStopBtn');

    if (appState.runStatus.isRunning) {
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

function updateRulesButtonStates() {
    const rulesEnableBtn = document.getElementById('rulesEnableBtn');
    const rulesDisableBtn = document.getElementById('rulesDisableBtn');

    if (rulesEnableBtn && rulesDisableBtn) {
        if (appState.ruleEngineStatus.isEnabled) {
            rulesEnableBtn.disabled = true;
            rulesEnableBtn.classList.add('disabled');
            rulesDisableBtn.disabled = false;
            rulesDisableBtn.classList.remove('disabled');
        } else {
            rulesEnableBtn.disabled = false;
            rulesEnableBtn.classList.remove('disabled');
            rulesDisableBtn.disabled = true;
            rulesDisableBtn.classList.add('disabled');
        }
    }
}

async function loadDashboard() {
    try {
        const [runStatus, proxyList, doorMembers, rulesStatus] = await Promise.all([
            apiGet('/run/status'),
            apiGet('/proxy/list'),
            apiGet('/proxy/door/members').catch(() => ({ members: [] })),
            apiGet('/rules/engine/status').catch(() => null)
        ]);

        let currentProxyDisplay = '-';
        try {
            const currentProxy = await apiGet('/proxy/current/get');
            if (currentProxy && currentProxy.name && !currentProxy.error) {
                currentProxyDisplay = currentProxy.show_name || currentProxy.name;
            }
        } catch (error) {
            console.log('当前代理未设置:', error.message);
        }

        const isRunning = runStatus.is_running || false;
        
        appState.runStatus.isRunning = isRunning;
        appState.runStatus.currentMode = runStatus.current_mode || 'unknown';
        appState.runStatus.lastUpdated = Date.now();

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
        const proxyCount = proxyList?.proxies ? Object.keys(proxyList.proxies).length : (proxyList?.count || 0);
        document.getElementById('dashProxyCount').textContent = proxyCount;
        document.getElementById('dashDoorCount').textContent = doorMembers?.members?.length || 0;
        document.getElementById('dashLastUpdate').textContent = new Date().toLocaleTimeString();

        const indicator = document.getElementById('statusIndicator');
        const statusText = document.getElementById('statusText');
        if (isRunning) {
            indicator.className = 'status-indicator running';
            statusText.textContent = 'NonelaneCore - 运行中';
        } else {
            indicator.className = 'status-indicator stopped';
            statusText.textContent = 'NonelaneCore - 已停止';
        }
        
        updateButtonStates();

        await loadStatsData().catch(error => console.error('Failed to load stats:', error));

        initChart();
    } catch (error) {
        console.error('加载仪表板数据失败:', error);
    }
}

document.getElementById('dashStartBtn').addEventListener('click', (event) => {
    const btn = event.currentTarget;
    showLoading(btn);
    apiPost('/run/start')
        .then((result) => {
            if (result && (result.status === 'success' || result.status === 'already_running')) {
                appState.runStatus.isRunning = true;
                updateStatusIndicator();
                updateButtonStates();
                showSuccess(result.message || '服务启动成功');
            } else {
                showError(result?.msg || result?.message || '服务启动失败');
            }
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

document.getElementById('dashStopBtn').addEventListener('click', (event) => {
    const btn = event.currentTarget;
    showLoading(btn);
    apiPost('/run/stop')
        .then((result) => {
            if (result && result.status === 'success') {
                appState.runStatus.isRunning = false;
                updateStatusIndicator();
                updateButtonStates();
                showSuccess(result.message || '服务停止成功');
            } else {
                showError(result?.msg || result?.message || '服务停止失败');
            }
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
