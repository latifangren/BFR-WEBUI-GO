<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { Terminal as XTerminal } from '@xterm/xterm'
  import { FitAddon } from '@xterm/addon-fit'
  import '@xterm/xterm/css/xterm.css'
  import {
    Terminal,
    RefreshCw,
    Maximize2,
    Trash2,
    Keyboard,
  } from '@lucide/svelte'
  import { WebSocketClient } from '../../../ws/socket'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import TerminalVirtualKeys from './TerminalVirtualKeys.svelte'

  let terminalContainer: HTMLDivElement | null = $state(null)
  let term: XTerminal | null = null
  let fitAddon: FitAddon | null = null
  let wsClient: WebSocketClient | null = null
  let resizeObserver: ResizeObserver | null = null

  let isConnected = $state(false)
  let isConnecting = $state(false)
  let showVirtualKeys = $state(false)

  // Mobile detection for virtual keys ergonomics
  function isMobileClient(): boolean {
    if (typeof window === 'undefined' || typeof navigator === 'undefined') return false
    const uaMatches = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini|Mobile/i.test(navigator.userAgent)
    const touchMatches = navigator.maxTouchPoints > 0 && window.innerWidth < 1024
    return uaMatches || touchMatches
  }

  onMount(() => {
    showVirtualKeys = isMobileClient()

    initTerminal()
    connectWebSocket()

    const handleResize = () => {
      handleFitAndResize()
    }
    window.addEventListener('resize', handleResize)

    return () => {
      window.removeEventListener('resize', handleResize)
      cleanup()
    }
  })

  onDestroy(() => {
    cleanup()
  })

  function handleFitAndResize() {
    if (fitAddon && term) {
      try {
        fitAddon.fit()
        if (wsClient && wsClient.readyState === WebSocket.OPEN && term.cols > 0 && term.rows > 0) {
          wsClient.send(
            JSON.stringify({
              type: 'resize',
              cols: term.cols,
              rows: term.rows,
            })
          )
        }
      } catch {
        // Suppress fit errors during transition or zero-dimension states
      }
    }
  }

  function toggleVirtualKeys() {
    showVirtualKeys = !showVirtualKeys
    setTimeout(() => {
      handleFitAndResize()
      term?.focus()
    }, 60)
  }

  function handleVirtualKey(data: string) {
    if (wsClient && wsClient.readyState === WebSocket.OPEN) {
      wsClient.send(data)
    }
    term?.focus()
  }

  function cleanup() {
    if (resizeObserver) {
      resizeObserver.disconnect()
      resizeObserver = null
    }
    if (wsClient) {
      wsClient.disconnect()
      wsClient = null
    }
    if (term) {
      term.dispose()
      term = null
    }
    if (fitAddon) {
      fitAddon.dispose()
      fitAddon = null
    }
    isConnected = false
    isConnecting = false
  }

  function initTerminal() {
    if (!terminalContainer) return

    term = new XTerminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
      theme: {
        background: '#090d16',
        foreground: '#f8fafc',
        cursor: '#38bdf8',
        selectionBackground: 'rgba(56, 189, 248, 0.3)',
        black: '#090d16',
        red: '#ef4444',
        green: '#10b981',
        yellow: '#f59e0b',
        blue: '#3b82f6',
        magenta: '#d946ef',
        cyan: '#06b6d4',
        white: '#f8fafc',
      },
    })

    fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(terminalContainer)

    // Attach ResizeObserver to keep cols/rows synchronized on any dimension shift
    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(() => {
        handleFitAndResize()
      })
      resizeObserver.observe(terminalContainer)
    }

    setTimeout(() => {
      handleFitAndResize()
    }, 100)

    term.onData((data) => {
      if (wsClient && wsClient.readyState === WebSocket.OPEN) {
        wsClient.send(data)
      }
    })
  }

  function connectWebSocket() {
    if (wsClient && wsClient.readyState === WebSocket.OPEN) {
      return
    }

    isConnecting = true

    wsClient = new WebSocketClient({
      path: '/api/terminal/ws',
      autoReconnect: true,
      binaryType: 'arraybuffer',
      onOpen: () => {
        isConnected = true
        isConnecting = false
        term?.write('\r\n\x1b[32m[+] Interactive Root Terminal Connected\x1b[0m\r\n\r\n')
        handleFitAndResize()
      },
      onMessage: (data: string | Blob | ArrayBuffer) => {
        if (typeof data === 'string') {
          term?.write(data)
        } else if (data instanceof ArrayBuffer) {
          const decoder = new TextDecoder()
          term?.write(decoder.decode(data))
        } else if (data instanceof Blob) {
          data.text().then((text) => term?.write(text))
        }
      },
      onError: () => {
        term?.write('\r\n\x1b[31m[-] WebSocket Error\x1b[0m\r\n')
      },
      onClose: () => {
        isConnected = false
        isConnecting = false
        term?.write('\r\n\x1b[33m[*] Terminal Session Closed.\x1b[0m\r\n')
      },
    })

    wsClient.connect()
  }

  function reconnect() {
    cleanup()
    initTerminal()
    connectWebSocket()
  }

  function clearTerminal() {
    term?.clear()
  }
</script>

<div class="space-y-4">
  <!-- Header & Toolbar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div class="flex items-center gap-2">
      <Terminal class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Interactive Root Terminal
      </h2>
      <Badge variant={isConnected ? 'success' : isConnecting ? 'warning' : 'default'}>
        {isConnected ? 'Connected (Root)' : isConnecting ? 'Connecting...' : 'Disconnected'}
      </Badge>
    </div>

    <div class="flex items-center gap-2 flex-wrap">
      <!-- Virtual Keys Toggle -->
      <Button
        variant={showVirtualKeys ? 'secondary' : 'outline'}
        size="sm"
        onclick={toggleVirtualKeys}
        title="Toggle Virtual Keypad Bar"
      >
        <Keyboard class="w-3.5 h-3.5 mr-1" />
        <span>{showVirtualKeys ? 'Keys: On' : 'Keys: Off'}</span>
      </Button>

      <!-- Clear Terminal -->
      <Button variant="outline" size="sm" onclick={clearTerminal} title="Clear Terminal">
        <Trash2 class="w-3.5 h-3.5 mr-1" />
        <span>Clear</span>
      </Button>

      <!-- Reconnect Session -->
      <Button variant="secondary" size="sm" onclick={reconnect} title="Reconnect Terminal">
        <RefreshCw class="w-3.5 h-3.5 mr-1 {isConnecting ? 'animate-spin' : ''}" />
        <span>Reconnect</span>
      </Button>
    </div>
  </div>

  <!-- Terminal Window (Neo-Brutalist Full-Height Frame) -->
  <div class="neo-card bg-card border-2 border-border shadow-neobrutal rounded-lg overflow-hidden h-[calc(100vh-210px)] min-h-[460px] flex flex-col">
    <!-- Terminal Header Bar -->
    <div class="bg-card-sub border-b border-border px-3 py-2 flex items-center justify-between font-mono text-xs text-muted shrink-0">
      <div class="flex items-center gap-2">
        <span class="w-3 h-3 rounded-full bg-red-500/80 inline-block"></span>
        <span class="w-3 h-3 rounded-full bg-amber-500/80 inline-block"></span>
        <span class="w-3 h-3 rounded-full bg-emerald-500/80 inline-block"></span>
        <span class="ml-2 font-bold text-foreground">root@android (PTY / su)</span>
      </div>
      <div class="flex items-center gap-2 text-[10px]">
        <span>UTF-8</span>
        <span>|</span>
        <button
          type="button"
          class="hover:text-foreground cursor-pointer flex items-center gap-1"
          onclick={() => handleFitAndResize()}
          title="Force Fit Terminal Canvas"
        >
          <Maximize2 class="w-3 h-3" />
          <span>Fit</span>
        </button>
      </div>
    </div>

    <!-- Terminal Canvas Container -->
    <div
      bind:this={terminalContainer}
      class="flex-1 w-full p-2 bg-[#090d16] font-mono overflow-hidden"
    ></div>

    <!-- Virtual Touch Keypad (Mobile / Toggleable) -->
    {#if showVirtualKeys}
      <TerminalVirtualKeys onKey={handleVirtualKey} />
    {/if}
  </div>
</div>
