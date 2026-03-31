import { defineRegistry } from '@json-render/react'
import { shadcnComponents } from '@json-render/shadcn'
import { messageJsonRenderCatalog } from './messageJsonRenderCatalog'

export const { registry: messageJsonRenderRegistry } = defineRegistry(messageJsonRenderCatalog, {
  components: {
    Alert: shadcnComponents.Alert,
    Avatar: shadcnComponents.Avatar,
    Badge: shadcnComponents.Badge,
    Card: shadcnComponents.Card,
    Grid: shadcnComponents.Grid,
    Heading: shadcnComponents.Heading,
    Image: shadcnComponents.Image,
    Progress: shadcnComponents.Progress,
    Separator: shadcnComponents.Separator,
    Skeleton: shadcnComponents.Skeleton,
    Spinner: shadcnComponents.Spinner,
    Stack: shadcnComponents.Stack,
    Table: shadcnComponents.Table,
    Text: shadcnComponents.Text,
  },
})
