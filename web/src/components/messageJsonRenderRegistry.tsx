import { defineRegistry } from '@json-render/react'
import { shadcnComponents } from '@json-render/shadcn'
import { MessageAlert } from './MessageAlert'
import { MessageApprovalCard } from './MessageApprovalCard'
import { MessageAskQuestion } from './MessageAskQuestion'
import { MessageCommitLog } from './MessageCommitLog'
import { MessageGif } from './MessageGif'
import { MessageBarGraph, MessageLineGraph } from './MessageGraph'
import { MessageImage } from './MessageImage'
import { MessageMarkdown } from './MessageMarkdown'
import { MessageTable } from './MessageTable'
import { messageJsonRenderCatalog } from './messageJsonRenderCatalog'

export const { registry: messageJsonRenderRegistry } = defineRegistry(messageJsonRenderCatalog, {
  components: {
    ApprovalCard: MessageApprovalCard,
    Alert: MessageAlert,
    AskQuestion: MessageAskQuestion,
    Avatar: shadcnComponents.Avatar,
    Badge: shadcnComponents.Badge,
    BarGraph: MessageBarGraph,
    Card: shadcnComponents.Card,
    Gif: MessageGif,
    GitCommitLog: MessageCommitLog,
    Grid: shadcnComponents.Grid,
    Heading: shadcnComponents.Heading,
    Image: MessageImage,
    LineGraph: MessageLineGraph,
    Markdown: MessageMarkdown,
    Progress: shadcnComponents.Progress,
    Separator: shadcnComponents.Separator,
    Skeleton: shadcnComponents.Skeleton,
    Spinner: shadcnComponents.Spinner,
    Stack: shadcnComponents.Stack,
    Table: MessageTable,
    Text: shadcnComponents.Text,
  },
})
