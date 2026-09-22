// Wire types of the BFF API (docs/prd.md § 5). Kept in one file so the
// shape of every response is visible at a glance.

export interface ErrorBody {
  code: string
  message: string
}

export type AgentStatus = 'healthy' | 'degraded' | 'unreachable' | 'unauthorized'

export interface PlatformState {
  state: string
  error_code: string | null
  error_message: string | null
  updated_at: string
  needs_attention?: boolean
}

export interface AgentCard {
  profile: string
  status: AgentStatus
  latency_ms: number
  error?: ErrorBody
  platforms?: Record<string, PlatformState>
}

export interface Gateway {
  source_profile: string
  version: string
  state: string
  readiness: string
  active_agents: number
  active_api_runs: number
  active_delegations: number
  process_completions: number
  busy: boolean
  disk: { status: string; used_percent: number; free_bytes: number }
  checks: Record<string, string>
  updated_at: string
}

export interface Overview {
  generated_at: string
  gateway: Gateway | null
  agents: AgentCard[]
}

export interface Toolset {
  name: string
  label: string
  enabled: boolean
  configured: boolean
  tools: string[]
}

export interface AgentDetail extends AgentCard {
  generated_at: string
  model?: string
  models: string[]
  capabilities: Record<string, unknown> | null
  toolsets: Toolset[]
  skills: unknown
  warnings: string[]
}

export interface Usage {
  input_tokens: number
  output_tokens: number
  cache_read_tokens: number
  cache_write_tokens: number
  reasoning_tokens: number
}

export interface Session {
  id: string
  source: string
  model: string
  title: string | null
  preview: string | null
  started_at: string
  ended_at: string | null
  end_reason: string | null
  open: boolean
  last_active: string
  message_count: number
  tool_call_count: number
  api_call_count: number
  parent_session_id: string | null
  usage: Usage
  estimated_cost_usd: number | null
  actual_cost_usd: number | null
}

export interface SessionList {
  profile: string
  data: Session[]
  has_more: boolean
  limit: number
  offset: number
}

export interface ToolCall {
  id: string
  type: string
  function: { name: string; arguments: string }
}

export interface Message {
  id: number
  session_id: string
  role: 'user' | 'assistant' | 'tool' | 'session_meta' | string
  content: string | null
  tool_call_id: string
  tool_calls: ToolCall[] | null
  tool_name: string
  timestamp: string | null
  token_count: number | null
  finish_reason: string | null
  display_kind: string | null
}

export interface SessionDetail {
  profile: string
  session: Session
  messages: {
    session_id: string
    pagination: { limit: number; offset: number; order: string; returned: number }
    data: Message[]
  }
}

export interface Job {
  profile: string
  id: string
  name: string
  schedule: { kind: string; expr: string; display: string }
  schedule_display: string
  enabled: boolean
  state: string
  created_at: string | null
  next_run_at: string | null
  last_run_at: string | null
  last_status: string | null
  last_error: string | null
  failure_streak: number
  deliver: string | null
  model_snapshot: string | null
}

export interface JobsResponse {
  generated_at: string
  jobs: Job[]
  errors: { profile: string; error: ErrorBody }[]
}

export interface UsageProfile {
  profile: string
  session_count: number
  usage: Usage
  estimated_cost_usd: number | null
  actual_cost_usd: number | null
  // Sessions kept coming for the whole window (safety cap hit): totals are
  // a lower bound for this profile, not the full window.
  truncated: boolean
}

export interface UsageResponse {
  generated_at: string
  window_hours: number
  profiles: UsageProfile[]
  errors: { profile: string; error: ErrorBody }[]
}

export type ActivityEventType =
  | 'session.snapshot'
  | 'session.started'
  | 'session.ended'
  | 'subagent.start'
  | 'subagent.complete'
  | 'tool.started'
  | 'tool.completed'
  | 'message'
  | 'error'

export interface ActivityEvent {
  type: ActivityEventType
  profile: string
  session_id?: string
  at: string
  session?: Session
  parent_session_id?: string
  end_reason?: string
  tool?: string
  call_id?: string
  preview?: string
  role?: string
  message_id?: number
  error?: ErrorBody
}

export const activityEventTypes: ActivityEventType[] = [
  'session.snapshot',
  'session.started',
  'session.ended',
  'subagent.start',
  'subagent.complete',
  'tool.started',
  'tool.completed',
  'message',
  'error',
]
