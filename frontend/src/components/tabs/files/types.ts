export type {
  FileEntry,
  FileStorageInfo,
  FileListResponse,
  FileReadResponse,
  FilePermissionsPayload,
  FileCompressPayload,
  FileExtractPayload,
  FileCopyMovePayload,
  FileBatchPayload,
  FileSavePayload,
  FileRenamePayload,
} from '../../../types/files'

import type { FileEntry } from '../../../types/files'

export interface Bookmark {
  name: string
  path: string
  isPreset?: boolean
}

export type SortField = 'name' | 'size' | 'modified' | 'type'

export type SortOrder = 'asc' | 'desc'

export type ViewMode = 'single' | 'dual'

export interface FileModalState {
  editor: {
    isOpen: boolean
    path: string
    content: string
    isLoading: boolean
    isSaving: boolean
  }
  create: {
    isOpen: boolean
    type: 'file' | 'folder'
    name: string
  }
  rename: {
    isOpen: boolean
    target: FileEntry | null
    newName: string
  }
  delete: {
    isOpen: boolean
    target: FileEntry | null
  }
  chmod: {
    isOpen: boolean
    target: FileEntry | null
    mode: string
    owner: string
  }
  compress: {
    isOpen: boolean
    target: FileEntry | null
    destName: string
  }
  extract: {
    isOpen: boolean
    target: FileEntry | null
    destDir: string
  }
  copyMove: {
    isOpen: boolean
    target: FileEntry | null
    action: 'copy' | 'move'
    destPath: string
  }
  batch: {
    deleteOpen: boolean
    copyOpen: boolean
    moveOpen: boolean
    compressOpen: boolean
    destDir: string
    zipName: string
  }
}
