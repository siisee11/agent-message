import { defineCatalog } from '@json-render/core'
import { schema } from '@json-render/react/schema'
import { shadcnComponentDefinitions } from '@json-render/shadcn/catalog'
import { z } from 'zod'

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
    Card: shadcnComponentDefinitions.Card,
    Grid: shadcnComponentDefinitions.Grid,
    Heading: shadcnComponentDefinitions.Heading,
    Image: shadcnComponentDefinitions.Image,
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
