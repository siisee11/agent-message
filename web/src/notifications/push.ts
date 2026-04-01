import { apiClient } from '../api/runtime'

export interface PushState {
  supported: boolean
  configured: boolean
  enabled: boolean
  permission: NotificationPermission | 'unsupported'
}

export function supportsPushNotifications(): boolean {
  return (
    typeof window !== 'undefined' &&
    'serviceWorker' in navigator &&
    'PushManager' in window &&
    'Notification' in window
  )
}

export async function getPushState(): Promise<PushState> {
  if (!supportsPushNotifications()) {
    return {
      supported: false,
      configured: false,
      enabled: false,
      permission: 'unsupported',
    }
  }

  const config = await apiClient.getPushConfig()
  const registration = await navigator.serviceWorker.ready
  const subscription = await registration.pushManager.getSubscription()

  return {
    supported: true,
    configured: config.enabled,
    enabled: subscription !== null,
    permission: Notification.permission,
  }
}

export async function enablePushNotifications(): Promise<PushState> {
  if (!supportsPushNotifications()) {
    throw new Error('Push notifications are not supported on this device.')
  }

  const config = await apiClient.getPushConfig()
  if (!config.enabled || !config.vapid_public_key) {
    return {
      supported: true,
      configured: false,
      enabled: false,
      permission: Notification.permission,
    }
  }

  const permission = await Notification.requestPermission()
  if (permission !== 'granted') {
    return {
      supported: true,
      configured: true,
      enabled: false,
      permission,
    }
  }

  const registration = await navigator.serviceWorker.ready
  let subscription = await registration.pushManager.getSubscription()
  if (!subscription) {
    subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: decodeVAPIDPublicKey(config.vapid_public_key),
    })
  }

  await apiClient.savePushSubscription(toSubscriptionPayload(subscription))
  return {
    supported: true,
    configured: true,
    enabled: true,
    permission,
  }
}

export async function disablePushNotifications(): Promise<void> {
  if (!supportsPushNotifications()) {
    return
  }

  const registration = await navigator.serviceWorker.ready
  const subscription = await registration.pushManager.getSubscription()
  if (!subscription) {
    return
  }

  try {
    await apiClient.deletePushSubscription({ endpoint: subscription.endpoint })
  } catch {
    // Keep local unsubscribe best-effort even if the server call fails.
  }

  await subscription.unsubscribe()
}

function toSubscriptionPayload(subscription: PushSubscription) {
  const json = subscription.toJSON()
  const p256dh = json.keys?.p256dh?.trim()
  const auth = json.keys?.auth?.trim()
  const endpoint = json.endpoint?.trim()

  if (!endpoint || !p256dh || !auth) {
    throw new Error('Push subscription is missing required keys.')
  }

  return {
    endpoint,
    keys: {
      p256dh,
      auth,
    },
  }
}

function decodeVAPIDPublicKey(value: string): Uint8Array<ArrayBuffer> {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
  const raw = window.atob(padded)
  const out = new Uint8Array(raw.length)
  for (let index = 0; index < raw.length; index += 1) {
    out[index] = raw.charCodeAt(index)
  }
  return out
}
