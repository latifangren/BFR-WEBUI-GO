export class ApiError extends Error {
  status: number
  data?: unknown

  constructor(message: string, status: number, data?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.data = data
  }
}

export async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const url = endpoint.startsWith('/') ? endpoint : `/${endpoint}`

  const defaultHeaders: Record<string, string> = {
    Accept: 'application/json',
  }

  if (options.body && !(options.body instanceof FormData)) {
    defaultHeaders['Content-Type'] = 'application/json'
  }

  const response = await fetch(url, {
    ...options,
    credentials: 'include', // essential for cookie auth on Android daemon
    headers: {
      ...defaultHeaders,
      ...options.headers,
    },
  })

  if (!response.ok) {
    if (response.status === 401 && typeof window !== 'undefined') {
      window.dispatchEvent(new CustomEvent('bfr:unauthorized'))
    }

    let errorData: unknown
    try {
      errorData = await response.json()
    } catch {
      errorData = await response.text()
    }
    const message =
      typeof errorData === 'object' && errorData !== null && 'error' in errorData
        ? String((errorData as { error: unknown }).error)
        : typeof errorData === 'object' && errorData !== null && 'message' in errorData
          ? String((errorData as { message: unknown }).message)
          : `HTTP ${response.status}: ${response.statusText}`

    throw new ApiError(message, response.status, errorData)
  }

  const contentType = response.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    const data = await response.json()
    // Guard against silent failures where backend returns HTTP 200 with { success: false, error: "..." }
    if (
      typeof data === 'object' &&
      data !== null &&
      'success' in data &&
      (data as { success: boolean }).success === false
    ) {
      const errMsg = String(
        (data as { error?: unknown }).error ||
          (data as { message?: unknown }).message ||
          'Operation failed'
      )
      throw new ApiError(errMsg, response.status, data)
    }
    return data as T
  }
  return (await response.text()) as unknown as T
}

export const api = {
  get: <T>(endpoint: string, options?: RequestInit) =>
    request<T>(endpoint, { ...options, method: 'GET' }),
  post: <T>(endpoint: string, body?: unknown, options?: RequestInit) =>
    request<T>(endpoint, {
      ...options,
      method: 'POST',
      body: body instanceof FormData ? body : body !== undefined ? JSON.stringify(body) : undefined,
    }),
  delete: <T>(endpoint: string, options?: RequestInit) =>
    request<T>(endpoint, { ...options, method: 'DELETE' }),
}
