export function buildSelectedFileKey(file: File): string {
  return `${file.name}-${file.size}-${file.lastModified}`
}

export function mergeSelectedFiles(existing: File[], incoming: File[]): File[] {
  if (incoming.length === 0) {
    return existing
  }

  const merged = [...existing]
  const seen = new Set(existing.map(buildSelectedFileKey))
  for (const file of incoming) {
    const key = buildSelectedFileKey(file)
    if (seen.has(key)) {
      continue
    }
    seen.add(key)
    merged.push(file)
  }
  return merged
}
