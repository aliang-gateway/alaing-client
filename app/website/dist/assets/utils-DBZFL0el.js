// ============ 通知功能 ============
function showNotification(message, type = 'success') {
    // 确保通知容器存在
    let container = document.getElementById('notification-container');
    if (container && container.closest('.compat-anchors')) {
        container.removeAttribute('id');
        container = null;
    }
    if (!container) {
        container = document.createElement('div');
        container.id = 'notification-container';
        document.body.appendChild(container);
    }

    const alertClass = type === 'success' ? 'alert-success' : 'alert-danger';
    const notificationId = 'notification-' + Date.now();
    const alertHtml = `
        <div class="alert ${alertClass} alert-dismissible fade show notification" role="alert" id="${notificationId}">
            ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        </div>
    `;

    container.insertAdjacentHTML('beforeend', alertHtml);

    // 自动移除通知
    setTimeout(() => {
        const alert = document.getElementById(notificationId);
        if (alert) {
            // 添加淡出动画
            alert.classList.add('removing');
            // 等待动画完成后移除元素
            setTimeout(() => alert.remove(), 300); // 等待淡出动画完成
        }
    }, 5000);
}

function showSuccess(message) {
    showNotification(message, 'success');
}

function showError(message) {
    showNotification(message, 'error');
}

// ============ 加载状态管理 ============
function showLoading(element) {
    element.classList.add('loading');
    element.disabled = true;
}

function hideLoading(element) {
    element.classList.remove('loading');
    element.disabled = false;
}

// ============ 模态加载框 ============
let modalLoadingInstance = null;

/**
 * 显示模态加载框
 * @param {string} message - 提示文字 (默认: "加载中，请稍候...")
 */
function showModalLoading(message = '加载中，请稍候...') {
    // 防止重复创建
    if (modalLoadingInstance) {
        return;
    }

    // 创建DOM结构
    const overlay = document.createElement('div');
    overlay.id = 'modal-loading-overlay';
    overlay.className = 'modal-loading-overlay';
    overlay.innerHTML = `
        <div class="modal-loading-container">
            <div class="modal-loading-spinner"></div>
            <div class="modal-loading-text">${message}</div>
        </div>
    `;

    // 插入到body
    document.body.appendChild(overlay);
    modalLoadingInstance = overlay;

    // 防止背景滚动
    document.body.style.overflow = 'hidden';
}

/**
 * 隐藏模态加载框
 */
function hideModalLoading() {
    if (!modalLoadingInstance) {
        return;
    }

    // 添加淡出动画
    modalLoadingInstance.classList.add('fade-out');

    // 等待动画完成后移除
    setTimeout(() => {
        if (modalLoadingInstance && modalLoadingInstance.parentNode) {
            modalLoadingInstance.parentNode.removeChild(modalLoadingInstance);
            modalLoadingInstance = null;
            document.body.style.overflow = '';
        }
    }, 200); // 与CSS动画时长一致
}

// ============ 格式化工具函数 ============

// 格式化日期时间
function formatDateTime(isoString) {
    try {
        const date = new Date(isoString);
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
    } catch (error) {
        return isoString || '-';
    }
}

// 格式化相对时间（从Unix时间戳）
function formatRelativeTime(timestamp) {
    if (!timestamp || timestamp === 0) return '未测试';

    const now = Math.floor(Date.now() / 1000);
    const diff = now - parseInt(timestamp);

    if (diff < 0) return '刚才';
    if (diff < 60) return '刚才';
    if (diff < 3600) return Math.floor(diff / 60) + '分钟前';
    if (diff < 86400) return Math.floor(diff / 3600) + '小时前';
    if (diff < 2592000) return Math.floor(diff / 86400) + '天前';
    return Math.floor(diff / 2592000) + '月前';
}

// 格式化数字，添加千位符
function formatNumber(num) {
    return num.toString().replace(/\B(?=(\\d{3})+(?!\\d))/g, ',');
}

// 获取绑定源的显示颜色（Bootstrap 颜色）
function getBadgeColor(source) {
    const colors = {
        'sni': 'primary',
        'http_host': 'warning',
        'dns': 'success',
        'connect': 'danger'
    };
    return colors[source] || 'primary';
}

function getCssColorChannels(tokenName, fallbackTokenName = '--color-black-rgb') {
    if (!tokenName) {
        return getCssColorChannels(fallbackTokenName, null);
    }

    const root = document.documentElement;
    if (!root) {
        return '';
    }

    const value = getComputedStyle(root).getPropertyValue(tokenName).trim();
    if (value) {
        return value;
    }

    if (fallbackTokenName && fallbackTokenName !== tokenName) {
        return getCssColorChannels(fallbackTokenName, null);
    }

    return '';
}

function cssColor(tokenName, alpha, fallbackTokenName = '--color-black-rgb') {
    const channels = getCssColorChannels(tokenName, fallbackTokenName);
    if (alpha === undefined || alpha === null) {
        return `rgb(${channels})`;
    }

    return `rgb(${channels} / ${alpha})`;
}

// 格式化字节为可读的大小
function formatBytes(bytes, decimals = 2) {
    if (bytes === 0 || bytes === undefined || bytes === null) return '0 B';

    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

// HTML转义函数
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
