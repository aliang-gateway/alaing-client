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

function displayUserInfo(userInfo) {
    const container = document.getElementById('authUserInfoContainer');
    if (!container) return;
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

    const logoutBtn = document.getElementById('authLogoutBtn');
    if (logoutBtn) logoutBtn.style.display = 'inline-block';

    const refreshCard = document.getElementById('authRefreshStatusCard');
    if (refreshCard) refreshCard.style.display = 'block';

    if (balanceEl) {
        const balance = Number(userInfo.balance || userInfo.amount || 0);
        balanceEl.textContent = `${balance.toFixed(2)} CNY`;
    }
}

function displayNoUserInfo() {
    const container = document.getElementById('authUserInfoContainer');
    if (!container) return;
    const balanceEl = document.getElementById('userBalance');
    container.innerHTML = `
        <div class="text-center text-muted py-4">
            <p>暂无用户信息，请先激活 Token</p>
        </div>
    `;

    const logoutBtn = document.getElementById('authLogoutBtn');
    if (logoutBtn) logoutBtn.style.display = 'none';

    const refreshCard = document.getElementById('authRefreshStatusCard');
    if (refreshCard) refreshCard.style.display = 'none';

    if (balanceEl) {
        balanceEl.textContent = '--';
    }
}

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

function updateRefreshStatus(status) {
    const runningBadge = document.getElementById('authRefreshRunning');
    const lastUpdateEl = document.getElementById('authLastUpdate');
    const refreshIntervalEl = document.getElementById('authRefreshInterval');
    const errorEl = document.getElementById('authRefreshError');
    const errorMsgEl = document.getElementById('authRefreshErrorMsg');

    if (!runningBadge) return;

    runningBadge.className = status.is_running ? 'badge bg-success' : 'badge bg-secondary';
    runningBadge.textContent = status.is_running ? '运行中' : '未运行';

    if (lastUpdateEl && status.last_update_time) {
        lastUpdateEl.textContent = formatDateTime(status.last_update_time);
    }

    if (refreshIntervalEl) {
        refreshIntervalEl.textContent = status.refresh_interval || '1 分钟';
    }

    if (status.last_error) {
        if (errorEl) errorEl.style.display = 'block';
        if (errorMsgEl) errorMsgEl.textContent = status.last_error;
    } else {
        if (errorEl) errorEl.style.display = 'none';
        if (errorMsgEl) errorMsgEl.textContent = '';
    }
}

document.getElementById('authActivateBtn')?.addEventListener('click', async () => {
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

document.getElementById('authLogoutBtn')?.addEventListener('click', async () => {
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
        hideLoading(btn);
    }
});
