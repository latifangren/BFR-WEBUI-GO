import { api } from '../../../api/client'
import { toastStore } from '../../../stores/toast.svelte'
import type {
  FileEntry,
  FileStorageInfo,
  FileListResponse,
  SortField,
  SortOrder,
} from './types'

export class PaneState {
  paneId: 'left' | 'right'
  currentPath = $state('/sdcard')
  files = $state<FileEntry[]>([])
  isLoading = $state(false)
  storageInfo = $state<FileStorageInfo | null>(null)
  searchQuery = $state('')
  isSearching = $state(false)
  searchResults = $state<FileEntry[] | null>(null)
  selectedPaths = $state<string[]>([])
  sortField = $state<SortField>('name')
  sortOrder = $state<SortOrder>('asc')

  constructor(paneId: 'left' | 'right', defaultPath = '/sdcard') {
    this.paneId = paneId
    let initialPath = defaultPath
    if (typeof window !== 'undefined' && typeof localStorage !== 'undefined') {
      try {
        const saved = localStorage.getItem('bfr_fm_path_' + paneId)
        if (saved && saved.startsWith('/')) {
          initialPath = saved
        }
      } catch {
        // localStorage not available or permission denied
      }
    }
    this.currentPath = initialPath
  }

  get displayedFiles(): FileEntry[] {
    const list = this.searchResults !== null ? this.searchResults : this.files
    const sorted = [...list].sort((a, b) => {
      // Folders always on top
      if (a.is_dir && !b.is_dir) return -1
      if (!a.is_dir && b.is_dir) return 1

      let cmp = 0
      switch (this.sortField) {
        case 'name':
          cmp = a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })
          break
        case 'size':
          cmp = a.size - b.size
          break
        case 'modified':
          cmp = new Date(a.mod_time).getTime() - new Date(b.mod_time).getTime()
          break
        case 'type': {
          const extA = a.name.includes('.') ? a.name.split('.').pop() || '' : ''
          const extB = b.name.includes('.') ? b.name.split('.').pop() || '' : ''
          cmp = extA.localeCompare(extB) || a.name.localeCompare(b.name)
          break
        }
      }
      return this.sortOrder === 'asc' ? cmp : -cmp
    })
    return sorted
  }

  get allSelected(): boolean {
    const activeList = this.searchResults !== null ? this.searchResults : this.files
    return activeList.length > 0 && this.selectedPaths.length === activeList.length
  }

  get breadcrumbs(): { name: string; path: string }[] {
    const segs = this.currentPath.split('/').filter(Boolean)
    return segs.map((seg, idx, arr) => ({
      name: seg,
      path: '/' + arr.slice(0, idx + 1).join('/'),
    }))
  }

  async navigateTo(path: string): Promise<void> {
    const cleanPath = path.trim() || '/'
    try {
      this.isLoading = true
      const res = await api.get<FileListResponse>(`/api/files/list?path=${encodeURIComponent(cleanPath)}`)
      this.currentPath = res.path || res.current_path || cleanPath
      this.files = Array.isArray(res.files) ? res.files : []
      if (typeof window !== 'undefined' && typeof localStorage !== 'undefined') {
        try {
          localStorage.setItem('bfr_fm_path_' + this.paneId, this.currentPath)
        } catch {
          // Ignored
        }
      }
      this.clearSearch()
      this.clearSelection()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to list directory'
      toastStore.error(msg)
    } finally {
      this.isLoading = false
    }
  }

  async refresh(): Promise<void> {
    await this.navigateTo(this.currentPath)
  }

  async fetchStorage(): Promise<void> {
    try {
      const res = await api.get<FileStorageInfo>('/api/files/storage')
      if (res) {
        this.storageInfo = res
      }
    } catch {
      // Ignored if storage query unsupported
    }
  }

  goUp(): void {
    if (this.currentPath === '/' || !this.currentPath) return
    const parts = this.currentPath.split('/').filter(Boolean)
    parts.pop()
    const parent = '/' + parts.join('/')
    this.navigateTo(parent || '/')
  }

  async runSearch(): Promise<void> {
    const q = this.searchQuery.trim()
    if (!q) {
      this.clearSearch()
      return
    }
    try {
      this.isSearching = true
      this.clearSelection()
      const res = await api.get<{ files: FileEntry[] }>(
        `/api/files/search?path=${encodeURIComponent(this.currentPath)}&query=${encodeURIComponent(q)}`
      )
      this.searchResults = Array.isArray(res?.files) ? res.files : []
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Search failed'
      toastStore.error(msg)
    } finally {
      this.isSearching = false
    }
  }

  clearSearch(): void {
    this.searchQuery = ''
    this.searchResults = null
    this.isSearching = false
  }

  toggleSelect(path: string): void {
    if (this.selectedPaths.includes(path)) {
      this.selectedPaths = this.selectedPaths.filter((p) => p !== path)
    } else {
      this.selectedPaths = [...this.selectedPaths, path]
    }
  }

  toggleSelectAll(): void {
    const activeList = this.searchResults !== null ? this.searchResults : this.files
    if (this.selectedPaths.length === activeList.length && activeList.length > 0) {
      this.selectedPaths = []
    } else {
      this.selectedPaths = activeList.map((f) => f.path)
    }
  }

  clearSelection(): void {
    this.selectedPaths = []
  }

  setSort(field: SortField): void {
    if (this.sortField === field) {
      this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc'
    } else {
      this.sortField = field
      this.sortOrder = 'asc'
    }
  }

  static formatBytes(bytes: number): string {
    if (!bytes || bytes <= 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i] || 'B'}`
  }

  static isArchive(file: FileEntry): boolean {
    if (!file || file.is_dir) return false
    const n = file.name.toLowerCase()
    return n.endsWith('.zip') || n.endsWith('.tar.gz') || n.endsWith('.tgz') || n.endsWith('.tar')
  }
}

export function createPaneState(paneId: 'left' | 'right', defaultPath = '/sdcard'): PaneState {
  return new PaneState(paneId, defaultPath)
}
