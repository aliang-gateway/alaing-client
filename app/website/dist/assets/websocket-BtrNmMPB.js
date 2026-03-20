const logWebSocket = {
    ws: null,
    isConnected: false,
    reconnectAttempts: 0,
    maxReconnectAttempts: 5,
    reconnectInterval: 3000,
    isManualDisconnect: false,

    url: function() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        return `${protocol}//${window.location.host}/api/logs/stream`;
    },

    connect: function() {
        if (this.ws && (this.ws.readyState === WebSocket.CONNECTING ||
                       this.ws.readyState === WebSocket.OPEN)) {
            return;
        }

        this.updateConnectionStatus('connecting');

        try {
            this.ws = new WebSocket(this.url());

            this.ws.onopen = () => {
                this.isConnected = true;
                this.reconnectAttempts = 0;
                this.updateConnectionStatus('connected');
                console.log('WebSocket 日志连接已建立');
                showSuccess('实时日志连接已建立');
            };

            this.ws.onmessage = (event) => {
                this.handleLogMessage(event);
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket 错误:', error);
                this.updateConnectionStatus('error');
                showError('WebSocket 连接错误');
            };

            this.ws.onclose = (event) => {
                this.isConnected = false;
                this.updateConnectionStatus('disconnected');

                if (!this.isManualDisconnect &&
                    event.code !== 1000 &&
                    this.reconnectAttempts < this.maxReconnectAttempts) {
                    console.log(`WebSocket 连接断开，${this.reconnectInterval/1000}秒后重试 (${this.reconnectAttempts + 1}/${this.maxReconnectAttempts})`);
                    setTimeout(() => {
                        this.reconnectAttempts++;
                        this.connect();
                    }, this.reconnectInterval);
                } else if (this.reconnectAttempts >= this.maxReconnectAttempts) {
                    showError('WebSocket 连接失败，请检查网络连接或手动刷新页面');
                }
            };
        } catch (error) {
            console.error('创建 WebSocket 连接失败:', error);
            this.updateConnectionStatus('error');
            showError('创建 WebSocket 连接失败');
        }
    },

    disconnect: function() {
        this.isManualDisconnect = true;
        if (this.ws) {
            this.ws.close(1000, 'Client disconnect');
            this.ws = null;
        }
        this.isConnected = false;
        this.updateConnectionStatus('disconnected');
        console.log('WebSocket 连接已断开');
    },

    handleLogMessage: function(event) {
        try {
            const logEntry = JSON.parse(event.data);
            this.appendLogToOutput(logEntry);
            appendDashboardRequestLog(logEntry);
        } catch (error) {
            console.error('解析日志消息失败:', error);
        }
    },

    appendLogToOutput: function(logEntry) {
        const output = document.getElementById('logsOutput');
        if (!output) return;

        const timestamp = logEntry.timestamp || new Date().toLocaleString();
        const level = logEntry.level || 'INFO';
        const message = logEntry.message || '';
        const source = logEntry.source || '';

        const formattedLog = `[${timestamp}] [${level}]`;
        const sourcePart = source ? ` [${source}]` : '';
        const fullLogLine = `${formattedLog}${sourcePart} ${message}\n`;

        const shouldScroll = output.scrollTop + output.clientHeight >= output.scrollHeight - 10;

        const readText = () => {
            if (output instanceof HTMLTextAreaElement || output instanceof HTMLInputElement) {
                return output.value || '';
            }
            return output.textContent || '';
        };

        const writeText = (value) => {
            if (output instanceof HTMLTextAreaElement || output instanceof HTMLInputElement) {
                output.value = value;
                return;
            }
            output.textContent = value;
        };

        const nextText = `${readText()}${fullLogLine}`;
        writeText(nextText);

        const lines = readText().split('\n');
        if (lines.length > 1000) {
            writeText(lines.slice(-1000).join('\n'));
        }

        if (shouldScroll) {
            output.scrollTop = output.scrollHeight;
        }
    },

    updateConnectionStatus: function(status) {
        const statusElement = document.getElementById('wsConnectionStatus');
        const connectBtn = document.getElementById('wsConnectBtn');
        const disconnectBtn = document.getElementById('wsDisconnectBtn');
        const logOutput = document.getElementById('logsOutput');

        if (!statusElement) return;

        let statusText = '';
        let statusClass = '';

        switch (status) {
            case 'connected':
                statusText = '实时连接';
                statusClass = 'badge bg-danger';
                if (connectBtn) {
                    connectBtn.disabled = true;
                    connectBtn.classList.add('disabled');
                }
                if (disconnectBtn) {
                    disconnectBtn.disabled = false;
                    disconnectBtn.classList.remove('disabled');
                }
                if (logOutput) {
                    logOutput.classList.add('ws-connected');
                }
                break;
            case 'connecting':
                statusText = '连接中...';
                statusClass = 'badge bg-warning';
                if (connectBtn) {
                    connectBtn.disabled = true;
                    connectBtn.classList.add('disabled');
                }
                if (disconnectBtn) {
                    disconnectBtn.disabled = true;
                    disconnectBtn.classList.add('disabled');
                }
                break;
            case 'disconnected':
                statusText = '已断开';
                statusClass = 'badge bg-secondary';
                if (connectBtn) {
                    connectBtn.disabled = false;
                    connectBtn.classList.remove('disabled');
                }
                if (disconnectBtn) {
                    disconnectBtn.disabled = true;
                    disconnectBtn.classList.add('disabled');
                }
                if (logOutput) {
                    logOutput.classList.remove('ws-connected');
                }
                break;
            case 'error':
                statusText = '连接错误';
                statusClass = 'badge bg-danger';
                if (connectBtn) {
                    connectBtn.disabled = false;
                    connectBtn.classList.remove('disabled');
                }
                if (disconnectBtn) {
                    disconnectBtn.disabled = true;
                    disconnectBtn.classList.add('disabled');
                }
                if (logOutput) {
                    logOutput.classList.remove('ws-connected');
                }
                break;
            default:
                statusText = '未知状态';
                statusClass = 'badge bg-dark';
                if (connectBtn) {
                    connectBtn.disabled = false;
                    connectBtn.classList.remove('disabled');
                }
                if (disconnectBtn) {
                    disconnectBtn.disabled = true;
                    disconnectBtn.classList.add('disabled');
                }
        }

        statusElement.className = statusClass;
        statusElement.textContent = statusText;
    }
};
