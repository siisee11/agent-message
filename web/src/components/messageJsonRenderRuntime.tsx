import { createContext, useContext } from 'react'

export interface MessageJsonRenderRuntime {
  interactionDisabled: boolean
  onReplyAction?: (value: string) => Promise<void>
}

export const MessageJsonRenderRuntimeContext = createContext<MessageJsonRenderRuntime>({
  interactionDisabled: false,
})

export function useMessageJsonRenderRuntime(): MessageJsonRenderRuntime {
  return useContext(MessageJsonRenderRuntimeContext)
}
