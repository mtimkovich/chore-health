<script setup>
import { ref } from 'vue'

const props = defineProps({
  chore: { type: Object, default: null },
})
const emit = defineEmits(['submit', 'cancel'])

const isEdit = !!props.chore
const name = ref(props.chore?.name ?? '')
const days = ref(props.chore ? Math.floor(props.chore.interval_hours / 24) : 0)
const hours = ref(props.chore ? Math.round(props.chore.interval_hours % 24) : 24)
const recurring = ref(props.chore ? props.chore.recurring : true)
const error = ref('')

function submit() {
  const intervalHours = Number(days.value) * 24 + Number(hours.value)
  if (!name.value.trim()) {
    error.value = 'Give the chore a name.'
    return
  }
  if (!intervalHours || intervalHours <= 0) {
    error.value = 'Set how much time is allowed before it is due.'
    return
  }
  emit('submit', { name: name.value.trim(), intervalHours, recurring: recurring.value })
}
</script>

<template>
  <div class="modal-backdrop" @click.self="emit('cancel')">
    <div class="modal">
      <h2>{{ isEdit ? 'Edit Chore' : 'Add Chore' }}</h2>

      <div class="field">
        <label>Name</label>
        <input type="text" v-model="name" placeholder="e.g. Clean litter box" />
      </div>

      <div class="interval-row">
        <div class="field">
          <label>Days</label>
          <input type="number" min="0" v-model="days" />
        </div>
        <div class="field">
          <label>Hours</label>
          <input type="number" min="0" max="23" v-model="hours" />
        </div>
      </div>

      <div class="checkbox-row">
        <input type="checkbox" id="recurring" v-model="recurring" />
        <label for="recurring">Repeats automatically when marked done</label>
      </div>

      <p v-if="error" class="error-text">{{ error }}</p>

      <div class="modal-actions">
        <button class="btn-secondary" @click="emit('cancel')">Cancel</button>
        <button class="btn-primary" @click="submit">{{ isEdit ? 'Save' : 'Add' }}</button>
      </div>
    </div>
  </div>
</template>
