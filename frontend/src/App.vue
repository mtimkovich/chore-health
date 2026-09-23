<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue';
import ChoreCard from './components/ChoreCard.vue';
import ChoreFormModal from './components/ChoreFormModal.vue';
import LoginScreen from './components/LoginScreen.vue';
import * as api from './api';
import { getTheme, setTheme } from './theme';

const authChecked = ref(false);
const passwordSet = ref(false);
const authenticated = ref(true);

const theme = ref(getTheme());
const year = new Date().getFullYear();

function toggleTheme() {
  const next = theme.value === 'dark' ? 'light' : 'dark';
  setTheme(next);
  theme.value = next;
}

const tab = ref('active');
const chores = ref([]);
const completedChores = ref([]);
const loadError = ref('');
const showForm = ref(false);
const editingChore = ref(null);
let refreshTimer;

// Buckets for grouping the active list by how far out a chore is due. SOON
// and THIS WEEK are rolling windows off "now"; THIS MONTH is a calendar-month
// boundary instead of a rolling 30 days - a chore due in 22 days can still
// land in next month on the calendar, which isn't "this month" even though
// it's under 30 days out.
function startOfNextLocalMonth(d) {
  return new Date(d.getFullYear(), d.getMonth() + 1, 1);
}

function bucketFor(chore, now) {
  if (chore.hours_left < 0) return 'OVERDUE';
  if (chore.hours_left < 24) return 'DUE SOON';
  if (chore.hours_left < 24 * 7) return 'DUE THIS WEEK';
  const dueAt = new Date(now.getTime() + chore.hours_left * 3_600_000);
  if (dueAt < startOfNextLocalMonth(now)) return 'DUE THIS MONTH';
  return 'DUE LATER';
}

// chores is already sorted soonest-first by the API, so grouping it is a
// single pass: only start a new group when the bucket actually changes.
const groupedChores = computed(() => {
  const now = new Date();
  const groups = [];
  for (const chore of chores.value) {
    const label = bucketFor(chore, now);
    const current = groups[groups.length - 1];
    if (current && current.label === label) {
      current.chores.push(chore);
    } else {
      groups.push({ label, chores: [chore] });
    }
  }
  return groups;
});

// A short list doesn't need to be told everything in it is "due soon" - only
// show the group headers once there's an actual split to call out.
const showGroupLabels = computed(() => groupedChores.value.length > 1);

async function checkAuth() {
  const status = await api.authStatus();
  passwordSet.value = status.password_set;
  authenticated.value = status.authenticated;
  authChecked.value = true;
}

async function refresh() {
  if (!authenticated.value) return;
  try {
    const [active, completed] = await Promise.all([api.listChores(), api.listCompletedChores()]);
    chores.value = active;
    completedChores.value = completed;
    loadError.value = '';
  } catch (e) {
    if (e.status === 401) {
      // Session expired or the server restarted; re-check rather than
      // assuming - checkAuth is the source of truth for both flags.
      await checkAuth();
      return;
    }
    loadError.value = e.message;
  }
}

function openAdd() {
  editingChore.value = null;
  showForm.value = true;
}

function openEdit(chore) {
  editingChore.value = chore;
  showForm.value = true;
}

// ChoreFormModal makes the create/update API call itself and only emits
// this once that actually succeeded, so there's nothing to catch here.
async function handleSubmit() {
  showForm.value = false;
  editingChore.value = null;
  await refresh();
}

async function handleComplete(chore) {
  try {
    await api.completeChore(chore.id);
    await refresh();
  } catch (e) {
    loadError.value = e.message;
  }
}

async function handleDelete(chore) {
  try {
    await api.deleteChore(chore.id);
    await refresh();
  } catch (e) {
    loadError.value = e.message;
  }
}

async function handleUndo(chore) {
  try {
    await api.undoComplete(chore.id);
    await refresh();
  } catch (e) {
    loadError.value = e.message;
  }
}

async function handleSnooze(chore, hours) {
  try {
    await api.snoozeChore(chore.id, hours);
    await refresh();
  } catch (e) {
    loadError.value = e.message;
  }
}

async function handleLoginSuccess() {
  authenticated.value = true;
  await refresh();
}

async function handleLogout() {
  try {
    await api.logout();
  } catch (e) {
    loadError.value = e.message;
    return;
  }
  authenticated.value = false;
}

onMounted(async () => {
  await checkAuth();
  await refresh();
  // Bars are time-based, so keep them ticking down without a manual refresh.
  refreshTimer = setInterval(refresh, 60_000);
});
onUnmounted(() => clearInterval(refreshTimer));
</script>

<template>
  <template v-if="!authChecked"></template>

  <LoginScreen v-else-if="passwordSet && !authenticated" @success="handleLoginSuccess" />

  <template v-else>
    <div class="header">
      <h1>Chore Health</h1>
      <div class="header-actions">
        <button
          class="icon-btn"
          :aria-label="theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'"
          @click="toggleTheme"
        >
          <svg v-if="theme === 'dark'" viewBox="0 0 24 24" fill="none" width="20" height="20">
            <circle cx="12" cy="12" r="4" stroke="currentColor" stroke-width="2" />
            <path
              d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
            />
          </svg>
          <svg v-else viewBox="0 0 24 24" fill="none" width="20" height="20">
            <path
              d="M20 14.5A8.5 8.5 0 0 1 9.5 4a8.5 8.5 0 1 0 10.5 10.5z"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
        <button v-if="passwordSet" class="icon-btn" aria-label="Log out" @click="handleLogout">
          <svg viewBox="0 0 24 24" fill="none" width="20" height="20">
            <path
              d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4M10 17l5-5-5-5M15 12H3"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
      </div>
    </div>

    <div class="tabs">
      <button :class="{ active: tab === 'active' }" @click="tab = 'active'">Active</button>
      <button :class="{ active: tab === 'completed' }" @click="tab = 'completed'">
        Completed
        <span v-if="completedChores.length" class="tab-badge">{{ completedChores.length }}</span>
      </button>
    </div>

    <p v-if="loadError" class="error-text" style="margin: 0 20px 16px">{{ loadError }}</p>

    <template v-if="tab === 'active'">
      <div class="section-label">ACTIVE CHORES<span v-if="chores.length" class="section-count"> ({{ chores.length }})</span></div>

      <button class="add-chore-btn" @click="openAdd">+ Add Chore</button>

      <div v-if="!chores.length && !loadError" class="empty-state">
        No chores yet. Tap + to add one.
      </div>

      <template v-for="group in groupedChores" :key="group.label">
        <div v-if="showGroupLabels" class="group-label">{{ group.label }}</div>
        <div class="chore-list">
          <ChoreCard
            v-for="chore in group.chores"
            :key="chore.id"
            :chore="chore"
            @complete="handleComplete"
            @delete="handleDelete"
            @edit="openEdit"
            @snooze="handleSnooze"
          />
        </div>
      </template>
    </template>

    <template v-else>
      <div class="section-label">COMPLETED</div>

      <div v-if="!completedChores.length" class="empty-state">Nothing completed recently.</div>

      <div class="chore-list">
        <ChoreCard
          v-for="chore in completedChores"
          :key="chore.id"
          :chore="chore"
          completed
          @undo="handleUndo"
          @delete="handleDelete"
          @edit="openEdit"
        />
      </div>
    </template>

    <ChoreFormModal
      v-if="showForm"
      :chore="editingChore"
      @submit="handleSubmit"
      @cancel="showForm = false"
    />

    <footer class="app-footer">&copy; {{ year }} Max Timkovich</footer>
  </template>
</template>
