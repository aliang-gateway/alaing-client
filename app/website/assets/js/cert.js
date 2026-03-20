function updateCertModalLastRefreshed() {
    const refreshed = document.getElementById('certModalLastRefreshed');
    if (refreshed) {
        refreshed.textContent = `Last refreshed: ${new Date().toLocaleString()}`;
    }
}

function openCertManagementModal() {
    const modal = document.getElementById('certManagementModal');
    if (!modal) {
        return;
    }
    modal.classList.remove('hidden');
    modal.classList.add('flex');
    loadCertStatus();
}

function closeCertManagementModal() {
    const modal = document.getElementById('certManagementModal');
    if (!modal) {
        return;
    }
    modal.classList.add('hidden');
    modal.classList.remove('flex');
}

async function loadCertStatus() {
    try {
        const certType = document.getElementById('cert-type-select')?.value || 'mitm-ca';
        const response = await apiGet(`/cert/status?cert_type=${encodeURIComponent(certType)}`);
        updateCertStatusDisplay(response);
        updateCertModalLastRefreshed();
    } catch (error) {
        console.error('Failed to load cert status:', error);
    }
}

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

async function exportCert() {
    const certType = document.getElementById('cert-type-select').value;
    const btn = document.getElementById('btn-export-cert');
    showLoading(btn);

    try {
        const result = await apiPost('/cert/export', { cert_type: certType });
        showSuccess(`证书已导出到: ${result.export_path}`);
        loadCertStatus();
    } catch (error) {
        showError('导出失败: ' + error.message);
    } finally {
        hideLoading(btn);
    }
}

async function downloadCert() {
    const certType = document.getElementById('cert-type-select').value;
    downloadFile(`/cert/download?cert_type=${certType}`, `${certType}.pem`);
}

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
        loadCertStatus();
    } catch (error) {
        showError('安装失败: ' + error.message);
    } finally {
        hideLoading(btn);
    }
}

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
        loadCertStatus();
    } catch (error) {
        showError('移除失败: ' + error.message);
    } finally {
        hideLoading(btn);
    }
}

function updateCertStatusDisplay(certStatus) {
    const container = document.getElementById('cert-status-content') || document.getElementById('cert-status-container');
    if (!certStatus || !certStatus.cert_type) {
        container.innerHTML = '<div class="alert alert-warning">暂无证书信息</div>';
        return;
    }

    const exportedBadge = certStatus.is_exported
        ? '<span class="badge bg-danger"><i class="bi bi-check-circle"></i> 已导出</span>'
        : '<span class="badge bg-danger"><i class="bi bi-x-circle"></i> 未导出</span>';

    const installedBadge = certStatus.is_installed
        ? '<span class="badge bg-danger"><i class="bi bi-check-circle"></i> 已安装</span>'
        : '<span class="badge bg-danger"><i class="bi bi-x-circle"></i> 未安装</span>';

    const trustedBadge = certStatus.is_trusted
        ? '<span class="badge bg-info"><i class="bi bi-shield-check"></i> 已信任</span>'
        : '<span class="badge bg-warning"><i class="bi bi-exclamation-triangle"></i> 未信任</span>';

    const trustStatusText = getTrustStatusText(certStatus.trust_status);
    const trustStatusBadge = getTrustStatusBadge(certStatus.trust_status);

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

function getTrustStatusText(trustStatus) {
    const statusMap = {
        'not_found': '证书不存在',
        'installed_not_trusted': '已安装但未信任',
        'system_trusted': '系统信任',
        'unsupported_platform': '不支持的平台'
    };
    return statusMap[trustStatus] || trustStatus;
}

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
