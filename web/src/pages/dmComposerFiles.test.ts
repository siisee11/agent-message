import { describe, expect, it } from 'vitest'
import { buildSelectedFileKey, extractImageFilesFromClipboardData, isImageFile, mergeSelectedFiles } from './dmComposerFiles'

function createFile(name: string, size = 100, lastModified = 1): File {
  return new File([new Uint8Array(size)], name, { lastModified, type: 'image/png' })
}

function createClipboardItem(file: File): DataTransferItem {
  return {
    kind: 'file',
    type: file.type,
    getAsFile: () => file,
    getAsString: () => undefined,
    webkitGetAsEntry: () => null,
  }
}

describe('dm composer file helpers', () => {
  it('builds a stable file key from file metadata', () => {
    expect(buildSelectedFileKey(createFile('diagram.png', 42, 99))).toBe('diagram.png-42-99')
  })

  it('detects image files for preview rendering', () => {
    expect(isImageFile(createFile('preview.png'))).toBe(true)
    expect(isImageFile(new File(['plain'], 'note.txt', { type: 'text/plain' }))).toBe(false)
  })

  it('extracts pasted image files from clipboard items', () => {
    const image = createFile('clipboard.png', 10, 5)
    const nonImage = new File(['plain'], 'note.txt', { type: 'text/plain' })
    const clipboardData = {
      items: [createClipboardItem(image), createClipboardItem(nonImage)],
      files: [],
    } as unknown as DataTransfer

    expect(extractImageFilesFromClipboardData(clipboardData)).toEqual([image])
  })

  it('falls back to clipboard files when items are unavailable', () => {
    const image = createFile('fallback.png', 20, 6)
    const nonImage = new File(['plain'], 'note.txt', { type: 'text/plain' })
    const clipboardData = {
      items: [],
      files: [image, nonImage],
    } as unknown as DataTransfer

    expect(extractImageFilesFromClipboardData(clipboardData)).toEqual([image])
  })

  it('returns an empty list when no pasted images exist', () => {
    const clipboardData = {
      items: [],
      files: [],
    } as unknown as DataTransfer

    expect(extractImageFilesFromClipboardData(clipboardData)).toEqual([])
    expect(extractImageFilesFromClipboardData(null)).toEqual([])
  })

  it('appends newly selected files instead of replacing existing ones', () => {
    const first = createFile('first.png', 100, 1)
    const second = createFile('second.png', 200, 2)

    expect(mergeSelectedFiles([first], [second])).toEqual([first, second])
  })

  it('deduplicates files that were already selected', () => {
    const first = createFile('first.png', 100, 1)
    const duplicate = createFile('first.png', 100, 1)
    const second = createFile('second.png', 200, 2)

    expect(mergeSelectedFiles([first], [duplicate, second])).toEqual([first, second])
  })

  it('returns the current selection when no new files were picked', () => {
    const first = createFile('first.png', 100, 1)

    expect(mergeSelectedFiles([first], [])).toEqual([first])
  })
})
