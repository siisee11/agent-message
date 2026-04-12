import type {
  AuthResponse,
  ConversationDetails,
  ConversationSummary,
  EditMessageRequest,
  DeletePushSubscriptionRequest,
  ListConversationsQuery,
  ListMessagesQuery,
  LoginRequest,
  Message,
  MessageDetails,
  PushConfigResponse,
  Reaction,
  RegisterRequest,
  SavePushSubscriptionRequest,
  SearchUsersQuery,
  SendMessageInput,
  StartConversationRequest,
  ToggleReactionRequest,
  ToggleReactionResult,
  UploadResponse,
  UserProfile,
} from './types'
import {
  normalizeConversationSummaryProtocol,
  normalizeMessageDetailsProtocol,
  normalizeMessageProtocol,
} from './messageProtocol'

interface ApiErrorBody {
  error?: string
}

export class ApiError extends Error {
  readonly status: number
  readonly path: string

  constructor(message: string, status: number, path: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.path = path
  }
}

type TokenProvider = () => string | null | undefined
type UnauthorizedHandler = () => void

export interface ApiClientOptions {
  baseUrl?: string
  getToken?: TokenProvider
  onUnauthorized?: UnauthorizedHandler
}

interface RequestOptions {
  method: 'GET' | 'POST' | 'PATCH' | 'DELETE'
  path: string
  auth?: boolean
  query?: Record<string, string | number | undefined>
  headers?: Record<string, string>
  body?: BodyInit
}

const DEFAULT_API_BASE_URL = import.meta.env.VITE_API_BASE_URL?.trim() ?? ''

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '')
}

function joinPath(baseUrl: string, path: string): string {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  if (baseUrl === '') {
    return normalizedPath
  }
  return `${baseUrl}${normalizedPath}`
}

function withQuery(path: string, query?: Record<string, string | number | undefined>): string {
  if (!query) {
    return path
  }

  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined) {
      continue
    }
    params.set(key, String(value))
  }

  const queryString = params.toString()
  if (queryString === '') {
    return path
  }
  return `${path}?${queryString}`
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isSendMessageFileInput(payload: SendMessageInput): payload is { content?: string; attachment: File } {
  return 'attachment' in payload
}

function isSendMessageFilesInput(payload: SendMessageInput): payload is { content?: string; attachments: File[] } {
  return 'attachments' in payload
}

function isSendMessageAttachmentURLInput(
  payload: SendMessageInput,
): payload is { content?: string; attachmentUrl: string; attachmentType?: 'image' | 'file' } {
  return 'attachmentUrl' in payload
}

export class ApiClient {
  private readonly baseUrl: string

  private readonly getToken?: TokenProvider

  private readonly onUnauthorized?: UnauthorizedHandler

  private token: string | null = null

  constructor(options: ApiClientOptions = {}) {
    const envOrDefaultBaseUrl = options.baseUrl ?? DEFAULT_API_BASE_URL
    this.baseUrl = trimTrailingSlash(envOrDefaultBaseUrl)
    this.getToken = options.getToken
    this.onUnauthorized = options.onUnauthorized
  }

  setAuthToken(token: string | null): void {
    this.token = token
  }

  getAuthToken(): string | null {
    if (this.getToken) {
      return this.getToken() ?? this.token
    }
    return this.token
  }

  async register(input: RegisterRequest): Promise<AuthResponse> {
    return this.requestJSON<AuthResponse>({
      method: 'POST',
      path: '/api/auth/register',
      auth: false,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
  }

  async login(input: LoginRequest): Promise<AuthResponse> {
    return this.requestJSON<AuthResponse>({
      method: 'POST',
      path: '/api/auth/login',
      auth: false,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
  }

  async logout(): Promise<void> {
    await this.requestVoid({
      method: 'DELETE',
      path: '/api/auth/logout',
    })
  }

  async getMe(): Promise<UserProfile> {
    return this.requestJSON<UserProfile>({
      method: 'GET',
      path: '/api/users/me',
    })
  }

  async searchUsers(query: SearchUsersQuery): Promise<UserProfile[]> {
    return this.requestJSON<UserProfile[]>({
      method: 'GET',
      path: '/api/users',
      query: {
        username: query.username,
        limit: query.limit,
      },
    })
  }

  async listConversations(query: ListConversationsQuery = {}): Promise<ConversationSummary[]> {
    const summaries = await this.requestJSON<ConversationSummary[]>({
      method: 'GET',
      path: '/api/conversations',
      query: {
        limit: query.limit,
      },
    })
    return summaries.map(normalizeConversationSummaryProtocol)
  }

  async startConversation(input: StartConversationRequest): Promise<ConversationDetails> {
    return this.requestJSON<ConversationDetails>({
      method: 'POST',
      path: '/api/conversations',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
  }

  async getConversation(conversationId: string): Promise<ConversationDetails> {
    return this.requestJSON<ConversationDetails>({
      method: 'GET',
      path: `/api/conversations/${encodeURIComponent(conversationId)}`,
    })
  }

  async deleteConversation(conversationId: string): Promise<void> {
    await this.requestVoid({
      method: 'DELETE',
      path: `/api/conversations/${encodeURIComponent(conversationId)}`,
    })
  }

  async listMessages(conversationId: string, query: ListMessagesQuery = {}): Promise<MessageDetails[]> {
    const detailsList = await this.requestJSON<MessageDetails[]>({
      method: 'GET',
      path: `/api/conversations/${encodeURIComponent(conversationId)}/messages`,
      query: {
        before: query.before,
        limit: query.limit,
      },
    })
    return detailsList.map(normalizeMessageDetailsProtocol)
  }

  async sendMessage(conversationId: string, payload: SendMessageInput): Promise<Message> {
    const path = `/api/conversations/${encodeURIComponent(conversationId)}/messages`

    if (isSendMessageFilesInput(payload)) {
      const formData = new FormData()
      if (payload.content && payload.content.trim() !== '') {
        formData.set('content', payload.content)
      }
      for (const attachment of payload.attachments) {
        formData.append('attachment', attachment)
      }
      return this.requestJSON<Message>({
        method: 'POST',
        path,
        body: formData,
      }).then(normalizeMessageProtocol)
    }

    if (isSendMessageFileInput(payload)) {
      const formData = new FormData()
      if (payload.content && payload.content.trim() !== '') {
        formData.set('content', payload.content)
      }
      formData.set('attachment', payload.attachment)
      return this.requestJSON<Message>({
        method: 'POST',
        path,
        body: formData,
      }).then(normalizeMessageProtocol)
    }

    if (isSendMessageAttachmentURLInput(payload)) {
      const formData = new FormData()
      if (payload.content && payload.content.trim() !== '') {
        formData.set('content', payload.content)
      }
      formData.set('attachment_url', payload.attachmentUrl)
      if (payload.attachmentType) {
        formData.set('attachment_type', payload.attachmentType)
      }
      return this.requestJSON<Message>({
        method: 'POST',
        path,
        body: formData,
      }).then(normalizeMessageProtocol)
    }

    return this.requestJSON<Message>({
      method: 'POST',
      path,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: payload.content }),
    }).then(normalizeMessageProtocol)
  }

  async editMessage(messageId: string, input: EditMessageRequest): Promise<Message> {
    return this.requestJSON<Message>({
      method: 'PATCH',
      path: `/api/messages/${encodeURIComponent(messageId)}`,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    }).then(normalizeMessageProtocol)
  }

  async deleteMessage(messageId: string): Promise<Message> {
    return this.requestJSON<Message>({
      method: 'DELETE',
      path: `/api/messages/${encodeURIComponent(messageId)}`,
    }).then(normalizeMessageProtocol)
  }

  async toggleReaction(messageId: string, input: ToggleReactionRequest): Promise<ToggleReactionResult> {
    return this.requestJSON<ToggleReactionResult>({
      method: 'POST',
      path: `/api/messages/${encodeURIComponent(messageId)}/reactions`,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
  }

  async removeReaction(messageId: string, emoji: string): Promise<Reaction> {
    return this.requestJSON<Reaction>({
      method: 'DELETE',
      path: `/api/messages/${encodeURIComponent(messageId)}/reactions/${encodeURIComponent(emoji)}`,
    })
  }

  async uploadFile(file: File, fieldName: 'file' | 'attachment' = 'file'): Promise<UploadResponse> {
    const formData = new FormData()
    formData.set(fieldName, file)
    return this.requestJSON<UploadResponse>({
      method: 'POST',
      path: '/api/upload',
      body: formData,
    })
  }

  async getPushConfig(): Promise<PushConfigResponse> {
    return this.requestJSON<PushConfigResponse>({
      method: 'GET',
      path: '/api/push/config',
    })
  }

  async savePushSubscription(input: SavePushSubscriptionRequest): Promise<void> {
    await this.requestVoid({
      method: 'POST',
      path: '/api/push/subscriptions',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
  }

  async deletePushSubscription(input: DeletePushSubscriptionRequest): Promise<void> {
    await this.requestVoid({
      method: 'DELETE',
      path: '/api/push/subscriptions',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
  }

  private async requestJSON<TResponse>(request: RequestOptions): Promise<TResponse> {
    const response = await this.request(request)
    const parsed = (await response.json()) as TResponse
    return parsed
  }

  private async requestVoid(request: RequestOptions): Promise<void> {
    await this.request(request)
  }

  private async request(request: RequestOptions): Promise<Response> {
    const url = withQuery(joinPath(this.baseUrl, request.path), request.query)
    const headers = new Headers(request.headers ?? {})
    const auth = request.auth ?? true

    if (auth) {
      const token = this.getAuthToken()
      if (token) {
        headers.set('Authorization', `Bearer ${token}`)
      }
    }

    const response = await fetch(url, {
      method: request.method,
      headers,
      body: request.body,
      credentials: 'include',
    })

    if (response.ok) {
      return response
    }

    if (response.status === 401 && this.onUnauthorized) {
      this.onUnauthorized()
    }

    throw await this.toApiError(response, request.path)
  }

  private async toApiError(response: Response, path: string): Promise<ApiError> {
    const contentType = response.headers.get('Content-Type') ?? ''
    let message = `Request failed with status ${response.status}`

    if (contentType.includes('application/json')) {
      try {
        const body = (await response.json()) as ApiErrorBody
        if (isObject(body) && typeof body.error === 'string' && body.error.trim() !== '') {
          message = body.error
        }
      } catch {
        // Ignore parse failures and keep fallback message.
      }
    } else {
      try {
        const bodyText = (await response.text()).trim()
        if (bodyText !== '') {
          message = bodyText
        }
      } catch {
        // Ignore read failures and keep fallback message.
      }
    }

    return new ApiError(message, response.status, path)
  }
}

export function createApiClient(options: ApiClientOptions = {}): ApiClient {
  return new ApiClient(options)
}
