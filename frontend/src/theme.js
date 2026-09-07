const STORAGE_KEY = 'theme'

export function getTheme() {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'dark' ? 'dark' : 'light'
  } catch {
    return 'light' // e.g. storage disabled/blocked - fall back quietly
  }
}

export function applyTheme(theme) {
  if (theme === 'dark') {
    document.documentElement.setAttribute('data-theme', 'dark')
  } else {
    document.documentElement.removeAttribute('data-theme')
  }
}

export function setTheme(theme) {
  try {
    localStorage.setItem(STORAGE_KEY, theme)
  } catch {
    // Preference just won't persist across reloads; the toggle still works.
  }
  applyTheme(theme)
}
