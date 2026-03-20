function destroyCharts() {
    Object.values(charts).forEach(chart => {
        if (chart && typeof chart.destroy === 'function') {
            chart.destroy();
        }
    });
    charts = {};
}

function getChartColors() {
    return {
        uploadBorder: cssColor('--color-chart-upload-rgb'),
        uploadBg: cssColor('--color-chart-upload-rgb', 0.1),
        downloadBorder: cssColor('--color-chart-download-rgb'),
        downloadBg: cssColor('--color-chart-download-rgb', 0.1),
        pieSlices: [
            cssColor('--color-chart-pie-1-rgb', 0.72),
            cssColor('--color-chart-pie-2-rgb', 0.72),
            cssColor('--color-chart-pie-3-rgb', 0.72),
            cssColor('--color-chart-pie-4-rgb', 0.72)
        ],
        pieBorder: cssColor('--color-chart-pie-border-rgb', 0.88),
        legendText: cssColor('--color-chart-legend-rgb')
    };
}

function initChart() {
    const chartColors = getChartColors();

    if (document.getElementById('trafficChart')) {
        if (charts.traffic && typeof charts.traffic.destroy === 'function') {
            charts.traffic.destroy();
            charts.traffic = null;
        }

        const ctx = document.getElementById('trafficChart').getContext('2d');
        charts.traffic = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [
                    {
                        label: '上传 (Bytes/s)',
                        data: [],
                        borderColor: chartColors.uploadBorder,
                        backgroundColor: chartColors.uploadBg,
                        borderWidth: 2,
                        tension: 0.3,
                        fill: true,
                        pointRadius: 0,
                        pointHoverRadius: 5
                    },
                    {
                        label: '下载 (Bytes/s)',
                        data: [],
                        borderColor: chartColors.downloadBorder,
                        backgroundColor: chartColors.downloadBg,
                        borderWidth: 2,
                        tension: 0.3,
                        fill: true,
                        pointRadius: 0,
                        pointHoverRadius: 5
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: { mode: 'index', intersect: false },
                plugins: {
                    legend: { display: false },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                let label = context.dataset.label || '';
                                if (label) {
                                    label += ': ';
                                }
                                label += formatBytes(context.parsed.y) + '/s';
                                return label;
                            }
                        }
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            callback: function(value) {
                                return formatBytes(value) + '/s';
                            }
                        }
                    },
                    x: {
                        ticks: {
                            maxTicksLimit: 10
                        }
                    }
                }
            }
        });
    }

    if (document.getElementById('domainPieChart')) {
        if (charts.domainPie && typeof charts.domainPie.destroy === 'function') {
            charts.domainPie.destroy();
            charts.domainPie = null;
        }

        const pieCtx = document.getElementById('domainPieChart').getContext('2d');
        charts.domainPie = new Chart(pieCtx, {
            type: 'pie',
            data: {
                labels: ['暂无数据'],
                datasets: [{
                    data: [1],
                    backgroundColor: chartColors.pieSlices,
                    borderColor: chartColors.pieBorder,
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'bottom',
                        labels: {
                            color: chartColors.legendText,
                            boxWidth: 10
                        }
                    }
                }
            }
        });
    }
}

function updateChartData(statsData) {
    chartDataManager.addPoint(
        statsData.uploadTotal || 0,
        statsData.downloadTotal || 0
    );

    if (charts.traffic) {
        const timeLabels = chartDataManager.data.timestamps.map(ts => {
            const date = new Date(ts);
            return date.toLocaleTimeString();
        });

        charts.traffic.data.labels = timeLabels;
        charts.traffic.data.datasets[0].data = chartDataManager.data.uploadSpeeds;
        charts.traffic.data.datasets[1].data = chartDataManager.data.downloadSpeeds;
        charts.traffic.update('none');
    }
}

async function loadStatsData() {
    try {
        const response = await apiGet('/stats/traffic/current');
        if (!response) {
            console.error('Failed to load stats data');
            return;
        }

        document.getElementById('statsUploadTotal').textContent = formatBytes(response.uploadTotal || 0);
        document.getElementById('statsDownloadTotal').textContent = formatBytes(response.downloadTotal || 0);

        let totalConnections = 0;
        if (response.byRoute) {
            Object.values(response.byRoute).forEach(route => {
                totalConnections += route.connectionCount || 0;
            });
        }
        document.getElementById('statsConnectionCount').textContent = formatNumber(totalConnections);

        const totalTraffic = (response.uploadTotal || 0) + (response.downloadTotal || 0);
        document.getElementById('statsTotalTraffic').textContent = formatBytes(totalTraffic);

        renderStatsTable(response.byRoute);
        updateDomainPieChartByLogs();

        updateChartData(response);
    } catch (error) {
        console.error('Failed to load stats data:', error);
    }
}

function renderStatsTable(byRoute) {
    const tbody = document.getElementById('statsTableBody');

    if (!byRoute || Object.keys(byRoute).length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">暂无数据</td></tr>';
        return;
    }

    const routeNames = {
        'RouteToCursor': 'MITM (Cursor/Nonelane)',
        'RouteToDoor': '代理 (VLESS/Shadowsocks)',
        'RouteDirect': '直连'
    };

    let html = '';

    Object.entries(byRoute).forEach(([routeType, stats]) => {
        const displayName = routeNames[routeType] || routeType;
        const connectionCount = stats.connectionCount || 0;
        const uploadTotal = stats.uploadTotal || 0;
        const downloadTotal = stats.downloadTotal || 0;
        const averageUpload = stats.averageUpload || 0;
        const averageDownload = stats.averageDownload || 0;

        html += `<tr>
            <td><strong>${displayName}</strong></td>
            <td class="text-right">${formatNumber(connectionCount)}</td>
            <td class="text-right">${formatBytes(uploadTotal)}</td>
            <td class="text-right">${formatBytes(downloadTotal)}</td>
            <td class="text-right">${formatBytes(averageUpload)}</td>
            <td class="text-right">${formatBytes(averageDownload)}</td>
        </tr>`;
    });

    tbody.innerHTML = html;
}
