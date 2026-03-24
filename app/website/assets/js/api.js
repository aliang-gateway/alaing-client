const API_BASE = '/api';

function firstNonEmptyString(values) {
    for (const value of values) {
        if (typeof value === 'string' && value.trim()) {
            return value;
        }
    }
    return '';
}

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
            const detail = firstNonEmptyString([
                data?.data?.details?.error,
                data?.data?.error_msg,
                data?.data?.error,
                data?.msg,
                data?.message,
                data?.error
            ]);
            throw new Error(detail || '请求失败');
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

async function downloadFile(endpoint, filename) {
    try {
        const response = await fetch(`${API_BASE}${endpoint}`, {
            method: 'GET'
        });

        if (!response.ok) {
            let msg = '下载失败';
            const contentType = response.headers.get('content-type') || '';
            if (contentType.includes('application/json')) {
                const errorData = await response.json();
                msg = errorData.msg || errorData.message || msg;
            } else {
                const errorText = await response.text();
                if (errorText && errorText.trim()) {
                    msg = errorText.trim();
                }
            }
            throw new Error(msg);
        }

        const blob = await response.blob();
        const blobUrl = window.URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = blobUrl;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        link.remove();
        window.URL.revokeObjectURL(blobUrl);
        showSuccess('证书下载成功');
    } catch (error) {
        showError('下载失败: ' + error.message);
    }
}

async function softwareConfigList(software = '') {
    const query = software ? `?software=${encodeURIComponent(software)}` : '';
    return apiGet(`/software-config/list${query}`);
}

async function softwareConfigSave(payload) {
    return apiPost('/software-config/save', payload);
}

async function softwareConfigDelete(uuid) {
    return apiPost('/software-config/delete', { uuid });
}

async function softwareConfigSelect(uuid, selected) {
    return apiPost('/software-config/select', { uuid, selected });
}

async function softwareConfigCompare(cloud_url, auth_token = '') {
    return apiPost('/software-config/compare', { cloud_url, auth_token });
}

async function softwareConfigPushSelected(cloud_url, auth_token = '') {
    return apiPost('/software-config/cloud/push-selected', { cloud_url, auth_token });
}

async function softwareConfigLog(action, software, config_uuid = '', config_name = '', detail = '') {
    return apiPost('/software-config/log', {
        action,
        software,
        config_uuid,
        config_name,
        detail
    });
}

async function customerConfigGet() {
    return apiGet('/config/customer');
}

async function customerConfigSave(payload) {
    return apiPost('/config/customer', payload);
}

async function customerConfigGetProviders() {
    return apiGet('/config/customer/providers');
}

window.customerConfigGet = customerConfigGet;
window.customerConfigSave = customerConfigSave;
window.customerConfigGetProviders = customerConfigGetProviders;
