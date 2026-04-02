import { createContext, useContext } from 'react'

export interface MessageJsonRenderRuntime {
  approvalDisabled: boolean
  onApprovalAction?: (value: string) => Promise<void>
}

export const MessageJsonRenderRuntimeContext = createContext<MessageJsonRenderRuntime>({
  approvalDisabled: false,
})

export function useMessageJsonRenderRuntime(): MessageJsonRenderRuntime {
  return useContext(MessageJsonRenderRuntimeContext)
}
