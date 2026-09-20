// A thin EventSource wrapper with exponential-backoff reconnection.
//
// The BFF never buffers or replays (PRD § 5), so reconnecting is entirely the
// browser's job: on any error the source is closed and reopened after
// 1 s, 2 s, 4 s … capped at 30 s; a successful open resets the delay.
// Callers resynchronise through the next session.snapshot / status poll.

import type { ErrorBody } from './types'

export type StreamStatus = 'connecting' | 'open' | 'reconnecting' | 'closed'

export interface StreamCallbacks {
  onEvent: (type: string, data: unknown) => void
  onStatus: (status: StreamStatus) => void
  onError?: (err: ErrorBody) => void
}

export interface StreamHandle {
  close: () => void
}

const initialDelay = 1_000
const maxDelay = 30_000

export function connectStream(url: string, events: string[], cb: StreamCallbacks): StreamHandle {
  let es: EventSource | null = null
  let timer: ReturnType<typeof setTimeout> | null = null
  let delay = initialDelay
  let closed = false

  const open = () => {
    if (closed) return
    cb.onStatus(es === null && delay === initialDelay ? 'connecting' : 'reconnecting')
    const source = new EventSource(url)
    es = source
    source.onopen = () => {
      delay = initialDelay
      cb.onStatus('open')
    }
    source.onerror = () => {
      source.close()
      if (closed) return
      cb.onStatus('reconnecting')
      timer = setTimeout(() => {
        delay = Math.min(delay * 2, maxDelay)
        open()
      }, delay)
    }
    for (const type of events) {
      source.addEventListener(type, (e) => cb.onEvent(type, parse((e as MessageEvent).data)))
    }
    // The BFF's in-band failure signal (upstream broke mid-stream).
    source.addEventListener('error', (e) => {
      const data = (e as MessageEvent).data
      if (typeof data === 'string' && cb.onError) {
        const body = parse(data) as ErrorBody | null
        if (body && typeof body === 'object' && 'code' in body) cb.onError(body)
      }
    })
  }

  open()
  return {
    close() {
      closed = true
      if (timer) clearTimeout(timer)
      es?.close()
      cb.onStatus('closed')
    },
  }
}

function parse(data: unknown): unknown {
  if (typeof data !== 'string') return data
  try {
    return JSON.parse(data)
  } catch {
    return data
  }
}
