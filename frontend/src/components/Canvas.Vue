<script setup lang="ts">
import { ref, onMounted } from 'vue'

const canvasRef = ref<HTMLCanvasElement | null>(null)
const ctxRef = ref<CanvasRenderingContext2D | null>(null)

const scale = ref(1)
const offsetX = ref(0)
const offsetY = ref(0)
const isDragging = ref(false)
const startX = ref(0)
const startY = ref(0)

const handleMouseDown = (e: MouseEvent) => {
  isDragging.value = true
  startX.value = e.clientX - offsetX.value
  startY.value = e.clientY - offsetY.value
}

const handleMouseMove = (e: MouseEvent) => {
  if (isDragging.value) {
    offsetX.value = e.clientX - startX.value
    offsetY.value = e.clientY - startY.value
    drawCanvas()
  }
}

const handleMouseUp = () => {
  isDragging.value = false
}

const handleWheel = (e: WheelEvent) => {
  if (e.deltaY < 0) {
    scale.value += 0.1
  } else {
    scale.value -= 0.1
    if (scale.value < 0.1) scale.value = 0.1 // Prevent zooming out too much
  }
  drawCanvas()
  e.preventDefault()
}

const drawCanvas = () => {
  const canvas = canvasRef.value
  const ctx = ctxRef.value
  if (canvas && ctx) {
    ctx.clearRect(0, 0, canvas.width, canvas.height) // Clear the canvas
    ctx.save()

    // Apply zoom and pan transformations
    ctx.translate(offsetX.value, offsetY.value)
    ctx.scale(scale.value, scale.value)

    // Example: Draw a grid
    ctx.beginPath()
    for (let i = 0; i < canvas.width; i += 50) {
      ctx.moveTo(i, 0)
      ctx.lineTo(i, canvas.height)
      ctx.moveTo(0, i)
      ctx.lineTo(canvas.width, i)
    }
    ctx.strokeStyle = '#ddd'
    ctx.stroke()

    ctx.restore()
  }
}

onMounted(() => {
  const canvas = canvasRef.value
  const ctx = canvas?.getContext('2d')
  if (canvas && ctx) {
    ctxRef.value = ctx
    drawCanvas()

    // Event listeners for drag and zoom
    canvas.addEventListener('mousedown', handleMouseDown)
    canvas.addEventListener('mousemove', handleMouseMove)
    canvas.addEventListener('mouseup', handleMouseUp)
    canvas.addEventListener('wheel', handleWheel)
  }
})
</script>

<template>
  <div class="canvas-container">
    <canvas ref="canvasRef" width="1000" height="1000"></canvas>
  </div>
</template>

<style scoped>
.canvas-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background-color: #f0f0f0;
}

canvas {
  border: 2px solid #333;
  cursor: grab;
}
</style>
