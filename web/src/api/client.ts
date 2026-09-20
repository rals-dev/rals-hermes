// Fetch wrapper for the BFF. Same-origin cookies carry the session, so no
// token ever lives in JavaScript. A 401 anywhere notifies the app, which
// routes to the login page.

import type { ErrorBody } from './types'

export class ApiError extends Error {
  status: number
  code: string
  constructor(status: number, body: ErrorBody) {
    super(body.message)
    this.status = status
    this.code = body.code
  }
}

type UnauthorizedHook = () => void
const hooks = new Set<UnauthorizedHook>()

/** Registers a listener for 401 responses; returns an unsubscribe function. */
export function onUnauthorized(fn: UnauthorizedHook): () => void {
  hooks.add(fn)
  return () => hooks.delete(fn)
}

async function request<T>(url: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(url, {
    credentials: 'same-origin',
    headers: { Accept: 'application/json', ...(init.body ? { 'Content-Type': 'application/json' } : {}) },
    ...init,
  })
  if (res.status === 204) return undefined as T
  const text = await res.text()
  let body: unknown = null
  try {
    body = text ? JSON.parse(text) : null
  } catch {
    body = null
  }
  if (!res.ok) {
    const eb = (body && typeof body === 'object' && 'code' in body ? body : { code: 'http_' + res.status, message: res.statusText || 'request failed' }) as ErrorBody
    if (res.status === 401) for (const h of hooks) h()
    throw new ApiError(res.status, eb)
  }
  return body as T
}

export const api = {
  get<T>(url: string): Promise<T> {
    return request<T>(url)
  },
  login(key: string): Promise<void> {
    return request<void>('/api/auth/session', { method: 'POST', body: JSON.stringify({ key }) })
  },
  logout(): Promise<void> {
    return request<void>('/api/auth/session', { method: 'DELETE' })
  },
}
