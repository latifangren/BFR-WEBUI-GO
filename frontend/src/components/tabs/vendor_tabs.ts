import type { Component } from 'svelte'

import TabQualcomm from './qualcomm/TabQualcomm.svelte'
import TabSamsung from './samsung/TabSamsung.svelte'
import TabShannon from './shannon/TabShannon.svelte'

export const VENDOR_TABS: Record<string, Component> = {
  qualcomm: TabQualcomm,
  samsung: TabSamsung,
  shannon: TabShannon,
}
