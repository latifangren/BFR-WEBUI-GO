<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import {
    Smartphone,
    Play,
    Square,
    RotateCcw,
    Home,
    ArrowLeft,
    Menu,
    Power,
    Volume2,
    VolumeX,
    Send,
    Activity,
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'
  import { WebSocketClient } from '../../../ws/socket'

  let isMirroring = $state(false)
  let fps = $state(0)
  let currentImgUrl = $state('')
  let textInput = $state('')
  let screenImgEl: HTMLImageElement | null = $state(null)

  let wsClient: WebSocketClient | null = null
  let lastFrameTime = 0
  let swipeStart: { x: number; y: number; time: number } | null = null

  onMount(() => {
    return () => {
      stopMirroring()
    }
  })

  onDestroy(() => {
    stopMirroring()
  })

  function startMirroring() {
    if (wsClient && wsClient.readyState === WebSocket.OPEN) {
      return
    }

    stopMirroring()

    try {
      wsClient = new WebSocketClient({
        path: '/api/scrcpy/ws',
        autoReconnect: false,
        binaryType: 'blob',
        onOpen: () => {
          isMirroring = true
          lastFrameTime = Date.now()
        },
        onMessage: (data) => {
          if (data instanceof Blob) {
            const now = Date.now()
            if (lastFrameTime > 0) {
              const delta = now - lastFrameTime
              if (delta > 0) {
                fps = Math.round(1000 / delta)
              }
            }
            lastFrameTime = now

            const oldUrl = currentImgUrl
            currentImgUrl = URL.createObjectURL(data)
            if (oldUrl) {
              URL.revokeObjectURL(oldUrl)
            }
          }
        },
        onClose: () => {
          stopMirroring()
        },
        onError: () => {
          stopMirroring()
        },
      })

      wsClient.connect()
    } catch {
      stopMirroring()
    }
  }

  function stopMirroring() {
    if (wsClient) {
      wsClient.disconnect()
      wsClient = null
    }
    if (currentImgUrl) {
      URL.revokeObjectURL(currentImgUrl)
      currentImgUrl = ''
    }
    if (screenImgEl) {
      screenImgEl.src = ''
    }
    isMirroring = false
    fps = 0
    swipeStart = null
  }

  function sendEvent(evt: Record<string, unknown>) {
    if (wsClient && wsClient.readyState === WebSocket.OPEN) {
      wsClient.send(JSON.stringify(evt))
    }
  }

  function sendNavKey(action: string, keycode = 0) {
    sendEvent({ action, keycode })
  }

  function sendText() {
    if (!textInput) return
    sendEvent({ action: 'text', text: textInput })
    textInput = ''
  }

  function handleScreenClick(event: MouseEvent) {
    if (!screenImgEl) return
    const rect = screenImgEl.getBoundingClientRect()
    const clickX = event.clientX - rect.left
    const clickY = event.clientY - rect.top

    const natW = screenImgEl.naturalWidth || 1080
    const natH = screenImgEl.naturalHeight || 2400

    const targetX = Math.round((clickX / rect.width) * natW)
    const targetY = Math.round((clickY / rect.height) * natH)

    sendEvent({ action: 'click', x: targetX, y: targetY })
  }

  function handleTouchStart(event: TouchEvent) {
    if (!screenImgEl || !event.touches || event.touches.length === 0) return
    const touch = event.touches[0]
    const rect = screenImgEl.getBoundingClientRect()
    swipeStart = {
      x: touch.clientX - rect.left,
      y: touch.clientY - rect.top,
      time: Date.now(),
    }
  }

  function handleTouchEnd(event: TouchEvent) {
    if (!screenImgEl || !swipeStart) return
    const rect = screenImgEl.getBoundingClientRect()
    const touch = event.changedTouches[0]
    const endX = touch.clientX - rect.left
    const endY = touch.clientY - rect.top
    const duration = Date.now() - swipeStart.time

    const natW = screenImgEl.naturalWidth || 1080
    const natH = screenImgEl.naturalHeight || 2400

    const x1 = Math.round((swipeStart.x / rect.width) * natW)
    const y1 = Math.round((swipeStart.y / rect.height) * natH)
    const x2 = Math.round((endX / rect.width) * natW)
    const y2 = Math.round((endY / rect.height) * natH)

    if (Math.abs(x2 - x1) < 10 && Math.abs(y2 - y1) < 10) {
      sendEvent({ action: 'click', x: x1, y: y1 })
    } else {
      sendEvent({ action: 'swipe', x: x1, y: y1, x2, y2, duration: Math.max(duration, 100) })
    }
    swipeStart = null
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Smartphone class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Scrcpy Web Screen Mirroring
      </h2>
      <Badge variant={isMirroring ? 'success' : 'default'}>
        {isMirroring ? `${fps} FPS` : 'Idle'}
      </Badge>
    </div>

    <div class="flex items-center gap-2">
      {#if isMirroring}
        <Button variant="danger" size="sm" onclick={stopMirroring}>
          <Square class="w-3.5 h-3.5 mr-1" />
          <span>Stop Stream</span>
        </Button>
      {:else}
        <Button variant="primary" size="sm" onclick={startMirroring}>
          <Play class="w-3.5 h-3.5 mr-1" />
          <span>Start Mirror</span>
        </Button>
      {/if}
    </div>
  </div>

  <!-- Screen View & Remote Controls -->
  <div class="grid grid-cols-1 lg:grid-cols-4 gap-6">
    <!-- Left: Display Canvas Frame -->
    <div class="lg:col-span-3 flex justify-center">
      <div class="neo-card bg-card border-2 border-border shadow-neobrutal rounded-xl p-3 flex flex-col items-center max-w-sm w-full">
        <!-- Phone Frame Header -->
        <div class="w-full flex items-center justify-between font-mono text-[10px] text-muted pb-2 mb-2 border-b border-border">
          <span>Android Display Stream</span>
          <span class="text-accent font-bold">{isMirroring ? 'LIVE' : 'OFFLINE'}</span>
        </div>

        <!-- Screen Surface -->
        <div class="relative w-full aspect-[9/19.5] bg-black rounded-lg overflow-hidden border border-border flex items-center justify-center">
          {#if isMirroring && currentImgUrl}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
            <img
              bind:this={screenImgEl}
              src={currentImgUrl}
              alt="Android Screen Stream"
              class="w-full h-full object-contain cursor-crosshair select-none"
              onclick={handleScreenClick}
              ontouchstart={handleTouchStart}
              ontouchend={handleTouchEnd}
            />
          {:else}
            <div class="text-center font-mono p-6 text-muted space-y-2">
              <Smartphone class="w-10 h-10 mx-auto opacity-40 text-accent" />
              <p class="text-xs">Stream is stopped.</p>
              <Button variant="secondary" size="sm" onclick={startMirroring}>
                Launch Scrcpy
              </Button>
            </div>
          {/if}
        </div>

        <!-- Android Virtual Navigation Bar -->
        <div class="w-full grid grid-cols-3 gap-2 mt-3 pt-2 border-t border-border">
          <Button
            variant="outline"
            size="sm"
            disabled={!isMirroring}
            onclick={() => sendNavKey('back', 4)}
            title="Back"
          >
            <ArrowLeft class="w-4 h-4" />
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={!isMirroring}
            onclick={() => sendNavKey('home', 3)}
            title="Home"
          >
            <Home class="w-4 h-4" />
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={!isMirroring}
            onclick={() => sendNavKey('recents', 187)}
            title="Recents"
          >
            <Menu class="w-4 h-4" />
          </Button>
        </div>
      </div>
    </div>

    <!-- Right: Quick Remote Shortcuts & Text Injection -->
    <div class="space-y-4">
      <Card title="Hardware Buttons" subtitle="Inject keycodes directly">
        <div class="grid grid-cols-2 gap-2 font-mono text-xs">
          <Button
            variant="outline"
            size="sm"
            disabled={!isMirroring}
            onclick={() => sendNavKey('power', 26)}
          >
            <Power class="w-3.5 h-3.5 mr-1 text-red-400" />
            <span>Power</span>
          </Button>

          <Button
            variant="outline"
            size="sm"
            disabled={!isMirroring}
            onclick={() => sendNavKey('volume_up', 24)}
          >
            <Volume2 class="w-3.5 h-3.5 mr-1 text-emerald-400" />
            <span>Vol Up</span>
          </Button>

          <Button
            variant="outline"
            size="sm"
            disabled={!isMirroring}
            onclick={() => sendNavKey('volume_down', 25)}
          >
            <VolumeX class="w-3.5 h-3.5 mr-1 text-amber-400" />
            <span>Vol Down</span>
          </Button>

          <Button
            variant="outline"
            size="sm"
            disabled={!isMirroring}
            onclick={() => sendNavKey('enter', 66)}
          >
            <span>Enter</span>
          </Button>
        </div>
      </Card>

      <Card title="Text Injection" subtitle="Send keyboard strings to phone">
        <form onsubmit={(e) => { e.preventDefault(); sendText() }} class="space-y-2 font-mono text-xs">
          <Input
            placeholder="Type text to send..."
            bind:value={textInput}
            disabled={!isMirroring}
          />
          <Button
            type="submit"
            variant="primary"
            size="sm"
            fullWidth={true}
            disabled={!isMirroring || !textInput}
          >
            <Send class="w-3.5 h-3.5 mr-1" />
            <span>Inject Text</span>
          </Button>
        </form>
      </Card>
    </div>
  </div>
</div>
