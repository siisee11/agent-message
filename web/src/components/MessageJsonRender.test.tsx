import ReactDOMServer from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { MessageJsonRender } from './MessageJsonRender'

describe('MessageJsonRender', () => {
  it('renders minimal json-render specs even when props are omitted on container elements', () => {
    const html = ReactDOMServer.renderToStaticMarkup(
      <MessageJsonRender
        spec={{
          root: 'stack-1',
          elements: {
            'stack-1': {
              type: 'Stack',
              children: ['badge-1', 'text-1'],
            },
            'badge-1': {
              type: 'Badge',
              props: { text: 'Agent' },
            },
            'text-1': {
              type: 'Text',
              props: { text: 'Hello from CLI' },
            },
          },
        }}
      />,
    )

    expect(html).toContain('Agent')
    expect(html).toContain('Hello from CLI')
  })

  it('renders approval cards with action buttons', () => {
    const html = ReactDOMServer.renderToStaticMarkup(
      <MessageJsonRender
        spec={{
          root: 'approval-1',
          elements: {
            'approval-1': {
              type: 'ApprovalCard',
              props: {
                badge: 'Approval Needed',
                title: 'Command approval requested',
                details: ['Command: npm test', 'CWD: /repo'],
                replyHint: 'approve | session | deny | cancel',
                actions: [
                  { label: 'Approve', value: 'approve', variant: 'primary' },
                  { label: 'Deny', value: 'deny', variant: 'destructive' },
                ],
              },
            },
          },
        }}
      />,
    )

    expect(html).toContain('Command approval requested')
    expect(html).toContain('Approve')
    expect(html).toContain('Deny')
    expect(html).toContain('Fallback reply: approve | session | deny | cancel')
  })

  it('renders ask-question cards with options and a freeform response area', () => {
    const html = ReactDOMServer.renderToStaticMarkup(
      <MessageJsonRender
        spec={{
          root: 'question-1',
          elements: {
            'question-1': {
              type: 'AskQuestion',
              props: {
                question: 'Which environment should I use?',
                options: [
                  { label: 'Production', value: 'production' },
                  { label: 'Staging', value: 'staging' },
                ],
                freeformPlaceholder: 'Type a custom environment',
                confirmLabel: 'Send answer',
              },
            },
          },
        }}
      />,
    )

    expect(html).toContain('Which environment should I use?')
    expect(html).toContain('Production')
    expect(html).toContain('Staging')
    expect(html).toContain('Type a custom environment')
    expect(html).toContain('Send answer')
    expect(html).toContain('<textarea')
  })

  it('renders alerts with title and message', () => {
    const html = ReactDOMServer.renderToStaticMarkup(
      <MessageJsonRender
        spec={{
          root: 'alert-1',
          elements: {
            'alert-1': {
              type: 'Alert',
              props: {
                title: 'Heads up',
                message: 'This wrapper normalizes DM workflow before the final send.',
                type: 'info',
              },
            },
          },
        }}
      />,
    )

    expect(html).toContain('Heads up')
    expect(html).toContain('This wrapper normalizes DM workflow before the final send.')
  })

  it('renders markdown content through the markdown component', () => {
    const html = ReactDOMServer.renderToStaticMarkup(
      <MessageJsonRender
        spec={{
          root: 'markdown-1',
          elements: {
            'markdown-1': {
              type: 'Markdown',
              props: {
                content: '# Release Notes\n\n- Added `Markdown`\n- Supports [links](https://example.com)',
              },
            },
          },
        }}
      />,
    )

    expect(html).toContain('<h1>Release Notes</h1>')
    expect(html).toContain('<li>Added <code>Markdown</code></li>')
    expect(html).toContain('<a href="https://example.com">links</a>')
  })

  it('renders json tables inside a dedicated scroll container', () => {
    const html = ReactDOMServer.renderToStaticMarkup(
      <MessageJsonRender
        spec={{
          root: 'table-1',
          elements: {
            'table-1': {
              type: 'Table',
              props: {
                columns: ['Name', 'Value'],
                rows: [
                  ['Alpha', '123'],
                  ['Beta', '456'],
                ],
              },
            },
          },
        }}
      />,
    )

    expect(html).toContain('data-slot="table-container"')
    expect(html).toContain('data-slot="table"')
    expect(html).toContain('>Name</th>')
    expect(html).toContain('>456</td>')
  })

  it('renders custom bar graphs', () => {
    const html = ReactDOMServer.renderToStaticMarkup(
      <MessageJsonRender
        spec={{
          root: 'bar-graph-1',
          elements: {
            'bar-graph-1': {
              type: 'BarGraph',
              props: {
                title: 'Weekly volume',
                format: 'number',
                data: [
                  { label: 'Mon', value: 12 },
                  { label: 'Tue', value: 18 },
                  { label: 'Wed', value: 9 },
                ],
              },
            },
          },
        }}
      />,
    )

    expect(html).toContain('Weekly volume')
    expect(html).toContain('Mon')
    expect(html).toContain('Tue')
    expect(html).toContain('18')
  })

  it('renders custom line graphs', () => {
    const html = ReactDOMServer.renderToStaticMarkup(
      <MessageJsonRender
        spec={{
          root: 'line-graph-1',
          elements: {
            'line-graph-1': {
              type: 'LineGraph',
              props: {
                title: 'MRR trend',
                format: 'currency',
                currency: 'USD',
                data: [
                  { label: 'Jan', value: 1200 },
                  { label: 'Feb', value: 1550 },
                  { label: 'Mar', value: 1675 },
                ],
              },
            },
          },
        }}
      />,
    )

    expect(html).toContain('MRR trend')
    expect(html).toContain('Jan')
    expect(html).toContain('Mar')
    expect(html).toContain('$1,675')
    expect(html).toContain('<svg')
  })

  it('renders git commit logs as a visual timeline', () => {
    const html = ReactDOMServer.renderToStaticMarkup(
      <MessageJsonRender
        spec={{
          root: 'commit-log-1',
          elements: {
            'commit-log-1': {
              type: 'GitCommitLog',
              props: {
                title: 'Recent history',
                repository: 'agent-message',
                branch: 'main',
                commits: [
                  {
                    sha: '5e7f6b8f7c2b9a12f6b0c10b46c2cd884973a001',
                    subject: 'Add commit log renderer',
                    body: 'Introduces a dedicated timeline for git history payloads.',
                    authorName: 'jay',
                    authoredAt: '2026-04-06T09:12:00Z',
                    isHead: true,
                    refs: ['origin/main', 'tag:v0.4.1'],
                    stats: {
                      filesChanged: 3,
                      insertions: 124,
                      deletions: 9,
                    },
                  },
                  {
                    sha: '4d6c8e1bb3b9a11ce80c9e5e2f08e10ab6381234',
                    subject: 'Tighten json-render preview extraction',
                    authorName: 'jay',
                    authoredAt: '2026-04-05T18:40:00Z',
                    isMerge: true,
                  },
                ],
              },
            },
          },
        }}
      />,
    )

    expect(html).toContain('Recent history')
    expect(html).toContain('Add commit log renderer')
    expect(html).toContain('HEAD')
    expect(html).toContain('origin/main')
    expect(html).toContain('3 files changed')
    expect(html).toContain('+124 insertions')
    expect(html).toContain('merge')
  })
})
