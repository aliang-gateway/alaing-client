import { computed, reactive, readonly, toRefs } from 'vue';
import { login as loginRequest, logout as logoutRequest, restoreSession as restoreSessionRequest } from '../services/authApi';

const state = reactive({
  user: null,
  status: 'idle',
  isAuthenticated: false,
  isReady: false,
  loginPending: false,
  logoutPending: false,
  restorePending: false,
  loginError: '',
  restoreError: '',
  lastActionMessage: ''
});

function normalizeUser(user) {
  if (!user || typeof user !== 'object') {
    return null;
  }

  return {
    id: Number(user.id || 0),
    username: typeof user.username === 'string' ? user.username : '',
    email: typeof user.email === 'string' ? user.email : '',
    role: typeof user.role === 'string' ? user.role : '',
    status: typeof user.status === 'string' ? user.status : '',
    balance: Number(user.balance || 0),
    concurrency: Number(user.concurrency || 0),
    allowedGroups: Array.isArray(user.allowed_groups)
      ? user.allowed_groups.map((value) => Number(value)).filter((value) => Number.isFinite(value))
      : [],
    createdAt: typeof user.created_at === 'string' ? user.created_at : '',
    profileUpdatedAt: typeof user.profile_updated_at === 'string' ? user.profile_updated_at : '',
    updatedAt: typeof user.updated_at === 'string' ? user.updated_at : ''
  };
}

function applyAuthenticatedState(user, message = '') {
  state.user = normalizeUser(user);
  state.isAuthenticated = Boolean(state.user);
  state.status = state.isAuthenticated ? 'authenticated' : 'unauthenticated';
  state.loginError = '';
  state.restoreError = '';
  state.lastActionMessage = message;
}

function applyUnauthenticatedState(message = '', options = {}) {
  state.user = null;
  state.isAuthenticated = false;
  state.status = 'unauthenticated';
  state.loginError = options.preserveLoginError ? state.loginError : '';
  state.restoreError = message;
  state.lastActionMessage = message;
}

export function mergeAuthUser(partialUser, message = '') {
  if (!state.user || !partialUser || typeof partialUser !== 'object') {
    return;
  }

  state.user = {
    ...state.user,
    ...partialUser
  };

  if (message) {
    state.lastActionMessage = message;
  }
}

export async function restoreAuthSession() {
  if (state.restorePending) {
    return state.isAuthenticated;
  }

  state.restorePending = true;
  state.isReady = false;
  state.status = 'restoring';
  state.restoreError = '';
  state.lastActionMessage = '';

  try {
    const result = await restoreSessionRequest();
    if (result.status === 'success' && result.data) {
      applyAuthenticatedState(result.data, result.message || 'Session restored.');
      return true;
    }

    applyUnauthenticatedState(result.message || 'Please log in to continue.');
    return false;
  } catch (error) {
    applyUnauthenticatedState(error instanceof Error ? error.message : 'Please log in to continue.');
    return false;
  } finally {
    state.restorePending = false;
    state.isReady = true;
  }
}

export async function loginWithPassword(credentials) {
  state.loginPending = true;
  state.loginError = '';
  state.lastActionMessage = '';

  try {
    const result = await loginRequest(credentials);
    if (result.status !== 'success' || !result.data) {
      throw new Error(result.message || 'Login failed.');
    }

    applyAuthenticatedState(result.data, result.message || 'Login successful.');
    state.isReady = true;
    return true;
  } catch (error) {
    state.user = null;
    state.isAuthenticated = false;
    state.status = 'unauthenticated';
    state.loginError = error instanceof Error ? error.message : 'Login failed.';
    state.lastActionMessage = '';
    state.isReady = true;
    return false;
  } finally {
    state.loginPending = false;
  }
}

export async function logoutUser() {
  if (state.logoutPending) {
    return;
  }

  state.logoutPending = true;

  try {
    const result = await logoutRequest();
    applyUnauthenticatedState(result.message || 'Logged out successfully.');
  } catch (error) {
    applyUnauthenticatedState(error instanceof Error ? error.message : 'Logged out locally.');
  } finally {
    state.logoutPending = false;
    state.isReady = true;
  }
}

export function useAuthStore() {
  const userDisplayName = computed(() => {
    if (state.user?.username) {
      return state.user.username;
    }
    return 'Guest';
  });

  const planLabel = computed(() => {
    if (state.user?.status) {
      return state.user.status;
    }
    if (state.user?.email) {
      return state.user.email;
    }
    return 'Login required';
  });

  const authNotice = computed(() => {
    if (state.isAuthenticated) {
      return state.lastActionMessage || 'Authenticated session is active.';
    }
    if (state.restorePending) {
      return 'Restoring your saved session...';
    }
    return state.loginError || state.restoreError || 'Log in to unlock proxy controls, quick chat, quick setup, and settings changes.';
  });

  return {
    ...toRefs(readonly(state)),
    userDisplayName,
    planLabel,
    authNotice,
    restoreAuthSession,
    loginWithPassword,
    logoutUser,
    mergeAuthUser
  };
}
