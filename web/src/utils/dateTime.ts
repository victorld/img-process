const DISPLAY_TIME_ZONE = 'Asia/Shanghai'
const EMPTY_PLACEHOLDER = '-'

const formatter = new Intl.DateTimeFormat('sv-SE', {
  timeZone: DISPLAY_TIME_ZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
})

export function formatDateTime(value?: string | null): string {
  if (!value) {
    return EMPTY_PLACEHOLDER
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  const parts = formatter.formatToParts(parsed)
  const mapped = Object.fromEntries(parts.filter((part) => part.type !== 'literal').map((part) => [part.type, part.value]))
  return `${mapped.year}-${mapped.month}-${mapped.day} ${mapped.hour}:${mapped.minute}:${mapped.second}`
}
