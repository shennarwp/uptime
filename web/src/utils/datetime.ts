const DAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'] as const;
const MONTH_NAMES = [
  'Jan',
  'Feb',
  'Mar',
  'Apr',
  'May',
  'Jun',
  'Jul',
  'Aug',
  'Sep',
  'Oct',
  'Nov',
  'Dec',
] as const;

const pad = (n: number): string => String(n).padStart(2, '0');

// formatDateTime formats a Date in RFC 2822 style (e.g. "Wed, 06 Aug 2026
// 20:00:00") using the user's local timezone, without a UTC offset.
export function formatDateTime(date: Date): string {
  return (
    `${DAY_NAMES[date.getDay()]}, ${pad(date.getDate())} ${MONTH_NAMES[date.getMonth()]} ` +
    `${date.getFullYear()} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
  );
}
