async function loadRunStatus() {
    try {
        const status = await apiGet('/run/status');
        
        const isRunning = status.is_running || false;
        
        document.getElementById('runCurrentMode').textContent = status.current_mode || '-';
        document.getElementById('runServiceStatus').textContent = isRunning ? '运行中' : '已停止';
        document.getElementById('runAvailableModes').textContent = status.available_modes?.join(' / ') || '-';
        document.getElementById('runStatusInfo').textContent = status.status || status.description || '-';
        
        appState.runStatus.isRunning = isRunning;
        appState.runStatus.currentMode = status.current_mode || 'unknown';
        
        updateButtonStates();
    } catch (error) {
        console.error('加载运行状态失败:', error);
    }
}

document.getElementById('runStartBtn').addEventListener('click', (event) => {
    const btn = event.currentTarget;
    showLoading(btn);
    apiPost('/run/start')
        .then((result) => {
            if (result && (result.status === 'success' || result.status === 'already_running')) {
                showSuccess(result.message || '服务启动成功');
            } else {
                showError(result?.msg || result?.message || '服务启动失败');
                return;
            }
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

document.getElementById('runStopBtn').addEventListener('click', (event) => {
    const btn = event.currentTarget;
    showLoading(btn);
    apiPost('/run/stop')
        .then((result) => {
            if (result && result.status === 'success') {
                showSuccess(result.message || '服务停止成功');
            } else {
                showError(result?.msg || result?.message || '服务停止失败');
                return;
            }
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

document.getElementById('runModeBtn').addEventListener('click', (event) => {
    const btn = event.currentTarget;
    const currentMode = document.querySelector('input[name="runMode"]:checked').value;

    var mode = currentMode;
    if (currentMode === "http") {
        mode = "tun";
    } else {
        mode = "http";
    }
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
