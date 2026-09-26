// The lamp colour for a status word, shared by StatusWord and the office
// view's desk nameplates. The word carries the meaning; the lamp reinforces.

/** Tailwind background class for a status lamp. Unknown words get an unlit lamp. */
export function lampTone(status: string): string {
  switch (status) {
    case 'healthy': case 'ok': case 'connected': case 'running': case 'completed': case 'live': case 'open': case 'working': return 'bg-ok'
    case 'degraded': case 'warn': case 'waiting_for_approval': case 'stopping': case 'blocked': case 'reconnecting': case 'delegating': return 'bg-warn'
    case 'unreachable': case 'failed': case 'error': case 'interrupted': case 'offline': return 'bg-bad'
    case 'unauthorized': return 'bg-auth'
    default: return 'bg-lamp-off'
  }
}
