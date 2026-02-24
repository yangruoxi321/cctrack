<template>
  <div class="session-detail" v-if="session">
    <div class="detail-header">
      <h2 class="detail-title">{{ session.project || session.id.slice(0, 12) }}</h2>
      <Badge :label="formatModel(session.model)" />
    </div>

    <div class="detail-meta">
      <div class="meta-row">
        <span class="meta-label">Session ID</span>
        <span class="meta-value mono">{{ session.id }}</span>
      </div>
      <div class="meta-row">
        <span class="meta-label">Project</span>
        <span class="meta-value">{{ session.project }}</span>
      </div>
      <div class="meta-row" v-if="session.slug">
        <span class="meta-label">Slug</span>
        <span class="meta-value mono">{{ session.slug }}</span>
      </div>
      <div class="meta-row">
        <span class="meta-label">Started</span>
        <span class="meta-value mono">{{ formatDate(session.started_at) }}</span>
      </div>
      <div class="meta-row">
        <span class="meta-label">Last Activity</span>
        <span class="meta-value mono">{{ formatDate(session.last_activity) }}</span>
      </div>
      <div class="meta-row" v-if="requests.length">
        <span class="meta-label">API Calls</span>
        <span class="meta-value mono">{{ requests.length }}</span>
      </div>
    </div>

    <!-- Request Timeline -->
    <div class="detail-section" v-if="requests.length > 1">
      <div class="section-label">Cost Timeline</div>
      <div class="timeline-wrap">
        <svg :viewBox="`0 0 ${timelineWidth} ${timelineHeight}`" class="timeline-svg">
          <!-- Area fill -->
          <path :d="areaPath" fill="rgba(245,158,11,0.12)" />
          <!-- Line -->
          <path :d="linePath" fill="none" stroke="#f59e0b" stroke-width="1.5" />
          <!-- Dots on each request -->
          <circle
            v-for="(pt, i) in points"
            :key="i"
            :cx="pt.x"
            :cy="pt.y"
            r="2.5"
            fill="#f59e0b"
            class="timeline-dot"
          >
            <title>{{ pt.tooltip }}</title>
          </circle>
        </svg>
        <div class="timeline-axis">
          <span>{{ formatTime(requests[0].timestamp) }}</span>
          <span>{{ formatTime(requests[requests.length - 1].timestamp) }}</span>
        </div>
      </div>
    </div>

    <div class="detail-section">
      <div class="section-label">Token Breakdown</div>
      <table class="breakdown-table">
        <tr>
          <td>Input</td>
          <td class="mono right">{{ formatTokensRaw(session.total_input) }}</td>
        </tr>
        <tr>
          <td>Output</td>
          <td class="mono right">{{ formatTokensRaw(session.total_output) }}</td>
        </tr>
        <tr>
          <td>Cache Read</td>
          <td class="mono right">{{ formatTokensRaw(session.total_cache_read) }}</td>
        </tr>
        <tr>
          <td>Cache Write</td>
          <td class="mono right">{{ formatTokensRaw(session.total_cache_write) }}</td>
        </tr>
        <tr class="total-row">
          <td>Total</td>
          <td class="mono right">{{ formatTokensRaw(totalTokens) }}</td>
        </tr>
      </table>
    </div>

    <div class="detail-section">
      <div class="section-label">Cost</div>
      <div class="cost-display">{{ formatCostDisplay(session.total_cost) }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Session, RequestRecord } from '../../types'
import Badge from '../primitives/Badge.vue'
import { formatCostDisplay, formatTokensRaw, formatModel, formatDate } from '../../composables/useFormatCost'
import { fetchSessionRequests } from '../../api'

const props = defineProps<{ session: Session | null }>()
const requests = ref<RequestRecord[]>([])

const timelineWidth = 380
const timelineHeight = 80
const padding = { top: 8, bottom: 8, left: 4, right: 4 }

watch(() => props.session?.id, async (id) => {
  if (!id) { requests.value = []; return }
  try {
    requests.value = await fetchSessionRequests(id) || []
  } catch {
    requests.value = []
  }
}, { immediate: true })

const totalTokens = computed(() => {
  if (!props.session) return 0
  return props.session.total_input + props.session.total_output +
    props.session.total_cache_read + props.session.total_cache_write
})

// Build cumulative cost points for the sparkline
const points = computed(() => {
  if (requests.value.length < 2) return []
  const w = timelineWidth - padding.left - padding.right
  const h = timelineHeight - padding.top - padding.bottom

  let cumCost = 0
  const raw = requests.value.map((r, i) => {
    cumCost += r.cost
    return { idx: i, cumCost, timestamp: r.timestamp, cost: r.cost }
  })

  const maxCost = cumCost || 1
  const n = raw.length

  return raw.map((r, i) => ({
    x: padding.left + (i / (n - 1)) * w,
    y: padding.top + h - (r.cumCost / maxCost) * h,
    tooltip: `$${r.cumCost.toFixed(2)} (+$${r.cost.toFixed(3)})`,
  }))
})

const linePath = computed(() => {
  if (points.value.length < 2) return ''
  return points.value.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x},${p.y}`).join(' ')
})

const areaPath = computed(() => {
  if (points.value.length < 2) return ''
  const base = timelineHeight - padding.bottom
  const first = points.value[0]
  const last = points.value[points.value.length - 1]
  return `M${first.x},${base} ` +
    points.value.map(p => `L${p.x},${p.y}`).join(' ') +
    ` L${last.x},${base} Z`
})

function formatTime(ts: string): string {
  try {
    const d = new Date(ts)
    return d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })
  } catch {
    return ts.slice(11, 16)
  }
}
</script>

<style scoped>
.session-detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
}
.detail-header {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}
.detail-title {
  font-family: 'Bebas Neue', sans-serif;
  font-size: 28px;
  letter-spacing: 0.02em;
  color: var(--text-primary);
}
.detail-meta {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.meta-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
}
.meta-label {
  color: var(--text-tertiary);
}
.meta-value {
  color: var(--text-secondary);
}
.meta-value.mono {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
}
.detail-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.section-label {
  font-size: 10.5px;
  font-weight: 500;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-tertiary);
}

/* Timeline */
.timeline-wrap {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.timeline-svg {
  width: 100%;
  height: 80px;
}
.timeline-dot {
  opacity: 0.7;
  transition: opacity 150ms;
}
.timeline-dot:hover {
  opacity: 1;
  r: 4;
}
.timeline-axis {
  display: flex;
  justify-content: space-between;
  font-family: 'JetBrains Mono', monospace;
  font-size: 9px;
  color: var(--text-disabled);
}

.breakdown-table {
  width: 100%;
}
.breakdown-table td {
  padding: var(--space-2) 0;
  font-size: 13px;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-subtle);
}
.breakdown-table .right {
  text-align: right;
}
.breakdown-table .mono {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
}
.total-row td {
  color: var(--text-primary);
  font-weight: 500;
  border-bottom: none;
  border-top: 1px solid var(--border-default);
}
.cost-display {
  font-family: 'Bebas Neue', sans-serif;
  font-size: 48px;
  color: var(--amber-400);
  line-height: 1;
}
</style>
