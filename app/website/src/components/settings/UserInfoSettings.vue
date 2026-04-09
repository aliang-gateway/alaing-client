<template>
  <div class="settings-pane" data-pane="userinfo">
    <div class="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-background-dark sm:p-4.5">
      <div class="mb-3 flex items-start justify-between gap-3">
        <div>
          <h3 class="flex items-center gap-2 font-bold">
            <span class="material-symbols-outlined text-primary">person</span>
            {{ t('user_title') }}
          </h3>
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t('user_description') }}</p>
        </div>
        <button
          v-if="isAuthenticated"
          type="button"
          class="inline-flex min-h-10 items-center justify-center rounded border border-slate-200 px-3 py-2 text-xs font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:text-slate-100 dark:hover:bg-slate-800"
          :disabled="isRefreshing"
          @click="refreshAll"
        >
          {{ isRefreshing ? t('user_refreshing') : t('user_refresh') }}
        </button>
      </div>

      <div
        class="mb-3 rounded-lg border px-3 py-2 text-[11px]"
        :class="isAuthenticated ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-300' : 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-300'"
      >
        {{ authNotice }}
      </div>

      <form v-if="!isAuthenticated" class="mx-auto max-w-sm space-y-4 py-4" @submit.prevent="submitLogin">
        <div class="rounded-xl border border-slate-200 bg-slate-50/80 p-5 dark:border-slate-800 dark:bg-slate-900/50">
          <div class="flex flex-col items-center gap-3 text-center">
            <div class="flex size-12 items-center justify-center rounded-full bg-primary/10">
              <span class="material-symbols-outlined text-2xl text-primary">lock</span>
            </div>
            <div>
              <p class="text-sm font-semibold text-slate-900 dark:text-white">{{ t('user_signInTitle') }}</p>
              <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
                {{ t('user_signInDesc') }}
              </p>
            </div>
          </div>
        </div>

        <div class="space-y-3">
          <div>
            <label class="mb-1.5 block text-xs font-semibold uppercase tracking-wide text-slate-500">{{ t('user_email') }}</label>
            <div class="relative">
              <span class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-slate-400">
                <span class="material-symbols-outlined text-[18px]">mail</span>
              </span>
              <input
                v-model.trim="email"
                type="email"
                autocomplete="username"
                class="h-10 w-full rounded-lg border border-slate-200 bg-white pl-9 pr-3 text-sm text-slate-700 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/10 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:focus:ring-primary/20"
                placeholder="name@example.com"
                :disabled="loginPending"
              />
            </div>
          </div>
          <div>
            <label class="mb-1.5 block text-xs font-semibold uppercase tracking-wide text-slate-500">{{ t('user_password') }}</label>
            <div class="relative">
              <span class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-slate-400">
                <span class="material-symbols-outlined text-[18px]">key</span>
              </span>
              <input
                v-model="password"
                type="password"
                autocomplete="current-password"
                class="h-10 w-full rounded-lg border border-slate-200 bg-white pl-9 pr-3 text-sm text-slate-700 outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/10 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:focus:ring-primary/20"
                :placeholder="t('user_passwordPh')"
                :disabled="loginPending"
              />
            </div>
          </div>
        </div>

        <div v-if="loginError" class="flex items-center gap-2 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-xs text-rose-700 dark:border-rose-900 dark:bg-rose-950/30 dark:text-rose-300">
          <span class="material-symbols-outlined text-sm">error</span>
          {{ loginError }}
        </div>

        <button
          type="submit"
          class="inline-flex h-10 w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-white shadow-sm transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
          :disabled="loginPending"
        >
          <span v-if="loginPending" class="inline-block size-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
          {{ loginPending ? t('user_signingIn') : t('user_signIn') }}
        </button>

        <p class="text-center text-[11px] text-slate-400">
          {{ t('user_loginHint') }}
        </p>
      </form>

      <template v-else>
        <div v-if="loadError" class="mb-4 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-xs text-rose-700 dark:border-rose-900 dark:bg-rose-950/30 dark:text-rose-300">
          {{ loadError }}
        </div>

        <div class="grid gap-2.5 sm:grid-cols-2">
          <article class="rounded-xl border border-slate-200 bg-slate-50/80 p-3 dark:border-slate-800 dark:bg-slate-900/50">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-[11px] font-semibold uppercase tracking-wide text-slate-500">{{ t('user_subscriptions') }}</p>
                <p class="mt-1.5 text-xl font-bold text-slate-900 dark:text-white">{{ activeSubscriptionsText }}</p>
              </div>
              <span class="material-symbols-outlined text-[20px] text-primary">stacked_bar_chart</span>
            </div>
            <p class="mt-1.5 text-[11px] leading-5 text-slate-500 dark:text-slate-400">{{ t('user_subscriptionsDesc') }}</p>
          </article>

          <article class="rounded-xl border border-slate-200 bg-slate-50/80 p-3 dark:border-slate-800 dark:bg-slate-900/50">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-[11px] font-semibold uppercase tracking-wide text-slate-500">{{ t('user_usageCost') }}</p>
                <p id="userBalance" class="mt-1.5 text-xl font-bold text-primary">{{ totalUsageText }}</p>
              </div>
              <span class="material-symbols-outlined text-[20px] text-primary">payments</span>
            </div>
            <p class="mt-1.5 text-[11px] leading-5 text-slate-500 dark:text-slate-400">{{ t('user_usageCostDesc') }}</p>
          </article>
        </div>

        <div class="mt-3 grid gap-3 lg:grid-cols-3">
          <div class="rounded-xl border border-slate-200 bg-white p-3.5 dark:border-slate-800 dark:bg-background-dark lg:col-span-2">
            <div class="flex items-center justify-between gap-3">
              <div>
                <p class="text-xs font-semibold uppercase tracking-wide text-slate-500">{{ t('user_accountOverview') }}</p>
                <p class="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">{{ t('user_accountOverviewDesc') }}</p>
              </div>
              <span class="rounded bg-primary/10 px-2 py-1 text-[11px] font-bold uppercase tracking-wide text-primary">{{ profileStatusText }}</span>
            </div>

            <div class="mt-3 grid gap-2 sm:grid-cols-2">
              <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-900/50">
                <p class="text-[11px] font-semibold uppercase tracking-wide text-slate-500">{{ t('user_email') }}</p>
                <p class="mt-0.5 text-sm font-medium text-slate-900 dark:text-white break-all">{{ profile.email || '-' }}</p>
              </div>
              <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-900/50">
                <p class="text-[11px] font-semibold uppercase tracking-wide text-slate-500">{{ t('user_role') }}</p>
                <p class="mt-0.5 text-sm font-medium text-slate-900 dark:text-white">{{ profile.role || '-' }}</p>
              </div>
              <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-900/50">
                <p class="text-[11px] font-semibold uppercase tracking-wide text-slate-500">{{ t('user_concurrency') }}</p>
                <p class="mt-0.5 text-sm font-medium text-slate-900 dark:text-white">{{ concurrencyText }}</p>
              </div>
              <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-900/50">
                <p class="text-[11px] font-semibold uppercase tracking-wide text-slate-500">{{ t('user_allowedGroups') }}</p>
                <p class="mt-0.5 text-sm font-medium text-slate-900 dark:text-white">{{ allowedGroupsText }}</p>
              </div>
            </div>
          </div>

          <div class="rounded-xl border border-slate-200 bg-white p-3.5 dark:border-slate-800 dark:bg-background-dark lg:col-span-1">
            <div class="flex items-start gap-3">
              <span class="material-symbols-outlined text-[20px] text-primary">badge</span>
              <div>
                <p class="text-sm font-semibold text-slate-900 dark:text-white">{{ t('user_updateUsername') }}</p>
                <p class="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">{{ t('user_updateUsernameDesc') }}</p>
              </div>
            </div>

            <form class="mt-3 space-y-2.5" @submit.prevent="submitProfileUpdate">
              <div>
                <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-500">{{ t('user_username') }}</label>
                <input
                  v-model.trim="usernameDraft"
                  type="text"
                  class="h-9 w-full rounded border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none transition focus:border-primary dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
                  :placeholder="t('user_usernamePh')"
                  :disabled="profileSaving"
                />
              </div>
              <p v-if="profileFeedback.text" :class="feedbackClass(profileFeedback.kind)">{{ profileFeedback.text }}</p>
              <button
                type="submit"
                class="inline-flex min-h-10 w-full items-center justify-center rounded bg-primary px-4 py-2 text-sm font-semibold text-white transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
                :disabled="profileSaving"
              >
                {{ profileSaving ? t('user_savingProfile') : t('user_saveProfile') }}
              </button>
            </form>
          </div>
        </div>

        <div class="mt-3 grid gap-3 lg:grid-cols-3">
          <div class="rounded-xl border border-slate-200 bg-white p-3.5 dark:border-slate-800 dark:bg-background-dark lg:col-span-2">
            <div class="mb-2.5 flex items-center justify-between gap-3">
              <div>
                <p class="text-xs font-semibold uppercase tracking-wide text-slate-500">{{ t('user_packageProgress') }}</p>
                <p class="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">{{ t('user_packageProgressDesc') }}</p>
              </div>
              <span class="rounded bg-slate-100 px-2 py-1 text-[11px] font-semibold text-slate-500 dark:bg-slate-800 dark:text-slate-300">{{ t('user_itemsCount', { count: progressItems.length }) }}</span>
            </div>

            <div v-if="isRefreshing && !hasLoaded" class="rounded-lg border border-dashed border-slate-300 bg-slate-50 px-4 py-5 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-400">
              {{ t('user_loadingData') }}
            </div>
            <div v-else-if="!progressItems.length" class="rounded-lg border border-dashed border-slate-300 bg-slate-50 px-4 py-5 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-400">
              {{ t('user_noProgressData') }}
            </div>
            <div v-else class="space-y-2.5">
              <article
                v-for="(item, index) in progressItems"
                :key="progressItemKey(item, index)"
                class="rounded-lg border border-slate-200 bg-slate-50/80 p-2.5 dark:border-slate-800 dark:bg-slate-900/50"
              >
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <p class="text-sm font-semibold text-slate-900 dark:text-white">{{ progressItemTitle(item, index) }}</p>
                    <p class="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">{{ progressItemSubtitle(item) }}</p>
                  </div>
                  <span class="text-xs font-semibold text-primary">{{ formatPercent(item.percent) }}</span>
                </div>
                <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800">
                  <div class="h-full rounded-full bg-primary transition-all" :style="{ width: `${clampPercent(item.percent)}%` }"></div>
                </div>
                <div class="mt-1.5 flex items-center justify-between text-[10px] text-slate-500 dark:text-slate-400">
                  <span>{{ progressItemCurrentLabel(item) }}</span>
                  <span>{{ progressItemTotalLabel(item) }}</span>
                </div>
              </article>
            </div>
          </div>

          <div class="rounded-xl border border-slate-200 bg-white p-3.5 dark:border-slate-800 dark:bg-background-dark lg:col-span-1">
            <div class="flex items-start gap-3">
              <span class="material-symbols-outlined text-[20px] text-primary">redeem</span>
              <div>
                <p class="text-sm font-semibold text-slate-900 dark:text-white">{{ t('user_redeemCode') }}</p>
                <p class="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">{{ t('user_redeemCodeDesc') }}</p>
              </div>
            </div>

            <form class="mt-3 space-y-2.5" @submit.prevent="submitRedeemCode">
              <div>
                <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-500">{{ t('user_redeemCodeLabel') }}</label>
                <input
                  v-model.trim="redeemCode"
                  type="text"
                  class="h-9 w-full rounded border border-slate-200 bg-white px-3 text-sm uppercase text-slate-700 outline-none transition focus:border-primary dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
                  :placeholder="t('user_redeemCodePh')"
                  :disabled="redeemPending"
                />
              </div>
              <p v-if="redeemFeedback.text" :class="feedbackClass(redeemFeedback.kind)">{{ redeemFeedback.text }}</p>
              <button
                type="submit"
                class="inline-flex min-h-10 w-full items-center justify-center rounded border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:text-slate-100 dark:hover:bg-slate-800"
                :disabled="redeemPending"
              >
                {{ redeemPending ? t('user_redeeming') : t('user_redeemBtn') }}
              </button>
            </form>
          </div>
        </div>

        <div class="mt-3">
          <button
            id="authLogoutBtn"
            type="button"
            class="inline-flex h-10 w-full items-center justify-center rounded border border-slate-200 px-4 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:text-slate-100 dark:hover:bg-slate-800"
            :disabled="logoutPending"
            @click="handleLogout"
          >
            {{ logoutPending ? t('user_loggingOut') : t('user_logOut') }}
          </button>
        </div>
      </template>

      <div class="compat-anchors" aria-hidden="true">
        <div id="authUserInfoContainer"></div>
        <div id="authRefreshStatusCard"></div>
        <div id="authRefreshRunning"></div>
        <div id="authLastUpdate"></div>
        <div id="authRefreshInterval"></div>
        <div id="authRefreshError"></div>
        <div id="authRefreshErrorMsg"></div>
        <button id="authActivateBtn" type="button"></button>
        <input id="authTokenInput" type="text" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useAuthStore } from '../../stores/auth';
import { useI18n } from '../../i18n';
import {
  getUserCenterProfile,
  getUserCenterUsageProgress,
  getUserCenterUsageSummary,
  redeemUserCenterCode,
  updateUserCenterProfile
} from '../../services/userCenterApi';

const { t } = useI18n();

const email = ref('');
const password = ref('');
const usernameDraft = ref('');
const redeemCode = ref('');
const isRefreshing = ref(false);
const hasLoaded = ref(false);
const profileSaving = ref(false);
const redeemPending = ref(false);
const loadError = ref('');

const profile = reactive({
  id: 0,
  email: '',
  username: '',
  role: '',
  balance: 0,
  concurrency: 0,
  status: '',
  allowedGroups: [],
  createdAt: '',
  updatedAt: ''
});

const usageSummary = reactive({
  activeCount: 0,
  totalUsedUsd: 0,
  subscriptions: []
});

const progressItems = ref([]);
const profileFeedback = reactive({ kind: '', text: '' });
const redeemFeedback = reactive({ kind: '', text: '' });

const {
  user,
  isAuthenticated,
  loginPending,
  logoutPending,
  loginError,
  authNotice,
  loginWithPassword,
  logoutUser,
  restoreAuthSession,
  mergeAuthUser
} = useAuthStore();

const activeSubscriptionsText = computed(() => String(Number(usageSummary.activeCount || 0)));

const totalUsageText = computed(() => formatCurrency(usageSummary.totalUsedUsd));

const profileStatusText = computed(() => profile.status || t('user_activeSession'));

const concurrencyText = computed(() => {
  const value = Number(profile.concurrency || 0);
  return value > 0 ? String(value) : '-';
});

const allowedGroupsText = computed(() => {
  if (!Array.isArray(profile.allowedGroups) || !profile.allowedGroups.length) {
    return '-';
  }
  return profile.allowedGroups.join(', ');
});

async function submitLogin() {
  const success = await loginWithPassword({
    email: email.value,
    password: password.value
  });

  if (success) {
    password.value = '';
    await refreshAll();
  }
}

async function handleLogout() {
  await logoutUser();
  resetUserCenterState();
}

async function refreshAll() {
  if (!isAuthenticated.value || isRefreshing.value) {
    return;
  }

  isRefreshing.value = true;
  loadError.value = '';

  try {
    const [profileEnvelope, usageSummaryEnvelope, usageProgressEnvelope] = await Promise.all([
      getUserCenterProfile(),
      getUserCenterUsageSummary(),
      getUserCenterUsageProgress()
    ]);

    const unauthenticatedMessage = findUnauthenticatedMessage([
      profileEnvelope,
      usageSummaryEnvelope,
      usageProgressEnvelope
    ]);

    if (unauthenticatedMessage) {
      resetUserCenterState();
      loadError.value = unauthenticatedMessage;
      await restoreAuthSession();
      return;
    }

    if (profileEnvelope.status !== 'success') {
      throw new Error(profileEnvelope.message || t('user_loadProfileFailed'));
    }
    if (usageSummaryEnvelope.status !== 'success') {
      throw new Error(usageSummaryEnvelope.message || t('user_loadUsageFailed'));
    }
    if (usageProgressEnvelope.status !== 'success') {
      throw new Error(usageProgressEnvelope.message || t('user_loadProgressFailed'));
    }

    applyProfile(profileEnvelope.data);
    applyUsageSummary(usageSummaryEnvelope.data);
    progressItems.value = normalizeProgressItems(usageProgressEnvelope.data?.items);
    usernameDraft.value = profile.username || user.value?.username || '';
    hasLoaded.value = true;
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : t('user_loadFailed');
  }
  finally {
    isRefreshing.value = false;
  }
}

async function submitProfileUpdate() {
  clearFeedback(profileFeedback);

  if (!usernameDraft.value.trim()) {
    profileFeedback.kind = 'error';
    profileFeedback.text = t('user_usernameEmpty');
    return;
  }

  profileSaving.value = true;

  try {
    const envelope = await updateUserCenterProfile(usernameDraft.value.trim());
    if (envelope.status === 'unauthenticated') {
      profileFeedback.kind = 'error';
      profileFeedback.text = envelope.message || t('user_sessionExpired');
      await restoreAuthSession();
      return;
    }
    if (envelope.status !== 'success') {
      throw new Error(envelope.message || t('user_profileUpdateFailed'));
    }

    applyProfile(envelope.data);
    usernameDraft.value = profile.username || '';
    mergeAuthUser({ username: profile.username }, envelope.message || t('user_profileUpdated'));
    profileFeedback.kind = 'success';
    profileFeedback.text = envelope.message || t('user_profileUpdated');
  } catch (error) {
    profileFeedback.kind = 'error';
    profileFeedback.text = error instanceof Error ? error.message : t('user_profileUpdateFailed');
  } finally {
    profileSaving.value = false;
  }
}

async function submitRedeemCode() {
  clearFeedback(redeemFeedback);

  if (!redeemCode.value.trim()) {
    redeemFeedback.kind = 'error';
    redeemFeedback.text = t('user_redeemCodeEmpty');
    return;
  }

  redeemPending.value = true;

  try {
    const envelope = await redeemUserCenterCode(redeemCode.value.trim());
    if (envelope.status === 'unauthenticated') {
      redeemFeedback.kind = 'error';
      redeemFeedback.text = envelope.message || t('user_sessionExpired');
      await restoreAuthSession();
      return;
    }
    if (envelope.status !== 'success') {
      throw new Error(envelope.message || t('user_redeemFailed'));
    }

    redeemFeedback.kind = 'success';
    redeemFeedback.text = formatRedeemSuccessMessage(envelope.data);
    redeemCode.value = '';
    await refreshAll();
  } catch (error) {
    redeemFeedback.kind = 'error';
    redeemFeedback.text = error instanceof Error ? error.message : t('user_redeemFailed');
  } finally {
    redeemPending.value = false;
  }
}

function applyProfile(incoming) {
  const nextProfile = incoming && typeof incoming === 'object' ? incoming : {};
  profile.id = Number(nextProfile.id || 0);
  profile.email = typeof nextProfile.email === 'string' ? nextProfile.email : '';
  profile.username = typeof nextProfile.username === 'string' ? nextProfile.username : '';
  profile.role = typeof nextProfile.role === 'string' ? nextProfile.role : '';
  profile.balance = Number(nextProfile.balance || 0);
  profile.concurrency = Number(nextProfile.concurrency || 0);
  profile.status = typeof nextProfile.status === 'string' ? nextProfile.status : '';
  profile.allowedGroups = Array.isArray(nextProfile.allowed_groups)
    ? nextProfile.allowed_groups.map((value) => Number(value)).filter((value) => Number.isFinite(value))
    : [];
  profile.createdAt = typeof nextProfile.created_at === 'string' ? nextProfile.created_at : '';
  profile.updatedAt = typeof nextProfile.updated_at === 'string' ? nextProfile.updated_at : '';
}

function applyUsageSummary(incoming) {
  const nextSummary = incoming && typeof incoming === 'object' ? incoming : {};
  usageSummary.activeCount = Number(nextSummary.active_count || 0);
  usageSummary.totalUsedUsd = Number(nextSummary.total_used_usd || 0);
  usageSummary.subscriptions = Array.isArray(nextSummary.subscriptions) ? nextSummary.subscriptions : [];
}

function normalizeProgressItems(items) {
  if (!Array.isArray(items)) {
    return [];
  }

  return items
    .flatMap((item, index) => normalizeProgressItem(item, index))
    .filter(Boolean);
}

function normalizeProgressItem(item, index) {
  const raw = item && typeof item === 'object' ? item : {};
  const progress = asObject(raw.progress);
  const subscription = asObject(raw.subscription);
  const group = asObject(subscription.group);

  const baseName = firstNestedString(raw, [
    ['progress', 'group_name'],
    ['subscription', 'group', 'name'],
    ['subscription', 'group_name'],
    ['group_name'],
    ['name'],
    ['title']
  ]) || t('user_usageItem', { index: index + 1 });

  const baseDescription = firstNestedString(raw, [
    ['subscription', 'group', 'description'],
    ['description'],
    ['subscription', 'description']
  ]);

  const scopeItems = ['daily', 'weekly', 'monthly']
    .map((scope) => normalizeScopedProgressItem(raw, progress, subscription, group, baseName, baseDescription, index, scope))
    .filter(Boolean);

  if (scopeItems.length > 0) {
    return scopeItems;
  }

  const durationItem = normalizeDurationProgressItem(raw, progress, subscription, group, baseName, baseDescription, index);
  if (durationItem) {
    return [durationItem];
  }

  const current = pickNumber(raw, ['current', 'used', 'value', 'consumed']);
  const total = pickNumber(raw, ['total', 'limit', 'max', 'quota']);
  const explicitPercent = pickOptionalNumber(raw, ['percent', 'progress', 'usage_percent']);
  const percent = Number.isFinite(explicitPercent)
    ? explicitPercent
    : total > 0
      ? (current / total) * 100
      : 0;

  return [{
    key: progressItemIdentifier(raw, index),
    title: baseName,
    subtitle: baseDescription || t('user_usageProgress'),
    current,
    total,
    percent,
    unit: pickString(raw, ['unit', 'display_unit']) || '',
    raw
  }];
}

function normalizeScopedProgressItem(raw, progress, subscription, group, baseName, baseDescription, index, scope) {
  const scoped = asObject(progress?.[scope]);
  const total = pickNumber(scoped, ['limit_usd', 'total', 'limit', 'max', 'quota']);
  if (total <= 0) {
    return null;
  }

  const current = pickNumber(scoped, ['used_usd', 'used', 'current', 'value', 'consumed']);
  const explicitPercent = pickOptionalNumber(scoped, ['percentage', 'percent', 'progress', 'usage_percent']);
  const percent = Number.isFinite(explicitPercent)
    ? explicitPercent
    : (current / total) * 100;

  const expiresInDays = pickNumber(progress, ['expires_in_days']);
  const scopeKey = `user_${scope}`;
  const scopeLabel = t(scopeKey);
  const windowDescription = firstNestedString(scoped, [['resets_at']])
    ? t('user_scopeLimitResets', { scope: scopeLabel, date: formatShortDate(scoped.resets_at) })
    : t('user_scopeUsage', { scope: scopeLabel });
  const packageDescription = buildPackageDescription(group, progress, subscription);
  const expiresDescription = expiresInDays > 0 ? t('user_expiresInDays', { count: expiresInDays }) : '';
  const subtitle = [baseDescription || packageDescription, windowDescription, expiresDescription]
    .filter(Boolean)
    .join(' • ');

  return {
    key: `${progressItemIdentifier(raw, index)}-${scope}`,
    title: `${baseName} \u00b7 ${scopeLabel}`,
    subtitle: subtitle || t('user_usageProgress'),
    current,
    total,
    percent,
    unit: 'USD',
    raw
  };
}

function normalizeDurationProgressItem(raw, progress, subscription, group, baseName, baseDescription, index) {
  const startsAt = firstNestedString(raw, [
    ['subscription', 'starts_at'],
    ['starts_at']
  ]);
  const expiresAt = firstNestedString(raw, [
    ['progress', 'expires_at'],
    ['subscription', 'expires_at'],
    ['expires_at']
  ]);

  const startTime = startsAt ? Date.parse(startsAt) : NaN;
  const endTime = expiresAt ? Date.parse(expiresAt) : NaN;
  if (!Number.isFinite(startTime) || !Number.isFinite(endTime) || endTime <= startTime) {
    return null;
  }

  const totalDays = Math.max(1, Math.ceil((endTime - startTime) / 86400000));
  const elapsedDays = Math.min(totalDays, Math.max(0, Math.floor((Date.now() - startTime) / 86400000)));
  const percent = totalDays > 0 ? (elapsedDays / totalDays) * 100 : 0;
  const expiresInDays = pickNumber(progress, ['expires_in_days']);
  const packageDescription = buildPackageDescription(group, progress, subscription);
  const expiryDescription = expiresInDays > 0
    ? t('user_expiresInDays', { count: expiresInDays })
    : t('user_expiresOn', { date: formatShortDate(expiresAt) });

  return {
    key: `${progressItemIdentifier(raw, index)}-duration`,
    title: baseName,
    subtitle: [baseDescription || packageDescription, expiryDescription].filter(Boolean).join(' \u2022 ') || t('user_activePackageDuration'),
    current: elapsedDays,
    total: totalDays,
    percent,
    unit: 'days',
    raw
  };
}

function progressItemIdentifier(item, index) {
  return firstNestedString(item, [
    ['subscription', 'id'],
    ['progress', 'id'],
    ['id'],
    ['subscription_id'],
    ['code'],
    ['name']
  ]) || `progress-${index}`;
}

function progressItemKey(item, index) {
  return item?.key || `progress-${index}`;
}

function progressItemTitle(item, index) {
  return item?.title || t('user_usageItem', { index: index + 1 });
}

function progressItemSubtitle(item) {
  return item?.subtitle || t('user_usageProgress');
}

function progressItemCurrentLabel(item) {
  if (!item) {
    return t('user_currentEmpty');
  }
  return t('user_currentLabel', { value: formatProgressValue(item.current, item.unit) });
}

function progressItemTotalLabel(item) {
  if (!item || !Number.isFinite(item.total) || item.total <= 0) {
    return t('user_totalEmpty');
  }
  return t('user_totalLabel', { value: formatProgressValue(item.total, item.unit) });
}

function clampPercent(percent) {
  const value = Number(percent || 0);
  if (value < 0) {
    return 0;
  }
  if (value > 100) {
    return 100;
  }
  return value;
}

function formatPercent(percent) {
  return `${clampPercent(percent).toFixed(0)}%`;
}

function formatProgressValue(value, unit = '') {
  const numericValue = Number(value || 0);
  if (!Number.isFinite(numericValue)) {
    return '-';
  }
  if (unit) {
    return `${numericValue} ${unit}`;
  }
  return String(numericValue);
}

function formatCurrency(value) {
  const amount = Number(value || 0);
  if (!Number.isFinite(amount)) {
    return '$0.00';
  }
  return `$${amount.toFixed(2)}`;
}

function formatRedeemSuccessMessage(data) {
  if (data && typeof data === 'object') {
    const code = pickString(data, ['code', 'redeem_code']);
    const detail = pickString(data, ['message', 'status', 'result']);
    if (code && detail) {
      return t('user_redeemSuccessFor', { code, detail });
    }
    if (detail) {
      return t('user_redeemSuccessDetail', { detail });
    }
  }
  return t('user_redeemSuccess');
}

function findUnauthenticatedMessage(envelopes) {
  const unauthenticatedEnvelope = envelopes.find((envelope) => envelope?.status === 'unauthenticated');
  return unauthenticatedEnvelope?.message || '';
}

function clearFeedback(target) {
  target.kind = '';
  target.text = '';
}

function feedbackClass(kind) {
  return kind === 'success'
    ? 'text-xs text-emerald-600 dark:text-emerald-400'
    : 'text-xs text-rose-500';
}

function resetUserCenterState() {
  hasLoaded.value = false;
  usernameDraft.value = '';
  redeemCode.value = '';
  clearFeedback(profileFeedback);
  clearFeedback(redeemFeedback);
  applyProfile({});
  applyUsageSummary({});
  progressItems.value = [];
}

function pickNumber(source, keys) {
  for (const key of keys) {
    const value = source?.[key];
    const numeric = Number(value);
    if (Number.isFinite(numeric)) {
      return numeric;
    }
  }
  return 0;
}

function pickOptionalNumber(source, keys) {
  for (const key of keys) {
    if (!Object.prototype.hasOwnProperty.call(source, key)) {
      continue;
    }

    const numeric = Number(source[key]);
    if (Number.isFinite(numeric)) {
      return numeric;
    }
  }

  return null;
}

function pickString(source, keys) {
  for (const key of keys) {
    const value = source?.[key];
    if (typeof value === 'string' && value.trim()) {
      return value.trim();
    }
  }
  return '';
}

function firstNestedString(source, paths) {
  for (const path of paths) {
    const value = getNestedValue(source, path);
    if (typeof value === 'string' && value.trim()) {
      return value.trim();
    }
    if (typeof value === 'number' && Number.isFinite(value)) {
      return String(value);
    }
  }
  return '';
}

function getNestedValue(source, path) {
  let current = source;
  for (const key of path) {
    if (!current || typeof current !== 'object') {
      return undefined;
    }
    current = current[key];
  }
  return current;
}

function asObject(value) {
  return value && typeof value === 'object' ? value : {};
}

function capitalize(value) {
  if (!value) {
    return '';
  }
  return `${value.charAt(0).toUpperCase()}${value.slice(1)}`;
}

function formatShortDate(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

function buildPackageDescription(group, progress, subscription) {
  const parts = [
    firstNestedString({ group }, [['group', 'platform']]),
    firstNestedString({ group }, [['group', 'subscription_type']]),
    firstNestedString({ progress }, [['progress', 'status']]),
    firstNestedString({ subscription }, [['subscription', 'status']])
  ]
    .filter(Boolean)
    .map((part) => capitalize(part));

  return parts.join(' • ');
}

watch(isAuthenticated, async (authenticated) => {
  if (authenticated) {
    await refreshAll();
    return;
  }

  loadError.value = '';
  resetUserCenterState();
});

onMounted(async () => {
  if (isAuthenticated.value) {
    await refreshAll();
  }
});
</script>
