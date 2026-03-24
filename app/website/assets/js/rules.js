async function loadRulesData() {
    try {
        const statusData = await apiGet('/rules/engine/status');

        const isEnabled = statusData.engineEnabled || false;
        appState.ruleEngineStatus.isEnabled = isEnabled;
        appState.ruleEngineStatus.lastUpdated = Date.now();

        const statusBadge = document.getElementById('rules-status-badge');
        const statusText = document.getElementById('rules-status-text');

        if (statusBadge && statusText) {
            if (isEnabled) {
                statusBadge.className = 'badge bg-danger';
                statusText.textContent = '启用';
            } else {
                statusBadge.className = 'badge bg-danger';
                statusText.textContent = '禁用';
            }
        }

        const cacheData = statusData.cache || null;
        if (cacheData) {
            updateRulesStatistics(cacheData);
        } else {
            try {
                const cacheStats = await apiGet('/rules/cache/stats');
                if (cacheStats) {
                    updateRulesStatistics(cacheStats);
                }
            } catch (error) {
                console.warn('获取缓存统计失败:', error);
            }
        }

        updateRulesButtonStates();

    } catch (error) {
        console.error('加载规则数据失败:', error);
        showError('加载规则引擎数据失败: ' + error.message);

        const statusElement = document.getElementById('rulesStatus');
        const statusBadge = document.getElementById('rules-status-badge');
        const statusText = document.getElementById('rules-status-text');

        if (statusElement) {
            statusElement.textContent = '未知';
        }

        if (statusBadge && statusText) {
            statusBadge.className = 'badge bg-secondary';
            statusText.textContent = '未知';
        }
    }
}

function updateRulesStatistics(cacheData) {
    const cacheHitRateElement = document.getElementById('cache-hit-rate');
    const totalCacheElement = document.getElementById('total-cache');
    const cacheSizeElement = document.getElementById('rulesCacheSize');
    const cacheCountElement = document.getElementById('rulesCacheCount');
    const lastUpdateElement = document.getElementById('cache-last-update');

    if (cacheData) {
        const cacheHits = cacheData.cache_hits || 0;
        const cacheMisses = cacheData.cache_misses || 0;
        const totalHits = cacheHits + cacheMisses;
        const hitRate = totalHits > 0 ? (cacheHits / totalHits * 100).toFixed(2) : 0;

        if (cacheHitRateElement) {
            cacheHitRateElement.textContent = hitRate + '%';
        }

        if (totalCacheElement) {
            totalCacheElement.textContent = totalHits.toLocaleString();
        }

        if (cacheSizeElement) {
            cacheSizeElement.textContent = cacheData.size || 0;
        }

        if (cacheCountElement) {
            cacheCountElement.textContent = cacheData.count || 0;
        }

        if (lastUpdateElement) {
            lastUpdateElement.textContent = new Date().toLocaleTimeString();
        }
    }
}

async function loadRoutingConfig() {
    try {
        const config = await apiGet('/config/routing');
        if (config) {
            populateRulesUI(toCanonicalRoutingConfig(config));
        }
    } catch (error) {
        console.error('加载路由配置失败:', error);
        populateRulesUI(buildDefaultCanonicalRoutingConfig());
    }
}

function buildDefaultCanonicalRoutingConfig() {
    return {
        version: 1,
        ingress: { mode: 'tun' },
        egress: {
            direct: { enabled: true },
            toAliang: { enabled: false },
            toSocks: { enabled: false, upstream: { type: 'socks' } }
        },
        routing: { rules: [] }
    };
}

function toCanonicalRoutingConfig(config) {
    if (config?.ingress && config?.egress && Array.isArray(config?.routing?.rules)) {
        return config;
    }
    return buildDefaultCanonicalRoutingConfig();
}

function rulesByTarget(config, target) {
    return (config?.routing?.rules || []).filter(rule => rule.target === target);
}

function populateRulesUI(config) {
    const geoipSwitch = document.getElementById('geoipEnabledSwitch');

    if (geoipSwitch) {
        geoipSwitch.checked = false;
    }

    populateRuleTable('rulesTableBody', rulesByTarget(config, 'toSocks'), 'toSocks');
    populateRuleTable('rulesTableBody', rulesByTarget(config, 'toAliang'), 'toAliang');

    appState.routingConfig = config;
}

function populateRuleTable(tableBodyId, rules, ruleSet) {
    const tbody = document.getElementById(tableBodyId);
    if (!tbody) return;

    if (!rules || rules.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="text-center text-muted py-3">暂无规则</td></tr>';
        return;
    }

    tbody.innerHTML = rules.map((rule) => {
        const typeText = getTypeText(rule.type);
        const enabledChecked = rule.enabled ? 'checked' : '';

        return `
            <tr data-rule-id="${rule.id}">
                <td><span class="badge bg-info">${typeText}</span></td>
                <td><code>${escapeHtml(rule.condition)}</code></td>
                <td class="text-center">
                    <div class="form-check form-switch d-flex justify-content-center">
                        <input class="form-check-input rule-toggle" type="checkbox" ${enabledChecked}
                               data-rule-id="${rule.id}" data-rule-set="${ruleSet}">
                    </div>
                </td>
                <td class="text-center">
                    <button class="btn btn-sm btn-outline-primary edit-rule-btn"
                            data-rule-id="${rule.id}" data-rule-set="${ruleSet}">
                        <i class="bi bi-pencil"></i>
                    </button>
                    <button class="btn btn-sm btn-outline-danger delete-rule-btn ms-1"
                            data-rule-id="${rule.id}" data-rule-set="${ruleSet}">
                        <i class="bi bi-trash"></i>
                    </button>
                </td>
            </tr>
        `;
    }).join('');

    bindRuleTableEvents(tbody);
}

function bindRuleTableEvents(tbody) {
    tbody.querySelectorAll('.rule-toggle').forEach(toggle => {
        toggle.addEventListener('change', async (e) => {
            const ruleId = e.target.dataset.ruleId;
            const enabled = e.target.checked;
            try {
                await apiCall(`/config/routing/rules/${ruleId}/toggle`, {
                    method: 'PUT',
                    body: JSON.stringify({ enabled })
                });
                showSuccess('规则状态已更新');
            } catch (error) {
                showError('更新规则状态失败: ' + error.message);
                e.target.checked = !enabled;
            }
        });
    });

    tbody.querySelectorAll('.edit-rule-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const ruleId = e.currentTarget.dataset.ruleId;
            const ruleSet = e.currentTarget.dataset.ruleSet;
            editRule(ruleId, ruleSet);
        });
    });

    tbody.querySelectorAll('.delete-rule-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const ruleId = e.currentTarget.dataset.ruleId;
            const ruleSet = e.currentTarget.dataset.ruleSet;
            deleteRule(ruleId, ruleSet);
        });
    });
}

function getTypeText(type) {
    const typeMap = {
        'domain': '域名',
        'ip': 'IP段',
        'geoip': 'GeoIP'
    };
    return typeMap[type] || type;
}

function generateRuleId(type) {
    const timestamp = Math.floor(Date.now() / 1000);
    return `rule_${type}_${timestamp}`;
}

function addRule(ruleSet) {
    const modal = new bootstrap.Modal(document.getElementById('ruleEditModal'));

    document.getElementById('ruleTypeSelect').value = 'domain';
    document.getElementById('ruleConditionInput').value = '';
    document.getElementById('ruleEnabledCheckbox').checked = true;
    document.getElementById('ruleIdInput').value = '';
    document.getElementById('ruleSetInput').value = ruleSet;

    document.getElementById('ruleEditModalTitle').textContent = '添加规则';

    modal.show();
}

function editRule(ruleId, ruleSet) {
    const config = appState.routingConfig;
    if (!config) {
        showError('配置未加载');
        return;
    }

    let rule = null;
    const rules = rulesByTarget(config, ruleSet);
    if (rules) {
        rule = rules.find(r => r.id === ruleId);
    }

    if (!rule) {
        showError('规则未找到');
        return;
    }

    document.getElementById('ruleTypeSelect').value = rule.type;
    document.getElementById('ruleConditionInput').value = rule.condition;
    document.getElementById('ruleEnabledCheckbox').checked = rule.enabled;
    document.getElementById('ruleIdInput').value = rule.id;
    document.getElementById('ruleSetInput').value = ruleSet;

    document.getElementById('ruleEditModalTitle').textContent = '编辑规则';

    const modal = new bootstrap.Modal(document.getElementById('ruleEditModal'));
    modal.show();
}

async function deleteRule(ruleId, ruleSet) {
    if (!confirm('确定要删除此规则吗？')) {
        return;
    }

    try {
        const config = appState.routingConfig;
        if (!config) {
            showError('配置未加载');
            return;
        }

        config.routing.rules = (config.routing.rules || []).filter(rule => {
            if (rule.id !== ruleId) {
                return true;
            }
            return rule.target !== ruleSet;
        });

        await apiPost('/config/routing', config);
        showSuccess('规则已删除');

        loadRoutingConfig();
    } catch (error) {
        showError('删除规则失败: ' + error.message);
    }
}

function validateRule(type, condition) {
    if (!condition || condition.trim() === '') {
        return '条件不能为空';
    }

    const trimmed = condition.trim();

    switch (type) {
        case 'domain':
            if (!/^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/.test(trimmed)) {
                return '域名格式无效（例: *.google.com 或 example.com）';
            }
            break;

        case 'ip': {
            const cidrRegex = /^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/;
            if (!cidrRegex.test(trimmed)) {
                return 'IP段格式无效（例: 192.168.0.0/16）';
            }
            const parts = trimmed.split('/');
            const ipParts = parts[0].split('.');
            if (ipParts.some(p => parseInt(p, 10) > 255)) {
                return 'IP地址范围无效';
            }
            const prefix = parseInt(parts[1], 10);
            if (prefix < 0 || prefix > 32) {
                return '子网掩码范围无效（0-32）';
            }
            break;
        }

        case 'geoip':
            if (!/^[A-Z]{2}$/.test(trimmed)) {
                return 'GeoIP格式无效（例: US, CN）';
            }
            break;

        default:
            return '未知的规则类型';
    }

    return null;
}

async function saveRuleFromModal() {
    const ruleId = document.getElementById('ruleIdInput').value;
    const ruleSet = document.getElementById('ruleSetInput').value;
    const type = document.getElementById('ruleTypeSelect').value;
    const condition = document.getElementById('ruleConditionInput').value.trim();
    const enabled = document.getElementById('ruleEnabledCheckbox').checked;

    const validationError = validateRule(type, condition);
    if (validationError) {
        const errorDiv = document.getElementById('conditionError');
        errorDiv.textContent = validationError;
        errorDiv.style.display = 'block';
        document.getElementById('ruleConditionInput').classList.add('is-invalid');
        return;
    }

    document.getElementById('conditionError').style.display = 'none';
    document.getElementById('ruleConditionInput').classList.remove('is-invalid');

    try {
        const config = appState.routingConfig;
        if (!config) {
            showError('配置未加载');
            return;
        }

        const rule = {
            id: ruleId || generateRuleId(type),
            type: type,
            condition: condition,
            enabled: enabled,
            target: ruleSet
        };

        if (!Array.isArray(config.routing?.rules)) {
            config.routing = config.routing || {};
            config.routing.rules = [];
        }

        if (ruleId) {
            const index = config.routing.rules.findIndex(r => r.id === ruleId && r.target === ruleSet);
            if (index !== -1) {
                config.routing.rules[index] = rule;
            }
        } else {
            config.routing.rules.push(rule);
        }

        await apiPost('/config/routing', config);
        showSuccess(ruleId ? '规则已更新' : '规则已添加');

        const modal = bootstrap.Modal.getInstance(document.getElementById('ruleEditModal'));
        modal.hide();

        loadRoutingConfig();
    } catch (error) {
        showError('保存规则失败: ' + error.message);
    }
}

function updateRuleTypeHelpText() {
    const type = document.getElementById('ruleTypeSelect').value;
    const helpText = document.getElementById('typeHelpText');

    const helpMap = {
        'domain': '示例: *.google.com 或 example.com',
        'ip': '示例: 192.168.0.0/16 或 10.0.0.0/8',
        'geoip': '示例: US (美国), CN (中国), JP (日本) - 使用ISO 3166-1 alpha-2代码'
    };

    helpText.textContent = helpMap[type] || '';
}

async function saveGlobalRoutingConfig() {
    try {
        const config = appState.routingConfig;
        if (!config) {
            showError('配置未加载');
            return;
        }

        config.egress = config.egress || {};
        config.egress.direct = config.egress.direct || { enabled: true };
        config.egress.toSocks = config.egress.toSocks || { enabled: false, upstream: { type: 'socks' } };
        config.egress.toAliang = config.egress.toAliang || { enabled: false };

        await apiPost('/config/routing', config);
        showSuccess('配置已保存');

        loadRoutingConfig();
    } catch (error) {
        showError('保存配置失败: ' + error.message);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const ruleEditSaveBtn = document.getElementById('ruleEditSaveBtn');
    if (ruleEditSaveBtn) {
        ruleEditSaveBtn.addEventListener('click', saveRuleFromModal);
    }

    const ruleTypeSelect = document.getElementById('ruleTypeSelect');
    if (ruleTypeSelect) {
        ruleTypeSelect.addEventListener('change', updateRuleTypeHelpText);
        updateRuleTypeHelpText();
    }

    const rulesConfigSaveBtn = document.getElementById('rulesConfigSaveBtn');
    if (rulesConfigSaveBtn) {
        rulesConfigSaveBtn.addEventListener('click', saveGlobalRoutingConfig);
    }
});

document.getElementById('rulesEnableBtn')?.addEventListener('click', (event) => {
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

document.getElementById('rulesDisableBtn')?.addEventListener('click', (event) => {
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

document.getElementById('rulesReloadBtn')?.addEventListener('click', (event) => {
    const btn = event.target.closest('button');
    showLoading(btn);
    showError('重新加载功能暂未实现');
    hideLoading(btn);
});

document.getElementById('rulesClearCacheBtn')?.addEventListener('click', (event) => {
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

document.getElementById('rulesLookupBtn')?.addEventListener('click', (event) => {
    const btn = event.target.closest('button');
    const ip = document.getElementById('rulesLookupDomain').value.trim();

    if (!ip) {
        showError('请输入IP地址');
        return;
    }

    showLoading(btn);
    apiPost('/rules/geoip/lookup', { ip: ip })
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
