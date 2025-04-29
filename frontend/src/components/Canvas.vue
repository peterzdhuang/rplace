<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, reactive, computed, nextTick } from 'vue'

// --- Configuration ---
const PIXEL_SIZE = 15 // Base size of each pixel block
const GRID_WIDTH = 10 // Grid width in pixels
const GRID_HEIGHT = 10 // Grid height in pixels
const WS_URL = 'ws://localhost:8000/ws' // MAKE SURE THIS IS YOUR CORRECT WEBSOCKET URL

// Changed from hex to RGB objects
const COLOR_PALETTE = [
 { r: 255, g: 255, b: 255 }, // White
 { r: 228, g: 228, b: 228 }, // Light Grey
 { r: 136, g: 136, b: 136 }, // Grey
 { r: 34, g: 34, b: 34 },   // Black
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

// --- App State ---
const isLoggedIn = ref(false) // Added for login state
const username = ref('') // Added for login input

// --- Canvas Refs and State ---
const canvasRef = ref<HTMLCanvasElement | null>(null)
const ctxRef = ref<CanvasRenderingContext2D | null>(null)
const canvasWidth = ref(window.innerWidth)
const canvasHeight = ref(window.innerHeight)

// --- Pan & Zoom State ---
const scale = ref(0.5)
const offsetX = ref(0)
const offsetY = ref(0)
const isPanning = ref(false)
const startX = ref(0)
const startY = ref(0)
const hasMoved = ref(false)

// --- Grid & Pixel State ---
const pixels = reactive<{ data: {r: number, g: number, b: number}[][] }>({ data: [] })
const ws = ref<WebSocket | null>(null)
const isLoading = ref(false) // Initially not loading until login attempt
const loadError = ref<string | null>(null)

// --- Modal & Selection State ---
const isModalVisible = ref(false)
const selectedPixelCoords = reactive<{ x:number; y:number }>({ x:-1, y:-1 })
const draftColour = ref<{ r:number; g:number; b:number } | null>(null) // Holds the temporarily selected colour

// --- Computed Properties ---
const canvasContentWidth = computed(() => GRID_WIDTH * PIXEL_SIZE)
const canvasContentHeight = computed(() => GRID_HEIGHT * PIXEL_SIZE)

// --- Coordinate Transformation ---
const screenToGridCoords = (clientX: number, clientY: number): { x: number; y: number } | null => {
  const canvas = canvasRef.value
  if (!canvas) return null
  const rect = canvas.getBoundingClientRect()
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
const rgbToString = (colour: {r: number, g: number, b: number} | null): string => {
  if (!colour) return 'transparent'; // Handle null case if needed
  return `rgb(${colour.r}, ${colour.g}, ${colour.b})`;
}

// --- Login Logic ---
const handleLogin = () => {
  if (!username.value.trim()) {
    alert('Please enter a username.');
    return;
  }
  isLoggedIn.value = true;
  // Wait for the canvas element to be rendered after login
  nextTick(() => {
    setupCanvas(); // Setup canvas context
    connectWebSocket(); // Connect WebSocket AFTER login and canvas setup
  });
};

// --- WebSocket Logic ---
const connectWebSocket = () => {
  if (ws.value || !isLoggedIn.value) return; // Only connect if logged in

  console.log('Attempting to connect to WebSocket:', WS_URL);
  isLoading.value = true; // Start loading now
  loadError.value = null;
  ws.value = new WebSocket(WS_URL);

  ws.value.onopen = () => {
    console.log('WebSocket connection established.');
    // Optional: Send username or token if backend needs it
    // sendWebSocketMessage({ type: 'auth', username: username.value });
    // Optional: Request initial state if needed (backend might send it automatically onopen)
    // sendWebSocketMessage({ type: 'get_initial_state' });
  };

  ws.value.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data);
      console.log('WebSocket message received:', message.type);

      if (message.type === 'init' && Array.isArray(message.pixels)) {
        console.log(`Received initial state with ${message.pixels.length} rows.`);
        if (message.pixels.length > 0 && message.pixels[0]?.length > 0) {
          pixels.data = message.pixels;
          isLoading.value = false;
          loadError.value = null;
          console.log('Initial pixel state loaded.');
          nextTick(() => { // Ensure data is set before centering/drawing
            centerView(); // Center view after getting initial data
            drawCanvas(); // Draw initial state
          });
        } else {
           throw new Error('Received initial state pixel data is empty or invalid.');
        }
      } else if (message.type === 'update') {
        const { x, y, pixel } = message; // Expecting pixel: {r, g, b}
        console.log(`Update received for (${x}, ${y}):`, pixel);
        if (
          pixels.data[y] &&
          pixel && typeof pixel === 'object' &&
          x >= 0 && x < GRID_WIDTH &&
          y >= 0 && y < GRID_HEIGHT
        ) {
            // Update the reactive data structure - Vue handles reactivity
          pixels.data[y][x] = pixel;
          drawCanvas()
        } else {
            console.warn('Invalid update received:', message);
        }
      } else {
        console.warn('Unknown or unhandled WebSocket message type:', message.type);
      }
    } catch (error: any) {
      console.error('Failed to parse WebSocket message or handle update:', error);
      loadError.value = `Error processing message: ${error.message || 'Unknown error'}`;
      isLoading.value = false;
      if (!pixels.data.length) initializeEmptyGrid(true);
      drawCanvas(); // Draw the error state or empty grid
    }
  };

  ws.value.onerror = (error) => {
    console.error('WebSocket error:', error);
    loadError.value = 'WebSocket connection error. Please try refreshing.';
    isLoading.value = false;
    ws.value = null;
    if (!pixels.data.length) initializeEmptyGrid(true);
    drawCanvas(); // Draw error state
  };

  ws.value.onclose = (event) => {
    console.log('WebSocket connection closed:', event.code, event.reason);
    if (!event.wasClean && !loadError.value) {
       loadError.value = 'WebSocket connection closed unexpectedly.';
    }
    isLoading.value = false;
    ws.value = null;
    // Don't initialize empty grid here necessarily, user might be logged out
     if (isLoggedIn.value && !pixels.data.length) {
         initializeEmptyGrid(true); // Initialize if logged in and data is missing
     }
     drawCanvas(); // Draw current state (maybe error message)
  };
};

const sendWebSocketMessage = (message: object) => {
  if (ws.value && ws.value.readyState === WebSocket.OPEN) {
    ws.value.send(JSON.stringify(message));
    console.log("WS message sent: ", message);
  } else {
    console.error('WebSocket is not connected or not open.');
    loadError.value = 'Cannot send update: Disconnected from server.'; // Notify user
    // Optional: Attempt to reconnect automatically, but be careful of loops
     // if (!ws.value && isLoggedIn.value) connectWebSocket();
     drawCanvas(); // Redraw to show error message
  }
};

const centerView = () => {
   const canvas = canvasRef.value;
   if (!canvas || !pixels.data.length) return; // Don't center if canvas/data not ready

   // Ensure canvas dimensions are known
   const viewWidth = canvas.clientWidth;
   const viewHeight = canvas.clientHeight;
   if (viewWidth === 0 || viewHeight === 0) {
       console.warn("Cannot center view: Canvas dimensions are zero.");
       return; // Avoid division by zero or nonsensical centering
   }

   // Scale to fit the entire grid (or a portion) initially
   const fitScale = Math.min(
     viewWidth / canvasContentWidth.value,
     viewHeight / canvasContentHeight.value
   ) * 0.9; // Add some padding
   scale.value = Math.max(0.1, fitScale); // Limit minimum zoom

   offsetX.value = (viewWidth - canvasContentWidth.value * scale.value) / 2;
   offsetY.value = (viewHeight - canvasContentHeight.value * scale.value) / 2;
   console.log(`Centering view. Scale: ${scale.value}, Offset: (${offsetX.value}, ${offsetY.value})`);
}

// --- Event Handlers ---
const handleMouseDown = (e: MouseEvent) => {
  if (!isLoggedIn.value) return; // Ignore clicks if not logged in
  // Prevent starting pan if clicking inside the modal
  if ((e.target as HTMLElement)?.closest('.colour-modal')) return;
  isPanning.value = true;
  hasMoved.value = false;
  startX.value = e.clientX - offsetX.value;
  startY.value = e.clientY - offsetY.value;
  if (canvasRef.value) canvasRef.value.style.cursor = 'grabbing';
};

const handleMouseMove = (e: MouseEvent) => {
  if (!isLoggedIn.value || !isPanning.value) return;

  const currentX = e.clientX - startX.value;
  const currentY = e.clientY - startY.value;
  // Add a threshold to distinguish click from drag
  if (!hasMoved.value && (Math.abs(e.clientX - (startX.value + offsetX.value)) > 5 || Math.abs(e.clientY - (startY.value + offsetY.value)) > 5)) {
    hasMoved.value = true;
  }
  if (hasMoved.value) {
    offsetX.value = currentX;
    offsetY.value = currentY;
    requestAnimationFrame(drawCanvas); // Redraw during panning
  }
};

const handleMouseUp = (e: MouseEvent) => {
  if (!isLoggedIn.value) return;

  // Check if mouseup occurred over the modal - if so, do nothing here
  if ((e.target as HTMLElement)?.closest('.colour-modal')) {
     if (isPanning.value) {
       isPanning.value = false;
       if (canvasRef.value) canvasRef.value.style.cursor = 'grab';
     }
     return; // Don't process click if mouse up is on modal
  }

  if (isPanning.value) {
    if (!hasMoved.value && e.target === canvasRef.value) {
      handleCanvasClick(e); // Treat as click if no movement
    }
    isPanning.value = false;
    if (canvasRef.value) canvasRef.value.style.cursor = 'grab';
  }
  hasMoved.value = false;
  // Optional: Redraw on mouse up even if not panning, to potentially show updates
  // requestAnimationFrame(drawCanvas);
};

const handleCanvasClick = (e: MouseEvent) => {
  if (!isLoggedIn.value) return;

  const coords = screenToGridCoords(e.clientX, e.clientY);
  if (coords) {
    const previousX = selectedPixelCoords.x;
    const previousY = selectedPixelCoords.y;

    // Update selection
    selectedPixelCoords.x = coords.x;
    selectedPixelCoords.y = coords.y;
    draftColour.value = null; // Reset draft colour when selecting new pixel
    isModalVisible.value = true; // Show the modal
    console.log(`Selected pixel: (${coords.x}, ${coords.y})`);

    // Redraw needed to show new selection highlight and remove old one
     // This will also display any pending updates received via WebSocket
    requestAnimationFrame(drawCanvas);

  } else {
    // Click outside grid - close modal if open
    if (isModalVisible.value) {
       closeModal();
    }
    // Optional: Redraw even if clicking outside grid, to potentially show updates
    // requestAnimationFrame(drawCanvas);
  }
};

const handleWheel = (e: WheelEvent) => {
  if (!isLoggedIn.value) return;

  e.preventDefault();
  const canvas = canvasRef.value;
  if (!canvas) return;

  const rect = canvas.getBoundingClientRect();
  const mouseX = e.clientX - rect.left;
  const mouseY = e.clientY - rect.top;

  const worldXBefore = (mouseX - offsetX.value) / scale.value;
  const worldYBefore = (mouseY - offsetY.value) / scale.value;

  const scaleFactor = e.deltaY < 0 ? 1.1 : 1 / 1.1;
  const newScale = Math.max(0.05, Math.min(scale.value * scaleFactor, 50)); // Clamp zoom

  offsetX.value = mouseX - worldXBefore * newScale;
  offsetY.value = mouseY - worldYBefore * newScale;
  scale.value = newScale;

  requestAnimationFrame(drawCanvas); // Redraw on zoom
};

// --- Modal Actions ---

// Called when a colour swatch is clicked in the modal
const chooseColour = (colour: {r:number; g:number; b:number}) => {
  console.log("Draft colour chosen:", colour)
  draftColour.value = colour; // Set the draft colour, DON'T send yet
}

// Called when the 'Confirm' button is clicked
const confirmColour = () => {
  if (
    draftColour.value &&
    selectedPixelCoords.x !== -1 &&
    selectedPixelCoords.y !== -1
  ) {
    console.log(`Confirming colour ${rgbToString(draftColour.value)} at (${selectedPixelCoords.x}, ${selectedPixelCoords.y})`);
    // Optimistically update local state (Vue handles reactivity)
    pixels.data[selectedPixelCoords.y][selectedPixelCoords.x] = draftColour.value;

     // Send the update message to the WebSocket server
    sendWebSocketMessage({
      type: 'update',
      x: selectedPixelCoords.x,
      y: selectedPixelCoords.y,
      pixel: {...draftColour.value} // Send a copy
    });

     // Close modal and redraw to show the optimistic update immediately
    closeModal();
     requestAnimationFrame(drawCanvas); // Explicitly redraw after optimistic update + close

  } else {
    console.warn("Confirm clicked but no draft colour or pixel selected.");
  }
}

// Called when 'Cancel' is clicked or modal is closed otherwise
const closeModal = () => {
  const needsRedraw = selectedPixelCoords.x !== -1; // Need redraw if something was selected
  isModalVisible.value = false;
  selectedPixelCoords.x = -1; // Deselect pixel
  selectedPixelCoords.y = -1;
  draftColour.value = null; // Clear the draft colour
  if (needsRedraw) {
     // Redraw to remove selection highlight - this might also show pending WS updates
      requestAnimationFrame(drawCanvas);
  }
};

// --- Drawing Logic ---
const initializeEmptyGrid = (showWarning = false) => {
   if(showWarning) console.warn(`Initializing fallback empty ${GRID_WIDTH}x${GRID_HEIGHT} grid locally.`);
   const white = { r: 255, g: 255, b: 255 };
   pixels.data = Array.from({ length: GRID_HEIGHT }, () =>
       Array(GRID_WIDTH).fill(null).map(() => ({ ...white })) // Ensure deep copy
   );
   isLoading.value = false;
};

const drawCanvas = () => {
  const canvas = canvasRef.value;
  const ctx = ctxRef.value;
  // Only draw if logged in and canvas is ready
  if (!isLoggedIn.value || !canvas || !ctx) {
     console.log("Draw skipped: Not logged in or canvas/context not ready.");
     return;
  }

  const dpr = window.devicePixelRatio || 1;
  const viewWidth = canvas.clientWidth;
  const viewHeight = canvas.clientHeight;

  // Set actual buffer size based on DPR if it changed
  if (canvas.width !== viewWidth * dpr || canvas.height !== viewHeight * dpr) {
      canvas.width = viewWidth * dpr;
      canvas.height = viewHeight * dpr;
      console.log(`Canvas buffer resized to: ${canvas.width}x${canvas.height} (DPR: ${dpr})`);
  }

  // Clear canvas
  ctx.save();
  ctx.resetTransform();
  // Explicitly set the fill style before clearing for the canvas background
  ctx.fillStyle = '#ffffff'; // Match the CSS background for the canvas element itself
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.restore();

  // Handle loading/error state overlay (drawn without transforms)
  if (isLoading.value || loadError.value) {
      ctx.save();
      ctx.scale(dpr, dpr); // Scale context for drawing text consistently
      ctx.font = "16px Arial";
      ctx.fillStyle = "#ffffff";
      ctx.textAlign = "center";
      const message = loadError.value || "Loading pixels...";
      ctx.fillText(message, viewWidth / 2, viewHeight / 2);
      ctx.restore();
      return; // Don't draw grid if loading/error
  }

  // If we got here, we are logged in, not loading, and have no errors, but maybe no data yet
  if (!pixels.data || pixels.data.length === 0) {
      console.warn("Draw skipped: Pixel data is not available yet.");
      // Optionally draw a "waiting for data" message
       ctx.save();
       ctx.scale(dpr, dpr);
       ctx.font = "16px Arial";
       ctx.fillStyle = "#ffffff";
       ctx.textAlign = "center";
       ctx.fillText("Waiting for pixel data...", viewWidth / 2, viewHeight / 2);
       ctx.restore();
      return;
  }


  // Apply transforms for drawing grid content
  ctx.save();
  ctx.scale(dpr, dpr); // Apply DPR scaling for all drawing operations
  ctx.translate(offsetX.value, offsetY.value);
  ctx.scale(scale.value, scale.value);

  // --- Viewport Culling ---
  const viewLeftWorld = -offsetX.value / scale.value;
  const viewTopWorld = -offsetY.value / scale.value;
  const viewRightWorld = (viewWidth - offsetX.value) / scale.value;
  const viewBottomWorld = (viewHeight - offsetY.value) / scale.value;

  const startCol = Math.max(0, Math.floor(viewLeftWorld / PIXEL_SIZE));
  const endCol = Math.min(GRID_WIDTH, Math.ceil(viewRightWorld / PIXEL_SIZE));
  const startRow = Math.max(0, Math.floor(viewTopWorld / PIXEL_SIZE));
  const endRow = Math.min(GRID_HEIGHT, Math.ceil(viewBottomWorld / PIXEL_SIZE));

  const scaledPixelSize = PIXEL_SIZE * scale.value;
  const drawGridLines = scaledPixelSize > MIN_ZOOM_FOR_GRID_LINES;

  const defaultLineWidth = 0.5 / scale.value; // Adjusted for scale
  const defaultStrokeStyle = '#000000';
  const selectedLineWidth = 1.5 / scale.value; // Adjusted for scale
  const selectedStrokeStyleOuter = '#FFFFFF'; 
  const selectedStrokeStyleInner = '#000000'; 

  // --- Draw Visible Pixels ---
  for (let y = startRow; y < endRow; y++) {
    if (!pixels.data[y]) {
        // console.warn(`Row ${y} not found in pixel data during draw.`);
        continue; // Safety check
    }
    for (let x = startCol; x < endCol; x++) {
      const colour = pixels.data[y][x];
      if (!colour) {
        // console.warn(`Pixel data missing at (${x}, ${y}) during draw.`);
        ctx.fillStyle = 'rgb(255,255,255)'; // Draw black if data unexpectedly missing
      } else {
        ctx.fillStyle = rgbToString(colour);
      }
      ctx.fillRect(x * PIXEL_SIZE, y * PIXEL_SIZE, PIXEL_SIZE, PIXEL_SIZE);
    }
  }

  // --- Draw Grid Lines (if zoomed in enough) ---
  if (drawGridLines) {
      ctx.beginPath();
      ctx.lineWidth = defaultLineWidth;
      ctx.strokeStyle = defaultStrokeStyle;
      // Vertical lines
      for (let x = startCol; x <= endCol; x++) {
          ctx.moveTo(x * PIXEL_SIZE, startRow * PIXEL_SIZE);
          ctx.lineTo(x * PIXEL_SIZE, endRow * PIXEL_SIZE);
      }
      // Horizontal lines
      for (let y = startRow; y <= endRow; y++) {
          ctx.moveTo(startCol * PIXEL_SIZE, y * PIXEL_SIZE);
          ctx.lineTo(endCol * PIXEL_SIZE, y * PIXEL_SIZE);
      }
      ctx.stroke();
  }

  // --- Draw Selection Highlight (double border for visibility) ---
  if (
    selectedPixelCoords.x !== -1 && selectedPixelCoords.y !== -1 && // Ensure selection exists
    selectedPixelCoords.x >= startCol && selectedPixelCoords.x < endCol &&
    selectedPixelCoords.y >= startRow && selectedPixelCoords.y < endRow // Ensure selection is visible
  ) {
     // Calculate stroke positions carefully to avoid blurry lines due to scaling
     const strokeX = selectedPixelCoords.x * PIXEL_SIZE;
     const strokeY = selectedPixelCoords.y * PIXEL_SIZE;
     const strokeW = PIXEL_SIZE;
     const strokeH = PIXEL_SIZE;

     // Outer White Border (draw slightly outside the pixel bounds)
     ctx.strokeStyle = selectedStrokeStyleOuter;
     ctx.lineWidth = selectedLineWidth;
     // Adjust coordinates by half the line width to center the stroke *on* the edge
     ctx.strokeRect(
         strokeX - selectedLineWidth / 2,
         strokeY - selectedLineWidth / 2,
         strokeW + selectedLineWidth,
         strokeH + selectedLineWidth
     );

     // Inner Dark Border (draw slightly inside the pixel bounds)
     ctx.strokeStyle = selectedStrokeStyleInner;
     ctx.lineWidth = selectedLineWidth / 2; // Make inner border thinner
     // Adjust coordinates by half the (thinner) line width
     ctx.strokeRect(
         strokeX + selectedLineWidth / 4,
         strokeY + selectedLineWidth / 4,
         strokeW - selectedLineWidth / 2,
         strokeH - selectedLineWidth / 2
     );
  }

  ctx.restore(); // Restore from DPR scale + transforms
};


// --- Lifecycle Hooks ---
const updateCanvasSize = () => {
  canvasWidth.value = window.innerWidth;
  canvasHeight.value = window.innerHeight;

  nextTick(() => {
    const canvas = canvasRef.value;
    if (canvas) {
      // Set display size
      canvas.style.width = `${canvasWidth.value}px`;
      canvas.style.height = `${canvasHeight.value}px`;
      // Buffer size handled in drawCanvas, which will be called if needed
       if(isLoggedIn.value) { // Only redraw if logged in
           requestAnimationFrame(drawCanvas);
       }
    }
  });
};

// Sets up canvas context and listeners - called after login
const setupCanvas = () => {
  const canvas = canvasRef.value;
  if (canvas && !ctxRef.value) { // Only setup if not already done
    const ctx = canvas.getContext('2d', { alpha: false });
    if (ctx) {
      ctxRef.value = ctx;
      console.log('Canvas context obtained.');
      updateCanvasSize(); // Set initial size

      // Add event listeners
      canvas.addEventListener('mousedown', handleMouseDown);
      canvas.addEventListener('mousemove', handleMouseMove);
      window.addEventListener('mouseup', handleMouseUp); // Use window for mouseup
      canvas.addEventListener('wheel', handleWheel, { passive: false });

    } else {
      console.error("Failed to get 2D rendering context.");
      loadError.value = "Canvas context not available.";
      isLoading.value = false; // Stop loading if context fails
      drawCanvas(); // Attempt to draw error state
    }
  } else if (!canvas) {
      console.error("Canvas element not found during setupCanvas.");
      loadError.value = "Canvas element failed to mount properly after login.";
      isLoading.value = false;
  }
};

onMounted(() => {
  // Don't setup canvas or connect WS here anymore.
  // Wait for login.
  // We still need the resize listener for the overall page layout.
  window.addEventListener('resize', updateCanvasSize);
  // Initial size update for potential login screen styling
  updateCanvasSize();
});

onBeforeUnmount(() => {
  const canvas = canvasRef.value;
  if (canvas) {
    canvas.removeEventListener('mousedown', handleMouseDown);
    canvas.removeEventListener('mousemove', handleMouseMove);
    canvas.removeEventListener('wheel', handleWheel);
  }
  // Mouseup listener is on window
  window.removeEventListener('mouseup', handleMouseUp);
  window.removeEventListener('resize', updateCanvasSize);

  if (ws.value) {
    ws.value.onclose = null;
    ws.value.onerror = null;
    ws.value.onmessage = null;
    ws.value.onopen = null;
    ws.value.close();
    console.log('WebSocket connection closed during unmount.');
  }
  // Reset state if component is unmounted
  isLoggedIn.value = false;
  username.value = '';
  ctxRef.value = null;
  pixels.data = [];
  loadError.value = null;
  isLoading.value = false;
});
</script>

<template>
  <div class="canvas-wrapper">
    <div v-if="!isLoggedIn" class="login-container">
        <div class="login-form">
            <h2>Enter Canvas</h2>
            <input
                type="text"
                v-model="username"
                placeholder="Username"
                @keyup.enter="handleLogin"
                class="login-input"
            />
            <button @click="handleLogin" class="login-button">Enter</button>
        </div>
    </div>

    <div v-else class="main-content-area">
        <canvas
          ref="canvasRef"
          class="main-canvas"
          :width="canvasWidth"
          :height="canvasHeight"
          > </canvas>

        <div class="colour-modal"
             :class="{ 'modal-visible': isModalVisible }"
             @mousedown.stop @click.stop @wheel.stop> <div class="modal-content">
            <div class="colour-palette">
              <div v-for="colour in COLOR_PALETTE"
                   :key="`${colour.r}-${colour.g}-${colour.b}`"
                   class="colour-swatch"
                   :class="{ selected: draftColour && colour.r === draftColour.r && colour.g === draftColour.g && colour.b === draftColour.b }"
                   :style="{ backgroundColor: rgbToString(colour) }"
                   @click="chooseColour(colour)">
              </div>
            </div>
            <div class="modal-actions">
              <button class="cancel-btn" @click="closeModal">Cancel</button>
              <button class="confirm-btn"
                      :disabled="!draftColour"
                      @click="confirmColour">Confirm</button>
            </div>
          </div>
        </div>
    </div>   </div>
</template>

<style scoped>
/* General Wrapper - Now Black Background */
.canvas-wrapper {
  position: relative;
  width: 100vw;
  height: 100vh;
  background-color: #ffffff; /* Changed background to black */
  overflow: hidden; /* Prevent scrollbars */
  display: flex;
  justify-content: center;
  align-items: center;
}

/* Login Screen Styles */
.login-container {
    display: flex;
    justify-content: center;
    align-items: center;
    width: 100%;
    height: 100%;
}

.login-form {
    background-color: #f0f0f0;
    padding: 30px 40px;
    border-radius: 8px;
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.2);
    text-align: center;
    border: 1px solid #ccc;
}

.login-form h2 {
    margin-bottom: 20px;
    color: #333;
}

.login-input {
    width: 100%;
    padding: 12px;
    margin-bottom: 20px;
    border: 1px solid #ccc;
    border-radius: 4px;
    font-size: 1rem;
    box-sizing: border-box; /* Include padding in width */
}

.login-button {
    padding: 12px 25px;
    background-color: #007bff;
    color: white;
    border: none;
    border-radius: 4px;
    font-size: 1rem;
    cursor: pointer;
    transition: background-color 0.2s ease;
}

.login-button:hover {
    background-color: #0056b3;
}

/* Container for Canvas and Modal when logged in */
.main-content-area {
    background-color: #ffffff;
    width: 100%;
    height: 100%;
    position: relative; /* Needed for modal positioning relative to this */
    display: flex; /* Use flex if needed, or just let canvas fill */
    justify-content: center;
    align-items: center;
}

/* Main Canvas Styles */
.main-canvas {
  display: block;
  cursor: grab;
  background-color: #c72323; /* Canvas background BEFORE drawing occurs */
  image-rendering: pixelated;
  image-rendering: crisp-edges;
  /* width/height style set dynamically */
}

.main-canvas:active {
    cursor: grabbing;
}

.main-canvas::selection { background: transparent; }
.main-canvas::-moz-selection { background: transparent; }


/* Colour Modal Styles */
.colour-modal {
  position: fixed; /* Fixed relative to viewport */
  bottom: 0;
  left: 0;
  right: 0;
  width: 100%;
  background-color: rgba(240, 240, 240, 0.95);
  border-top: 1px solid #bbb;
  box-shadow: 0 -4px 15px rgba(0, 0, 0, 0.15);
  padding: 15px 0;
  z-index: 1000;
  transform: translateY(100%); /* Start hidden */
  transition: transform 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
  display: flex;
  justify-content: center;
  box-sizing: border-box;
}

.colour-modal.modal-visible {
  transform: translateY(0); /* Slide in */
}

.modal-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: auto;
  max-width: 90%;
}

.colour-palette {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(30px, 1fr));
  gap: 8px;
  margin-bottom: 15px;
  padding: 0 10px;
  width: 100%;
  max-width: 400px;
  box-sizing: border-box;
}

.colour-swatch {
  width: 30px;
  height: 30px;
  border: 1px solid #ccc;
  cursor: pointer;
  border-radius: 5px;
  transition: all 0.15s ease;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  box-sizing: border-box;
}

.colour-swatch:hover {
  transform: scale(1.1);
  box-shadow: 0 3px 6px rgba(0,0,0,0.2);
  border-color: #888;
}

.colour-swatch.selected {
  border: 2px solid #ffffff;
  outline: 2px solid #007bff;
  box-shadow: 0 0 0 3px rgba(0, 123, 255, 0.3);
  transform: scale(1.05);
}

.modal-actions {
  display: flex;
  gap: 15px;
}

.modal-actions button {
  padding: 10px 20px;
  border-radius: 5px;
  border: none;
  font-size: 1rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s ease, box-shadow 0.2s ease, transform 0.1s ease;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}

.modal-actions button:hover {
    box-shadow: 0 2px 5px rgba(0,0,0,0.2);
    transform: translateY(-1px);
}
.modal-actions button:active {
    transform: translateY(0px);
    box-shadow: 0 1px 2px rgba(0,0,0,0.15);
}


.modal-actions .cancel-btn {
  background-color: #f8f9fa;
  border: 1px solid #ced4da;
  color: #495057;
}
.modal-actions .cancel-btn:hover {
  background-color: #e9ecef;
}

.modal-actions .confirm-btn {
  background-color: #007bff;
  color: white;
}
.modal-actions .confirm-btn:hover {
  background-color: #0056b3;
}

.modal-actions .confirm-btn:disabled {
  background-color: #cccccc;
  color: #666666;
  cursor: not-allowed;
  box-shadow: none;
  transform: none;
}

</style>