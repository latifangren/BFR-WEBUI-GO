<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import {
    Smartphone,
    Play,
    Square,
    Camera,
    Maximize2,
    Minimize2,
    SlidersHorizontal,
    Keyboard,
    ArrowLeft,
    Home,
    Menu,
    Power,
    Volume1,
    Volume2,
    VolumeX,
    Eye,
    EyeOff,
    Send,
    Activity,
    Zap,
    Cpu,
    Monitor,
    RefreshCw,
    Sparkles,
    Check,
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'
  import { WebSocketClient } from '../../../ws/socket'
  import { toastStore } from '../../../stores/toast.svelte'

  // --- Runtime Streaming & Device State ---
  let isMirroring = $state(false)
  let isConnecting = $state(false)
  let fps = $state(0)
  let totalFrames = $state(0)
  let streamEngine = $state<'webcodecs' | 'mjpeg' | 'idle'>('idle')
  let detectedWidth = $state(1080)
  let detectedHeight = $state(2400)
  let streamWidth = $state(1080)
  let streamHeight = $state(2400)
  let codecName = $state('avc1.640033')
  let hardwareEncoderName = $state('Qualcomm c2.qti.avc (Hardware GPU/VPU)')

  // --- Stream Customization Configurations ---
  let maxSize = $state(1280) // Native (0), 1440p (2560), 1080p (1920), 720p (1280), 540p (960)
  let bitRate = $state(8000000) // 2M, 4M, 8M, 16M, 24M, 32M or custom
  let customBitrateMbps = $state(8)
  let maxFps = $state(60) // 30 FPS vs 60 FPS
  let displayId = $state(0) // Main (0) vs Cover (1)
  let turnScreenOff = $state(false)
  let stayAwake = $state(true)
  let settingsChanged = $state(false)

  // --- Interactive Control States ---
  let isPhysicalScreenOff = $state(false)
  let isKeyboardActive = $state(false)
  let isFullscreen = $state(false)
  let showSettings = $state(true)
  let textInput = $state('')

  // --- DOM & Engine References ---
  let canvasEl: HTMLCanvasElement | null = $state(null)
  let phoneStageEl: HTMLDivElement | null = $state(null)
  let wsClient: WebSocketClient | null = null
  let videoDecoder: any = null
  let cachedSpsPps: Uint8Array | null = null
  let isDecoderConfigured = false

  let frameCounter = 0
  let fpsInterval: ReturnType<typeof setInterval> | null = null
  let lastMoveTime = 0

  // Dynamic Aspect Ratio calculated from stream resolution
  let aspectRatio = $derived(
    detectedWidth > 0 && detectedHeight > 0
      ? `${detectedWidth} / ${detectedHeight}`
      : '9 / 19.5'
  )

  // Resolution options
  const resolutionOptions = [
    { label: 'Native 1:1', value: 0, desc: 'Original (No Scale)' },
    { label: '1440p', value: 2560, desc: 'Ultra (2560)' },
    { label: '1080p', value: 1920, desc: 'FHD (1920)' },
    { label: '720p', value: 1280, desc: 'HD (1280)' },
    { label: '540p', value: 960, desc: 'Fast (960)' },
  ]

  // Bitrate options
  const bitrateOptions = [
    { label: '2M', value: 2000000, desc: 'Low' },
    { label: '4M', value: 4000000, desc: 'Balanced' },
    { label: '8M', value: 8000000, desc: 'High' },
    { label: '16M', value: 16000000, desc: 'Very High' },
    { label: '24M', value: 24000000, desc: 'Ultra' },
    { label: '32M', value: 32000000, desc: 'Near Lossless' },
  ]

  onMount(() => {
    window.addEventListener('keydown', handleGlobalKeyDown)
    document.addEventListener('fullscreenchange', handleFullscreenChange)

    return () => {
      cleanup()
    }
  })

  onDestroy(() => {
    cleanup()
  })

  function cleanup() {
    window.removeEventListener('keydown', handleGlobalKeyDown)
    document.removeEventListener('fullscreenchange', handleFullscreenChange)
    stopMirroring()
  }

  function handleFullscreenChange() {
    isFullscreen = !!document.fullscreenElement
  }

  // --- Helper: Parse H.264 SPS profile & level to RFC 6381 codec string ---
  function parseH264Sps(nalData: Uint8Array): string | null {
    let i = 0
    while (i < nalData.length - 4) {
      if (
        (nalData[i] === 0 && nalData[i + 1] === 0 && nalData[i + 2] === 1) ||
        (nalData[i] === 0 && nalData[i + 1] === 0 && nalData[i + 2] === 0 && nalData[i + 3] === 1)
      ) {
        const offset = nalData[i + 2] === 1 ? i + 3 : i + 4
        const nalType = nalData[offset] & 0x1f
        if (nalType === 7 && offset + 3 < nalData.length) {
          const profileIdc = nalData[offset + 1]
          const constraintFlags = nalData[offset + 2]
          const levelIdc = nalData[offset + 3]
          const toHex = (n: number) => n.toString(16).padStart(2, '0')
          return `avc1.${toHex(profileIdc)}${toHex(constraintFlags)}${toHex(levelIdc)}`
        }
      }
      i++
    }
    return null
  }

  // Helper: extract 64-bit big-endian PTS safely
  function parsePts(view: DataView): number {
    try {
      return Number(view.getBigUint64(1, false))
    } catch {
      const hi = view.getUint32(1, false)
      const lo = view.getUint32(5, false)
      return hi * 4294967296 + lo
    }
  }

  // --- WebCodecs Engine Initialization & Configuration ---
  function initOrConfigureDecoder(codec: string) {
    if (typeof window === 'undefined' || !('VideoDecoder' in window)) {
      console.warn('[Scrcpy] Browser does not support WebCodecs VideoDecoder. Using fallback.')
      return false
    }

    try {
      if (!videoDecoder || videoDecoder.state === 'closed') {
        // @ts-ignore - VideoDecoder global
        videoDecoder = new VideoDecoder({
          output: (frame: any) => {
            renderVideoFrame(frame)
          },
          error: (err: any) => {
            console.error('[WebCodecs] Decoder error:', err)
          },
        })
      }

      if (videoDecoder.state === 'unconfigured' || !isDecoderConfigured) {
        videoDecoder.configure({
          codec: codec || 'avc1.640033',
          optimizeForLatency: true,
          hardwareAcceleration: 'prefer-hardware',
        })
        isDecoderConfigured = true
      }
      return true
    } catch (err) {
      console.warn('[WebCodecs] Configuration error:', err)
      return false
    }
  }

  // Render decoded video frame to canvas
  function renderVideoFrame(frame: any) {
    try {
      if (canvasEl) {
        const frameW = frame.displayWidth || frame.codedWidth || streamWidth
        const frameH = frame.displayHeight || frame.codedHeight || streamHeight

        if (canvasEl.width !== frameW || canvasEl.height !== frameH) {
          canvasEl.width = frameW
          canvasEl.height = frameH
          detectedWidth = frameW
          detectedHeight = frameH
        }

        const ctx = canvasEl.getContext('2d', { alpha: false, desynchronized: true })
        if (ctx) {
          ctx.drawImage(frame, 0, 0, frameW, frameH)
        }
      }
      frameCounter++
      totalFrames++
      if (isConnecting) {
        isConnecting = false
      }
    } catch (e) {
      console.error('[Scrcpy] Frame render error:', e)
    } finally {
      try {
        frame.close()
      } catch {}
    }
  }

  // --- WebSocket Stream Lifecycle ---
  function startMirroring() {
    if (wsClient && wsClient.readyState === WebSocket.OPEN) {
      return
    }

    stopMirroring()
    isConnecting = true
    totalFrames = 0
    frameCounter = 0
    fps = 0
    settingsChanged = false

    const queryParams = new URLSearchParams({
      max_size: maxSize.toString(),
      bit_rate: bitRate.toString(),
      max_fps: maxFps.toString(),
      display_id: displayId.toString(),
      stay_awake: stayAwake ? 'true' : 'false',
      screen_off: turnScreenOff ? 'true' : 'false',
    })

    const path = `/api/scrcpy/ws?${queryParams.toString()}`

    try {
      wsClient = new WebSocketClient({
        path,
        autoReconnect: false,
        binaryType: 'arraybuffer',
        onOpen: () => {
          isConnecting = false
          isMirroring = true
          streamEngine = 'webcodecs'
          initOrConfigureDecoder(codecName)

          // Start 1-second FPS ticker
          if (fpsInterval) clearInterval(fpsInterval)
          fpsInterval = setInterval(() => {
            fps = frameCounter
            frameCounter = 0
          }, 1000)

          toastStore.success('Hardware screen stream connected', 'Scrcpy Live')
        },
        onMessage: (data) => {
          handleIncomingMessage(data)
        },
        onClose: () => {
          stopMirroring()
        },
        onError: () => {
          stopMirroring()
          toastStore.error('Connection to scrcpy server lost', 'Connection Error')
        },
      })

      wsClient.connect()
    } catch (err) {
      console.error('[Scrcpy] Connect failed:', err)
      stopMirroring()
    }
  }

  function handleIncomingMessage(data: string | ArrayBuffer | Blob) {
    // 1. Text Message: Metadata JSON
    if (typeof data === 'string') {
      try {
        const meta = JSON.parse(data)
        if (meta.type === 'metadata') {
          if (meta.width) detectedWidth = meta.width
          if (meta.height) detectedHeight = meta.height
          if (meta.videoWidth) streamWidth = meta.videoWidth
          if (meta.videoHeight) streamHeight = meta.videoHeight
          if (meta.engine === 'mjpeg') {
            streamEngine = 'mjpeg'
          } else {
            streamEngine = 'webcodecs'
          }
          if (meta.codec) {
            codecName = meta.codec
          }
          if (meta.encoder) {
            hardwareEncoderName = meta.encoder
          }
          if (canvasEl) {
            canvasEl.width = meta.videoWidth || meta.width
            canvasEl.height = meta.videoHeight || meta.height
          }
        }
      } catch (err) {
        console.warn('[Scrcpy] Malformed metadata JSON:', err)
      }
      return
    }

    // 2. Binary Message: WebCodecs Annex-B H.264 or Fallback MJPEG
    if (data instanceof ArrayBuffer) {
      const u8 = new Uint8Array(data)
      if (u8.length === 0) return

      // Fallback check: JPEG SOI (0xFF, 0xD8) or engine is mjpeg
      if (streamEngine === 'mjpeg' || (u8.length > 2 && u8[0] === 0xff && u8[1] === 0xd8)) {
        streamEngine = 'mjpeg'
        renderJpegFallback(data)
        return
      }

      // H.264 WebCodecs packet format:
      // [0]: Flag (2 = Config SPS/PPS, 1 = Keyframe, 0 = Delta)
      // [1..8]: PTS (uint64 big-endian)
      // [9..]: NAL bytes
      if (u8.length < 9) return
      const flag = u8[0]
      const view = new DataView(data)
      const pts = parsePts(view)
      const nalData = u8.subarray(9)

      if (flag === 2) {
        // CONFIG packet (SPS/PPS)
        cachedSpsPps = nalData
        const detectedCodec = parseH264Sps(nalData)
        if (detectedCodec) {
          codecName = detectedCodec
        }
        initOrConfigureDecoder(codecName)
        return
      }

      if (flag === 1 || flag === 0) {
        if (!videoDecoder || videoDecoder.state !== 'configured') {
          initOrConfigureDecoder(codecName)
        }

        // Assemble NAL chunk. Prepend SPS/PPS if this is an IDR keyframe
        let chunkData: Uint8Array
        if (flag === 1 && cachedSpsPps && cachedSpsPps.length > 0) {
          chunkData = new Uint8Array(cachedSpsPps.length + nalData.length)
          chunkData.set(cachedSpsPps, 0)
          chunkData.set(nalData, cachedSpsPps.length)
        } else {
          chunkData = nalData
        }

        try {
          // @ts-ignore - EncodedVideoChunk global
          const chunk = new EncodedVideoChunk({
            type: flag === 1 ? 'key' : 'delta',
            timestamp: pts,
            data: chunkData,
          })

          if (videoDecoder && videoDecoder.state === 'configured') {
            videoDecoder.decode(chunk)
          }
        } catch (err) {
          console.warn('[WebCodecs] Error decoding chunk:', err)
        }
      }
    } else if (data instanceof Blob) {
      renderJpegFallback(data)
    }
  }

  function renderJpegFallback(source: ArrayBuffer | Blob) {
    const blob = source instanceof Blob ? source : new Blob([source], { type: 'image/jpeg' })
    if (typeof createImageBitmap === 'function') {
      createImageBitmap(blob)
        .then((bmp) => {
          if (canvasEl) {
            if (canvasEl.width !== bmp.width || canvasEl.height !== bmp.height) {
              canvasEl.width = bmp.width
              canvasEl.height = bmp.height
              detectedWidth = bmp.width
              detectedHeight = bmp.height
            }
            const ctx = canvasEl.getContext('2d', { alpha: false })
            if (ctx) {
              ctx.drawImage(bmp, 0, 0)
            }
          }
          bmp.close()
          frameCounter++
          totalFrames++
        })
        .catch(() => {})
    }
  }

  function stopMirroring() {
    if (wsClient) {
      wsClient.disconnect()
      wsClient = null
    }

    if (videoDecoder) {
      try {
        if (videoDecoder.state !== 'closed') {
          videoDecoder.close()
        }
      } catch {}
      videoDecoder = null
    }

    if (fpsInterval) {
      clearInterval(fpsInterval)
      fpsInterval = null
    }

    isDecoderConfigured = false
    cachedSpsPps = null
    isMirroring = false
    isConnecting = false
    fps = 0
    streamEngine = 'idle'
  }

  // --- Remote Input & Control Events ---
  function sendEvent(evt: Record<string, unknown>) {
    if (wsClient && wsClient.readyState === WebSocket.OPEN) {
      wsClient.send(JSON.stringify(evt))
    }
  }

  function sendNavKey(action: string, keycode = 0) {
    sendEvent({ action, keycode })
  }

  function sendText() {
    if (!textInput.trim()) return
    sendEvent({ action: 'text', text: textInput })
    textInput = ''
    toastStore.success('Text sent to device', 'Injected')
  }

  function sendQuickText(str: string) {
    sendEvent({ action: 'text', text: str })
    toastStore.success(`Sent: ${str}`, 'Quick Input')
  }

  // --- Dynamic Canvas Coordinate Mapping ---
  function getCanvasCoordinates(e: PointerEvent): { x: number; y: number } {
    if (!canvasEl) return { x: 0, y: 0 }
    const rect = canvasEl.getBoundingClientRect()
    if (rect.width === 0 || rect.height === 0) return { x: 0, y: 0 }

    // Physical screen dimensions of the target device
    const devW = detectedWidth || 1080
    const devH = detectedHeight || 2400

    // Compute exact rendered video area within canvas (handling any letterbox / aspect-ratio offsets)
    const containerAspect = rect.width / rect.height
    const screenAspect = devW / devH

    let renderW = rect.width
    let renderH = rect.height
    let offsetX = 0
    let offsetY = 0

    if (containerAspect > screenAspect) {
      // Container is wider than phone aspect ratio: letterbox on left and right
      renderW = rect.height * screenAspect
      offsetX = (rect.width - renderW) / 2
    } else {
      // Container is taller than phone aspect ratio: letterbox on top and bottom
      renderH = rect.width / screenAspect
      offsetY = (rect.height - renderH) / 2
    }

    const clickX = e.clientX - rect.left - offsetX
    const clickY = e.clientY - rect.top - offsetY

    const clampedX = Math.max(0, Math.min(clickX, renderW))
    const clampedY = Math.max(0, Math.min(clickY, renderH))

    const x = Math.round((clampedX / renderW) * devW)
    const y = Math.round((clampedY / renderH) * devH)
    return { x, y }
  }

  function handlePointerDown(e: PointerEvent) {
    if (!isMirroring || !canvasEl) return
    try {
      canvasEl.setPointerCapture(e.pointerId)
    } catch {}
    const { x, y } = getCanvasCoordinates(e)
    sendEvent({ action: 'down', x, y, pointerId: e.pointerId })
  }

  function handlePointerMove(e: PointerEvent) {
    if (!isMirroring || !canvasEl) return
    if (e.buttons === 0 && e.pointerType === 'mouse') return

    const now = performance.now()
    if (now - lastMoveTime < 14) return // Rate-limit to ~70Hz for smooth WS transfer
    lastMoveTime = now

    const { x, y } = getCanvasCoordinates(e)
    sendEvent({ action: 'move', x, y, pointerId: e.pointerId })
  }

  function handlePointerUp(e: PointerEvent) {
    if (!isMirroring || !canvasEl) return
    try {
      if (canvasEl.hasPointerCapture(e.pointerId)) {
        canvasEl.releasePointerCapture(e.pointerId)
      }
    } catch {}
    const { x, y } = getCanvasCoordinates(e)
    sendEvent({ action: 'up', x, y, pointerId: e.pointerId })
  }

  function handlePointerCancel(e: PointerEvent) {
    if (!isMirroring || !canvasEl) return
    try {
      if (canvasEl.hasPointerCapture(e.pointerId)) {
        canvasEl.releasePointerCapture(e.pointerId)
      }
    } catch {}
    const { x, y } = getCanvasCoordinates(e)
    sendEvent({ action: 'up', x, y, pointerId: e.pointerId })
  }

  // --- Physical Keyboard Interceptor ---
  function handleGlobalKeyDown(e: KeyboardEvent) {
    if (!isMirroring || !isKeyboardActive) return

    // Never capture when typing inside form fields
    const target = e.target as HTMLElement | null
    if (
      target &&
      (target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.isContentEditable)
    ) {
      return
    }

    const keycodeMap: Record<string, number> = {
      Backspace: 67,
      Enter: 66,
      Tab: 61,
      Escape: 4, // Back
      ArrowUp: 19,
      ArrowDown: 20,
      ArrowLeft: 21,
      ArrowRight: 22,
      Home: 3,
      Delete: 112,
    }

    if (e.key in keycodeMap) {
      e.preventDefault()
      sendNavKey('key', keycodeMap[e.key])
      return
    }

    if (e.key === ' ' || e.code === 'Space') {
      e.preventDefault()
      sendNavKey('key', 62)
      return
    }

    // Printable characters
    if (e.key.length === 1 && !e.ctrlKey && !e.altKey && !e.metaKey) {
      e.preventDefault()
      sendEvent({ action: 'text', text: e.key })
    }
  }

  // --- Screen Power Toggle ---
  function togglePhysicalScreen() {
    isPhysicalScreenOff = !isPhysicalScreenOff
    sendEvent({ action: 'screen_power', on: !isPhysicalScreenOff })
    if (isPhysicalScreenOff) {
      toastStore.info('Device physical panel powered off (touch & mirror active)', 'Display Power')
    } else {
      toastStore.info('Device physical panel powered on', 'Display Power')
    }
  }

  // --- Screenshot Capture ---
  function captureScreenshot() {
    if (!canvasEl || !isMirroring) {
      toastStore.warning('Start mirroring before taking a screenshot', 'Screenshot')
      return
    }

    try {
      const dataUrl = canvasEl.toDataURL('image/png')
      const a = document.createElement('a')
      const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
      a.href = dataUrl
      a.download = `scrcpy-${detectedWidth}x${detectedHeight}-${timestamp}.png`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      toastStore.success('Saved screenshot to downloads folder', 'Screenshot Saved')
    } catch (err) {
      console.error('Screenshot error:', err)
      toastStore.error('Failed to capture frame from canvas', 'Screenshot Error')
    }
  }

  // --- Fullscreen Toggle ---
  function toggleFullscreen() {
    if (!phoneStageEl) return
    if (!document.fullscreenElement) {
      phoneStageEl
        .requestFullscreen()
        .then(() => {
          isFullscreen = true
        })
        .catch(() => {})
    } else {
      document
        .exitFullscreen()
        .then(() => {
          isFullscreen = false
        })
        .catch(() => {})
    }
  }

  // Parameter update helper
  function onSettingModified() {
    if (isMirroring) {
      settingsChanged = true
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls (Header) -->
  <div class="neo-card bg-card p-4 rounded-xl border-2 border-border shadow-neobrutal">
    <div class="flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
      <!-- Title & Live Badges -->
      <div class="flex flex-wrap items-center gap-2.5">
        <div class="w-9 h-9 rounded-lg bg-accent/20 border-2 border-accent/40 flex items-center justify-center text-accent">
          <Smartphone class="w-5 h-5" />
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
              SCRCPY REMOTE CONTROL
            </h2>
            {#if isMirroring}
              <Badge variant="success" class="flex items-center gap-1.5 animate-pulse">
                <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
                <span>{fps > 0 ? `LIVE ${fps} FPS` : 'LIVE (Idle 60Hz)'}</span>
              </Badge>
            {:else if isConnecting}
              <Badge variant="warning" class="flex items-center gap-1.5 animate-pulse">
                <RefreshCw class="w-3 h-3 animate-spin" />
                <span>CONNECTING...</span>
              </Badge>
            {:else}
              <Badge variant="default">OFFLINE</Badge>
            {/if}
          </div>
          <p class="text-xs text-muted font-mono mt-0.5">
            Low-latency hardware video mirroring with zero-delay pointer injection
          </p>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center flex-wrap gap-2 w-full md:w-auto justify-end">
        <!-- Settings Toggle (Mobile focus) -->
        <Button
          variant={showSettings ? 'secondary' : 'outline'}
          size="sm"
          onclick={() => (showSettings = !showSettings)}
          title="Toggle stream customization sidebar"
        >
          <SlidersHorizontal class="w-3.5 h-3.5 mr-1" />
          <span class="hidden sm:inline">Settings</span>
        </Button>

        <!-- Screenshot Button -->
        <Button
          variant="outline"
          size="sm"
          disabled={!isMirroring}
          onclick={captureScreenshot}
          title="Capture instant screenshot to PNG"
        >
          <Camera class="w-3.5 h-3.5 mr-1 text-cyan-400" />
          <span class="hidden sm:inline">Snap</span>
        </Button>

        <!-- Fullscreen Button -->
        <Button
          variant="outline"
          size="sm"
          onclick={toggleFullscreen}
          title="Toggle phone stage fullscreen"
        >
          {#if isFullscreen}
            <Minimize2 class="w-3.5 h-3.5 mr-1" />
            <span class="hidden sm:inline">Exit</span>
          {:else}
            <Maximize2 class="w-3.5 h-3.5 mr-1" />
            <span class="hidden sm:inline">Full</span>
          {/if}
        </Button>

        <!-- Start / Stop Stream Main CTA -->
        {#if isMirroring}
          <Button variant="danger" size="sm" onclick={stopMirroring}>
            <Square class="w-3.5 h-3.5 mr-1 fill-current" />
            <span>Stop Stream</span>
          </Button>
        {:else}
          <Button
            variant="primary"
            size="sm"
            disabled={isConnecting}
            onclick={startMirroring}
          >
            {#if isConnecting}
              <RefreshCw class="w-3.5 h-3.5 mr-1 animate-spin" />
              <span>Starting...</span>
            {:else}
              <Play class="w-3.5 h-3.5 mr-1 fill-current" />
              <span>Start Mirroring</span>
            {/if}
          </Button>
        {/if}
      </div>
    </div>

    <!-- Parameter Changed Alert if modified on the fly -->
    {#if settingsChanged && isMirroring}
      <div class="mt-3 pt-3 border-t border-border/80 flex items-center justify-between gap-2 text-xs font-mono bg-amber-950/30 border border-amber-800/60 text-amber-300 px-3 py-2 rounded-lg">
        <div class="flex items-center gap-2">
          <Zap class="w-4 h-4 text-amber-400 shrink-0" />
          <span>Stream settings modified. Restart session to apply new bitrate/resolution.</span>
        </div>
        <Button variant="outline" size="sm" class="border-amber-700 hover:bg-amber-900/40 text-amber-200" onclick={startMirroring}>
          <RefreshCw class="w-3 h-3 mr-1" />
          <span>Restart</span>
        </Button>
      </div>
    {/if}
  </div>

  <!-- Screen View & Remote Controls Layout -->
  <div class="grid grid-cols-1 {showSettings ? 'lg:grid-cols-12' : ''} gap-6 items-start">
    <!-- Left/Main Stage: Phone Screen Frame -->
    <div class="{showSettings ? 'lg:col-span-7 xl:col-span-8' : 'w-full'} flex flex-col items-center">
      <div
        bind:this={phoneStageEl}
        class="neo-card bg-card border-2 border-border shadow-neobrutal rounded-2xl p-2.5 sm:p-4 flex flex-col items-center w-full max-w-sm sm:max-w-md transition-all relative {isFullscreen ? 'fixed inset-0 z-50 max-w-none rounded-none justify-center bg-black border-none' : ''}"
      >
        <!-- Phone Speaker & Camera Notch Top Accent -->
        <div class="w-full flex items-center justify-between font-mono text-[10px] text-muted pb-1.5 sm:pb-2 mb-1.5 sm:mb-2 border-b border-border">
          <div class="flex items-center gap-2">
            <span class="inline-block w-2 h-2 rounded-full {isMirroring ? 'bg-emerald-500' : 'bg-zinc-600'}"></span>
            <span class="font-bold tracking-widest text-foreground">BFR-DEV</span>
          </div>

          <!-- Front bezel speaker notch -->
          <div class="flex items-center gap-1.5 px-3 py-0.5 rounded-full bg-card-sub border border-border/60">
            <div class="w-8 h-1 bg-border rounded-full"></div>
            <div class="w-2 h-2 rounded-full bg-black border border-border/80"></div>
          </div>

          <div class="flex items-center gap-1.5">
            {#if isMirroring}
              <span class="text-emerald-400 font-bold">{detectedWidth}x{detectedHeight}</span>
            {:else}
              <span>OFFLINE</span>
            {/if}
          </div>
        </div>

        <!-- Phone Screen Surface Area -->
        <div
          class="relative w-full bg-black rounded-xl overflow-hidden border-2 border-border flex items-center justify-center select-none shadow-inner"
          style="aspect-ratio: {aspectRatio}; touch-action: none;"
        >
          <!-- Canvas rendering element -->
          <canvas
            bind:this={canvasEl}
            class="w-full h-full object-contain cursor-crosshair {isMirroring ? 'block' : 'hidden'}"
            onpointerdown={handlePointerDown}
            onpointermove={handlePointerMove}
            onpointerup={handlePointerUp}
            onpointercancel={handlePointerCancel}
            onpointerleave={handlePointerCancel}
          ></canvas>

          <!-- Empty / Offline Backdrop Overlay -->
          {#if !isMirroring}
            <div class="absolute inset-0 flex flex-col items-center justify-center p-6 text-center bg-gradient-to-b from-card/90 via-card/95 to-black font-mono space-y-4">
              <!-- Geometric Grid Ambient Accent -->
              <div class="w-16 h-16 rounded-2xl bg-card-sub border-2 border-border flex items-center justify-center shadow-neobrutal">
                <Smartphone class="w-8 h-8 text-accent animate-pulse" />
              </div>

              <div>
                <h3 class="text-sm sm:text-base font-bold text-foreground uppercase tracking-wider">
                  Mirroring Inactive
                </h3>
                <p class="text-xs text-muted mt-1 max-w-xs">
                  Connect to your device's native low-latency scrcpy server engine
                </p>
              </div>

              <div class="flex flex-wrap justify-center gap-1.5 max-w-xs text-[10px]">
                <Badge variant="info">WebCodecs H.264</Badge>
                <Badge variant="default">&lt;15ms Latency</Badge>
                <Badge variant="default">Zero ADB Lag</Badge>
                <Badge variant="default">Hardware Touch</Badge>
              </div>

              <div class="pt-2">
                <Button
                  variant="primary"
                  size="md"
                  disabled={isConnecting}
                  onclick={startMirroring}
                  class="shadow-neobrutal"
                >
                  {#if isConnecting}
                    <RefreshCw class="w-4 h-4 mr-1.5 animate-spin" />
                    <span>Connecting...</span>
                  {:else}
                    <Play class="w-4 h-4 mr-1.5 fill-current" />
                    <span>Start Mirroring</span>
                  {/if}
                </Button>
              </div>
            </div>
          {/if}

          <!-- Floating Keyboard Overlay Indicator when active -->
          {#if isMirroring && isKeyboardActive}
            <div class="absolute top-2 left-2 pointer-events-none bg-emerald-950/90 text-emerald-300 border border-emerald-700/80 px-2 py-1 rounded text-[10px] font-mono flex items-center gap-1.5 shadow-md">
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-ping"></span>
              <span>PC Keyboard Captured</span>
            </div>
          {/if}
        </div>

        <!-- Android Virtual Navigation & Hardware Control Dock -->
        <div class="w-full mt-2 sm:mt-3 pt-2 sm:pt-3 border-t-2 border-border flex flex-col gap-1.5 sm:gap-2">
          <!-- Main Android Navigation Bar -->
          <div class="grid grid-cols-3 gap-1.5 sm:gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={!isMirroring}
              onclick={() => sendNavKey('back', 4)}
              title="Android Back (Keycode 4 / Esc)"
              class="hover:bg-card-sub py-1.5 sm:py-2 px-1.5 sm:px-3 text-[11px] sm:text-xs"
            >
              <ArrowLeft class="w-3.5 h-3.5 sm:w-4 sm:h-4 mr-1" />
              <span>Back</span>
            </Button>

            <Button
              variant="outline"
              size="sm"
              disabled={!isMirroring}
              onclick={() => sendNavKey('home', 3)}
              title="Android Home (Keycode 3)"
              class="hover:bg-card-sub py-1.5 sm:py-2 px-1.5 sm:px-3 text-[11px] sm:text-xs"
            >
              <Home class="w-3.5 h-3.5 sm:w-4 sm:h-4 mr-1" />
              <span>Home</span>
            </Button>

            <Button
              variant="outline"
              size="sm"
              disabled={!isMirroring}
              onclick={() => sendNavKey('recents', 187)}
              title="Android Recents / App Switcher (Keycode 187)"
              class="hover:bg-card-sub py-1.5 sm:py-2 px-1.5 sm:px-3 text-[11px] sm:text-xs"
            >
              <Menu class="w-3.5 h-3.5 sm:w-4 sm:h-4 mr-1" />
              <span>Recents</span>
            </Button>
          </div>

          <!-- Extended Quick Hardware Dock -->
          <div class="grid grid-cols-5 gap-1 sm:gap-1.5 font-mono text-xs">
            <Button
              variant="outline"
              size="sm"
              disabled={!isMirroring}
              onclick={() => sendNavKey('vol_down', 25)}
              title="Volume Down (Keycode 25)"
              class="py-1.5 sm:py-2 px-1 sm:px-1.5 flex justify-center items-center"
            >
              <Volume1 class="w-3.5 h-3.5 text-amber-400" />
            </Button>

            <Button
              variant="outline"
              size="sm"
              disabled={!isMirroring}
              onclick={() => sendNavKey('vol_up', 24)}
              title="Volume Up (Keycode 24)"
              class="py-1.5 sm:py-2 px-1 sm:px-1.5 flex justify-center items-center"
            >
              <Volume2 class="w-3.5 h-3.5 text-emerald-400" />
            </Button>

            <Button
              variant="outline"
              size="sm"
              disabled={!isMirroring}
              onclick={() => sendNavKey('power', 26)}
              title="Physical Power Key (Keycode 26)"
              class="py-1.5 sm:py-2 px-1 sm:px-1.5 flex justify-center items-center"
            >
              <Power class="w-3.5 h-3.5 text-red-400" />
            </Button>

            <!-- Toggle Physical Device Screen Power -->
            <Button
              variant={isPhysicalScreenOff ? 'secondary' : 'outline'}
              size="sm"
              disabled={!isMirroring}
              onclick={togglePhysicalScreen}
              title={isPhysicalScreenOff ? 'Turn Physical Screen ON' : 'Turn Physical Screen OFF (Save Battery & Heat)'}
              class="py-1.5 sm:py-2 px-1 sm:px-1.5 flex justify-center items-center {isPhysicalScreenOff ? 'border-accent bg-accent/20 text-accent' : ''}"
            >
              {#if isPhysicalScreenOff}
                <EyeOff class="w-3.5 h-3.5 text-accent" />
              {:else}
                <Eye class="w-3.5 h-3.5 text-muted hover:text-foreground" />
              {/if}
            </Button>

            <!-- Direct PC Keyboard Capture Toggle -->
            <Button
              variant={isKeyboardActive ? 'primary' : 'outline'}
              size="sm"
              disabled={!isMirroring}
              onclick={() => {
                isKeyboardActive = !isKeyboardActive
                if (isKeyboardActive) {
                  toastStore.info('PC Keyboard captured: typing sends keys directly to device', 'Keyboard ON')
                }
              }}
              title={isKeyboardActive ? 'Disable PC Keyboard Capture' : 'Enable PC Keyboard Capture'}
              class="py-1.5 sm:py-2 px-1 sm:px-1.5 flex justify-center items-center"
            >
              <Keyboard class="w-3.5 h-3.5" />
            </Button>
          </div>
        </div>
      </div>
    </div>

    <!-- Right Column: Stream Customization & Diagnostics Card -->
    {#if showSettings}
      <div class="lg:col-span-5 xl:col-span-4 space-y-4">
        <!-- 1. Stream Customization Card -->
        <Card title="Stream Customization" subtitle="Adjust encoder quality & parameters">
          <div class="space-y-4 font-mono text-xs">
            <!-- Resolution Selector -->
            <div>
              <div class="block text-[11px] font-bold text-muted uppercase tracking-wider mb-1.5">
                Resolution Limit
              </div>
              <div class="grid grid-cols-2 gap-1.5">
                {#each resolutionOptions as opt}
                  <button
                    type="button"
                    class="px-2 py-2 rounded-lg border text-left transition-all cursor-pointer {maxSize === opt.value ? 'bg-accent text-accent-text border-accent font-bold shadow-sm' : 'bg-card-sub border-border text-foreground hover:bg-card hover:border-accent/40'}"
                    onclick={() => {
                      maxSize = opt.value
                      onSettingModified()
                    }}
                  >
                    <div class="text-xs font-bold">{opt.label}</div>
                    <div class="text-[10px] opacity-75">{opt.desc}</div>
                  </button>
                {/each}
              </div>
            </div>

            <!-- Bitrate Selector & Custom Input -->
            <div>
              <div class="flex items-center justify-between mb-1.5">
                <span class="text-[11px] font-bold text-muted uppercase tracking-wider">
                  Video Bitrate
                </span>
                <span class="text-[11px] font-mono text-accent font-bold">
                  {(bitRate / 1000000).toFixed(0)} Mbps
                </span>
              </div>

              <!-- Presets Grid -->
              <div class="grid grid-cols-3 sm:grid-cols-6 gap-1.5 mb-2">
                {#each bitrateOptions as b}
                  <button
                    type="button"
                    class="py-1.5 px-1 text-center rounded-md border text-xs transition-all cursor-pointer {bitRate === b.value ? 'bg-accent text-accent-text border-accent font-bold shadow-sm' : 'bg-card-sub border-border text-foreground hover:bg-card'}"
                    onclick={() => {
                      bitRate = b.value
                      customBitrateMbps = Math.round(b.value / 1000000)
                      onSettingModified()
                    }}
                  >
                    {b.label}
                  </button>
                {/each}
              </div>

              <!-- Custom Bitrate Input -->
              <div class="flex items-center gap-2 p-2 bg-card-sub rounded-lg border border-border">
                <span class="text-[11px] text-muted whitespace-nowrap font-bold">Custom:</span>
                <input
                  type="number"
                  min="1"
                  max="100"
                  bind:value={customBitrateMbps}
                  onchange={() => {
                    const mbps = Math.max(1, Math.min(100, Number(customBitrateMbps) || 8))
                    customBitrateMbps = mbps
                    bitRate = mbps * 1000000
                    onSettingModified()
                  }}
                  class="w-20 px-2 py-1 bg-card border border-border rounded text-xs text-foreground font-mono focus:outline-none focus:border-accent"
                  placeholder="Mbps"
                />
                <span class="text-xs text-muted font-mono">Mbps</span>
                <button
                  type="button"
                  class="ml-auto px-2.5 py-1 text-[11px] bg-accent text-accent-text font-bold rounded border border-accent hover:opacity-90 transition-opacity cursor-pointer"
                  onclick={() => {
                    const mbps = Math.max(1, Math.min(100, Number(customBitrateMbps) || 8))
                    customBitrateMbps = mbps
                    bitRate = mbps * 1000000
                    onSettingModified()
                  }}
                >
                  Apply
                </button>
              </div>
            </div>

            <!-- Framerate & Display Selectors -->
            <div class="grid grid-cols-2 gap-3">
              <div>
                <div class="block text-[11px] font-bold text-muted uppercase tracking-wider mb-1.5">
                  Target FPS
                </div>
                <div class="grid grid-cols-2 gap-1.5">
                  <button
                    type="button"
                    class="py-1.5 text-center rounded border cursor-pointer {maxFps === 30 ? 'bg-accent text-accent-text border-accent font-bold' : 'bg-card-sub border-border'}"
                    onclick={() => {
                      maxFps = 30
                      onSettingModified()
                    }}
                  >
                    30 FPS
                  </button>
                  <button
                    type="button"
                    class="py-1.5 text-center rounded border cursor-pointer {maxFps === 60 ? 'bg-accent text-accent-text border-accent font-bold' : 'bg-card-sub border-border'}"
                    onclick={() => {
                      maxFps = 60
                      onSettingModified()
                    }}
                  >
                    60 FPS
                  </button>
                </div>
              </div>

              <div>
                <div class="block text-[11px] font-bold text-muted uppercase tracking-wider mb-1.5">
                  Target Display
                </div>
                <div class="grid grid-cols-2 gap-1.5">
                  <button
                    type="button"
                    class="py-1.5 text-center rounded border cursor-pointer {displayId === 0 ? 'bg-accent text-accent-text border-accent font-bold' : 'bg-card-sub border-border'}"
                    onclick={() => {
                      displayId = 0
                      onSettingModified()
                    }}
                  >
                    Main (0)
                  </button>
                  <button
                    type="button"
                    class="py-1.5 text-center rounded border cursor-pointer {displayId === 1 ? 'bg-accent text-accent-text border-accent font-bold' : 'bg-card-sub border-border'}"
                    onclick={() => {
                      displayId = 1
                      onSettingModified()
                    }}
                  >
                    Cover (1)
                  </button>
                </div>
              </div>
            </div>

            <!-- Device Behavior Checkboxes -->
            <div class="pt-2 border-t border-border/80 space-y-2">
              <label class="flex items-center gap-2.5 cursor-pointer select-none">
                <input
                  type="checkbox"
                  bind:checked={turnScreenOff}
                  onchange={onSettingModified}
                  class="rounded border-border text-accent focus:ring-accent bg-card-sub w-4 h-4 cursor-pointer"
                />
                <span class="text-xs text-foreground">Turn off physical screen on stream launch</span>
              </label>

              <label class="flex items-center gap-2.5 cursor-pointer select-none">
                <input
                  type="checkbox"
                  bind:checked={stayAwake}
                  onchange={onSettingModified}
                  class="rounded border-border text-accent focus:ring-accent bg-card-sub w-4 h-4 cursor-pointer"
                />
                <span class="text-xs text-foreground">Keep device awake while mirroring</span>
              </label>
            </div>
          </div>
        </Card>

        <!-- 2. Real-Time Diagnostics Card -->
        <Card title="Diagnostics & Telemetry" subtitle="Hardware decoding & pipeline metrics">
          <div class="space-y-3 font-mono text-xs">
            <!-- Digital FPS Gauge -->
            <div class="p-3 bg-card-sub rounded-xl border border-border flex items-center justify-between">
              <div>
                <span class="text-[10px] text-muted uppercase font-bold tracking-wider">Live Refresh Rate</span>
                <div class="flex items-baseline gap-1 mt-0.5">
                  <span class="text-2xl font-bold {fps >= 50 ? 'text-emerald-400' : fps >= 25 ? 'text-amber-400' : isMirroring ? 'text-emerald-400/80' : 'text-muted'}">
                    {fps > 0 ? fps : (isMirroring ? 'Idle' : '0')}
                  </span>
                  <span class="text-xs text-muted">{fps > 0 ? 'FPS' : (isMirroring ? '(60Hz Standby)' : 'FPS')}</span>
                </div>
              </div>

              <div class="text-right">
                <Badge variant={streamEngine === 'webcodecs' ? 'success' : streamEngine === 'mjpeg' ? 'warning' : 'default'}>
                  {#if streamEngine === 'webcodecs'}
                    WEBCODECS H.264
                  {:else if streamEngine === 'mjpeg'}
                    MJPEG FALLBACK
                  {:else}
                    IDLE
                  {/if}
                </Badge>
                <div class="text-[10px] text-muted mt-1">
                  {totalFrames > 0 ? `${totalFrames.toLocaleString()} frames decoded` : 'No frames'}
                </div>
              </div>
            </div>

            <!-- Detailed Grid Readouts -->
            <div class="grid grid-cols-2 gap-2 text-[11px]">
              <div class="p-2.5 bg-card-sub rounded-lg border border-border col-span-2">
                <div class="text-muted text-[10px] uppercase">Active Hardware Encoder</div>
                <div class="font-bold text-foreground mt-0.5 truncate text-[11px]" title={hardwareEncoderName}>
                  {hardwareEncoderName || 'Detecting...'}
                </div>
              </div>

              <div class="p-2.5 bg-card-sub rounded-lg border border-border">
                <div class="text-muted text-[10px] uppercase">Display Output</div>
                <div class="font-bold text-foreground mt-0.5">
                  {isMirroring ? `${detectedWidth} × ${detectedHeight}` : '—'}
                </div>
              </div>

              <div class="p-2.5 bg-card-sub rounded-lg border border-border">
                <div class="text-muted text-[10px] uppercase">Codec Profile</div>
                <div class="font-bold text-foreground mt-0.5 truncate" title={codecName}>
                  {codecName || 'avc1.640033'}
                </div>
              </div>

              <div class="p-2.5 bg-card-sub rounded-lg border border-border">
                <div class="text-muted text-[10px] uppercase">Encoder Bitrate</div>
                <div class="font-bold text-foreground mt-0.5">
                  {(bitRate / 1000000).toFixed(0)} Mbps
                </div>
              </div>

              <div class="p-2.5 bg-card-sub rounded-lg border border-border">
                <div class="text-muted text-[10px] uppercase">Pipeline Latency</div>
                <div class="font-bold text-emerald-400 mt-0.5">
                  {isMirroring ? '< 15 ms' : '—'}
                </div>
              </div>
            </div>
          </div>
        </Card>

        <!-- 3. Direct Text Injection & Quick URL Bar -->
        <Card title="Direct Text Injection" subtitle="Send strings, passwords or URLs to phone">
          <form
            onsubmit={(e) => {
              e.preventDefault()
              sendText()
            }}
            class="space-y-3 font-mono text-xs"
          >
            <div class="flex items-center gap-2">
              <Input
                placeholder="Paste URL or type string to send..."
                bind:value={textInput}
                disabled={!isMirroring}
                class="text-xs"
              />
              <Button
                type="submit"
                variant="primary"
                size="md"
                disabled={!isMirroring || !textInput.trim()}
                title="Inject string to active input field"
              >
                <Send class="w-3.5 h-3.5" />
              </Button>
            </div>

            <!-- Quick Paste Presets -->
            <div class="flex items-center flex-wrap gap-1.5 pt-1">
              <span class="text-[10px] text-muted uppercase font-bold mr-1">Quick:</span>
              <button
                type="button"
                disabled={!isMirroring}
                class="px-2 py-0.5 rounded bg-card-sub hover:bg-card border border-border text-[10px] text-foreground disabled:opacity-50 cursor-pointer"
                onclick={() => sendQuickText('https://')}
              >
                https://
              </button>
              <button
                type="button"
                disabled={!isMirroring}
                class="px-2 py-0.5 rounded bg-card-sub hover:bg-card border border-border text-[10px] text-foreground disabled:opacity-50 cursor-pointer"
                onclick={() => sendQuickText('www.')}
              >
                www.
              </button>
              <button
                type="button"
                disabled={!isMirroring}
                class="px-2 py-0.5 rounded bg-card-sub hover:bg-card border border-border text-[10px] text-foreground disabled:opacity-50 cursor-pointer"
                onclick={() => sendNavKey('key', 66)}
              >
                [Enter]
              </button>
              <button
                type="button"
                disabled={!isMirroring}
                class="px-2 py-0.5 rounded bg-card-sub hover:bg-card border border-border text-[10px] text-foreground disabled:opacity-50 cursor-pointer"
                onclick={() => sendNavKey('key', 67)}
              >
                [Del]
              </button>
            </div>
          </form>
        </Card>
      </div>
    {/if}
  </div>
</div>
