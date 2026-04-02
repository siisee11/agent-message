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
})
