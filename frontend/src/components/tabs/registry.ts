import type { Component } from 'svelte'

import TabOverview from './overview/TabOverview.svelte'
import TabSysinfo from './sysinfo/TabSysinfo.svelte'
import TabKernel from './kernel/TabKernel.svelte'
import TabNetwork from './network/TabNetwork.svelte'
import TabHotspot from './hotspot/TabHotspot.svelte'
import TabPower from './power/TabPower.svelte'
import TabCharger from './charger/TabCharger.svelte'
import TabModem from './modem/TabModem.svelte'
import TabQualcomm from './qualcomm/TabQualcomm.svelte'
import TabSamsung from './samsung/TabSamsung.svelte'
import TabShannon from './shannon/TabShannon.svelte'
import TabTerminal from './terminal/TabTerminal.svelte'
import TabScrcpy from './scrcpy/TabScrcpy.svelte'
import TabFileManager from './files/TabFileManager.svelte'
import TabLogs from './logs/TabLogs.svelte'
import TabProxy from './proxy/TabProxy.svelte'
import TabQoS from './qos/TabQoS.svelte'
import TabVnstat from './vnstat/TabVnstat.svelte'
import TabNAS from './nas/TabNAS.svelte'
import TabTunnel from './tunnel/TabTunnel.svelte'
import TabSpeedtest from './speedtest/TabSpeedtest.svelte'
import TabTools from './tools/TabTools.svelte'
import TabTelegram from './telegram/TabTelegram.svelte'
import TabAbout from './about/TabAbout.svelte'
import TabSSH from './ssh/TabSSH.svelte'
import TabModules from './modules/TabModules.svelte'
import TabPartitions from './partitions/TabPartitions.svelte'
import TabSMS from './sms/TabSMS.svelte'

export const TAB_COMPONENTS: Record<string, Component> = {
  overview: TabOverview,
  sysinfo: TabSysinfo,
  kernel: TabKernel,
  network: TabNetwork,
  hotspot: TabHotspot,
  power: TabPower,
  charger: TabCharger,
  modem: TabModem,
  qualcomm: TabQualcomm,
  samsung: TabSamsung,
  shannon: TabShannon,
  terminal: TabTerminal,
  scrcpy: TabScrcpy,
  files: TabFileManager,
  logs: TabLogs,
  proxy: TabProxy,
  qos: TabQoS,
  vnstat: TabVnstat,
  nas: TabNAS,
  tunnel: TabTunnel,
  speedtest: TabSpeedtest,
  tools: TabTools,
  telegram: TabTelegram,
  about: TabAbout,
  ssh: TabSSH,
  modules: TabModules,
  partitions: TabPartitions,
  sms: TabSMS,
}

export const DEFAULT_TAB_COMPONENT: Component = TabOverview
