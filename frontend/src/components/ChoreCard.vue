<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  chore: { type: Object, required: true },
})
const emit = defineEmits(['complete', 'edit', 'delete'])

const open = ref(false)

const barColor = computed(() => {
  if (props.chore.overdue) return 'red'
  if (props.chore.percent_remaining <= 20) return 'red'
  if (props.chore.percent_remaining <= 50) return 'yellow'
  return 'green'
})

const statusText = computed(() => {
  const hrs = Math.abs(Math.round(props.chore.hours_left))
  return props.chore.overdue ? `OVERDUE BY ${hrs} HRS` : `~ ${hrs} HRS LEFT`
})
</script>

<template>
  <div class="card">
    <div class="card-top" @click="open = !open">
      <h2>{{ chore.name }}</h2>
      <span class="chevron" :class="{ open }">&#9660;</span>
    </div>

    <div class="bar-track">
      <div
        class="bar-fill"
        :class="barColor"
        :style="{ width: chore.percent_remaining + '%' }"
      ></div>
    </div>
    <div class="status-line" :class="{ overdue: chore.overdue }">{{ statusText }}</div>

    <div v-if="open" class="card-actions">
      <button class="btn-complete" @click="emit('complete', chore)">Done</button>
      <button class="btn-delete" @click="emit('delete', chore)">Delete</button>
      <button class="btn-delete" style="background:#eef0fb;color:#2b4ce0" @click="emit('edit', chore)">Edit</button>
    </div>
  </div>
</template>
