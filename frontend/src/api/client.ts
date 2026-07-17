// Minimal JSON client for the tournament API. On a non-2xx response it throws
// ApiError carrying the server's structured error envelope { code, message, field }.

export interface ApiErrorBody {
  code: string
  message: string
  field?: string
}

// ApiError surfaces the server's error envelope to callers.
export class ApiError extends Error {
  status: number
  code: string
  field?: string

  constructor(status: number, body: ApiErrorBody) {
    super(body?.message || `HTTP ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.code = body?.code ?? 'INTERNAL'
    this.field = body?.field
  }
}

// request performs one JSON call and throws ApiError on failure.
async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const options: RequestInit = { method }
  if (body !== undefined) {
    options.headers = { 'Content-Type': 'application/json' }
    options.body = JSON.stringify(body)
  }
  const response = await fetch(path, options)
  if (response.status === 204) {
    return null as T
  }
  const payload = await response.json()
  if (!response.ok) {
    throw new ApiError(response.status, (payload as { error: ApiErrorBody }).error)
  }
  return payload as T
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body: unknown) => request<T>('POST', path, body),
  patch: <T>(path: string, body: unknown) => request<T>('PATCH', path, body),
  del: (path: string) => request<void>('DELETE', path),
}
