import { defineCatalog } from '@json-render/core'
import { schema } from '@json-render/react/schema'
import { shadcnComponentDefinitions } from '@json-render/shadcn/catalog'
import { z } from 'zod'

const graphDatumSchema = z.object({
  color: z.string().optional(),
  label: z.string(),
  value: z.number(),
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
