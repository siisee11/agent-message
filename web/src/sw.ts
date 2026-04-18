/// <reference lib="webworker" />

import { cleanupOutdatedCaches, precacheAndRoute } from 'workbox-precaching'

declare let self: ServiceWorkerGlobalScope & {
  __WB_MANIFEST: Array<{ url: string; revision?: string | null }>
}

interface NotificationPayload {
  title?: string
  body?: string
  tag?: string
  data?: {
    url?: string
    conversationId?: string
    messageId?: string
  }
}

self.skipWaiting()
cleanupOutdatedCaches()
precacheAndRoute(self.__WB_MANIFEST)

self.addEventListener('push', (event) => {
  const payload = readNotificationPayload(event)
  event.waitUntil(handlePushEvent(payload))
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const targetURL = new URL(event.notification.data?.url ?? '/', self.location.origin).toString()
  event.waitUntil(focusOrOpenClient(targetURL))
})

async function handlePushEvent(payload: NotificationPayload): Promise<void> {
  const visibleClients = await self.clients.matchAll({
    type: 'window',
    includeUncontrolled: true,
  })

  const hasVisibleClient = visibleClients.some((client) => client.visibilityState === 'visible')
  if (hasVisibleClient) {
    return
  }

  await self.registration.showNotification(payload.title ?? 'Agent Message', {
    body: payload.body ?? 'New message',
    tag: payload.tag ?? 'chat',
    icon: '/pwa-192x192-0.6.21.png',
    badge: '/pwa-192x192-0.6.21.png',
    data: {
      url: payload.data?.url ?? '/',
      conversationId: payload.data?.conversationId,
      messageId: payload.data?.messageId,
    },
  })
}

function readNotificationPayload(event: PushEvent): NotificationPayload {
  if (!event.data) {
    return {}
  }

  try {
    return event.data.json() as NotificationPayload
  } catch {
    return {
      body: event.data.text(),
    }
  }
}

async function focusOrOpenClient(targetURL: string): Promise<void> {
  const windowClients = await self.clients.matchAll({
    type: 'window',
    includeUncontrolled: true,
  })

  for (const client of windowClients) {
    if (new URL(client.url).pathname === new URL(targetURL).pathname) {
      await client.focus()
      return
    }
  }

  await self.clients.openWindow(targetURL)
}
