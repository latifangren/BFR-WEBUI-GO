export type PartitionCategory = 'radio_efs' | 'boot_recovery' | 'system_os' | 'other'

export interface PartitionInfo {
  name: string
  device_path: string
  canonical_path: string
  size_bytes: number
  size_human: string
  category: PartitionCategory
  is_critical: boolean
  slot: string
  is_active_slot: boolean
}

export interface SystemSlotInfo {
  current_slot: string
  is_ab_device: boolean
  block_device_root: string
  total_partitions: number
}

export interface BackupFile {
  filename: string
  path: string
  partition_name: string
  size_bytes: number
  size_human: string
  sha256: string
  is_compressed: boolean
  created_at: string
}

export interface BackupResult {
  success: boolean
  partition_name: string
  filename?: string
  file_path?: string
  size_bytes?: number
  sha256?: string
  error?: string
}

export interface FlashResult {
  success: boolean
  partition_name: string
  backup_file?: string
  sha256?: string
  message?: string
  error?: string
}

export interface PartitionListResponse {
  partitions: PartitionInfo[]
  slot_info: SystemSlotInfo
}

export interface BackupListResponse {
  backups: BackupFile[]
}

export interface BackupGroupResponse {
  group: string
  results: BackupResult[]
}
