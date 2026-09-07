const BASE = '/api/chores'
const AUTH_BASE = '/api/auth'

async function handle(res) {
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    const err = new Error(body.error || `request failed: ${res.status}`)
    err.status = res.status
    throw err
  }
  if (res.status === 204) return null
  return res.json()
}

export function listChores() {
  return fetch(BASE).then(handle)
}

export function listCompletedChores() {
  return fetch(`${BASE}/completed`).then(handle)
}

export function createChore({ name, description, intervalHours, recurring }) {
  return fetch(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, interval_hours: intervalHours, recurring }),
  }).then(handle)
}

export function updateChore(id, { name, description, intervalHours, recurring }) {
  return fetch(`${BASE}/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, interval_hours: intervalHours, recurring }),
  }).then(handle)
}

export function completeChore(id) {
  return fetch(`${BASE}/${id}/complete`, { method: 'POST' }).then(handle)
}

export function undoComplete(id) {
  return fetch(`${BASE}/${id}/undo`, { method: 'POST' }).then(handle)
}

export function deleteChore(id) {
  return fetch(`${BASE}/${id}`, { method: 'DELETE' }).then(handle)
}

export function authStatus() {
  return fetch(`${AUTH_BASE}/status`).then(handle)
}

export function login(password) {
  return fetch(`${AUTH_BASE}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  }).then(handle)
}

export function logout() {
  return fetch(`${AUTH_BASE}/logout`, { method: 'POST' }).then(handle)
}
