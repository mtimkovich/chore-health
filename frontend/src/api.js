const BASE = '/api/chores'

async function handle(res) {
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || `request failed: ${res.status}`)
  }
  if (res.status === 204) return null
  return res.json()
}

export function listChores() {
  return fetch(BASE).then(handle)
}

export function createChore({ name, intervalHours, recurring }) {
  return fetch(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, interval_hours: intervalHours, recurring }),
  }).then(handle)
}

export function updateChore(id, { name, intervalHours, recurring }) {
  return fetch(`${BASE}/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, interval_hours: intervalHours, recurring }),
  }).then(handle)
}

export function completeChore(id) {
  return fetch(`${BASE}/${id}/complete`, { method: 'POST' }).then(handle)
}

export function deleteChore(id) {
  return fetch(`${BASE}/${id}`, { method: 'DELETE' }).then(handle)
}
