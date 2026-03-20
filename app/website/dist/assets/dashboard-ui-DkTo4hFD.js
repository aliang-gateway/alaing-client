function classifyDomainCategory(domain) {
    const normalized = String(domain || '').toLowerCase();
    for (const [category, patterns] of Object.entries(domainCategoryMap)) {
        if (patterns.some(pattern => normalized.includes(pattern))) {
            return category;
        }
    }
    return null;
}

function updateDomainPieChartByLogs() {
    if (!charts.domainPie) {
        return;
    }

    const nameMap = {
        cursor: 'Cursor',
        openai: 'OpenAI',
        claude: 'Claude',
        chatgpt: 'ChatGPT',
        copilot: 'Copilot'
    };

    const bucket = {
        cursor: 0,
        openai: 0,
        claude: 0,
        chatgpt: 0,
        copilot: 0
    };

    dashboardRequestLog.entries.forEach(item => {
        const category = classifyDomainCategory(item.domain);
        if (!category) {
            return;
        }

        if (activeDomainFilter !== 'all' && activeDomainFilter !== category) {
            return;
        }

        bucket[category] += Number(item.bytes || 0);
    });

    const labels = [];
    const values = [];

    Object.entries(bucket).forEach(([key, value]) => {
        if (value > 0) {
            labels.push(nameMap[key]);
            values.push(value);
        }
    });

    charts.domainPie.data.labels = labels.length > 0 ? labels : ['暂无数据'];
    charts.domainPie.data.datasets[0].data = values.length > 0 ? values : [1];
    charts.domainPie.update('none');
}

function deriveLogDomain(entry) {
    const candidates = [entry?.domain, entry?.host, entry?.server_name, entry?.target];
    const firstCandidate = candidates.find(v => typeof v === 'string' && v.trim() !== '');
    if (firstCandidate) {
        return firstCandidate;
    }

    const message = String(entry?.message || '');
    const match = message.match(/([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}/);
    return match ? match[0] : 'unknown';
}

function deriveLogPath(entry) {
    const candidates = [entry?.path, entry?.uri, entry?.url];
    const firstCandidate = candidates.find(v => typeof v === 'string' && v.trim() !== '');
    return firstCandidate || '/';
}

function deriveLogMethod(entry) {
    const method = typeof entry?.method === 'string' ? entry.method.toUpperCase() : '';
    if (method) {
        return method;
    }

    const message = String(entry?.message || '');
    const match = message.match(/\b(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\b/i);
    return match ? match[1].toUpperCase() : 'LOG';
}

function appendDashboardRequestLog(entry) {
    const listEl = document.getElementById('requestLogList');
    const countEl = document.getElementById('logCount');
    if (!listEl) {
        return;
    }

    const timestamp = entry?.timestamp || new Date().toLocaleTimeString();
    const level = (entry?.level || 'INFO').toUpperCase();
    const domain = deriveLogDomain(entry);
    const method = deriveLogMethod(entry);
    const path = deriveLogPath(entry);
    const source = entry?.source || '-';
    const message = entry?.message || '';
    const bytes = Number(entry?.bytes || entry?.size || 0);

    dashboardRequestLog.entries.unshift({ timestamp, level, domain, method, path, source, message, bytes });
    if (dashboardRequestLog.entries.length > dashboardRequestLog.maxItems) {
        dashboardRequestLog.entries = dashboardRequestLog.entries.slice(0, dashboardRequestLog.maxItems);
    }

    const fragment = document.createDocumentFragment();
    dashboardRequestLog.entries.forEach((item, index) => {
        const detailsId = `reqLogDetail-${index}`;
        const row = document.createElement('button');
        row.type = 'button';
        row.className = 'log-item';
        row.setAttribute('aria-expanded', 'false');
        row.innerHTML = `
            <span class="log-time">${item.timestamp}</span>
            <span class="log-domain">${item.domain}</span>
            <span class="log-status">${item.method}</span>
            <div class="log-detail" id="${detailsId}" hidden>
                <div><strong>路径:</strong> <code>${item.path}</code></div>
                <div><strong>级别:</strong> ${item.level}</div>
                <div><strong>来源:</strong> ${item.source}</div>
                <div><strong>流量:</strong> ${formatBytes(item.bytes || 0)}</div>
                <div><strong>消息:</strong> ${item.message}</div>
            </div>
        `;

        row.addEventListener('click', () => {
            const detail = row.querySelector('.log-detail');
            const expanded = row.getAttribute('aria-expanded') === 'true';
            row.setAttribute('aria-expanded', expanded ? 'false' : 'true');
            if (detail) {
                detail.hidden = expanded;
            }
        });

        fragment.appendChild(row);
    });

    listEl.innerHTML = '';
    listEl.appendChild(fragment);
    listEl.scrollTop = 0;

    if (countEl) {
        countEl.textContent = `${dashboardRequestLog.entries.length} 条记录`;
    }

    updateDomainPieChartByLogs();
}

function loadChatHistory() {
    try {
        const saved = window.localStorage.getItem(chatStore.storageKey);
        const parsed = saved ? JSON.parse(saved) : [];
        chatStore.entries = Array.isArray(parsed) ? parsed.slice(0, chatStore.maxItems) : [];
    } catch (error) {
        chatStore.entries = [];
    }
}

function persistChatHistory() {
    try {
        window.localStorage.setItem(chatStore.storageKey, JSON.stringify(chatStore.entries.slice(0, chatStore.maxItems)));
    } catch (error) {
        console.warn('Persist chat history failed:', error);
    }
}

function renderChatMessages(filterKeyword = '') {
    const container = document.getElementById('chatMessages');
    if (!container) {
        return;
    }

    const keyword = String(filterKeyword || '').trim().toLowerCase();
    const list = keyword
        ? chatStore.entries.filter(item => (item.text || '').toLowerCase().includes(keyword))
        : chatStore.entries;

    if (list.length === 0) {
        container.innerHTML = `
            <div class="chat-welcome">
                <i class="bi bi-robot"></i>
                <p>${keyword ? '没有匹配的历史记录' : '所有对话数据都存储在本地'}</p>
            </div>
        `;
        return;
    }

    container.innerHTML = list.map(item => `
        <div class="chat-message ${item.role === 'user' ? 'user' : 'assistant'}">
            <div class="chat-message-meta">${item.role === 'user' ? '我' : '助手'} · ${item.time}</div>
            <div class="chat-message-text">${item.text}</div>
        </div>
    `).join('');

    container.scrollTop = container.scrollHeight;
}

function addChatMessage(role, text) {
    const safeText = String(text || '').trim();
    if (!safeText) {
        return;
    }
    const item = {
        role,
        text: safeText,
        time: new Date().toLocaleString()
    };
    chatStore.entries.push(item);
    if (chatStore.entries.length > chatStore.maxItems) {
        chatStore.entries = chatStore.entries.slice(chatStore.entries.length - chatStore.maxItems);
    }
    persistChatHistory();
    renderChatMessages();
}

function initDomainFilterControls() {
    const filterButtons = document.querySelectorAll('[data-domain-filter]');
    if (filterButtons.length === 0) {
        return;
    }

    filterButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            const value = btn.getAttribute('data-domain-filter') || 'all';
            activeDomainFilter = value;
            filterButtons.forEach(node => {
                if (node === btn) {
                    node.classList.add('active');
                } else {
                    node.classList.remove('active');
                }
            });
            updateDomainPieChartByLogs();
        });
    });
}

function initQuickChat() {
    const quickChatBtn = document.getElementById('quickChatBtn');
    const chatModalEl = document.getElementById('chatModal');
    const chatInput = document.getElementById('chatInput');
    const chatSendBtn = document.getElementById('chatSendBtn');
    const chatSearchInput = document.getElementById('chatSearchInput');
    const chatClearHistoryBtn = document.getElementById('chatClearHistoryBtn');

    if (!chatModalEl || !chatInput || !chatSendBtn) {
        return;
    }

    loadChatHistory();
    renderChatMessages();

    let modalInstance = null;
    if (window.bootstrap && window.bootstrap.Modal) {
        modalInstance = window.bootstrap.Modal.getOrCreateInstance(chatModalEl);
    }

    if (quickChatBtn) {
        quickChatBtn.addEventListener('click', () => {
            if (modalInstance) {
                modalInstance.show();
            }
            renderChatMessages(chatSearchInput ? chatSearchInput.value : '');
        });
    }

    const send = () => {
        const text = chatInput.value.trim();
        if (!text) {
            return;
        }
        addChatMessage('user', text);
        addChatMessage('assistant', `已保存到本地：${text}`);
        chatInput.value = '';
    };

    chatSendBtn.addEventListener('click', send);
    chatInput.addEventListener('keydown', event => {
        if (event.key === 'Enter') {
            event.preventDefault();
            send();
        }
    });

    if (chatSearchInput) {
        chatSearchInput.addEventListener('input', () => {
            renderChatMessages(chatSearchInput.value);
        });
    }

    if (chatClearHistoryBtn) {
        chatClearHistoryBtn.addEventListener('click', () => {
            chatStore.entries = [];
            persistChatHistory();
            renderChatMessages(chatSearchInput ? chatSearchInput.value : '');
        });
    }
}
