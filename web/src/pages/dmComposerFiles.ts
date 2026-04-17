export function buildSelectedFileKey(file: File): string {
  return `${file.name}-${file.size}-${file.lastModified}`
}

export function isImageFile(file: File): boolean {
  return file.type.startsWith('image/')
}

export function extractImageFilesFromClipboardData(clipboardData: DataTransfer | null | undefined): File[] {
  if (!clipboardData) {
    return []
  }

  const itemFiles = Array.from(clipboardData.items ?? [])
    .filter((item) => item.kind === 'file' && item.type.startsWith('image/'))
    .map((item) => item.getAsFile())
    .filter((file): file is File => file !== null)

  if (itemFiles.length > 0) {
    return itemFiles
  }

  return Array.from(clipboardData.files ?? []).filter(isImageFile)
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
