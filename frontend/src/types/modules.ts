export interface ModuleInfo {
  id: string
  name: string
  version: string
  version_code?: string | number
  versionCode?: number
  author: string
  description: string
  enabled: boolean
  update_json?: string
  updateJson?: string
}

export interface ModulesResponse {
  modules: ModuleInfo[]
}
