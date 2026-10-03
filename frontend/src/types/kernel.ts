export interface CPUCluster {
  policy_id: number
  cores: string
  label: string
  governor: string
  available_governors: string[]
  min_freq: number
  max_freq: number
  cur_freq: number
  available_frequencies: number[]
}

export interface VMConfig {
  swappiness: number
  vfs_cache_pressure: number
  dirty_ratio: number
  dirty_background_ratio: number
  tcp_congestion: string
  available_tcp_congestion: string[]
}

export interface ClusterConfig {
  policy_id: number
  governor?: string
  min_freq?: number
  max_freq?: number
}

export interface KernelPersistentConfig {
  active_profile?: string
  global_governor?: string
  clusters?: ClusterConfig[]
  swappiness?: number
  vfs_cache_pressure?: number
  dirty_ratio?: number
  dirty_background_ratio?: number
  tcp_congestion?: string
}

export interface KernelStatusResponse {
  clusters: CPUCluster[]
  vm: VMConfig
  config: KernelPersistentConfig
  active_profile: string
  kernel_release: string
}
