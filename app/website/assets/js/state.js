// ============ 全局状态管理 ============
const appState = {
    currentPage: 'dashboard',
    proxies: [],
    currentProxy: null,
    doorMembers: [],
    loading: false,
    // 新增状态属性
    runStatus: {
        isRunning: false,
        currentMode: null,
        lastUpdated: null
    },
    ruleEngineStatus: {
        isEnabled: false,
        lastUpdated: null
    },
    logConfig: {
        level: 'info',
        source: '',
        autoScroll: true
    },
    statusPollingInterval: null
};

// ============ 流量图表数据管理 ============
const chartDataManager = {
    maxPoints: 60,

    data: {
        timestamps: [],
        uploadSpeeds: [],
        downloadSpeeds: []
    },

    lastValues: {
        uploadTotal: 0,
        downloadTotal: 0,
        timestamp: 0
    },

    addPoint(uploadTotal, downloadTotal) {
        const now = Date.now();
        const timeDelta = (now - this.lastValues.timestamp) / 1000;

        const uploadSpeed = timeDelta > 0 ?
            Math.max(0, (uploadTotal - this.lastValues.uploadTotal) / timeDelta) : 0;
        const downloadSpeed = timeDelta > 0 ?
            Math.max(0, (downloadTotal - this.lastValues.downloadTotal) / timeDelta) : 0;

        this.data.timestamps.push(now);
        this.data.uploadSpeeds.push(uploadSpeed);
        this.data.downloadSpeeds.push(downloadSpeed);

        if (this.data.timestamps.length > this.maxPoints) {
            this.data.timestamps.shift();
            this.data.uploadSpeeds.shift();
            this.data.downloadSpeeds.shift();
        }

        this.lastValues.uploadTotal = uploadTotal;
        this.lastValues.downloadTotal = downloadTotal;
        this.lastValues.timestamp = now;
    },

    clear() {
        this.data = { timestamps: [], uploadSpeeds: [], downloadSpeeds: [] };
        this.lastValues = { uploadTotal: 0, downloadTotal: 0, timestamp: 0 };
    }
};

const dashboardRequestLog = {
    maxItems: 50,
    entries: []
};

const chatStore = {
    storageKey: 'alianggate-chat-history',
    maxItems: 200,
    entries: []
};

const domainCategoryMap = {
    cursor: ['cursor.sh', 'api2.cursor.sh', 'cursor.com'],
    openai: ['openai.com', 'api.openai.com'],
    claude: ['anthropic.com', 'claude.ai'],
    chatgpt: ['chatgpt.com'],
    copilot: ['githubcopilot.com', 'copilot.microsoft.com']
};

let activeDomainFilter = 'all';

// 全局图表实例
let charts = {};
