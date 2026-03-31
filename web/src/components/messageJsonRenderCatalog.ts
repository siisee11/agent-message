import { defineCatalog } from '@json-render/core'
import { schema } from '@json-render/react/schema'
import { shadcnComponentDefinitions } from '@json-render/shadcn/catalog'

export const messageJsonRenderCatalog = defineCatalog(schema, {
  components: {
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
