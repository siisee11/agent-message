export type ISODateString = string

export type AttachmentType = 'image' | 'file'
export type MessageKind = 'text' | 'json_render'
export type JsonRenderSpec = unknown

export interface UserProfile {
  id: string
  account_id: string
  username: string
  created_at: ISODateString
}

export interface AuthResponse {
  token: string
  user: UserProfile
}

export interface RegisterRequest {
  account_id: string
  username?: string
  password: string
}

export interface LoginRequest {
  account_id: string
  password: string
}

export interface Conversation {
  id: string
  participant_a: string
  participant_b: string
  title?: string
  created_at: ISODateString
}

export interface Message {
  id: string
  conversation_id: string
  sender_id: string
  content?: string
  kind?: MessageKind | null
  json_render_spec?: JsonRenderSpec | null
  attachment_url?: string
  attachment_type?: AttachmentType
  edited: boolean
  deleted: boolean
  created_at: ISODateString
  updated_at: ISODateString
}

export interface Reaction {
  id: string
  message_id: string
  user_id: string
  emoji: string
  created_at: ISODateString
}

export type ReactionMutationAction = 'added' | 'removed'

export interface ToggleReactionResult {
  action: ReactionMutationAction
  reaction: Reaction
}

export interface ConversationSummary {
  conversation: Conversation
  other_user: UserProfile
  last_message?: Message
  session_folder?: string
  session_hostname?: string
}

export type EventStreamClientKind = 'web' | 'watcher' | 'unknown'

export interface WatcherPresence {
  user_id: string
  client_kind: EventStreamClientKind
  online: boolean
}

export interface WatcherPresenceEvent extends WatcherPresence {
  conversation_id: string
}

export interface ConversationDetails {
  conversation: Conversation
  participant_a: UserProfile
  participant_b: UserProfile
  watcher_presence?: WatcherPresence
}

export interface MessageDetails {
  message: Message
  sender: UserProfile
  reactions: Reaction[]
}

export interface UploadResponse {
  url: string
}

export interface WebPushKeys {
  p256dh: string
  auth: string
}

export interface PushConfigResponse {
  enabled: boolean
  vapid_public_key?: string
}

export interface SavePushSubscriptionRequest {
  endpoint: string
  keys: WebPushKeys
}

export interface DeletePushSubscriptionRequest {
  endpoint: string
}

export interface StartConversationRequest {
  username: string
}

export interface EditMessageRequest {
  content: string
}

export interface ToggleReactionRequest {
  emoji: string
}

export interface ListMessagesQuery {
  before?: string
  limit?: number
}

export interface SearchUsersQuery {
  username: string
  limit?: number
}

export interface ListConversationsQuery {
  limit?: number
}

export interface SendMessageTextInput {
  content: string
}

export interface SendMessageFileInput {
  content?: string
  attachment: File
}

export interface SendMessageAttachmentURLInput {
  content?: string
  attachmentUrl: string
  attachmentType?: AttachmentType
}

export type SendMessageInput =
  | SendMessageTextInput
  | SendMessageFileInput
  | SendMessageAttachmentURLInput
