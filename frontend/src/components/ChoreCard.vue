<script setup>
import { computed, ref } from 'vue';
import { formatDuration } from '../duration';

const props = defineProps({
  chore: { type: Object, required: true },
  completed: { type: Boolean, default: false },
});
const emit = defineEmits(['complete', 'undo', 'edit', 'delete', 'snooze', 'toggle-open']);

const open = ref(false);
const confirmingDelete = ref(false);
const celebrating = ref(false);

const reduceMotion =
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

const barColor = computed(() => {
  if (props.chore.overdue) return 'red';
  if (props.chore.percent_remaining <= 20) return 'red';
  if (props.chore.percent_remaining <= 50) return 'yellow';
  return 'green';
});

const statusText = computed(() => {
  const label = formatDuration(Math.abs(props.chore.hours_left));
  // "~" signals an approximation, which reads oddly stacked in front of the
  // "< 1 HR" label's own approximation marker - so drop it in that case.
  const approx = label.startsWith('<') ? label : `~ ${label}`;
  return props.chore.overdue ? `OVERDUE BY ${label}` : `${approx} LEFT`;
});

const expiresAtLabel = computed(() => {
  const expiresAt = new Date(Date.now() + props.chore.hours_left * 3_600_000);
  const formatted = expiresAt.toLocaleString(undefined, {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
  return props.chore.overdue ? `Was due ${formatted}` : `Due ${formatted}`;
});

const completedText = computed(() => {
  const hoursAgo = (Date.now() - new Date(props.chore.last_completed_at).getTime()) / 3_600_000;
  if (hoursAgo < 1) return 'COMPLETED JUST NOW';
  return `COMPLETED ${formatDuration(hoursAgo)} AGO`;
});

function toggleOpen() {
  open.value = !open.value;
  confirmingDelete.value = false;
  emit('toggle-open', open.value);
}

// Briefly celebrates before actually completing, so marking something done
// feels like a small win instead of the card just vanishing. Skipped
// entirely for anyone who's asked for less motion.
function handleDone() {
  if (celebrating.value) return;
  if (reduceMotion) {
    emit('complete', props.chore);
    return;
  }
  celebrating.value = true;
  setTimeout(() => emit('complete', props.chore), 1000);
}
</script>

<template>
  <div class="card" :class="{ celebrating }">
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

    <div v-if="celebrating" class="celebrate-banner">
      <svg class="celebrate-check" viewBox="0 0 24 24" fill="none">
        <path d="M20 6 9 17l-5-5" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
      <span>Nice work!</span>
    </div>

    <div v-else-if="open && !confirmingDelete" class="card-actions">
      <button v-if="!completed" class="btn-complete" aria-label="Mark done" @click="handleDone">
        <svg viewBox="0 0 24 24" fill="none" width="18" height="18">
          <path d="M20 6 9 17l-5-5" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>
      <button v-if="completed && chore.can_undo" class="btn-complete" aria-label="Undo" @click="emit('undo', chore)">
        <svg viewBox="0 0 24 24" fill="none" width="18" height="18">
          <path d="M9 14 4 9l5-5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
          <path d="M20 20v-7a4 4 0 0 0-4-4H4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>
      <button v-if="!completed || chore.recurring" class="btn-edit" aria-label="Edit" @click="emit('edit', chore)">
        <svg viewBox="0 0 24 24" fill="none" width="18" height="18">
          <path
            d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
      <button class="btn-delete" aria-label="Delete" @click="confirmingDelete = true">
        <svg viewBox="0 0 24 24" fill="none" width="18" height="18">
          <path d="M3 6h18" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
          <path
            d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
          <path d="M10 11v6M14 11v6" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
        </svg>
      </button>
    </div>

    <div v-if="open && !completed && !confirmingDelete && !celebrating" class="snooze-actions">
      <button class="btn-snooze" @click="emit('snooze', chore, 24)">+1 DAY</button>
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
