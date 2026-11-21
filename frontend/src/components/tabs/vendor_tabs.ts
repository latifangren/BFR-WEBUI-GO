import type { Component } from 'svelte'

type VendorTabModule = { default: Component }
type VendorTabLoader = () => Promise<VendorTabModule>

export const VENDOR_TABS: Record<string, VendorTabLoader> = {
  qualcomm: () => import('./qualcomm/TabQualcomm.svelte'),
  samsung: () => import('./samsung/TabSamsung.svelte'),
  shannon: () => import('./shannon/TabShannon.svelte'),
}
