export interface FileEntry {
  name: string
  path: string
  size: number
  is_dir: boolean
  mod_time: string
  permissions: string
}

export interface FileListResponse {
  current_path: string
  files: FileEntry[]
}

export interface FileSavePayload {
  path: string
  content: string
}

export interface FileRenamePayload {
  old_path: string
  new_path: string
}

export interface FileCopyMovePayload {
  src: string
  dst: string
}

export interface FilePermissionsPayload {
  path: string
  mode: string
  owner?: string
}

export interface FileCompressPayload {
  paths: string[]
  dest_zip: string
}

export interface FileExtractPayload {
  zip_path: string
  dest_dir: string
}
