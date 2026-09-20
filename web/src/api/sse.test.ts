import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { connectStream, type StreamHandle } from './sse'

// A controllable stand-in for the browser EventSource.
class FakeEventSource {
  static instances: FakeEventSource[] = []
  url: string
  listeners = new Map<string, ((e: MessageEvent) => void)[]>()
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  closed = false
  readyState = 0
  constructor(url: string) {
    this.url = url
    FakeEventSource.instances.push(this)
  }
  addEventListener(type: string, fn: (e: MessageEvent) => void) {
    this.listeners.set(type, [...(this.listeners.get(type) ?? []), fn])
  }
  close() { this.closed = true; this.readyState = 2 }
  // test helpers
  open() { this.readyState = 1; this.onopen?.() }
  emit(type: string, data: unknown) {
    for (const fn of this.listeners.get(type) ?? []) fn(new MessageEvent(type, { data: JSON.stringify(data) }))
  }
  fail() { this.readyState = 2; this.onerror?.() }
}

describe('connectStream', () => {
  let handle: StreamHandle | undefined
  beforeEach(() => {
    vi.useFakeTimers()
    FakeEventSource.instances = []
    vi.stubGlobal('EventSource', FakeEventSource)
  })
  afterEach(() => {
    handle?.close()
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('dispatches named events as parsed JSON and reports status', () => {
    const seen: unknown[] = []
    const status: string[] = []
    handle = connectStream('/api/x/stream', ['tool.started', 'tool.completed'], {
      onEvent: (type, data) => seen.push([type, data]),
      onStatus: (s) => status.push(s),
    })
    const es = FakeEventSource.instances[0]!
    expect(es.url).toBe('/api/x/stream')
    es.open()
    es.emit('tool.started', { tool: 'terminal' })
    expect(seen).toEqual([['tool.started', { tool: 'terminal' }]])
    expect(status).toEqual(['connecting', 'open'])
  })

  it('reconnects with exponential backoff, capped, and resets after a successful open', () => {
    const status: string[] = []
    handle = connectStream('/api/x/stream', ['message'], { onEvent: () => {}, onStatus: (s) => status.push(s) })

    FakeEventSource.instances[0]!.fail()
    expect(FakeEventSource.instances[0]!.closed).toBe(true)
    expect(status.at(-1)).toBe('reconnecting')
    expect(FakeEventSource.instances).toHaveLength(1)
    vi.advanceTimersByTime(999)
    expect(FakeEventSource.instances).toHaveLength(1)
    vi.advanceTimersByTime(1)
    expect(FakeEventSource.instances).toHaveLength(2) // 1 s

    FakeEventSource.instances[1]!.fail()
    vi.advanceTimersByTime(2000)
    expect(FakeEventSource.instances).toHaveLength(3) // 2 s
    FakeEventSource.instances[2]!.fail()
    vi.advanceTimersByTime(4000)
    expect(FakeEventSource.instances).toHaveLength(4) // 4 s
    for (let i = 3; i < 9; i++) {
      FakeEventSource.instances[i]!.fail()
      vi.advanceTimersByTime(30_000) // cap
    }
    expect(FakeEventSource.instances).toHaveLength(10)

    // A successful open resets the backoff to 1 s.
    FakeEventSource.instances[9]!.open()
    FakeEventSource.instances[9]!.fail()
    vi.advanceTimersByTime(1000)
    expect(FakeEventSource.instances).toHaveLength(11)
  })

  it('stops reconnecting once closed', () => {
    handle = connectStream('/api/x/stream', ['message'], { onEvent: () => {}, onStatus: () => {} })
    FakeEventSource.instances[0]!.fail()
    handle.close()
    vi.advanceTimersByTime(60_000)
    expect(FakeEventSource.instances).toHaveLength(1)
  })

  it('surfaces in-band error events through onError', () => {
    const errors: unknown[] = []
    handle = connectStream('/api/x/stream', ['message'], { onEvent: () => {}, onStatus: () => {}, onError: (e) => errors.push(e) })
    const es = FakeEventSource.instances[0]!
    es.open()
    es.emit('error', { code: 'upstream_unreachable', message: 'Hermes did not answer in time' })
    expect(errors).toEqual([{ code: 'upstream_unreachable', message: 'Hermes did not answer in time' }])
  })
})
