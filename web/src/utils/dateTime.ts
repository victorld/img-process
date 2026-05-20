const DISPLAY_TIME_ZONE = 'Asia/Shanghai'
const EMPTY_PLACEHOLDER = '-'
const SECOND_MS = 1000
const MINUTE_MS = 60 * SECOND_MS
const HOUR_MS = 60 * MINUTE_MS
const DAY_MS = 24 * HOUR_MS

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

export function formatDurationBetween(startValue?: string | null, endValue?: string | null): string {
  if (!startValue || !endValue) {
    return EMPTY_PLACEHOLDER
  }

  const start = new Date(startValue)
  const end = new Date(endValue)
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) {
    return EMPTY_PLACEHOLDER
  }

  const durationMs = end.getTime() - start.getTime()
  if (durationMs < 0) {
    return EMPTY_PLACEHOLDER
  }

  return formatDurationMs(durationMs)
}

function formatDurationMs(durationMs: number): string {
  const totalSeconds = Math.floor(durationMs / SECOND_MS)
  if (totalSeconds < 60) {
    return `${totalSeconds}秒`
  }

  const days = Math.floor(durationMs / DAY_MS)
  const hours = Math.floor((durationMs % DAY_MS) / HOUR_MS)
  const minutes = Math.floor((durationMs % HOUR_MS) / MINUTE_MS)
  const seconds = Math.floor((durationMs % MINUTE_MS) / SECOND_MS)
  const parts: string[] = []

  if (days > 0) {
    parts.push(`${days}天`)
  }
  if (hours > 0) {
    parts.push(`${hours}小时`)
  }
  if (minutes > 0) {
    parts.push(`${minutes}分钟`)
  }
  if (seconds > 0 || parts.length === 0) {
    parts.push(`${seconds}秒`)
  }

  return parts.join('')
}
