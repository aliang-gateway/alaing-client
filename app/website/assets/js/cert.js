const CERT_TYPES = {
    MITM_CA: 'mitm-ca',
    ROOT_CA: 'root-ca',
    MTLS_CERT: 'mtls-cert'
};

const CERT_TYPE_PARAM = 'cert_type';

const CERT_API_CONTRACT = {
    LOAD_STATUS: { method: 'GET', path: '/cert/status', paramsSource: 'query' },
    CHECK_STATUS: { method: 'GET', path: '/cert/status', paramsSource: 'query' },
    INFO_CERT: { method: 'GET', path: '/cert/info', paramsSource: 'query' },
    GENERATE_CERT: { method: 'POST', path: '/cert/generate', paramsSource: 'body' },
    EXPORT_CERT: { method: 'POST', path: '/cert/export', paramsSource: 'body' },
    DOWNLOAD_CERT: { method: 'GET', path: '/cert/download', paramsSource: 'query' },
    INSTALL_CERT: { method: 'POST', path: '/cert/install', paramsSource: 'body' },
    REMOVE_CERT: { method: 'POST', path: '/cert/remove', paramsSource: 'body' }
};

const DEFAULT_CERT_TYPE = CERT_TYPES.MITM_CA;
const CERT_AUDIT_STORAGE_KEY = 'cert-operation-audit-v1';

const certOverviewState = {};
let certOperationLockCount = 0;

const CERT_ACTION_BUTTON_IDS = [
    'btn-check-cert',
    'btn-export-cert',
    'btn-install-cert',
    'btn-download-cert',
    'btn-remove-cert',
    'btn-generate-cert',
    'btn-reinstall-cert'
];

function setCertActionButtonsDisabled(disabled) {
    CERT_ACTION_BUTTON_IDS.forEach((id) => {
        const button = document.getElementById(id);
        if (button) {
            button.disabled = disabled;
        }
    });
}

function beginCertOperation() {
    certOperationLockCount += 1;
    if (certOperationLockCount === 1) {
        setCertActionButtonsDisabled(true);
    }
}

function endCertOperation() {
    certOperationLockCount = Math.max(0, certOperationLockCount - 1);
    if (certOperationLockCount === 0) {
        setCertActionButtonsDisabled(false);
    }
}

async function buildAllCertOverview() {
    try {
        const fetchPromises = Object.values(CERT_TYPES).map(async (certType) => {
            try {
                const info = await getCertInfo(certType, true);
                certOverviewState[certType] = info;
            } catch (error) {
                certOverviewState[certType] = null;
            }
        });
        await Promise.all(fetchPromises);
    } catch (error) {
    }
}

function getSelectedCertType() {
    const selectedValue = document.getElementById('cert-type-select')?.value;
    return Object.values(CERT_TYPES).includes(selectedValue) ? selectedValue : DEFAULT_CERT_TYPE;
}

function buildCertQueryPath(contract, certType) {
    return `${contract.path}?${CERT_TYPE_PARAM}=${encodeURIComponent(certType)}`;
}

function buildCertBody(certType) {
    return { [CERT_TYPE_PARAM]: certType };
}

function normalizeCertResponseData(responseData) {
    if (!responseData || typeof responseData !== 'object') {
        return {};
    }

    if (responseData.data && typeof responseData.data === 'object') {
        return responseData.data;
    }

    if (responseData.result && typeof responseData.result === 'object') {
        return responseData.result;
    }

    return responseData;
}

function mapToCertUIModel(responseData) {
    const data = normalizeCertResponseData(responseData);
    return {
        cert_type: data.cert_type || '',
        is_exported: Boolean(data.is_exported),
        is_installed: Boolean(data.is_installed),
        is_trusted: Boolean(data.is_trusted),
        trust_status: data.trust_status || 'not_found',
        subject: data.subject || '',
        issuer: data.issuer || '',
        not_before: data.not_before || '',
        not_after: data.not_after || '',
        fingerprint: data.fingerprint || '',
        install_path: data.install_path || '',
        export_path: data.export_path || data.path || '',
        cert_path: data.cert_path || '',
        key_path: data.key_path || '',
        cn: data.cn || '',
        valid_years: Number.isFinite(Number(data.valid_years)) ? Number(data.valid_years) : null
    };
}

function hideGenerateResult() {
    const resultContainer = document.getElementById('cert-generate-result');
    if (!resultContainer) {
        return;
    }
    resultContainer.classList.add('hidden');
    resultContainer.innerHTML = '';
}

function showGenerateResult(result) {
    const resultContainer = document.getElementById('cert-generate-result');
    if (!resultContainer) {
        return;
    }

    const validYearsText = result.valid_years === null ? '-' : `${result.valid_years} 年`;
    resultContainer.innerHTML = `
        <div class="text-sm font-semibold text-amber-700 dark:text-amber-300 mb-2">重新生成结果</div>
        <div class="text-xs text-slate-700 dark:text-slate-300 space-y-1">
            <div><strong>CN:</strong> ${result.cn || '-'}</div>
            <div><strong>Issuer:</strong> ${result.issuer || '-'}</div>
            <div><strong>Valid Years:</strong> ${validYearsText}</div>
            <div><strong>Cert Path:</strong> <code style="word-break: break-all;">${result.cert_path || '-'}</code></div>
            <div><strong>Key Path:</strong> <code style="word-break: break-all;">${result.key_path || '-'}</code></div>
        </div>
    `;
    resultContainer.classList.remove('hidden');
}

function hideReinstallResult() {
    const container = document.getElementById('cert-reinstall-result');
    if (!container) {
        return;
    }
    container.classList.add('hidden');
    container.innerHTML = '';
}

function showReinstallResult(stages, success, finalMessage) {
    const container = document.getElementById('cert-reinstall-result');
    if (!container) {
        return;
    }

    const stageHtml = stages
        .map((stage) => {
            const icon = stage.ok ? '✓' : '✗';
            return `<div><strong>${icon} ${stage.name}:</strong> ${stage.message}</div>`;
        })
        .join('');

    container.innerHTML = `
        <div class="text-xs font-semibold mb-2 ${success ? 'text-emerald-700 dark:text-emerald-300' : 'text-red-600 dark:text-red-400'}">重新安装结果</div>
        <div class="space-y-1">${stageHtml || '<div>-</div>'}</div>
        <div class="mt-2"><strong>总结:</strong> ${finalMessage || (success ? '流程完成' : '流程失败')}</div>
    `;
    container.classList.remove('hidden');
}

function hideAuxResult() {
    const container = document.getElementById('cert-aux-result');
    if (!container) {
        return;
    }
    container.classList.add('hidden');
    container.classList.remove('border-red-200', 'dark:border-red-700/40', 'bg-red-50/60', 'dark:bg-red-900/10', 'text-red-600', 'dark:text-red-300');
    container.classList.add('border-emerald-200', 'dark:border-emerald-700/40', 'bg-emerald-50/60', 'dark:bg-emerald-900/10', 'text-emerald-700', 'dark:text-emerald-300');
    container.innerHTML = '';
}

function showAuxResult(message, isError = false) {
    const container = document.getElementById('cert-aux-result');
    if (!container) {
        return;
    }
    container.classList.remove('hidden');
    if (isError) {
        container.classList.remove('border-emerald-200', 'dark:border-emerald-700/40', 'bg-emerald-50/60', 'dark:bg-emerald-900/10', 'text-emerald-700', 'dark:text-emerald-300');
        container.classList.add('border-red-200', 'dark:border-red-700/40', 'bg-red-50/60', 'dark:bg-red-900/10', 'text-red-600', 'dark:text-red-300');
    } else {
        container.classList.remove('border-red-200', 'dark:border-red-700/40', 'bg-red-50/60', 'dark:bg-red-900/10', 'text-red-600', 'dark:text-red-300');
        container.classList.add('border-emerald-200', 'dark:border-emerald-700/40', 'bg-emerald-50/60', 'dark:bg-emerald-900/10', 'text-emerald-700', 'dark:text-emerald-300');
    }
    container.innerHTML = message || '-';
}

function safeReadCertAudit() {
    try {
        const raw = localStorage.getItem(CERT_AUDIT_STORAGE_KEY);
        if (!raw) {
            return null;
        }
        const parsed = JSON.parse(raw);
        return parsed && typeof parsed === 'object' ? parsed : null;
    } catch (error) {
        return null;
    }
}

function renderCertOperationAudit(entry) {
    const auditContainer = document.getElementById('cert-operation-audit');
    if (!auditContainer) {
        return;
    }

    const record = entry || safeReadCertAudit();
    if (!record) {
        auditContainer.textContent = '暂无记录';
        return;
    }

    const statusText = record.ok ? '成功' : '失败';
    const modeText = record.reinstallMode ? ` (${record.reinstallMode})` : '';
    auditContainer.innerHTML = `
        <div><strong>操作:</strong> ${record.operation || '-'}${modeText}</div>
        <div><strong>证书类型:</strong> ${record.certType || '-'}</div>
        <div><strong>结果:</strong> ${statusText}</div>
        <div><strong>时间:</strong> ${record.time || '-'}</div>
        <div><strong>信息:</strong> ${record.message || '-'}</div>
    `;
}

function recordCertOperation(operation, certType, ok, message, extra = {}) {
    const entry = {
        operation,
        certType,
        ok,
        message,
        time: new Date().toLocaleString(),
        ...extra
    };

    try {
        localStorage.setItem(CERT_AUDIT_STORAGE_KEY, JSON.stringify(entry));
    } catch (error) {
    }
    renderCertOperationAudit(entry);
}

function pickCertMessage(...values) {
    for (const value of values) {
        if (typeof value === 'string' && value.trim()) {
            return value.trim();
        }
    }
    return '';
}

function getCertActionSuccessMessage(responseData, fallbackMessage) {
    const normalized = normalizeCertResponseData(responseData);
    return pickCertMessage(
        normalized.message,
        normalized.msg,
        responseData?.message,
        responseData?.msg,
        fallbackMessage
    );
}

function getCertActionErrorMessage(error, fallbackMessage) {
    const normalizedError = normalizeCertResponseData(error?.response?.data || error?.data || error);
    const nestedData = normalizeCertResponseData(normalizedError?.data);
    const details = pickCertMessage(
        normalizedError.details,
        nestedData.details,
        nestedData.error_msg,
        normalizedError.error_msg
    );
    const code = normalizedError.code || nestedData.code;
    const detail = pickCertMessage(
        normalizedError.error,
        normalizedError.message,
        normalizedError.msg,
        nestedData.message,
        nestedData.msg,
        details,
        error?.message
    );
    const suffix = detail ? `${fallbackMessage}: ${detail}` : fallbackMessage;
    return code ? `${suffix} (code: ${code})` : suffix;
}

function getFriendlyCertErrorMessage(actionKey, fallbackMessage) {
    const map = {
        status: '状态查询失败，请检查后端服务是否可用',
        info: '详情查询失败，请检查后端服务是否可用',
        install: '安装失败，请检查系统权限或证书信任存储',
        remove: '移除失败，请确认该证书已安装且当前账号有权限',
        generate: '生成失败，请检查证书配置后重试',
        export: '导出失败，请检查目标目录写权限',
        download: '下载失败，请稍后重试或先导出证书',
        reinstall: '重新安装失败，请按阶段结果排查',
        missing_type: '证书类型缺失，请重新选择后再试'
    };
    return map[actionKey] || fallbackMessage;
}

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
    buildAllCertOverview();
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
        const certType = getSelectedCertType();
        const contract = CERT_API_CONTRACT.LOAD_STATUS;
        const response = await apiGet(buildCertQueryPath(contract, certType));
        updateCertStatusDisplay(mapToCertUIModel(response));
        updateCertModalLastRefreshed();
    } catch (error) {
        showError(getCertActionErrorMessage(error, getFriendlyCertErrorMessage('status', '状态加载失败')));
    }
}

async function getCertInfo(certType, silent = false) {
    const resolvedCertType = Object.values(CERT_TYPES).includes(certType) ? certType : getSelectedCertType();
    const contract = CERT_API_CONTRACT.INFO_CERT;

    try {
        const responseData = await apiGet(buildCertQueryPath(contract, resolvedCertType));
        return mapToCertUIModel(responseData);
    } catch (error) {
        if (!silent) {
            showError(getCertActionErrorMessage(error, getFriendlyCertErrorMessage('info', '获取详情失败')));
        }
        throw error;
    }
}

async function checkCertInstallation() {
    const certType = getSelectedCertType();
    const contract = CERT_API_CONTRACT.CHECK_STATUS;
    const btn = document.getElementById('btn-check-cert');
    beginCertOperation();
    showLoading(btn);

    try {
        const responseData = await apiGet(buildCertQueryPath(contract, certType));
        const result = mapToCertUIModel(responseData);
        updateCertStatusDisplay(result);
        showSuccess(getCertActionSuccessMessage(responseData, `证书状态: ${result.is_installed ? '✓ 已安装' : '✗ 未安装'}`));
        recordCertOperation('check-status', certType, true, '状态检查完成');
    } catch (error) {
        const message = getCertActionErrorMessage(error, getFriendlyCertErrorMessage('status', '检查失败'));
        showError(message);
        recordCertOperation('check-status', certType, false, message);
    } finally {
        hideLoading(btn);
        endCertOperation();
    }
}

async function generateCert(certType) {
    const resolvedCertType = Object.values(CERT_TYPES).includes(certType) ? certType : getSelectedCertType();
    const contract = CERT_API_CONTRACT.GENERATE_CERT;

    try {
        const responseData = await apiPost(contract.path, buildCertBody(resolvedCertType));
        const result = mapToCertUIModel(responseData);
        showSuccess(getCertActionSuccessMessage(responseData, '证书生成成功！'));
        showGenerateResult(result);
        await loadCertStatus();
        recordCertOperation('generate', resolvedCertType, true, '证书重新生成完成');
        return result;
    } catch (error) {
        const message = getCertActionErrorMessage(error, getFriendlyCertErrorMessage('generate', '生成失败'));
        showError(message);
        recordCertOperation('generate', resolvedCertType, false, message);
        throw error;
    }
}

async function regenerateCertAction() {
    if (!confirm('重新生成将覆盖当前证书文件，确定继续吗？')) {
        return;
    }

    const certType = getSelectedCertType();
    const btn = document.getElementById('btn-generate-cert');
    beginCertOperation();
    showLoading(btn);
    try {
        await generateCert(certType);
    } finally {
        hideLoading(btn);
        endCertOperation();
    }
}

async function reinstallCertAction() {
    const certType = getSelectedCertType();
    if (!certType) {
        const message = getFriendlyCertErrorMessage('missing_type', '请选择证书类型');
        showError(message);
        recordCertOperation('reinstall', certType, false, message);
        return;
    }

    const regenerateFirst = Boolean(document.getElementById('reinstall-regenerate-first')?.checked);
    const modeLabel = regenerateFirst ? 'generate -> install' : 'install-only';
    const confirmText = regenerateFirst
        ? '将先重新生成证书，再执行安装。确定继续吗？'
        : '将执行证书重新安装（幂等）。确定继续吗？';
    if (!confirm(confirmText)) {
        return;
    }

    const btn = document.getElementById('btn-reinstall-cert');
    const stages = [];
    beginCertOperation();
    showLoading(btn);
    hideReinstallResult();

    try {
        if (regenerateFirst) {
            try {
                await generateCert(certType);
                stages.push({ name: '生成', ok: true, message: '证书生成成功' });
            } catch (error) {
                const generateMessage = getCertActionErrorMessage(error, getFriendlyCertErrorMessage('generate', '生成失败'));
                stages.push({ name: '生成', ok: false, message: generateMessage });
                throw new Error(generateMessage);
            }
        }

        const contract = CERT_API_CONTRACT.INSTALL_CERT;
        try {
            const installResponse = await apiPost(contract.path, buildCertBody(certType));
            stages.push({ name: '安装', ok: true, message: getCertActionSuccessMessage(installResponse, '证书安装成功') });
        } catch (error) {
            const installMessage = getCertActionErrorMessage(error, getFriendlyCertErrorMessage('install', '安装失败'));
            stages.push({ name: '安装', ok: false, message: installMessage });
            throw new Error(installMessage);
        }

        await loadCertStatus();
        showReinstallResult(stages, true, '重新安装流程已完成');
        showSuccess('重新安装成功');
        recordCertOperation('reinstall', certType, true, '重新安装成功', { reinstallMode: modeLabel });
    } catch (error) {
        const failureMessage = error?.message || getFriendlyCertErrorMessage('reinstall', '重新安装失败');
        showReinstallResult(stages, false, failureMessage);
        showError(failureMessage);
        recordCertOperation('reinstall', certType, false, failureMessage, { reinstallMode: modeLabel });
    } finally {
        hideLoading(btn);
        endCertOperation();
    }
}

async function exportCert() {
    const certType = getSelectedCertType();
    const contract = CERT_API_CONTRACT.EXPORT_CERT;
    const btn = document.getElementById('btn-export-cert');
    beginCertOperation();
    showLoading(btn);

    try {
        const responseData = await apiPost(contract.path, buildCertBody(certType));
        const result = mapToCertUIModel(responseData);
        const exportPath = result.export_path;
        showSuccess(getCertActionSuccessMessage(responseData, exportPath ? `证书已导出到: ${exportPath}` : '证书导出成功！'));
        showAuxResult(exportPath ? `导出路径: ${exportPath}` : '导出成功，但未返回导出路径');
        loadCertStatus();
        recordCertOperation('export', certType, true, exportPath ? `导出路径: ${exportPath}` : '导出成功');
    } catch (error) {
        const message = getCertActionErrorMessage(error, getFriendlyCertErrorMessage('export', '导出失败'));
        showError(message);
        showAuxResult(message, true);
        recordCertOperation('export', certType, false, message);
    } finally {
        hideLoading(btn);
        endCertOperation();
    }
}

async function downloadCert() {
    const certType = getSelectedCertType();
    const contract = CERT_API_CONTRACT.DOWNLOAD_CERT;
    try {
        await downloadFile(buildCertQueryPath(contract, certType), `${certType}.pem`);
        showAuxResult(`下载已触发: ${certType}.pem`);
        recordCertOperation('download', certType, true, `下载已触发: ${certType}.pem`);
    } catch (error) {
        const message = getCertActionErrorMessage(error, getFriendlyCertErrorMessage('download', '下载失败'));
        showAuxResult(`${message}。是否重试请再次点击“下载 PEM”。`, true);
        recordCertOperation('download', certType, false, message);
    }
}

async function installCert() {
    const certType = getSelectedCertType();
    const contract = CERT_API_CONTRACT.INSTALL_CERT;

    if (!confirm('此操作需要管理员权限。继续吗？')) {
        return;
    }

    const btn = document.getElementById('btn-install-cert');
    beginCertOperation();
    showLoading(btn);

    try {
        const responseData = await apiPost(contract.path, buildCertBody(certType));
        showSuccess(getCertActionSuccessMessage(responseData, '证书安装成功！'));
        await loadCertStatus();
        recordCertOperation('install', certType, true, '证书安装成功');
    } catch (error) {
        const message = getCertActionErrorMessage(error, getFriendlyCertErrorMessage('install', '安装失败'));
        showError(message);
        recordCertOperation('install', certType, false, message);
    } finally {
        hideLoading(btn);
        endCertOperation();
    }
}

async function removeCert() {
    const certType = getSelectedCertType();
    const contract = CERT_API_CONTRACT.REMOVE_CERT;

    if (!confirm('确定要移除证书吗？')) {
        return;
    }

    const btn = document.getElementById('btn-remove-cert');
    beginCertOperation();
    showLoading(btn);

    try {
        const responseData = await apiPost(contract.path, buildCertBody(certType));
        showSuccess(getCertActionSuccessMessage(responseData, '证书已移除！'));
        await loadCertStatus();
        recordCertOperation('remove', certType, true, '证书移除成功');
    } catch (error) {
        const message = getCertActionErrorMessage(error, getFriendlyCertErrorMessage('remove', '移除失败'));
        showError(message);
        recordCertOperation('remove', certType, false, message);
    } finally {
        hideLoading(btn);
        endCertOperation();
    }
}

function updateCertStatusDisplay(certStatus) {
    const container = document.getElementById('cert-status-content') || document.getElementById('cert-status-container');
    if (!certStatus || !certStatus.cert_type) {
        container.innerHTML = '<div class="alert alert-warning">暂无证书信息</div>';
        return;
    }

    const exportedBadge = certStatus.is_exported
    ? '<span class="badge bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300"><i class="bi bi-check-circle-fill"></i> 已导出</span>'
    : '<span class="badge bg-slate-100 text-slate-500 dark:bg-slate-500/20 dark:text-slate-400"><i class="bi bi-x-circle"></i> 未导出</span>';
    const installedBadge = certStatus.is_installed
    ? '<span class="badge bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300"><i class="bi bi-check-circle-fill"></i> 已安装</span>'
    : '<span class="badge bg-slate-100 text-slate-500 dark:bg-slate-500/20 dark:text-slate-400"><i class="bi bi-x-circle"></i> 未安装</span>';
    const trustedBadge = certStatus.is_trusted
    ? '<span class="badge bg-blue-100 text-blue-700 dark:bg-blue-500/20 dark:text-blue-300"><i class="bi bi-shield-fill-check"></i> 已信任</span>'
    : '<span class="badge bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-300"><i class="bi bi-exclamation-triangle-fill"></i> 未信任</span>';

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
                        <div><strong>指纹:</strong> <code style="word-break: break-all;">${certStatus.fingerprint || '-'}</code></div>
                        <div><strong>安装路径:</strong> <code style="word-break: break-all;">${certStatus.install_path || '-'}</code></div>
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
        [CERT_TYPES.MITM_CA]: 'MITM CA (客户端拦截)',
        [CERT_TYPES.ROOT_CA]: 'Root CA (根证书)',
        [CERT_TYPES.MTLS_CERT]: 'mTLS Certificate (后端通信)'
    };
    return names[type] || type;
}

document.addEventListener('DOMContentLoaded', () => {
    buildAllCertOverview();

    document.addEventListener('change', (e) => {
        if (e.target && e.target.id === 'cert-type-select') {
            hideGenerateResult();
            hideReinstallResult();
            hideAuxResult();
            loadCertStatus();
        }
    });

    renderCertOperationAudit();
});
