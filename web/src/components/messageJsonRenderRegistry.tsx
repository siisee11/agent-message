import { defineRegistry } from '@json-render/react'
import { shadcnComponents } from '@json-render/shadcn'
import { MessageAlert } from './MessageAlert'
import { MessageApprovalCard } from './MessageApprovalCard'
import { MessageBarGraph, MessageLineGraph } from './MessageGraph'
import { MessageMarkdown } from './MessageMarkdown'
import { messageJsonRenderCatalog } from './messageJsonRenderCatalog'

export const { registry: messageJsonRenderRegistry } = defineRegistry(messageJsonRenderCatalog, {
  components: {
    ApprovalCard: MessageApprovalCard,
    Alert: MessageAlert,
    Avatar: shadcnComponents.Avatar,
    Badge: shadcnComponents.Badge,
    BarGraph: MessageBarGraph,
    Card: shadcnComponents.Card,
    Grid: shadcnComponents.Grid,
    Heading: shadcnComponents.Heading,
    Image: shadcnComponents.Image,
    LineGraph: MessageLineGraph,
    Markdown: MessageMarkdown,
    Progress: shadcnComponents.Progress,
    Separator: shadcnComponents.Separator,
    Skeleton: shadcnComponents.Skeleton,
    Spinner: shadcnComponents.Spinner,
    Stack: shadcnComponents.Stack,
    Table: shadcnComponents.Table,
    Text: shadcnComponents.Text,
  },
})
