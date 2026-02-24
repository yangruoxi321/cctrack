<template>
  <div class="heatmap-card">
    <div class="chart-header">
      <div class="chart-title">Activity Heatmap</div>
    </div>
    <div class="heatmap-grid">
      <!-- Hour labels across top -->
      <div class="heatmap-corner"></div>
      <div v-for="h in hourLabels" :key="'h'+h.hour" class="hour-label">{{ h.label }}</div>

      <!-- Rows: one per day of week -->
      <template v-for="d in dayLabels" :key="'d'+d.day">
        <div class="day-label">{{ d.label }}</div>
        <div
          v-for="h in 24"
          :key="d.day + '-' + (h-1)"
          class="heatmap-cell"
          :style="{ background: cellColor(d.day, h - 1) }"
          :title="cellTooltip(d.day, h - 1)"
        ></div>
      </template>
    </div>
    <div class="heatmap-legend">
      <span class="legend-label">Less</span>
      <div class="legend-swatch" style="background: var(--bg-subtle)"></div>
      <div class="legend-swatch" style="background: rgba(245,158,11,0.15)"></div>
      <div class="legend-swatch" style="background: rgba(245,158,11,0.35)"></div>
      <div class="legend-swatch" style="background: rgba(245,158,11,0.6)"></div>
      <div class="legend-swatch" style="background: #f59e0b"></div>
      <span class="legend-label">More</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { HeatmapCell } from '../../types'

const props = defineProps<{ cells: HeatmapCell[] }>()

const dayLabels = [
  { day: 1, label: 'Mon' },
  { day: 2, label: 'Tue' },
  { day: 3, label: 'Wed' },
  { day: 4, label: 'Thu' },
  { day: 5, label: 'Fri' },
  { day: 6, label: 'Sat' },
  { day: 0, label: 'Sun' },
]

const hourLabels = computed(() => {
  const labels = []
  for (let h = 0; h < 24; h++) {
    labels.push({
      hour: h,
      label: h % 3 === 0 ? (h === 0 ? '12a' : h < 12 ? `${h}a` : h === 12 ? '12p' : `${h-12}p`) : '',
    })
  }
  return labels
})

const cellMap = computed(() => {
  const m = new Map<string, number>()
  for (const c of props.cells) {
    m.set(`${c.day}-${c.hour}`, c.cost)
  }
  return m
})

const maxCost = computed(() => {
  let max = 0
  for (const c of props.cells) {
    if (c.cost > max) max = c.cost
  }
  return max || 1
})

function cellColor(day: number, hour: number): string {
  const cost = cellMap.value.get(`${day}-${hour}`) || 0
  if (cost === 0) return 'var(--bg-subtle)'
  const intensity = cost / maxCost.value
  if (intensity < 0.15) return 'rgba(245,158,11,0.10)'
  if (intensity < 0.35) return 'rgba(245,158,11,0.22)'
  if (intensity < 0.55) return 'rgba(245,158,11,0.38)'
  if (intensity < 0.75) return 'rgba(245,158,11,0.58)'
  return '#f59e0b'
}

function cellTooltip(day: number, hour: number): string {
  const cost = cellMap.value.get(`${day}-${hour}`) || 0
  const dayName = dayLabels.find(d => d.day === day)?.label || ''
  const hourStr = hour === 0 ? '12am' : hour < 12 ? `${hour}am` : hour === 12 ? '12pm' : `${hour-12}pm`
  return `${dayName} ${hourStr}: $${cost.toFixed(2)}`
}
</script>

<style scoped>
.heatmap-card {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  padding: var(--space-6);
  animation: fadeSlideUp 0.45s ease both;
  animation-delay: 450ms;
  display: flex;
  flex-direction: column;
}
.chart-header {
  margin-bottom: var(--space-4);
}
.chart-title {
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-tertiary);
}
.heatmap-grid {
  display: grid;
  grid-template-columns: 36px repeat(24, 1fr);
  gap: 2px;
}
.heatmap-corner {
  /* empty top-left corner */
}
.hour-label {
  font-family: 'JetBrains Mono', monospace;
  font-size: 8px;
  color: var(--text-disabled);
  text-align: center;
  padding-bottom: 2px;
}
.day-label {
  font-family: 'JetBrains Mono', monospace;
  font-size: 9px;
  color: var(--text-tertiary);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding-right: 4px;
}
.heatmap-cell {
  height: 24px;
  transition: background 300ms;
  cursor: default;
}
.heatmap-cell:hover {
  outline: 1px solid var(--amber-500);
  outline-offset: -1px;
  z-index: 1;
}
.heatmap-legend {
  display: flex;
  align-items: center;
  gap: 3px;
  justify-content: flex-end;
  margin-top: var(--space-3);
}
.legend-label {
  font-family: 'JetBrains Mono', monospace;
  font-size: 9px;
  color: var(--text-disabled);
  padding: 0 3px;
}
.legend-swatch {
  width: 10px;
  height: 10px;
}
</style>
