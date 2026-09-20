import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api, onUnauthorized } from './client'

function jsonResponse(status: number, body: unknown) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

describe('api client', () => {
  const fetchMock = vi.fn()
  beforeEach(() => vi.stubGlobal('fetch', fetchMock))
  afterEach(() => {
    vi.unstubAllGlobals()
    fetchMock.mockReset()
  })

  it('parses JSON and sends credentials for same-origin cookies', async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(200, { agents: [] }))
    const out = await api.get<{ agents: unknown[] }>('/api/agents')
    expect(out).toEqual({ agents: [] })
    expect(fetchMock).toHaveBeenCalledWith('/api/agents', expect.objectContaining({ credentials: 'same-origin' }))
  })

  it('throws ApiError carrying the BFF error body', async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(503, { code: 'upstream_unreachable', message: 'Hermes did not answer in time' }))
    await expect(api.get('/api/overview')).rejects.toMatchObject({ status: 503, code: 'upstream_unreachable' })
  })

  it('notifies the unauthorized hook on 401 so the app can route to login', async () => {
    const hook = vi.fn()
    const off = onUnauthorized(hook)
    fetchMock.mockResolvedValueOnce(jsonResponse(401, { code: 'unauthorized', message: 'a valid session is required' }))
    await expect(api.get('/api/overview')).rejects.toBeInstanceOf(ApiError)
    expect(hook).toHaveBeenCalledTimes(1)
    off()
  })

  it('login posts the key and logout deletes the session', async () => {
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }))
    await api.login('secret')
    expect(fetchMock).toHaveBeenLastCalledWith('/api/auth/session', expect.objectContaining({ method: 'POST', body: JSON.stringify({ key: 'secret' }) }))
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }))
    await api.logout()
    expect(fetchMock).toHaveBeenLastCalledWith('/api/auth/session', expect.objectContaining({ method: 'DELETE' }))
  })
})
