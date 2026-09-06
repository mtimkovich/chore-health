<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import ChoreCard from './components/ChoreCard.vue'
import ChoreFormModal from './components/ChoreFormModal.vue'
import * as api from './api'

const tab = ref('active')
const chores = ref([])
const completedChores = ref([])
const loadError = ref('')
const showForm = ref(false)
const editingChore = ref(null)
let refreshTimer

async function refresh() {
  try {
    const [active, completed] = await Promise.all([api.listChores(), api.listCompletedChores()])
    chores.value = active
    completedChores.value = completed
    loadError.value = ''
  } catch (e) {
    loadError.value = e.message
  }
}

function openAdd() {
  editingChore.value = null
  showForm.value = true
}

function openEdit(chore) {
  editingChore.value = chore
  showForm.value = true
}

async function handleSubmit(payload) {
  if (editingChore.value) {
    await api.updateChore(editingChore.value.id, payload)
  } else {
    await api.createChore(payload)
  }
  showForm.value = false
  editingChore.value = null
  await refresh()
}

async function handleComplete(chore) {
  await api.completeChore(chore.id)
  await refresh()
}

async function handleDelete(chore) {
  await api.deleteChore(chore.id)
  await refresh()
}

onMounted(() => {
  refresh()
  // Bars are time-based, so keep them ticking down without a manual refresh.
  refreshTimer = setInterval(refresh, 60_000)
})
onUnmounted(() => clearInterval(refreshTimer))
</script>

<template>
  <div class="header">
    <h1>Chore Health</h1>
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
    <div class="section-label">ACTIVE CHORES</div>

    <button class="add-chore-btn" @click="openAdd">+ Add Chore</button>

    <div v-if="!chores.length && !loadError" class="empty-state">
      No chores yet. Tap + to add one.
    </div>

    <div class="chore-list">
      <ChoreCard
        v-for="chore in chores"
        :key="chore.id"
        :chore="chore"
        @complete="handleComplete"
        @delete="handleDelete"
        @edit="openEdit"
      />
    </div>
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
</template>
