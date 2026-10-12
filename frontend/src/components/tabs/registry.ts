import type { Component } from 'svelte'

import { VENDOR_TABS } from './vendor_tabs'

export interface TabModule {
  default: Component
}

export type TabLoader = () => Promise<TabModule>

// Vendor tabs stay statically imported (per-flavor file, overwritten at build
// time for the nomodem variant). Wrap them so every tab resolves through the
// same async loader interface.
const vendorLoaders: Record<string, TabLoader> = Object.fromEntries(
  Object.entries(VENDOR_TABS).map(([id, component]) => [
    id,
    () => Promise.resolve({ default: component }),
  ])
)

// Dynamic per-tab imports: each tab becomes its own chunk so the initial
// bundle stays small on the phone's WebUI. The terminal chunk (xterm) in
// particular is only fetched when the tab is first opened.
export const TAB_LOADERS: Record<string, TabLoader> = {
  overview: () => import('./overview/TabOverview.svelte'),
  sysinfo: () => import('./sysinfo/TabSysinfo.svelte'),
  kernel: () => import('./kernel/TabKernel.svelte'),
  network: () => import('./network/TabNetwork.svelte'),
  hotspot: () => import('./hotspot/TabHotspot.svelte'),
  power: () => import('./power/TabPower.svelte'),
  charger: () => import('./charger/TabCharger.svelte'),
  modem: () => import('./modem/TabModem.svelte'),
  ...vendorLoaders,
  terminal: () => import('./terminal/TabTerminal.svelte'),
  scrcpy: () => import('./scrcpy/TabScrcpy.svelte'),
  files: () => import('./files/TabFileManager.svelte'),
  logs: () => import('./logs/TabLogs.svelte'),
  proxy: () => import('./proxy/TabProxy.svelte'),
  qos: () => import('./qos/TabQoS.svelte'),
  vnstat: () => import('./vnstat/TabVnstat.svelte'),
  nas: () => import('./nas/TabNAS.svelte'),
  tunnel: () => import('./tunnel/TabTunnel.svelte'),
  speedtest: () => import('./speedtest/TabSpeedtest.svelte'),
  tools: () => import('./tools/TabTools.svelte'),
  telegram: () => import('./telegram/TabTelegram.svelte'),
  about: () => import('./about/TabAbout.svelte'),
  ssh: () => import('./ssh/TabSSH.svelte'),
  modules: () => import('./modules/TabModules.svelte'),
  partitions: () => import('./partitions/TabPartitions.svelte'),
  sms: () => import('./sms/TabSMS.svelte'),
}

export const DEFAULT_TAB_LOADER: TabLoader = TAB_LOADERS.overview

// Dedicated loader for the keep-alive terminal tab (see App.svelte).
export const TERMINAL_TAB_LOADER: TabLoader = TAB_LOADERS.terminal
