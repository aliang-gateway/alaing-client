async function loadDNSCacheData() {
    try {
        const [statsResp, hotspotsResp] = await Promise.all([
            apiGet('/dns/stats').catch(() => null),
            apiGet('/dns/hotspots').catch(() => null)
        ]);

        if (statsResp) {
            document.getElementById('dnsTotalEntries').textContent = formatNumber(statsResp.size || 0);
            document.getElementById('dnsUniqueDomains').textContent = formatNumber(statsResp.uniqueDomains || 0);
            document.getElementById('dnsUniqueIPs').textContent = formatNumber(statsResp.uniqueIPs || 0);
            document.getElementById('dnsHitRate').textContent = (statsResp.hitRate || 0).toFixed(1) + '%';
        }

        if (hotspotsResp) {
            renderDNSHotDomains(hotspotsResp.topDomains);
            renderDNSHotIPs(hotspotsResp.topIPs);
        }
    } catch (error) {
        console.error('Failed to load DNS cache data:', error);
    }
}

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

async function performDNSSearch() {
    const query = document.getElementById('dnsSearchBox').value.trim();
    if (!query) {
        document.getElementById('dnsSearchResultsContainer').innerHTML =
            '<div class="text-center text-muted py-4">请输入要搜索的域名或IP地址</div>';
        return;
    }

    try {
        let response = await apiCall(`/dns/cache/query?domain=${encodeURIComponent(query)}`).catch(() => null);

        if (!response) {
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
