import { apiUrl } from '../config'
import type { ApiResponse } from '../types/api'

export class ApiError extends Error {
  readonly status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

interface ApiFetchOptions extends Omit<RequestInit, 'body'> {
  body?: unknown
}

export async function apiFetch<T>(path: string, options: ApiFetchOptions = {}): Promise<T> {
  const { body, ...requestOptions } = options
  const headers = new Headers(options.headers)
  const init: RequestInit = { ...requestOptions, headers }

  if (body !== undefined) {
    headers.set('Content-Type', 'application/json')
    init.body = JSON.stringify(body)
  }

  const response = await fetch(apiUrl(path), init)
  const payload = (await response.json().catch(() => null)) as ApiResponse<T> | null

  if (!response.ok || !payload?.success) {
    throw new ApiError(payload?.error || response.statusText || 'Request failed', response.status)
  }

  return payload.data as T
}

export async function apiFetchResponse<T>(path: string, options: ApiFetchOptions = {}): Promise<ApiResponse<T>> {
  const { body, ...requestOptions } = options
  const headers = new Headers(options.headers)
  const init: RequestInit = { ...requestOptions, headers }

  if (body !== undefined) {
    headers.set('Content-Type', 'application/json')
    init.body = JSON.stringify(body)
  }

  const response = await fetch(apiUrl(path), init)
  const payload = (await response.json().catch(() => null)) as ApiResponse<T> | null

  if (!response.ok || !payload?.success) {
    throw new ApiError(payload?.error || response.statusText || 'Request failed', response.status)
  }

  return payload
}
