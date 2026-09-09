function duration(count, word) {
  return `${count} ${word}${count === 1 ? '' : 'S'}`;
}

// Formats an hour count as the largest whole unit that keeps it readable,
// matching how the Roomba health screen phrases "time left" for a part.
export function formatDuration(hoursAbs) {
  const totalHours = Math.round(hoursAbs);
  if (totalHours < 24) {
    return duration(totalHours, 'HR');
  }

  const days = hoursAbs / 24;
  if (days < 7) {
    const rounded = Math.round(days);
    return duration(rounded, 'DAY');
  }

  const weeks = Math.round(days / 7);
  return duration(weeks, 'WEEK');
}
