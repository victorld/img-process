export function formatParentDirectory(path?: string | null): string | undefined {
  if (!path) {
    return path ?? undefined
  }

  const lastSlashIndex = path.lastIndexOf('/')
  if (lastSlashIndex < 0) {
    return path
  }
  if (lastSlashIndex === 0) {
    return '/'
  }
  return path.slice(0, lastSlashIndex)
}
