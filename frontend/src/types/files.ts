export interface FileEntry {
  name: string
  path: string
  size: number
  is_dir: boolean
  mod_time: string
  permissions: string
}

export interface FileListResponse {
  path: string
  current_path?: string
  files: FileEntry[]
}

export interface FileReadResponse {
  path: string
  content: string
}

export interface FileStorageInfo {
  total: number
  used: number
  free: number
  used_pct: number
  mount?: string
}

export interface FilePermissionsPayload {
  path: string
  mode: string
  owner?: string
}

export interface FileCompressPayload {
  paths: string[]
  dest_zip?: string
  destination?: string
}

export interface FileExtractPayload {
  path?: string
  zip_path?: string
  destination?: string
  dest_dir?: string
}

export interface FileCopyMovePayload {
  src: string
  dst: string
}

export interface FileBatchPayload {
  action: 'delete' | 'copy' | 'move'
  items?: string[]
  paths?: string[]
  destination?: string
  dest_dir?: string
}

export interface FileSavePayload {
  path: string
  content: string
}

export interface FileRenamePayload {
  old_path: string
  new_path: string
}
