<script setup>
import { ref } from 'vue';
import * as api from '../api';

const emit = defineEmits(['success']);

const password = ref('');
const error = ref('');
const submitting = ref(false);

async function submit() {
  if (!password.value || submitting.value) return;
  error.value = '';
  submitting.value = true;
  try {
    await api.login(password.value);
    emit('success');
  } catch (e) {
    error.value = e.status === 401 ? 'Incorrect password.' : e.message;
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <div class="login-screen">
    <div class="login-card">
      <h1>Chore Health</h1>
      <input
        type="password"
        v-model="password"
        placeholder="Password"
        autofocus
        @keyup.enter="submit"
      />
      <p v-if="error" class="error-text">{{ error }}</p>
      <button class="btn-primary" :disabled="submitting" @click="submit">
        {{ submitting ? 'Checking…' : 'Log In' }}
      </button>
    </div>
  </div>
</template>
