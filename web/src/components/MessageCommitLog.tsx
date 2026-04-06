import type { BaseComponentProps } from '@json-render/react'
import styles from './MessageJsonRender.module.css'

interface CommitStats {
  deletions?: number
  filesChanged?: number
  insertions?: number
}

interface GitCommitLogEntry {
  authorName?: string
  authoredAt?: string
  body?: string
  isHead?: boolean
  isMerge?: boolean
  refs?: string[]
  sha?: string
  stats?: CommitStats
  subject?: string
}

interface GitCommitLogProps {
  branch?: string
  commits?: GitCommitLogEntry[]
  description?: string
  repository?: string
  title?: string
}

function resolveText(value: unknown): string | null {
  if (typeof value !== 'string') {
    return null
  }

  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

function resolveCountLabel(count: number, singular: string, plural = `${singular}s`): string {
  return `${count} ${count === 1 ? singular : plural}`
}

function normalizeStats(value: unknown): CommitStats | null {
  if (typeof value !== 'object' || value === null) {
    return null
  }

  const candidate = value as Record<string, unknown>
  const filesChanged =
    typeof candidate.filesChanged === 'number' && Number.isFinite(candidate.filesChanged) && candidate.filesChanged >= 0
      ? Math.round(candidate.filesChanged)
      : null
  const insertions =
    typeof candidate.insertions === 'number' && Number.isFinite(candidate.insertions) && candidate.insertions >= 0
      ? Math.round(candidate.insertions)
      : null
  const deletions =
    typeof candidate.deletions === 'number' && Number.isFinite(candidate.deletions) && candidate.deletions >= 0
      ? Math.round(candidate.deletions)
      : null

  if (filesChanged === null && insertions === null && deletions === null) {
    return null
  }

  return {
    deletions: deletions ?? undefined,
    filesChanged: filesChanged ?? undefined,
    insertions: insertions ?? undefined,
  }
}

function normalizeEntries(entries: GitCommitLogProps['commits']): Array<Required<Pick<GitCommitLogEntry, 'sha' | 'subject'>> & GitCommitLogEntry> {
  if (!Array.isArray(entries)) {
    return []
  }

  return entries.flatMap((entry) => {
    if (typeof entry !== 'object' || entry === null) {
      return []
    }

    const sha = resolveText(entry.sha)
    const subject = resolveText(entry.subject)
    if (!sha || !subject) {
      return []
    }

    const refs = Array.isArray(entry.refs)
      ? entry.refs
          .map((ref) => resolveText(ref))
          .filter((ref): ref is string => ref !== null)
      : undefined

    return [
      {
        authorName: resolveText(entry.authorName) ?? undefined,
        authoredAt: resolveText(entry.authoredAt) ?? undefined,
        body: resolveText(entry.body) ?? undefined,
        isHead: entry.isHead === true,
        isMerge: entry.isMerge === true,
        refs,
        sha,
        stats: normalizeStats(entry.stats) ?? undefined,
        subject,
      },
    ]
  })
}

function formatShortSha(sha: string): string {
  return sha.slice(0, 7)
}

function renderStats(stats: CommitStats | undefined) {
  if (!stats) {
    return null
  }

  const items = [
    typeof stats.filesChanged === 'number' ? resolveCountLabel(stats.filesChanged, 'file', 'files') + ' changed' : null,
    typeof stats.insertions === 'number' ? `+${stats.insertions} insertions` : null,
    typeof stats.deletions === 'number' ? `-${stats.deletions} deletions` : null,
  ].filter((item): item is string => item !== null)

  if (items.length === 0) {
    return null
  }

  return (
    <div className={styles.commitLogStats}>
      {items.map((item) => (
        <span className={styles.commitLogStat} key={item}>
          {item}
        </span>
      ))}
    </div>
  )
}

function renderRefs(refs: string[] | undefined, isHead: boolean | undefined) {
  const labels = refs ?? []
  if (labels.length === 0 && !isHead) {
    return null
  }

  return (
    <div className={styles.commitLogRefs}>
      {isHead ? <span className={`${styles.commitLogRef} ${styles.commitLogRefHead}`}>HEAD</span> : null}
      {labels.map((label) => (
        <span className={styles.commitLogRef} key={label}>
          {label}
        </span>
      ))}
    </div>
  )
}

export function MessageCommitLog({ props }: BaseComponentProps<GitCommitLogProps>) {
  const title = resolveText(props.title) ?? 'Git commit log'
  const description = resolveText(props.description)
  const repository = resolveText(props.repository)
  const branch = resolveText(props.branch)
  const commits = normalizeEntries(props.commits)

  return (
    <section aria-label={title} className={styles.commitLogCard}>
      <header className={styles.commitLogHeader}>
        <div className={styles.commitLogTitleBlock}>
          <p className={styles.commitLogTitle}>{title}</p>
          {description ? <p className={styles.commitLogDescription}>{description}</p> : null}
        </div>
        <div className={styles.commitLogSummary}>
          {repository ? <span className={styles.commitLogSummaryPill}>{repository}</span> : null}
          {branch ? <span className={styles.commitLogSummaryPill}>{branch}</span> : null}
          {commits.length > 0 ? (
            <span className={styles.commitLogSummaryPill}>{resolveCountLabel(commits.length, 'commit')}</span>
          ) : null}
        </div>
      </header>
      {commits.length === 0 ? (
        <p className={styles.commitLogEmpty}>No commits were provided.</p>
      ) : (
        <ol className={styles.commitLogList}>
          {commits.map((commit) => (
            <li className={styles.commitLogEntry} key={commit.sha}>
              <div className={styles.commitLogRail} aria-hidden="true">
                <span className={`${styles.commitLogNode} ${commit.isHead ? styles.commitLogNodeHead : ''}`} />
                <span className={styles.commitLogLine} />
              </div>
              <article className={styles.commitLogContent}>
                <div className={styles.commitLogSubjectRow}>
                  <p className={styles.commitLogSubject}>{commit.subject}</p>
                  {renderRefs(commit.refs, commit.isHead)}
                </div>
                <div className={styles.commitLogMeta}>
                  <code className={styles.commitLogSha}>{formatShortSha(commit.sha)}</code>
                  {commit.authorName ? <span className={styles.commitLogMetaText}>{commit.authorName}</span> : null}
                  {commit.authoredAt ? (
                    <time className={styles.commitLogMetaText} dateTime={commit.authoredAt}>
                      {commit.authoredAt}
                    </time>
                  ) : null}
                  {commit.isMerge ? <span className={styles.commitLogMetaTag}>merge</span> : null}
                </div>
                {commit.body ? <p className={styles.commitLogBody}>{commit.body}</p> : null}
                {renderStats(commit.stats)}
              </article>
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}
