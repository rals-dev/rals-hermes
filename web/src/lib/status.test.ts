import { describe, expect, it } from 'vitest'
import { lampTone } from './status'

describe('lampTone', () => {
  it('maps every worker state and agent status to a lamp colour', () => {
    expect(lampTone('working')).toBe('bg-ok')
    expect(lampTone('healthy')).toBe('bg-ok')
    expect(lampTone('delegating')).toBe('bg-warn')
    expect(lampTone('degraded')).toBe('bg-warn')
    expect(lampTone('error')).toBe('bg-bad')
    expect(lampTone('offline')).toBe('bg-bad')
    expect(lampTone('unauthorized')).toBe('bg-auth')
  })

  it('leaves unknown words, including idle, unlit', () => {
    expect(lampTone('idle')).toBe('bg-lamp-off')
    expect(lampTone('something-new')).toBe('bg-lamp-off')
  })
})
