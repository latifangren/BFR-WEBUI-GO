import type { ToastMessage } from '../types/common'

class ToastStore {
  toasts = $state<ToastMessage[]>([])

  show(message: string, type: ToastMessage['type'] = 'info', title = '', duration = 4000) {
    const id = Math.random().toString(36).substring(2, 9)
    const toast: ToastMessage = {
      id,
      title: title || type.toUpperCase(),
      message,
      type,
      duration,
    }

    this.toasts = [...this.toasts, toast]

    if (duration > 0) {
      setTimeout(() => {
        this.dismiss(id)
      }, duration)
    }
  }

  dismiss(id: string) {
    this.toasts = this.toasts.filter((t) => t.id !== id)
  }

  success(message: string, title = 'Success') {
    this.show(message, 'success', title)
  }

  error(message: string, title = 'Error') {
    this.show(message, 'error', title)
  }

  warning(message: string, title = 'Warning') {
    this.show(message, 'warning', title)
  }

  info(message: string, title = 'Info') {
    this.show(message, 'info', title)
  }
}

export const toastStore = new ToastStore()
