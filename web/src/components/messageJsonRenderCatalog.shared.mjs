import { defineCatalog } from '@json-render/core'
import { schema } from '@json-render/react/schema'
import { shadcnComponentDefinitions } from '@json-render/shadcn/catalog'
import { z } from 'zod'

const graphDatumSchema = z.object({
  color: z.string().optional(),
  label: z.string(),
  value: z.number(),
})

const gitCommitStatsSchema = z.object({
  deletions: z.number().int().nonnegative().optional(),
  filesChanged: z.number().int().nonnegative().optional(),
  insertions: z.number().int().nonnegative().optional(),
})

const gitCommitEntrySchema = z.object({
  authorName: z.string().optional(),
  authoredAt: z.string().optional(),
  body: z.string().optional(),
  isHead: z.boolean().optional(),
  isMerge: z.boolean().optional(),
  refs: z.array(z.string()).optional(),
  sha: z.string(),
  stats: gitCommitStatsSchema.optional(),
  subject: z.string(),
})

const askQuestionOptionSchema = z.object({
  label: z.string(),
  value: z.string().optional(),
})

const askQuestionItemSchema = z.object({
  freeformPlaceholder: z.string().optional(),
  id: z.string().optional(),
  options: z.array(askQuestionOptionSchema).optional(),
  question: z.string(),
})

const mediaPropsSchema = z.object({
  alt: z.string(),
  height: z.number().optional(),
  src: z.string().optional(),
  width: z.number().optional(),
})

export const messageJsonRenderCatalog = defineCatalog(schema, {
  components: {
    ApprovalCard: {
      description: 'Approval request card with quick response buttons',
      props: z.object({
        badge: z.string().optional(),
        title: z.string(),
        details: z.array(z.string()).optional(),
        replyHint: z.string().optional(),
        actions: z
          .array(
            z.object({
              label: z.string(),
              value: z.string(),
              variant: z.enum(['primary', 'secondary', 'destructive']).optional(),
            }),
          )
          .optional(),
        }),
    },
    Alert: shadcnComponentDefinitions.Alert,
    AskQuestion: {
      description: 'Question prompt that supports either a single reply or a paginated multi-question flow',
      props: z.object({
        backLabel: z.string().optional(),
        confirmLabel: z.string().optional(),
        freeformPlaceholder: z.string().optional(),
        nextLabel: z.string().optional(),
        options: z.array(askQuestionOptionSchema).optional(),
        question: z.string().optional(),
        questions: z.array(askQuestionItemSchema).optional(),
      }),
    },
    Avatar: shadcnComponentDefinitions.Avatar,
    Badge: shadcnComponentDefinitions.Badge,
    BarGraph: {
      description: 'Compact bar chart for labeled numeric series',
      props: z.object({
        currency: z.string().optional(),
        data: z.array(graphDatumSchema),
        description: z.string().optional(),
        format: z.enum(['currency', 'number', 'percent']).optional(),
        maxValue: z.number().optional(),
        title: z.string().optional(),
      }),
    },
    Card: shadcnComponentDefinitions.Card,
    Gif: {
      description: 'Animated GIF component. Renders a GIF image when src is provided, otherwise a placeholder.',
      props: mediaPropsSchema,
    },
    GitCommitLog: {
      description: 'Visual git commit timeline with refs, authors, and stat badges',
      props: z.object({
        branch: z.string().optional(),
        commits: z.array(gitCommitEntrySchema),
        description: z.string().optional(),
        repository: z.string().optional(),
        title: z.string().optional(),
      }),
    },
    Grid: shadcnComponentDefinitions.Grid,
    Heading: shadcnComponentDefinitions.Heading,
    Image: shadcnComponentDefinitions.Image,
    LineGraph: {
      description: 'Compact line chart for labeled numeric series',
      props: z.object({
        currency: z.string().optional(),
        data: z.array(graphDatumSchema),
        description: z.string().optional(),
        format: z.enum(['currency', 'number', 'percent']).optional(),
        maxValue: z.number().optional(),
        title: z.string().optional(),
      }),
    },
    Markdown: {
      description: 'Markdown content rendered with react-markdown',
      props: z.object({
        content: z.string(),
      }),
    },
    Progress: shadcnComponentDefinitions.Progress,
    Separator: shadcnComponentDefinitions.Separator,
    Skeleton: shadcnComponentDefinitions.Skeleton,
    Spinner: shadcnComponentDefinitions.Spinner,
    Stack: shadcnComponentDefinitions.Stack,
    Table: shadcnComponentDefinitions.Table,
    Text: shadcnComponentDefinitions.Text,
  },
  actions: {},
})
