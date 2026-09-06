<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import ChoreCard from './components/ChoreCard.vue'
import ChoreFormModal from './components/ChoreFormModal.vue'
import * as api from './api'

const chores = ref([])
const loadError = ref('')
const showForm = ref(false)
const editingChore = ref(null)
let refreshTimer

async function refresh() {
  try {
    chores.value = await api.listChores()
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

  <div class="section-label">ACTIVE CHORES</div>

  <p v-if="loadError" class="error-text" style="margin: 0 20px 16px">{{ loadError }}</p>

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

  <button class="fab" @click="openAdd">+</button>

  <ChoreFormModal
    v-if="showForm"
    :chore="editingChore"
    @submit="handleSubmit"
    @cancel="showForm = false"
  />
</template>
