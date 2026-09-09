<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { Terminal as XTerminal } from '@xterm/xterm'
  import { FitAddon } from '@xterm/addon-fit'
  import '@xterm/xterm/css/xterm.css'
  import { Terminal, RefreshCw, Maximize2, Trash2 } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'

  let terminalContainer: HTMLDivElement | null = $state(null)
  let term: XTerminal | null = null
  let fitAddon: FitAddon | null = null
  let ws: WebSocket | null = null
  let isConnected = $state(false)
  let isConnecting = $state(false)

  onMount(() => {
    initTerminal()
    connectWebSocket()

    const handleResize = () => {
      if (fitAddon && term) {
        fitAddon.fit()
      }
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

  function cleanup() {
    if (ws) {
      ws.onopen = null
      ws.onclose = null
      ws.onerror = null
      ws.onmessage = null
      ws.close()
      ws = null
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

    setTimeout(() => {
      fitAddon?.fit()
    }, 100)

    term.onData((data) => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(data)
      }
    })
  }

  function connectWebSocket() {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    isConnecting = true
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const url = `${protocol}//${host}/api/terminal/ws`

    try {
      ws = new WebSocket(url)
      ws.binaryType = 'blob'

      ws.onopen = () => {
        isConnected = true
        isConnecting = false
        term?.write('\r\n\x1b[32m[+] Interactive Root Terminal Connected\x1b[0m\r\n\r\n')
        fitAddon?.fit()
      }

      ws.onmessage = async (event) => {
        if (typeof event.data === 'string') {
          term?.write(event.data)
        } else if (event.data instanceof Blob) {
          const text = await event.data.text()
          term?.write(text)
        } else if (event.data instanceof ArrayBuffer) {
          const decoder = new TextDecoder()
          term?.write(decoder.decode(event.data))
        }
      }

      ws.onerror = () => {
        term?.write('\r\n\x1b[31m[-] WebSocket Error\x1b[0m\r\n')
      }

      ws.onclose = () => {
        isConnected = false
        isConnecting = false
        term?.write('\r\n\x1b[33m[*] Terminal Session Closed. Reconnecting...\x1b[0m\r\n')
      }
    } catch {
      isConnecting = false
      isConnected = false
    }
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
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Terminal class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Interactive Root Terminal
      </h2>
      <Badge variant={isConnected ? 'success' : isConnecting ? 'warning' : 'default'}>
        {isConnected ? 'Connected (Root)' : isConnecting ? 'Connecting...' : 'Disconnected'}
      </Badge>
    </div>

    <div class="flex items-center gap-2">
      <Button variant="outline" size="sm" onclick={clearTerminal} title="Clear Terminal">
        <Trash2 class="w-3.5 h-3.5 mr-1" />
        <span>Clear</span>
      </Button>
      <Button variant="secondary" size="sm" onclick={reconnect} title="Reconnect Terminal">
        <RefreshCw class="w-3.5 h-3.5 mr-1 {isConnecting ? 'animate-spin' : ''}" />
        <span>Reconnect</span>
      </Button>
    </div>
  </div>

  <!-- Terminal Window (Neo-Brutalist Frame) -->
  <div class="neo-card bg-card border-2 border-border shadow-neobrutal rounded-lg overflow-hidden flex flex-col">
    <!-- Terminal Header Bar -->
    <div class="bg-card-sub border-b border-border px-3 py-2 flex items-center justify-between font-mono text-xs text-muted">
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
          onclick={() => fitAddon?.fit()}
        >
          <Maximize2 class="w-3 h-3" />
          <span>Fit</span>
        </button>
      </div>
    </div>

    <!-- Terminal Canvas Container -->
    <div
      bind:this={terminalContainer}
      class="w-full h-[65vh] p-2 bg-[#090d16] font-mono"
    ></div>
  </div>
</div>
