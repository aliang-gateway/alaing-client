<template>
  <div class="settings-content hidden" data-content="userinfo">
    <div class="settings-card">
      <h3 class="settings-card-title">用户信息</h3>
      <div id="authUserInfoContainer" class="space-y-3">
        <div class="flex justify-between p-3 rounded bg-slate-50 dark:bg-slate-800/40">
          <span class="text-slate-500">用户名</span>
          <span class="font-medium">-</span>
        </div>
        <div class="flex justify-between p-3 rounded bg-slate-50 dark:bg-slate-800/40">
          <span class="text-slate-500">余额</span>
          <span id="userBalance" class="font-semibold text-primary">-</span>
        </div>
      </div>
      <div class="mt-6">
        <div class="flex flex-wrap gap-2">
          <input type="text" id="authTokenInput" placeholder="Token" class="flex-1 min-w-[220px] px-3 py-2 border border-slate-300 rounded focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary" />
          <button type="button" id="authActivateBtn" class="settings-btn-primary">
            激活
          </button>
          <button type="button" id="authLogoutBtn" class="settings-btn-secondary">
            登出
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'UserInfoSettings'
}
</script>