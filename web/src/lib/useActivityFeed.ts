import { computed, onBeforeUnmount, ref, shallowRef, watch, type Ref } from 'vue'
import { connectStream, type StreamHandle, type StreamStatus } from '@/api/sse'
import { activityEventTypes, type ActivityEvent, type Session } from '@/api/types'
import { applyEvent, applySnapshot, type FeedRow } from './feed'

export type SessionRecord = Session & { profile: string }

/**
 * Subscribes to the activity stream of every given profile and folds the
 * events into feed rows plus a map of latest session snapshots. Streams are
 * reopened when the profile list changes and closed on unmount.
 */
export function useActivityFeed(profiles: Ref<string[]>) {
  const rows = shallowRef<FeedRow[]>([])
  const sessions = shallowRef(new Map<string, SessionRecord>())
  const status = ref<Record<string, StreamStatus>>({})
  const fresh = ref(new Set<string>())
  let handles: StreamHandle[] = []

  function connect() {
    for (const h of handles) h.close()
    handles = []
    for (const p of profiles.value) {
      status.value = { ...status.value, [p]: 'connecting' }
      handles.push(
        connectStream(`/api/agents/${encodeURIComponent(p)}/activity/stream`, activityEventTypes, {
          onStatus: (s) => { status.value = { ...status.value, [p]: s } },
          onEvent: (_type, data) => {
            const e = data as ActivityEvent
            if (e.type === 'session.snapshot') {
              const next = new Map(sessions.value)
              applySnapshot(next, e)
              sessions.value = next
              return
            }
            const before = rows.value
            const after = applyEvent(before, e)
            if (after !== before) {
              rows.value = after
              const key = after[0]?.key
              if (key && !before.some((r) => r.key === key)) markFresh(key)
            }
          },
          onError: (err) => {
            rows.value = applyEvent(rows.value, { type: 'error', profile: p, at: new Date().toISOString(), error: err })
          },
        }),
      )
    }
  }

  function markFresh(key: string) {
    fresh.value.add(key)
    fresh.value = new Set(fresh.value)
    setTimeout(() => {
      fresh.value.delete(key)
      fresh.value = new Set(fresh.value)
    }, 1600)
  }

  // Watch a stable key: the profile list is rebuilt on every overview
  // refetch, and reopening four streams every 5 s would defeat the point.
  const key = computed(() => [...profiles.value].sort().join(','))
  watch(key, connect, { immediate: true })
  onBeforeUnmount(() => { for (const h of handles) h.close() })

  return { rows, sessions, status, fresh }
}
