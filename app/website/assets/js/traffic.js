let currentTimescale = '1s';
let trafficStatsRefreshInterval = null;

async function refreshTrafficStats() {
    try {
        const statsResponse = await fetch(`/api/stats/traffic/${currentTimescale}`);
        if (!statsResponse.ok) {
            console.error('Failed to fetch traffic stats:', statsResponse.status);
            return;
        }

        const statsResult = await statsResponse.json();
        if (statsResult.success && statsResult.data && statsResult.data.stats) {
            updateTrafficChart(statsResult.data.stats);
        }

        const currentResponse = await fetch('/api/stats/traffic/current');
        if (currentResponse.ok) {
            const currentResult = await currentResponse.json();
            if (currentResult.success && currentResult.data) {
                updateConnectionInfo(currentResult.data.active_connections);
                updateCurrentRates(currentResult.data.upload_rate, currentResult.data.download_rate);
            }
        }
    } catch (error) {
        console.error('Error refreshing traffic stats:', error);
    }
}

function updateTrafficChart(statsArray) {
    if (!charts.traffic || !statsArray || statsArray.length === 0) {
        return;
    }

    const uploads = [];
    const downloads = [];
    const timestamps = [];

    for (let i = 0; i < statsArray.length; i++) {
        const stat = statsArray[i];
        const date = new Date(stat.timestamp * 1000);
        timestamps.push(formatTime(date));
        uploads.push(stat.upload_bytes);
        downloads.push(stat.download_bytes);
    }

    charts.traffic.data.labels = timestamps;
    charts.traffic.data.datasets[0].data = uploads;
    charts.traffic.data.datasets[1].data = downloads;
    charts.traffic.update('none');
}

function formatTime(date) {
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    const seconds = String(date.getSeconds()).padStart(2, '0');
    return `${hours}:${minutes}:${seconds}`;
}

function updateConnectionInfo(connectionCount) {
    const element = document.getElementById('statsConnectionCount');
    if (element) {
        element.textContent = connectionCount || '0';
    }
}

function updateCurrentRates(uploadRate, downloadRate) {
    const uploadElement = document.getElementById('statsUploadRate');
    const downloadElement = document.getElementById('statsDownloadRate');

    if (uploadElement) {
        uploadElement.textContent = formatBytes(uploadRate || 0) + '/s';
    }
    if (downloadElement) {
        downloadElement.textContent = formatBytes(downloadRate || 0) + '/s';
    }
}

function initRealTimeMonitoring() {
    if (trafficStatsRefreshInterval) {
        clearInterval(trafficStatsRefreshInterval);
    }

    refreshTrafficStats();

    trafficStatsRefreshInterval = setInterval(refreshTrafficStats, 1000);

    const timescaleButtons = document.querySelectorAll('[data-timescale]');
    timescaleButtons.forEach(btn => {
        btn.addEventListener('click', function() {
            const newTimescale = this.getAttribute('data-timescale');
            switchTrafficTimescale(newTimescale);
        });
    });
}

function switchTrafficTimescale(newTimescale) {
    if (['1s', '5s', '15s'].includes(newTimescale)) {
        currentTimescale = newTimescale;

        document.querySelectorAll('[data-timescale]').forEach(btn => {
            btn.classList.remove('active');
            if (btn.getAttribute('data-timescale') === newTimescale) {
                btn.classList.add('active');
            }
        });

        refreshTrafficStats();
    }
}

function stopRealTimeMonitoring() {
    if (trafficStatsRefreshInterval) {
        clearInterval(trafficStatsRefreshInterval);
        trafficStatsRefreshInterval = null;
    }
}

window.addEventListener('beforeunload', function() {
    stopRealTimeMonitoring();
});

document.addEventListener('DOMContentLoaded', function() {
    const dashboardPage = document.getElementById('dashboard-page');
    if (dashboardPage && dashboardPage.classList.contains('active')) {
        initRealTimeMonitoring();
    }

    const navLinks = document.querySelectorAll('.nav-link');
    navLinks.forEach(link => {
        link.addEventListener('click', function() {
            const targetPage = this.getAttribute('data-page');
            if (targetPage === 'dashboard') {
                setTimeout(initRealTimeMonitoring, 100);
            } else {
                stopRealTimeMonitoring();
            }
        });
    });
});
