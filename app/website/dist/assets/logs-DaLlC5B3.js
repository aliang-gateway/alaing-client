function setLogsOutputText(output, text) {
    if (!output) {
        return;
    }
    if (output instanceof HTMLTextAreaElement || output instanceof HTMLInputElement) {
        output.value = text;
        return;
    }
    output.textContent = text;
}

async function loadLogs() {
    try {
        const level = appState.logConfig.level || '';
        const source = appState.logConfig.source || '';
        
        const params = new URLSearchParams();
        if (level) params.append('level', level);
        if (source) params.append('source', source);
        params.append('limit', '1000');
        
        const queryString = params.toString();
        const endpoint = queryString ? `/logs?${queryString}` : '/logs';
        
        const response = await apiGet(endpoint);
        const output = document.getElementById('logsOutput');
        
        if (response.entries && Array.isArray(response.entries)) {
            const logText = response.entries.map(entry => {
                const level = entry.level || 'INFO';
                const timestamp = entry.timestamp || '';
                const source = entry.source ? `[${entry.source}]` : '';
                const message = entry.message || '';
                return `${timestamp} ${level} ${source} ${message}`;
            }).join('\n');
            setLogsOutputText(output, logText);
        } else {
            setLogsOutputText(output, '');
        }
        
        if (appState.logConfig.autoScroll) {
            output.scrollTop = output.scrollHeight;
        }
    } catch (error) {
        console.error('加载日志失败:', error);
        const output = document.getElementById('logsOutput');
        if (output) {
            setLogsOutputText(output, `加载日志失败: ${error.message}`);
        }
    }
}

document.getElementById('logsRefreshBtn').addEventListener('click', (event) => {
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

document.getElementById('logsClearBtn').addEventListener('click', (event) => {
    const btn = event.target.closest('button');
    if (!confirm('确定要清空所有日志吗？')) {
        return;
    }
    showLoading(btn);

    const output = document.getElementById('logsOutput');
    if (output) {
        setLogsOutputText(output, '');
    }

    apiPost('/logs/clear')
        .then(() => {
            showSuccess('日志已清空');
            loadLogs();
        })
        .catch(error => {
            showError('清空失败: ' + error.message);
        })
        .finally(() => {
            hideLoading(btn);
        });
});

async function loadLogConfig() {
    try {
        const config = await apiGet('/logs/config');
        
        const level = (config.level || 'INFO').toUpperCase();
        
        appState.logConfig = {
            level: level,
            source: appState.logConfig?.source || '',
            autoScroll: appState.logConfig?.autoScroll !== false
        };

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
        appState.logConfig = {
            level: 'INFO',
            source: '',
            autoScroll: true
        };
    }
}

async function saveLogConfig(event) {
    const btn = event.target.closest('button');
    showLoading(btn);

    const level = document.getElementById('logLevelSelect').value.toUpperCase();
    
    const configData = {
        level: level
    };

    try {
        await apiPost('/logs/config', configData);
        appState.logConfig.level = level;
        showSuccess('日志级别已更新');
        loadLogs();
    } catch (error) {
        showError('保存失败: ' + error.message);
    } finally {
        hideLoading(btn);
    }
}

function applyLogFilter() {
    const source = document.getElementById('logSourceSelect').value;
    appState.logConfig.source = source;
    appState.logConfig.autoScroll = document.getElementById('logAutoScroll').checked;
    
    loadLogs();
    showSuccess('过滤已应用');
}

document.getElementById('logConfigSaveBtn').addEventListener('click', saveLogConfig);
document.getElementById('logFilterBtn').addEventListener('click', applyLogFilter);

document.getElementById('wsConnectBtn').addEventListener('click', (event) => {
    const btn = event.target.closest('button');
    showLoading(btn);
    logWebSocket.connect();
    hideLoading(btn);
});

document.getElementById('wsDisconnectBtn').addEventListener('click', (event) => {
    const btn = event.target.closest('button');
    showLoading(btn);
    logWebSocket.disconnect();
    showSuccess('实时日志连接已断开');
    hideLoading(btn);
});

document.getElementById('logsFullscreenBtn').addEventListener('click', () => {
    const logsPage = document.getElementById('logs-page');
    const fullscreenBtn = document.getElementById('logsFullscreenBtn');
    const btnIcon = fullscreenBtn.querySelector('i');

    if (logsPage.classList.contains('fullscreen-mode')) {
        logsPage.classList.remove('fullscreen-mode');
        if (document.exitFullscreen) {
            document.exitFullscreen().catch(err => {
                console.log('Exit fullscreen failed:', err);
            });
        }
        btnIcon.className = 'bi bi-fullscreen';
        fullscreenBtn.innerHTML = '<i class="bi bi-fullscreen"></i> 全屏';
    } else {
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

document.addEventListener('fullscreenchange', () => {
    const logsPage = document.getElementById('logs-page');
    const fullscreenBtn = document.getElementById('logsFullscreenBtn');

    if (!document.fullscreenElement && logsPage.classList.contains('fullscreen-mode')) {
        logsPage.classList.remove('fullscreen-mode');
        fullscreenBtn.innerHTML = '<i class="bi bi-fullscreen"></i> 全屏';
    }
});
