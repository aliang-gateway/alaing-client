let settingsBindingsApplied = false;

function applySettingsBindings() {
    if (settingsBindingsApplied) {
        return;
    }

    const dashboardPage = document.getElementById('dashboard-page');
    const settingsPage = document.getElementById('settings-page');
    const goToSettingsBtn = document.getElementById('goToSettingsBtn');
    const headerSettingsBtn = document.getElementById('headerSettingsBtn');
    const backToDashboard = document.getElementById('backToDashboard');

    const settingsTabs = document.querySelectorAll('.settings-tab');
    const settingsContents = document.querySelectorAll('.settings-content');

    const showSettingsPage = () => {
        if (!dashboardPage || !settingsPage) {
            return;
        }
        appState.currentPage = 'settings';
        dashboardPage.classList.remove('active');
        dashboardPage.classList.add('hidden');
        settingsPage.classList.remove('hidden');
        settingsPage.classList.add('active');
    };

    const showDashboardPage = () => {
        if (!dashboardPage || !settingsPage) {
            return;
        }
        appState.currentPage = 'dashboard';
        settingsPage.classList.remove('active');
        settingsPage.classList.add('hidden');
        dashboardPage.classList.remove('hidden');
        dashboardPage.classList.add('active');
    };

    if (!dashboardPage || !settingsPage) {
        return;
    }

    appState.currentPage = 'dashboard';
    dashboardPage.classList.add('active');
    dashboardPage.classList.remove('hidden');
    settingsPage.classList.add('hidden');

    if (goToSettingsBtn) {
        goToSettingsBtn.addEventListener('click', showSettingsPage);
    }

    if (headerSettingsBtn) {
        headerSettingsBtn.addEventListener('click', showSettingsPage);
    }

    if (backToDashboard) {
        backToDashboard.addEventListener('click', showDashboardPage);
    }

    if (settingsTabs.length > 0 && settingsContents.length > 0) {
        settingsTabs.forEach(tab => {
            tab.addEventListener('click', () => {
                const targetTab = tab.getAttribute('data-tab');
                if (!targetTab) {
                    return;
                }

                settingsTabs.forEach(btn => {
                    btn.classList.remove('active');
                });
                tab.classList.add('active');

                settingsContents.forEach(content => {
                    if (content.getAttribute('data-content') === targetTab) {
                        content.classList.add('active');
                        content.classList.remove('hidden');
                    } else {
                        content.classList.remove('active');
                        content.classList.add('hidden');
                    }
                });
            });
        });
    }

    const dashboardCheckBtn = document.getElementById('dashBtnCheckCert');
    if (dashboardCheckBtn) {
        dashboardCheckBtn.addEventListener('click', () => {
            showSettingsPage();
            const tab = document.querySelector('.settings-tab[data-tab="system"]');
            if (tab instanceof HTMLElement) {
                tab.click();
            }
        });
    }

    const dashboardInstallBtn = document.getElementById('dashBtnInstallCert');
    if (dashboardInstallBtn) {
        dashboardInstallBtn.addEventListener('click', () => {
            showSettingsPage();
            const tab = document.querySelector('.settings-tab[data-tab="system"]');
            if (tab instanceof HTMLElement) {
                tab.click();
            }
            const installBtn = document.getElementById('btn-install-cert');
            if (installBtn instanceof HTMLElement) {
                installBtn.click();
            }
        });
    }

    const sidebarCertDetailsBtn = document.getElementById('sidebarCertDetailsBtn');
    if (sidebarCertDetailsBtn) {
        sidebarCertDetailsBtn.addEventListener('click', () => {
            showSettingsPage();
            const tab = document.querySelector('.settings-tab[data-tab="system"]');
            if (tab instanceof HTMLElement) {
                tab.click();
            }
            const openModalBtn = document.getElementById('openCertManagementModalBtn');
            if (openModalBtn instanceof HTMLElement) {
                openModalBtn.click();
            }
        });
    }

    const sidebarCertReinstallBtn = document.getElementById('sidebarCertReinstallBtn');
    if (sidebarCertReinstallBtn) {
        sidebarCertReinstallBtn.addEventListener('click', () => {
            showSettingsPage();
            const tab = document.querySelector('.settings-tab[data-tab="system"]');
            if (tab instanceof HTMLElement) {
                tab.click();
            }
            const openModalBtn = document.getElementById('openCertManagementModalBtn');
            if (openModalBtn instanceof HTMLElement) {
                openModalBtn.click();
            }
            const installBtn = document.getElementById('btn-install-cert');
            if (installBtn instanceof HTMLElement) {
                installBtn.click();
            }
        });
    }

    settingsBindingsApplied = true;
}

document.addEventListener('DOMContentLoaded', applySettingsBindings);
window.addEventListener('app:mounted', applySettingsBindings);
