<script setup lang="ts">
export interface RatingRadarPoint {
  label: string
  value: number | null
}

const props = defineProps<{
  points: RatingRadarPoint[]
}>()

const VIEW_SIZE = 260
const CENTER = 130
const RADIUS = 90
const LABEL_RADIUS = 108
const RING_LEVELS = [2.5, 5, 7.5, 10]

function angleAt(index: number): number {
  const count = props.points.length || 1
  return ((-90 + (360 / count) * index) * Math.PI) / 180
}

function coordinateAt(index: number, radius: number): { x: number; y: number } {
  const angle = angleAt(index)
  return {
    x: CENTER + radius * Math.cos(angle),
    y: CENTER + radius * Math.sin(angle)
  }
}

function polygonPoints(radiusFor: (index: number) => number): string {
  return props.points
    .map((_, index) => {
      const { x, y } = coordinateAt(index, radiusFor(index))
      return `${x.toFixed(2)},${y.toFixed(2)}`
    })
    .join(' ')
}

const rings = computed(() =>
  RING_LEVELS.map((level) =>
    polygonPoints(() => (RADIUS * level) / 10)
  )
)

const axes = computed(() =>
  props.points.map((_, index) => coordinateAt(index, RADIUS))
)

const labels = computed(() =>
  props.points.map((point, index) => {
    const { x, y } = coordinateAt(index, LABEL_RADIUS)
    const cos = Math.cos(angleAt(index))
    let anchor = 'middle'
    if (cos > 0.3) {
      anchor = 'start'
    } else if (cos < -0.3) {
      anchor = 'end'
    }
    const sin = Math.sin(angleAt(index))
    const dy = sin < -0.7 ? -4 : sin > 0.7 ? 12 : 4
    return { label: point.label, x, y, anchor, dy }
  })
)

const shape = computed(() => {
  const available = props.points
    .map((point, index) => ({ point, index }))
    .filter((entry) => entry.point.value !== null)
  if (available.length < 3) {
    return null
  }
  return available
    .map(({ point, index }) => {
      const { x, y } = coordinateAt(index, (RADIUS * (point.value ?? 0)) / 10)
      return `${x.toFixed(2)},${y.toFixed(2)}`
    })
    .join(' ')
})

const dots = computed(() =>
  props.points
    .map((point, index) => ({ point, index }))
    .filter((entry) => entry.point.value !== null)
    .map(({ point, index }) => ({
      ...coordinateAt(index, (RADIUS * (point.value ?? 0)) / 10)
    }))
)
</script>

<template>
  <svg
    :viewBox="`0 0 ${VIEW_SIZE} ${VIEW_SIZE}`"
    class="rating-radar"
    role="img"
    aria-label="多维评分雷达图"
  >
    <polygon
      v-for="(ring, index) in rings"
      :key="index"
      :points="ring"
      class="rating-radar-grid"
    />
    <line
      v-for="(axis, index) in axes"
      :key="`axis-${index}`"
      :x1="CENTER"
      :y1="CENTER"
      :x2="axis.x"
      :y2="axis.y"
      class="rating-radar-axis"
    />
    <polygon v-if="shape" :points="shape" class="rating-radar-shape" />
    <circle
      v-for="(dot, index) in dots"
      :key="`dot-${index}`"
      :cx="dot.x"
      :cy="dot.y"
      r="2.5"
      class="rating-radar-dot"
    />
    <text
      v-for="(label, index) in labels"
      :key="`label-${index}`"
      :x="label.x"
      :y="label.y"
      :text-anchor="label.anchor"
      :dy="label.dy"
      class="rating-radar-label"
    >
      {{ label.label }}
    </text>
  </svg>
</template>

<style scoped>
.rating-radar {
  display: block;
  width: 100%;
  max-width: 260px;
  margin-inline: auto;
}

.rating-radar-grid {
  fill: none;
  stroke: var(--color-default-200);
  stroke-width: 1;
}

.rating-radar-axis {
  stroke: var(--color-default-200);
  stroke-width: 1;
}

.rating-radar-shape {
  fill: color-mix(in srgb, var(--color-primary) 18%, transparent);
  stroke: var(--color-primary);
  stroke-width: 2;
  stroke-linejoin: round;
}

.rating-radar-dot {
  fill: var(--color-primary);
}

.rating-radar-label {
  fill: var(--color-default-500);
  font-size: 11px;
}
</style>
