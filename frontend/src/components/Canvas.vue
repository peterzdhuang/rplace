<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, reactive, computed, nextTick } from 'vue'

// --- Configuration ---
const PIXEL_SIZE = 15 // Base size of each pixel block
const GRID_WIDTH = 10// Grid width in pixels
const GRID_HEIGHT = 10 // Grid height in pixels
const WS_URL = 'ws://localhost:8000/ws'

// Changed from hex to RGB objects
const COLOR_PALETTE = [
  { r: 255, g: 255, b: 255 }, // White
  { r: 228, g: 228, b: 228 }, // Light Grey
  { r: 136, g: 136, b: 136 }, // Grey
  { r: 34, g: 34, b: 34 },    // Black
  { r: 255, g: 167, b: 209 }, // Pink
  { r: 229, g: 0, b: 0 },     // Red
  { r: 229, g: 149, b: 0 },   // Orange
  { r: 160, g: 106, b: 66 },  // Brown
  { r: 229, g: 217, b: 0 },   // Yellow
  { r: 148, g: 224, b: 68 },  // Light Green
  { r: 2, g: 190, b: 1 },     // Green
  { r: 0, g: 211, b: 221 },   // Cyan
  { r: 0, g: 131, b: 199 },   // Light Blue
  { r: 0, g: 0, b: 234 },     // Blue
  { r: 207, g: 110, b: 228 }, // Light Purple
  { r: 130, g: 0, b: 128 }    // Purple
]
const MIN_ZOOM_FOR_GRID_LINES = 5 // Only draw grid lines if scaled pixel size is > this

// --- Canvas Refs and State ---
const canvasRef = ref<HTMLCanvasElement | null>(null)
const ctxRef = ref<CanvasRenderingContext2D | null>(null)
// Canvas display size will be set dynamically
const canvasWidth = ref(window.innerWidth) // Use window size initially
const canvasHeight = ref(window.innerHeight)

// --- Pan & Zoom State ---
const scale = ref(0.5) // Start slightly zoomed out for large canvas
const offsetX = ref(0)
const offsetY = ref(0)
const isPanning = ref(false)
const startX = ref(0)
const startY = ref(0)
const hasMoved = ref(false)

// --- Grid & Pixel State ---
// Changed to store RGB objects instead of hex strings
const pixels = reactive<{ data: {r: number, g: number, b: number}[][] }>({ data: [] }) // Initialize empty
const ws = ref<WebSocket | null>(null)
const isLoading = ref(true) // Start in loading state
const loadError = ref<string | null>(null) // To show errors

// --- Modal & Selection State ---
const isModalVisible = ref(false)
const selectedPixelCoords = reactive<{ x:number; y:number }>({ x:-1, y:-1 })
const draftColour = ref<{ r:number; g:number; b:number } | null>(null)   // 👈 NEW

// --- Computed Properties ---
// These represent the total size of the grid content in pixels (at scale=1)
const canvasContentWidth = computed(() => GRID_WIDTH * PIXEL_SIZE)
const canvasContentHeight = computed(() => GRID_HEIGHT * PIXEL_SIZE)

// --- Coordinate Transformation ---
const screenToGridCoords = (clientX: number, clientY: number): { x: number; y: number } | null => {
  const canvas = canvasRef.value
  if (!canvas) return null
  const rect = canvas.getBoundingClientRect()
  // Inverse transform: Screen -> Canvas -> World -> Grid
  const canvasX = clientX - rect.left
  const canvasY = clientY - rect.top
  const worldX = (canvasX - offsetX.value) / scale.value
  const worldY = (canvasY - offsetY.value) / scale.value
  const gridX = Math.floor(worldX / PIXEL_SIZE)
  const gridY = Math.floor(worldY / PIXEL_SIZE)

  if (gridX >= 0 && gridX < GRID_WIDTH && gridY >= 0 && gridY < GRID_HEIGHT) {
    return { x: gridX, y: gridY }
  }
  return null
}

// Helper function to convert RGB to CSS color string
const rgbToString = (colour: {r: number, g: number, b: number}): string => {
  return `rgb(${colour.r}, ${colour.g}, ${colour.b})`;
}

// --- WebSocket Logic ---
const connectWebSocket = () => {
  if (ws.value) return; // Avoid multiple connections

  console.log('Attempting to connect to WebSocket:', WS_URL);
  isLoading.value = true;
  loadError.value = null;
  ws.value = new WebSocket(WS_URL);

  ws.value.onopen = () => {
    console.log('WebSocket connection established.');
    // Optional: Request initial state explicitly if backend requires it
    // sendWebSocketMessage({ type: 'get_initial_state' });
  };

  ws.value.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data);
      console.log(message)
      console.log('WebSocket message received:', message.type);

      if (message.type === 'init' && Array.isArray(message.pixels)) {
        console.log(`Received initial state with ${message.pixels.length} rows.`);
        // Basic validation (can be more robust)
        if (message.pixels.length > 0 && message.pixels[0]?.length > 0) {
            // Assuming backend sends full grid of RGB values
            pixels.data = message.pixels;
            isLoading.value = false; // Data loaded
            loadError.value = null;
            console.log('Initial pixel state loaded.');
            drawCanvas();
        } else {
             throw new Error('Received initial state pixel data is empty or invalid.');
        }
      } else if (message.type === 'update') {

        const { x, y, pixel } = message;  // Expecting RGB object
        console.log(x, y, pixel)
        if (
          pixels.data[y] && // Check row exists
          x >= 0 && x < GRID_WIDTH &&
          y >= 0 && y < GRID_HEIGHT
        ) {
          pixels.data[y][x] = pixel;  // Store RGB object
          // Optimize: Only redraw if the updated pixel is visible
          // For simplicity now, redraw full canvas
          drawCanvas();
          // Or implement drawPixel if performance is critical
          // drawPixelIfVisible(x, y, colour);
        }
      } else {
          console.warn('Unknown or unhandled WebSocket message type:', message.type);
      }
    } catch (error: any) {
      console.error('Failed to parse WebSocket message or handle update:', error);
      loadError.value = `Error processing message: ${error.message || 'Unknown error'}`;
      isLoading.value = false; // Stop loading on error
      // Maybe initialize empty grid as fallback?
       if (!pixels.data.length) initializeEmptyGrid(true); // Init if empty on error
       drawCanvas();
    }
  };

  ws.value.onerror = (error) => {
    console.error('WebSocket error:', error);
    loadError.value = 'WebSocket connection error. Please try refreshing.';
    isLoading.value = false;
    ws.value = null; // Clear WS ref on error
    if (!pixels.data.length) initializeEmptyGrid(true); // Init if empty on error
    drawCanvas();
  };

  ws.value.onclose = (event) => {
    console.log('WebSocket connection closed:', event.code, event.reason);
    // Only show error if it wasn't a clean closure maybe?
    if (!event.wasClean && !loadError.value) {
       loadError.value = 'WebSocket connection closed unexpectedly.';
    }
    isLoading.value = false; // Ensure loading stops
    ws.value = null;
     if (!pixels.data.length) initializeEmptyGrid(true); // Init if empty on close without data
     drawCanvas();
    // Optional: Implement reconnection logic here
  };
};

const sendWebSocketMessage = (message: object) => {
  if (ws.value && ws.value.readyState === WebSocket.OPEN) {
    ws.value.send(JSON.stringify(message));
  } else {
    console.error('WebSocket is not connected or not open.');
    // Optionally notify user
  }
};

const centerView = () => {
    const canvas = canvasRef.value;
    if (!canvas) return;
    // Calculate desired scale to fit maybe 1/4 of the grid width initially
    const initialScale = Math.min(
        canvas.clientWidth / (canvasContentWidth.value / 2),
        canvas.clientHeight / (canvasContentHeight.value / 2)
    );
     scale.value = Math.max(0.05, initialScale); // Limit minimum initial zoom

    // Center the grid in the canvas
    offsetX.value = (canvas.clientWidth - canvasContentWidth.value * scale.value) / 2;
    offsetY.value = (canvas.clientHeight - canvasContentHeight.value * scale.value) / 2;
    console.log(`Centering view. Scale: ${scale.value}, Offset: (${offsetX.value}, ${offsetY.value})`);
}

// --- Event Handlers ---
const handleMouseDown = (e: MouseEvent) => {
  if ((e.target as HTMLElement)?.closest('.colour-modal')) return;
  isPanning.value = true;
  hasMoved.value = false;
  startX.value = e.clientX - offsetX.value;
  startY.value = e.clientY - offsetY.value;
  if (canvasRef.value) canvasRef.value.style.cursor = 'grabbing';
};

const handleMouseMove = (e: MouseEvent) => {
  if (isPanning.value) {
    const currentX = e.clientX - startX.value;
    const currentY = e.clientY - startY.value;
    if (!hasMoved.value && (Math.abs(e.clientX - (startX.value + offsetX.value)) > 5 || Math.abs(e.clientY - (startY.value + offsetY.value)) > 5)) {
      hasMoved.value = true;
    }
    if (hasMoved.value) {
      offsetX.value = currentX;
      offsetY.value = currentY;
      requestAnimationFrame(drawCanvas); // Use rAF for smoother panning
    }
  }
};

const handleMouseUp = (e: MouseEvent) => {
  // Check if mouseup occurred over the modal - if so, do nothing here
  if ((e.target as HTMLElement)?.closest('.colour-modal')) {
     // If panning was active, reset it, but don't process click
     if (isPanning.value) {
         isPanning.value = false;
         if (canvasRef.value) canvasRef.value.style.cursor = 'grab';
     }
     return;
  }

  if (isPanning.value) {
    if (!hasMoved.value && e.target === canvasRef.value) { // Ensure click was on canvas
      handleCanvasClick(e);
    }
    isPanning.value = false;
    if (canvasRef.value) canvasRef.value.style.cursor = 'grab';
  }
  hasMoved.value = false; // Reset move flag
};

const handleCanvasClick = (e: MouseEvent) => {
  const coords = screenToGridCoords(e.clientX, e.clientY);
  if (coords) {
    // Deselect previous, select new
     const previousX = selectedPixelCoords.x;
     const previousY = selectedPixelCoords.y;
    selectedPixelCoords.x = coords.x;
    selectedPixelCoords.y = coords.y;
    isModalVisible.value = true;
    console.log(`Selected pixel: (${coords.x}, ${coords.y})`);
    // Redraw needed to show selection highlight
    // If previous pixel was different and visible, it also needs redraw to remove highlight
    drawCanvas(); // Simple redraw all for now
  } else {
    // Click outside grid
    if (isModalVisible.value) {
      closeModal();
    }
  }
};

const handleWheel = (e: WheelEvent) => {
  e.preventDefault();
  const canvas = canvasRef.value;
  if (!canvas) return;

  const rect = canvas.getBoundingClientRect();
  const mouseX = e.clientX - rect.left; // Mouse X relative to canvas
  const mouseY = e.clientY - rect.top;  // Mouse Y relative to canvas

  // World coordinates before zoom (point under mouse)
  const worldXBefore = (mouseX - offsetX.value) / scale.value;
  const worldYBefore = (mouseY - offsetY.value) / scale.value;

  // Calculate new scale (logarithmic zoom feels better)
  const scaleFactor = e.deltaY < 0 ? 1.1 : 1 / 1.1; // Zoom factor
  const newScale = Math.max(0.01, Math.min(scale.value * scaleFactor, 50)); // Clamp zoom level

   // Calculate new offset to keep mouse point stationary
  offsetX.value = mouseX - worldXBefore * newScale;
  offsetY.value = mouseY - worldYBefore * newScale;
  scale.value = newScale;

  requestAnimationFrame(drawCanvas); // Use rAF
};

// --- Modal Actions ---
const selectColour = (colour: {r: number, g: number, b: number}) => {
  if (selectedPixelCoords.x !== -1 && selectedPixelCoords.y !== -1) {
    console.log(`Placing colour RGB(${colour.r},${colour.g},${colour.b}) at (${selectedPixelCoords.x}, ${selectedPixelCoords.y})`);
    
    
    sendWebSocketMessage({
      type: 'update',
      payload: {
        x: selectedPixelCoords.x,
        y: selectedPixelCoords.y,
        colour: colour, // Send RGB object directly
      }
    });
    // Don't clear selection here - wait for WS update or potential immediate redraw
    closeModal(); // Close modal, redraw will happen
  }
};

const confirmColour = () => {
  if (
    draftColour.value &&
    selectedPixelCoords.x !== -1 &&
    selectedPixelCoords.y !== -1
  ) {
    sendWebSocketMessage({
      type: 'place_pixel',
      payload: {
        x: selectedPixelCoords.x,
        y: selectedPixelCoords.y,
        colour: draftColour.value
      }
    })
    closeModal()           // will also clear draftColour inside closeModal()
  }
}


const closeModal = () => {
  const needsRedraw = selectedPixelCoords.x !== -1; // Need redraw if something was selected
  isModalVisible.value = false;
  selectedPixelCoords.x = -1;
  selectedPixelCoords.y = -1;
  if (needsRedraw) {
      requestAnimationFrame(drawCanvas); // Redraw to remove selection highlight
  }
};

// --- Drawing Logic ---
const initializeEmptyGrid = (showWarning = false) => {
    if(showWarning) console.warn(`Initializing fallback empty ${GRID_WIDTH}x${GRID_HEIGHT} grid locally.`);
    // Create a large array efficiently (might still be slow)
    const white = { r: 255, g: 255, b: 255 };  // White in RGB
    pixels.data = Array(GRID_HEIGHT);
    for(let y=0; y<GRID_HEIGHT; y++){
        pixels.data[y] = Array(GRID_WIDTH).fill(white);
    }
    isLoading.value = false; // Ensure loading is set to false
};

const drawCanvas = () => {
  const canvas = canvasRef.value;
  const ctx = ctxRef.value;
  if (!canvas || !ctx) return;

  const dpr = window.devicePixelRatio || 1;
  const viewWidth = canvas.clientWidth; // Display width
  const viewHeight = canvas.clientHeight; // Display height

  // Clear the canvas (physical pixels)
  ctx.save();
  ctx.resetTransform(); // Ensure clean state before clearing
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.restore();

  // Apply DPR scaling for drawing
  ctx.save();
  ctx.scale(dpr, dpr);

  // Apply pan and zoom
  ctx.translate(offsetX.value, offsetY.value);
  ctx.scale(scale.value, scale.value);

  // --- Viewport Culling ---
  // Calculate the range of grid cells visible in the current viewport
  const viewLeftWorld = -offsetX.value / scale.value;
  const viewTopWorld = -offsetY.value / scale.value;
  const viewRightWorld = (viewWidth - offsetX.value) / scale.value;
  const viewBottomWorld = (viewHeight - offsetY.value) / scale.value;

  const startCol = Math.max(0, Math.floor(viewLeftWorld / PIXEL_SIZE));
  const endCol = Math.min(GRID_WIDTH, Math.ceil(viewRightWorld / PIXEL_SIZE));
  const startRow = Math.max(0, Math.floor(viewTopWorld / PIXEL_SIZE));
  const endRow = Math.min(GRID_HEIGHT, Math.ceil(viewBottomWorld / PIXEL_SIZE));
  // --- End Viewport Culling ---

  // If grid data hasn't loaded yet, don't try to draw pixels
  if (isLoading.value || !pixels.data.length) {
      ctx.restore(); // Restore from DPR scale
      // Optional: Draw loading text or spinner on canvas
      ctx.save();
      ctx.font = "16px Arial";
      ctx.fillStyle = "#555";
      ctx.textAlign = "center";
      ctx.fillText(loadError.value || "Loading pixels...", viewWidth / 2, viewHeight / 2);
      ctx.restore();
      return;
  }


  const scaledPixelSize = PIXEL_SIZE * scale.value;
  const drawGridLines = scaledPixelSize > MIN_ZOOM_FOR_GRID_LINES;

  const defaultLineWidth = 0.5 / scale.value; // Thinner lines
  const defaultStrokeStyle = '#E0E0E0'; // Lighter grey
  const selectedLineWidth = 2.0 / scale.value; // Thicker selected border
  const selectedStrokeStyle = '#FFFFFF';

  // Set default line style once if possible
  ctx.lineWidth = defaultLineWidth;
  ctx.strokeStyle = defaultStrokeStyle;

  // --- Draw Visible Pixels ---
  for (let y = startRow; y < endRow; y++) {
    // Ensure row exists (important if data is loaded partially/incorrectly)
    if (!pixels.data[y]) continue;

    for (let x = startCol; x < endCol; x++) {
      // Default to black if missing
      const colour = pixels.data[y][x] || { r: 0, g: 0, b: 0};
      ctx.fillStyle = rgbToString(colour);  // Convert RGB object to CSS string
      ctx.fillRect(x * PIXEL_SIZE, y * PIXEL_SIZE, PIXEL_SIZE, PIXEL_SIZE);
    }
  }

    // --- Draw Grid Lines (Optional and Optimized) ---
    if (drawGridLines) {
        ctx.beginPath();
        for (let x = startCol; x <= endCol; x++) {
            ctx.moveTo(x * PIXEL_SIZE, startRow * PIXEL_SIZE);
            ctx.lineTo(x * PIXEL_SIZE, endRow * PIXEL_SIZE);
        }
        for (let y = startRow; y <= endRow; y++) {
            ctx.moveTo(startCol * PIXEL_SIZE, y * PIXEL_SIZE);
            ctx.lineTo(endCol * PIXEL_SIZE, y * PIXEL_SIZE);
        }
        ctx.lineWidth = defaultLineWidth;
        ctx.strokeStyle = defaultStrokeStyle;
        ctx.stroke();
    }

  // --- Draw Selection Highlight (if selected pixel is visible) ---
  if (
    selectedPixelCoords.x >= startCol && selectedPixelCoords.x < endCol &&
    selectedPixelCoords.y >= startRow && selectedPixelCoords.y < endRow
  ) {
    ctx.lineWidth = selectedLineWidth;
    ctx.strokeStyle = selectedStrokeStyle;
    // Adjust rect slightly for better border visibility
    const borderOffset = selectedLineWidth / (2 * scale.value); // Adjust offset based on line width in world coords
     ctx.strokeRect(
       selectedPixelCoords.x * PIXEL_SIZE + borderOffset,
       selectedPixelCoords.y * PIXEL_SIZE + borderOffset,
       PIXEL_SIZE - 2 * borderOffset,
       PIXEL_SIZE - 2 * borderOffset
     );
  }

  ctx.restore(); // Restore from DPR scale + transforms
};

// --- Lifecycle Hooks ---
const updateCanvasSize = () => {
  // Update reactive vars for potential template bindings
  canvasWidth.value = window.innerWidth;
  canvasHeight.value = window.innerHeight;

  nextTick(() => {
    const canvas = canvasRef.value;
    if (canvas) {
      const dpr = window.devicePixelRatio || 1;
      // Set the display size of the canvas
      canvas.style.width = `${canvasWidth.value}px`;
      canvas.style.height = `${canvasHeight.value}px`;
      // Set the actual drawing buffer size accounting for DPR
      canvas.width = canvasWidth.value * dpr;
      canvas.height = canvasHeight.value * dpr;

      // No need to scale context here, handled in drawCanvas
      drawCanvas(); // Redraw after resize
    }
  });
};

onMounted(() => {
  const canvas = canvasRef.value;
  if (canvas) {
    const ctx = canvas.getContext('2d', { alpha: false }); // Improve perf if no transparency needed
    if (ctx) {
      ctxRef.value = ctx;
      console.log('Canvas context obtained.');
      updateCanvasSize(); // Set initial size

       // Initial draw might show loading screen
       drawCanvas();

      // Add event listeners
      canvas.addEventListener('mousedown', handleMouseDown);
      canvas.addEventListener('mousemove', handleMouseMove);
      window.addEventListener('mouseup', handleMouseUp); // Listen on window
      // canvas.addEventListener('mouseleave', handleMouseUp); // mouseup on window covers this
      canvas.addEventListener('wheel', handleWheel, { passive: false });

      // Connect WebSocket AFTER setting up canvas
      connectWebSocket();

    } else {
        console.error("Failed to get 2D rendering context.");
        loadError.value = "Canvas context not available.";
        isLoading.value = false;
    }
  } else {
      console.error("Canvas element not found.");
      loadError.value = "Canvas element failed to mount.";
      isLoading.value = false;
  }
  window.addEventListener('resize', updateCanvasSize);
});
const chooseColour = (colour:{r:number,g:number,b:number}) => {   // renamed
  if (
    selectedPixelCoords.x !== -1 &&
    selectedPixelCoords.y !== -1 &&
    pixels.data[selectedPixelCoords.y]             // row safety check
  ) {
    // clone so we don’t hold a reference
    pixels.data[selectedPixelCoords.y][selectedPixelCoords.x] = { ...colour }
    sendWebSocketMessage({
    pixel: colour,   // field name must be "pixel"
    x : selectedPixelCoords.x,
      y : selectedPixelCoords.y,
    }) 
    
    drawCanvas()
  }
}

onBeforeUnmount(() => {
  const canvas = canvasRef.value;
  if (canvas) {
    canvas.removeEventListener('mousedown', handleMouseDown);
    canvas.removeEventListener('mousemove', handleMouseMove);
    window.removeEventListener('mouseup', handleMouseUp); // Remove window listener
    canvas.removeEventListener('wheel', handleWheel);
  }
  window.removeEventListener('resize', updateCanvasSize);

  if (ws.value) {
    ws.value.onclose = null; // Prevent close handler triggers during unmount
    ws.value.onerror = null;
    ws.value.onmessage = null;
    ws.value.onopen = null;
    ws.value.close();
    console.log('WebSocket connection closed during unmount.');
  }
});
</script>

<template>
  <div class="canvas-wrapper">
    <canvas
      ref="canvasRef"
      class="main-canvas"
    ></canvas>

    <div v-if="isLoading || loadError" class="status-overlay">
       {{ loadError || 'Loading Canvas...' }}
     </div>

    <div class="colour-modal"
     :class="{ 'modal-visible': isModalVisible }"
     @mousedown.stop @click.stop @wheel.stop>

  <div class="modal-content">
    <div class="colour-palette">
      <div v-for="colour in COLOR_PALETTE"
           :key="`${colour.r}-${colour.g}-${colour.b}`"
           class="colour-swatch"
           :class="{ selected: draftColour && colour === draftColour }"
           :style="{ backgroundColor: rgbToString(colour),
            border: draftColour && colour === draftColour
              ? '2px solid #333'
              : '2px solid transparent'
           }"
           @click="chooseColour(colour)"></div>
    </div>

    <div class="modal-actions">
      <button @click="closeModal">Cancel</button>
      <button :disabled="!draftColour" @click="confirmColour">Confirm</button>
    </div>
  </div>
</div>

  </div>
</template>

<style scoped>
.canvas-wrapper {
  position: relative;
  width: 100vw;
  height: 100vh;
  background-color: #777; /* Darker background for contrast */
  overflow: hidden;
  display: flex;
  justify-content: center;
  align-items: center;
}

.main-canvas {
  /* Let JS set width/height style for DPR */
  max-width: 100%;
  max-height: 100%;
  display: block;
  cursor: grab;
  background-color: #ffffff; /* Set a canvas background for loading state */
  image-rendering: pixelated; /* Better for pixel art when zoomed */
  image-rendering: crisp-edges;
}

/* Prevent text selection during drag */
.main-canvas::selection { background: transparent; }
.main-canvas::-moz-selection { background: transparent; }

.status-overlay {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    display: flex;
    justify-content: center;
    align-items: center;
    background-color: rgba(255, 255, 255, 0.5);
    color: white;
    font-size: 1.5em;
    z-index: 500; /* Below modal */
    pointer-events: none; /* Allow clicks through if needed, though covered */
}

.colour-modal {
  position: fixed; /* Use fixed positioning */
  bottom: 0;
  left: 0;
  right: 0;
  width: 100%; /* Ensure full width */
  background-color: rgba(245, 245, 245, 0.98); /* Slightly opaque background */
  border-top: 1px solid #bbb;
  box-shadow: 0 -3px 12px rgba(255, 255, 255, 0.2);
  padding: 12px 0;
  z-index: 1000; /* Ensure it's above canvas */
  transform: translateY(100%); /* Start hidden below */
  transition: transform 0.25s ease-out; /* Animation */
  display: flex;
  justify-content: center;
  box-sizing: border-box; /* Include padding/border in width */
}

/* Class applied when isModalVisible is true */
.colour-modal.modal-visible {
  transform: translateY(0); /* Slide into view */
}

.modal-content {
   display: flex;
   flex-direction: column;
   align-items: center;
   width: auto;
   max-width: 95%; /* Limit width */
}

.colour-palette {
  display: grid;
  /* Adjust columns based on available space or keep fixed */
  grid-template-columns: repeat(8, 1fr); /* 8 columns */
  gap: 6px;
  margin-bottom: 12px;
}

.colour-swatch {
  width: 28px;
  height: 28px;
  border: 1px solid #ccc;
  cursor: pointer;
  border-radius: 4px;
  transition: transform 0.1s ease, box-shadow 0.1s ease;
  box-shadow: 0 1px 2px rgba(0,0,0,0.1);
}

/* highlight the chosen colour */
.colour-swatch.selected {
  border: 3px solid #000000;          /* visible white ring */
  box-shadow: 0 0 0 2px #000 inset;/* thin dark ring inside for contrast */
  box-sizing: border-box;          /* keep the overall size the same */
}
.colour-swatch:hover {
  transform: scale(1.1);
  box-shadow: 0 2px 4px rgba(0,0,0,0.2);
  border-color: #888;
}

.close-modal-btn {
    padding: 6px 14px;
    border: 1px solid #aaa;
    background-color: #eee;
    color: #333;
    cursor: pointer;
    border-radius: 4px;
    font-size: 0.9em;
    transition: background-color 0.1s ease;
}


</style>