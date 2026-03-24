import { nextTick } from 'vue';

import stateScriptUrl from '../../assets/js/state.js?url';
import utilsScriptUrl from '../../assets/js/utils.js?url';
import apiScriptUrl from '../../assets/js/api.js?url';
import dashboardUiScriptUrl from '../../assets/js/dashboard-ui.js?url';
import statusScriptUrl from '../../assets/js/status.js?url';
import chartsScriptUrl from '../../assets/js/charts.js?url';
import runControlScriptUrl from '../../assets/js/run-control.js?url';
import rulesScriptUrl from '../../assets/js/rules.js?url';
import websocketScriptUrl from '../../assets/js/websocket.js?url';
import logsScriptUrl from '../../assets/js/logs.js?url';
import dnsScriptUrl from '../../assets/js/dns.js?url';
import authScriptUrl from '../../assets/js/auth.js?url';
import certScriptUrl from '../../assets/js/cert.js?url';
import navigationScriptUrl from '../../assets/js/navigation.js?url';
import trafficScriptUrl from '../../assets/js/traffic.js?url';
import settingsScriptUrl from '../../assets/js/settings.js?url';
import mainScriptUrl from '../../assets/js/main.js?url';

const legacyScriptOrder = [
  stateScriptUrl,
  utilsScriptUrl,
  apiScriptUrl,
  dashboardUiScriptUrl,
  statusScriptUrl,
  chartsScriptUrl,
  runControlScriptUrl,
  rulesScriptUrl,
  websocketScriptUrl,
  logsScriptUrl,
  dnsScriptUrl,
  authScriptUrl,
  certScriptUrl,
  navigationScriptUrl,
  trafficScriptUrl,
  settingsScriptUrl,
  mainScriptUrl
];

function appendScript(src) {
  return new Promise((resolve, reject) => {
    const script = document.createElement('script');
    script.src = src;
    script.async = false;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error(`Failed to load legacy script: ${src}`));
    document.body.appendChild(script);
  });
}

let loaded = false;

export async function loadLegacyScripts() {
  if (loaded) {
    return;
  }

  await nextTick();

  for (const src of legacyScriptOrder) {
    await appendScript(src);
  }

  document.dispatchEvent(new Event('DOMContentLoaded'));

  loaded = true;
}
