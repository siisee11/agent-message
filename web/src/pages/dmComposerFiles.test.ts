import { describe, expect, it } from 'vitest'
import { buildSelectedFileKey, mergeSelectedFiles } from './dmComposerFiles'

function createFile(name: string, size = 100, lastModified = 1): File {
  return new File([new Uint8Array(size)], name, { lastModified, type: 'image/png' })
}

describe('dm composer file helpers', () => {
  it('builds a stable file key from file metadata', () => {
    expect(buildSelectedFileKey(createFile('diagram.png', 42, 99))).toBe('diagram.png-42-99')
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
