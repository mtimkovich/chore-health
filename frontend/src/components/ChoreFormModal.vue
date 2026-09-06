<script setup>
import { ref } from 'vue'

const props = defineProps({
  chore: { type: Object, default: null },
})
const emit = defineEmits(['submit', 'cancel'])

const isEdit = !!props.chore
const name = ref(props.chore?.name ?? '')
const description = ref(props.chore?.description ?? '')

// Editing an existing chore whose interval is a whole number of days starts
// the form in "days" mode so it round-trips cleanly instead of showing e.g. 72 hours.
const startInDays = !!props.chore && props.chore.interval_hours % 24 === 0 && props.chore.interval_hours > 0
const unit = ref(startInDays ? 'days' : 'hours')
const amount = ref(
  props.chore ? (startInDays ? props.chore.interval_hours / 24 : props.chore.interval_hours) : 24
)
const recurring = ref(props.chore ? props.chore.recurring : false)
const error = ref('')

function setUnit(newUnit) {
  if (unit.value === newUnit) return
  unit.value = newUnit
  amount.value = newUnit === 'days' ? 7 : 24
}

function submit() {
  const intervalHours = unit.value === 'days' ? Number(amount.value) * 24 : Number(amount.value)
  if (!name.value.trim()) {
    error.value = 'Give the chore a name.'
    return
  }
  if (!intervalHours || intervalHours <= 0) {
    error.value = 'Set how much time is allowed before it is due.'
    return
  }
  emit('submit', {
    name: name.value.trim(),
    description: description.value.trim(),
    intervalHours,
    recurring: recurring.value,
  })
}
</script>

<template>
  <div class="modal-backdrop">
    <div class="modal">
      <div class="modal-header">
        <h2>{{ isEdit ? 'Edit Chore' : 'Add Chore' }}</h2>
        <button type="button" class="modal-close" aria-label="Close" @click="emit('cancel')">&#10005;</button>
      </div>

      <div class="field">
        <label>Name</label>
        <input type="text" v-model="name" placeholder="e.g. Clean litter box" />
      </div>

      <div class="field">
        <label>Description <span class="label-optional">(optional)</span></label>
        <textarea
          v-model="description"
          rows="3"
          placeholder="Any notes — e.g. which supplies to use"
        ></textarea>
      </div>

      <div class="unit-toggle">
        <button type="button" :class="{ active: unit === 'hours' }" @click="setUnit('hours')">Hours</button>
        <button type="button" :class="{ active: unit === 'days' }" @click="setUnit('days')">Days</button>
      </div>

      <div class="field">
        <label>{{ unit === 'hours' ? 'Hours until due' : 'Days until due' }}</label>
        <input type="number" min="1" v-model="amount" />
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
