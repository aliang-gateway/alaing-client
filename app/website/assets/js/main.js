function initializeLegacyMain() {
    if (window.__legacyMainInitialized) {
        return;
    }
    window.__legacyMainInitialized = true;

    loadDashboard();

    loadAuthUserInfo().catch(error => {
        console.debug('Initial user info load failed (this is normal if no user is configured):', error);
    });

    console.log('Auto-connecting to log WebSocket on page load...');
    logWebSocket.connect();

    updateGlobalStatus();

    appState.statusPollingInterval = setInterval(updateGlobalStatus, 1000);

    setInterval(() => {
        if (appState.currentPage === 'dashboard') {
            loadDashboard();
        }
    }, 5000);

    window.addEventListener('beforeunload', () => {
        if (appState.statusPollingInterval) {
            clearInterval(appState.statusPollingInterval);
            appState.statusPollingInterval = null;
        }
    });

    setInterval(() => {
        if (appState.currentPage === 'userinfo') {
            loadAuthUserInfo();
            loadRefreshStatus();
        }
    }, 30000);

    if (document.getElementById('btn-check-cert')) {
        const openModalBtn = document.getElementById('openCertManagementModalBtn');
        const closeModalBtn = document.getElementById('certModalCloseBtn');
        const modalBackdrop = document.getElementById('certManagementModalBackdrop');
        const certModal = document.getElementById('certManagementModal');

        if (openModalBtn) {
            openModalBtn.addEventListener('click', openCertManagementModal);
        }

        if (closeModalBtn) {
            closeModalBtn.addEventListener('click', closeCertManagementModal);
        }

        if (modalBackdrop) {
            modalBackdrop.addEventListener('click', closeCertManagementModal);
        }

        if (certModal) {
            certModal.addEventListener('click', (event) => {
                if (event.target === certModal) {
                    closeCertManagementModal();
                }
            });
        }

        document.addEventListener('keydown', (event) => {
            if (event.key === 'Escape') {
                closeCertManagementModal();
            }
        });

        document.getElementById('btn-check-cert').addEventListener('click', checkCertInstallation);
        document.getElementById('btn-export-cert').addEventListener('click', exportCert);
        document.getElementById('btn-download-cert').addEventListener('click', downloadCert);
        document.getElementById('btn-install-cert').addEventListener('click', installCert);
        document.getElementById('btn-remove-cert').addEventListener('click', removeCert);
        document.getElementById('cert-type-select').addEventListener('change', loadCertStatus);

        loadCertStatus();

        let certStatusPolling = null;

        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                if (certStatusPolling) clearInterval(certStatusPolling);
            } else {
                loadCertStatus();
                if (!certStatusPolling) {
                    certStatusPolling = setInterval(loadCertStatus, 10000);
                }
            }
        });

        certStatusPolling = setInterval(loadCertStatus, 10000);
    }

    initQuickChat();
    initDomainFilterControls();
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializeLegacyMain, { once: true });
} else {
    initializeLegacyMain();
}
