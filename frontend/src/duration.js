// Formats an hour count as the largest whole unit that keeps it readable,
// matching how the Roomba health screen phrases "time left" for a part.
export function formatDuration(hoursAbs) {
  const totalHours = Math.round(hoursAbs)
  if (totalHours < 24) {
    return `${totalHours} HR${totalHours === 1 ? '' : 'S'}`
  }

  const days = hoursAbs / 24
  if (days < 7) {
    const rounded = Math.round(days)
    return `${rounded} DAY${rounded === 1 ? '' : 'S'}`
  }

  const weeks = Math.round(days / 7)
  return `${weeks} WEEK${weeks === 1 ? '' : 'S'}`
}
