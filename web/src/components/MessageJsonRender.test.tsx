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
})
