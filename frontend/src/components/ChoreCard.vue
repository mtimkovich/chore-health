<script setup>
import { computed, ref } from 'vue'
import { formatDuration } from '../duration'

const props = defineProps({
  chore: { type: Object, required: true },
  completed: { type: Boolean, default: false },
})
const emit = defineEmits(['complete', 'undo', 'edit', 'delete'])

const open = ref(false)
const confirmingDelete = ref(false)

const barColor = computed(() => {
  if (props.chore.overdue) return 'red'
  if (props.chore.percent_remaining <= 20) return 'red'
  if (props.chore.percent_remaining <= 50) return 'yellow'
  return 'green'
})

const statusText = computed(() => {
  const label = formatDuration(Math.abs(props.chore.hours_left))
  return props.chore.overdue ? `OVERDUE BY ${label}` : `~ ${label} LEFT`
})

const expiresAtLabel = computed(() => {
  const expiresAt = new Date(Date.now() + props.chore.hours_left * 3_600_000)
  const formatted = expiresAt.toLocaleString(undefined, {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  })
  return props.chore.overdue ? `Was due ${formatted}` : `Due ${formatted}`
})

const completedText = computed(() => {
  const hoursAgo = (Date.now() - new Date(props.chore.last_completed_at).getTime()) / 3_600_000
  if (hoursAgo < 1) return 'COMPLETED JUST NOW'
  return `COMPLETED ${formatDuration(hoursAgo)} AGO`
})

function toggleOpen() {
  open.value = !open.value
  confirmingDelete.value = false
}
</script>

<template>
  <div class="card">
    <div class="card-top" @click="toggleOpen">
      <h2>
        {{ chore.name }}
        <svg
          v-if="chore.recurring"
          class="recurring-icon"
          viewBox="0 0 24 24"
          fill="currentColor"
          aria-label="Repeats automatically"
        >
          <title>Repeats automatically</title>
          <path
            d="M17.65 6.35A7.958 7.958 0 0 0 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"
          />
        </svg>
      </h2>
      <svg class="chevron" :class="{ open }" viewBox="0 0 24 24" fill="none">
        <path d="M6 9l6 6 6-6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    </div>

    <template v-if="!completed">
      <div class="bar-track">
        <div
          class="bar-fill"
          :class="barColor"
          :style="{ width: chore.percent_remaining + '%' }"
        ></div>
      </div>
      <div class="status-line" :class="{ overdue: chore.overdue }" :title="expiresAtLabel">{{ statusText }}</div>
    </template>
    <div v-else class="status-line completed-line">{{ completedText }}</div>

    <p v-if="open && chore.description" class="chore-description">{{ chore.description }}</p>

    <div v-if="open && !confirmingDelete" class="card-actions">
      <button v-if="!completed" class="btn-complete" @click="emit('complete', chore)">Done</button>
      <button v-if="completed && chore.can_undo" class="btn-complete" @click="emit('undo', chore)">Undo</button>
      <button class="btn-delete" @click="confirmingDelete = true">Delete</button>
      <button v-if="!completed || chore.recurring" class="btn-edit" @click="emit('edit', chore)">Edit</button>
    </div>

    <div v-if="open && confirmingDelete" class="confirm-row">
      <p class="confirm-text">Delete "{{ chore.name }}"? This can't be undone.</p>
      <div class="confirm-buttons">
        <button class="btn-secondary" @click="confirmingDelete = false">Cancel</button>
        <button class="btn-confirm-delete" @click="emit('delete', chore)">Delete</button>
      </div>
    </div>
  </div>
</template>
