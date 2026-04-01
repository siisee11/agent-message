import type { EventStreamConnectionStatus } from '../hooks'

export function formatRealtimeStatusLabel(status: EventStreamConnectionStatus): string {
  if (status === 'open') {
    return 'Live'
  }
  if (status === 'connecting') {
    return 'Connecting'
  }
  return 'Offline'
}
