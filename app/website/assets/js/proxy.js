async function testProxyLatency() {
    const btn = document.getElementById('testLatencyBtn');
    if (!btn) return;

    const originalText = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>测试中...';

    try {
        const result = await apiPost('/proxy/door/test-latency', {});

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

async function loadProxyData() {
    try {
        const proxyListData = await apiGet('/proxy/list');

        let currentProxy = null;
        let isDoorAutoMode = false;
        let currentProxyInfo = null;

        if (proxyListData) {
            if (proxyListData.current_proxy) {
                currentProxy = proxyListData.current_proxy.name;
                currentProxyInfo = proxyListData.current_proxy;
            }
            isDoorAutoMode = proxyListData.is_door_auto_mode || false;
        }

        let allProxies = [];
        if (proxyListData) {
            if (proxyListData.proxies && typeof proxyListData.proxies === 'object') {
                allProxies = Object.keys(proxyListData.proxies).map(name => ({
                    name: name,
                    ...proxyListData.proxies[name]
                }));
            } else if (Array.isArray(proxyListData.proxies)) {
                allProxies = proxyListData.proxies;
            }
        }

        console.log('完整代理列表:', allProxies);
        console.log('当前代理:', currentProxy);
        console.log('Door自动模式:', isDoorAutoMode);

        let selectedProxy = currentProxy;

        const doorProxies = allProxies.filter(proxy => proxy.name && proxy.name.startsWith('door:'));

        const select = document.getElementById('proxySelect');
        select.innerHTML = '';
        if (doorProxies.length === 0) {
            select.innerHTML = '<option>暂无 Door 成员</option>';
        } else {
            const autoModeHint = isDoorAutoMode ? ' (自动模式)' : '';
            doorProxies.forEach(proxy => {
                const option = document.createElement('option');
                option.value = proxy.name;
                const displayName = proxy.name.substring(5);
                option.textContent = displayName + autoModeHint;
                if (proxy.name === selectedProxy) {
                    option.selected = true;
                }
                select.appendChild(option);
            });
        }

        const currentProxyDisplay = document.getElementById('currentProxyDisplay');
        if (currentProxyDisplay && currentProxyInfo) {
            const displayName = currentProxyInfo.show_name || currentProxyInfo.name;
            const autoModeLabel = isDoorAutoMode && currentProxyInfo.name.startsWith('door:')
                ? ' <span class="badge bg-info ms-2">自动模式</span>'
                : '';
            currentProxyDisplay.innerHTML = `当前代理: <strong>${displayName}</strong>${autoModeLabel}`;
        }

        const tbody = document.getElementById('proxyTableBody');
        if (allProxies.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">暂无代理</td></tr>';
        } else {
            tbody.innerHTML = allProxies.map(proxy => {
                const isCurrent = proxy.name === selectedProxy;
                const displayName = proxy.name && proxy.name.startsWith('door:')
                    ? proxy.name.substring(5)
                    : proxy.name;

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

async function showProxyDetail(proxyName) {
    try {
        const response = await apiGet(`/proxy/get?name=${encodeURIComponent(proxyName)}`);

        const displayName = proxyName.startsWith('door:') ? proxyName.substring(5) : proxyName;
        document.getElementById('proxyDetailTitle').textContent = displayName;

        const jsonStr = JSON.stringify(response, null, 2);
        document.getElementById('proxyDetailJson').textContent = jsonStr;

        const modal = new bootstrap.Modal(document.getElementById('proxyDetailModal'));
        modal.show();
    } catch (error) {
        showError('获取代理详情失败: ' + error.message);
    }
}

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
        console.log('正在切换代理到:', proxyName);

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

if (document.getElementById('testLatencyBtn')) {
    document.getElementById('testLatencyBtn').addEventListener('click', testProxyLatency);
}
