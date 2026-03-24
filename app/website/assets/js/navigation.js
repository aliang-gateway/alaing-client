function switchPage(page) {
    document.querySelectorAll('.content-section').forEach(el => {
        el.classList.remove('active');
        el.classList.add('hidden');
    });

    const pageEl = document.getElementById(`${page}-page`);
    if (pageEl) {
        pageEl.classList.add('active');
        pageEl.classList.remove('hidden');
    }

    document.querySelectorAll('.tab-item').forEach(link => {
        link.classList.remove('active');
        if (link.dataset.page === page) {
            link.classList.add('active');
        }
    });

    appState.currentPage = page;

    if (page === 'dashboard') {
        loadDashboard();
        const dashboardStatsInterval = setInterval(() => {
            if (appState.currentPage === 'dashboard') {
                loadStatsData().catch(error => console.error('Failed to load stats:', error));
            } else {
                clearInterval(dashboardStatsInterval);
            }
        }, 1500);
    } else if (page === 'rules') {
        loadRulesData();
        loadRoutingConfig();
    } else if (page === 'logs') {
        logWebSocket.connect();
        loadLogs();
        loadLogConfig();
    } else if (page === 'userinfo') {
        loadAuthUserInfo();
        loadRefreshStatus();
    } else if (page === 'stats') {
        loadStatsData();
        const statsRefreshInterval = setInterval(() => {
            if (appState.currentPage === 'stats') {
                loadStatsData();
            } else {
                clearInterval(statsRefreshInterval);
            }
        }, 1500);
    } else if (page === 'dnscache') {
        loadDNSCacheData();
        const dnsRefreshInterval = setInterval(() => {
            if (appState.currentPage === 'dnscache') {
                loadDNSCacheData();
            } else {
                clearInterval(dnsRefreshInterval);
            }
        }, 5000);
    }

    appState.currentPage = page;
}

function cleanupCurrentPage() {
}

document.querySelectorAll('.tab-item').forEach(link => {
    link.addEventListener('click', (e) => {
        e.preventDefault();
        const page = link.dataset.page;

        cleanupCurrentPage();

        switchPage(page);
    });
});
