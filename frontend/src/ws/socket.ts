export type SocketState = 'CONNECTING' | 'OPEN' | 'CLOSING' | 'CLOSED'

export interface WebSocketClientOptions {
  path: string
  autoReconnect?: boolean
  maxReconnectAttempts?: number
  reconnectIntervalMs?: number
  binaryType?: BinaryType
  onOpen?: (event: Event) => void
  onClose?: (event: CloseEvent) => void
  onError?: (event: Event) => void
  onMessage?: (data: string | ArrayBuffer | Blob) => void
}

export class WebSocketClient {
  private ws: WebSocket | null = null
  private options: WebSocketClientOptions
  private reconnectAttempts = 0
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private isManuallyClosed = false

  constructor(options: WebSocketClientOptions) {
    this.options = {
      autoReconnect: true,
      maxReconnectAttempts: 10,
      reconnectIntervalMs: 2000,
      binaryType: 'blob',
      ...options,
    }
  }

  public connect() {
    this.isManuallyClosed = false
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const fullUrl = this.options.path.startsWith('ws')
      ? this.options.path
      : `${protocol}//${host}${this.options.path.startsWith('/') ? '' : '/'}${this.options.path}`

    try {
      this.ws = new WebSocket(fullUrl)
      if (this.options.binaryType) {
        this.ws.binaryType = this.options.binaryType
      }

      this.ws.onopen = (e) => {
        this.reconnectAttempts = 0
        this.options.onOpen?.(e)
      }

      this.ws.onclose = (e) => {
        this.options.onClose?.(e)
        if (!this.isManuallyClosed && this.options.autoReconnect) {
          this.scheduleReconnect()
        }
      }

      this.ws.onerror = (e) => {
        this.options.onError?.(e)
      }

      this.ws.onmessage = (e) => {
        this.options.onMessage?.(e.data)
      }
    } catch (err) {
      console.error('[WebSocketClient] Connection creation failed:', err)
      this.scheduleReconnect()
    }
  }

  public send(data: Parameters<WebSocket['send']>[0]) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(data)
    } else {
      console.warn('[WebSocketClient] Cannot send, socket is not open')
    }
  }

  public close() {
    this.isManuallyClosed = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws) {
      this.ws.onopen = null
      this.ws.onclose = null
      this.ws.onerror = null
      this.ws.onmessage = null
      this.ws.close()
      this.ws = null
    }
  }

  public disconnect() {
    this.close()
  }

  private scheduleReconnect() {
    if (this.reconnectAttempts >= (this.options.maxReconnectAttempts || 10)) {
      console.warn('[WebSocketClient] Max reconnect attempts reached')
      return
    }

    const delay = Math.min((this.options.reconnectIntervalMs || 2000) * Math.pow(1.5, this.reconnectAttempts), 15000)
    this.reconnectAttempts++

    this.reconnectTimer = setTimeout(() => {
      this.connect()
    }, delay)
  }

  public get readyState(): number {
    return this.ws ? this.ws.readyState : WebSocket.CLOSED
  }
}
