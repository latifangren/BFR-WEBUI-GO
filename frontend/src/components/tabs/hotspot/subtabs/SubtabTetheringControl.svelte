<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Wifi,
    Usb,
    Network,
    Power,
    Check,
    Radio,
    Shield,
    Server,
    Sliders,
    Globe,
    Cpu,
  } from '@lucide/svelte'
  import { api } from '../../../../api/client'
  import { toastStore } from '../../../../stores/toast.svelte'
  import type { HotspotStatus, RouterConfig, RouterStatus, RouterSettingsResponse } from '../../../../types/hotspot'
  import { LL } from '../../../../i18n/i18n-svelte'
  import Card from '../../../ui/Card.svelte'
  import Button from '../../../ui/Button.svelte'
  import Badge from '../../../ui/Badge.svelte'
  import Input from '../../../ui/Input.svelte'

  interface Props {
    status: HotspotStatus
    onRefresh: () => Promise<void>
  }

  let { status = $bindable(), onRefresh }: Props = $props()

  // Wi-Fi Credentials state
  let ssidInput = $state(status.ssid || 'AndroidAP')
  let passwordInput = $state('')
  let securityType = $state<'wpa2' | 'open'>(status.security_type === 'open' ? 'open' : 'wpa2')
  let selectedBand = $state<'any' | '2' | '5'>('any')

  // Router Suite (L3 Gateway & DHCP) state
  let routerCfg = $state<RouterConfig>({
    gateway_mode: 'auto',
    gateway_ip: '192.168.1.1',
    subnet_mask: '255.255.255.0',
    dhcp_start: '192.168.1.10',
    dhcp_end: '192.168.1.100',
    lease_time: '12h',
    dns_primary: '1.1.1.1',
    dns_secondary: '8.8.8.8',
    band: 'any',
    upstream_interface: 'auto',
  })

  let routerStatus = $state<RouterStatus>({
    desired_mode: 'auto',
    desired_gateway: '',
    desired_subnet: '',
    desired_band: 'any',
    desired_upstream: 'auto',
    dhcp_managed_by_os: true,
    apply_state: 'inactive',
    applied: false,
    observed_iface: '',
    upstream_table: '',
    address_present: false,
    rule_to_subnet_present: false,
    rule_from_ap_present: false,
  })

  function applyStateLabel(state: string | undefined, mode: string): string {
    switch (state) {
      case 'applied':
        return $LL.hotspot.tethering.state.applied()
      case 'auto':
        return $LL.hotspot.tethering.state.androidManaged()
      case 'pending':
        return $LL.hotspot.tethering.state.savedPending()
      case 'failed':
        return $LL.hotspot.tethering.state.failed()
      case 'inactive':
        return mode === 'static' ? $LL.hotspot.tethering.state.hotspotOff() : $LL.hotspot.tethering.state.androidManaged()
      default:
        return $LL.hotspot.tethering.state.unknown()
    }
  }

  function applyStateVariant(state: string | undefined): 'success' | 'warning' | 'danger' | 'default' {
    switch (state) {
      case 'applied':
        return 'success'
      case 'pending':
        return 'warning'
      case 'failed':
        return 'danger'
      default:
        return 'default'
    }
  }

  let isTogglingWifi = $state(false)
  let isTogglingUsb = $state(false)
  let isTogglingEthernet = $state(false)
  let isSavingWifi = $state(false)
  let isSavingRouter = $state(false)
  let hasInitializedSSID = false

  onMount(async () => {
    await fetchRouterSettings()
  })

  $effect(() => {
    if (status.ssid && !hasInitializedSSID) {
      ssidInput = status.ssid
      hasInitializedSSID = true
    }
  })

  async function fetchRouterSettings() {
    try {
      const res = await api.get<RouterSettingsResponse>('/api/hotspot/settings')
      if (res && res.config) {
        routerCfg = { ...routerCfg, ...res.config }
        selectedBand = res.config.band || 'any'
        if (res.config.ssid && !hasInitializedSSID) {
          ssidInput = res.config.ssid
          hasInitializedSSID = true
        }
        if (res.config.security_type) {
          securityType = res.config.security_type === 'open' ? 'open' : 'wpa2'
        }
      }
      if (res && res.status) {
        routerStatus = res.status
      }
    } catch {
      // Ignored
    }
  }

  let isSavingAutoStart = $state(false)

  async function toggleAutoStart(channel: 'wifi' | 'usb' | 'ethernet') {
    try {
      isSavingAutoStart = true
      const nextCfg: RouterConfig = {
        ...routerCfg,
        band: selectedBand,
        ssid: ssidInput,
        security_type: securityType,
        auto_start_wifi: channel === 'wifi' ? !routerCfg.auto_start_wifi : routerCfg.auto_start_wifi,
        auto_start_usb: channel === 'usb' ? !routerCfg.auto_start_usb : routerCfg.auto_start_usb,
        auto_start_ethernet: channel === 'ethernet' ? !routerCfg.auto_start_ethernet : routerCfg.auto_start_ethernet,
      }
      if (passwordInput && securityType !== 'open') {
        nextCfg.password = passwordInput
      }

      await api.post('/api/hotspot/settings', nextCfg)
      routerCfg = nextCfg
      const isEnabled = channel === 'wifi' ? nextCfg.auto_start_wifi : channel === 'usb' ? nextCfg.auto_start_usb : nextCfg.auto_start_ethernet
      toastStore.success(
        isEnabled
          ? $LL.hotspot.tethering.toasts.autoStartOn({ channel: channel.toUpperCase() })
          : $LL.hotspot.tethering.toasts.autoStartOff({ channel: channel.toUpperCase() })
      )
      await fetchRouterSettings()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.hotspot.tethering.toasts.autoStartFailed())
    } finally {
      isSavingAutoStart = false
    }
  }

  async function toggleChannel(type: 'wifi' | 'usb' | 'ethernet') {
    let nextState = false
    if (type === 'wifi') {
      nextState = !status.wifi_enabled
      isTogglingWifi = true
    } else if (type === 'usb') {
      nextState = !status.usb_enabled
      isTogglingUsb = true
    } else if (type === 'ethernet') {
      nextState = !status.ethernet_enabled
      isTogglingEthernet = true
    }

    try {
      await api.post('/api/hotspot/control', {
        type,
        enable: nextState,
        ssid: type === 'wifi' ? ssidInput : undefined,
        security: type === 'wifi' ? securityType : undefined,
        band: type === 'wifi' ? selectedBand : undefined,
        password: type === 'wifi' ? (securityType === 'open' ? 'none' : (passwordInput || undefined)) : undefined,
      })
      toastStore.success(
        nextState
          ? $LL.hotspot.tethering.toasts.tetherEnabled({ channel: type.toUpperCase() })
          : $LL.hotspot.tethering.toasts.tetherDisabled({ channel: type.toUpperCase() })
      )
      await onRefresh()
      await fetchRouterSettings()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.hotspot.tethering.toasts.toggleFailed({ channel: type }))
    } finally {
      if (type === 'wifi') isTogglingWifi = false
      if (type === 'usb') isTogglingUsb = false
      if (type === 'ethernet') isTogglingEthernet = false
    }
  }

  async function saveWifiConfig() {
    if (!ssidInput) return
    try {
      isSavingWifi = true
      const nextCfg: RouterConfig = {
        ...routerCfg,
        band: selectedBand,
        ssid: ssidInput,
        security_type: securityType,
      }
      if (securityType === 'open') {
        nextCfg.password = 'none'
      } else if (passwordInput) {
        nextCfg.password = passwordInput
      }
      await api.post('/api/hotspot/settings', nextCfg)
      routerCfg = nextCfg

      if (status.wifi_enabled) {
        await api.post('/api/hotspot/control', {
          type: 'wifi',
          enable: true,
          ssid: ssidInput,
          security: securityType,
          band: selectedBand,
          password: securityType === 'open' ? 'none' : (passwordInput || undefined),
        })
      }
      status.ssid = ssidInput
      status.security_type = securityType
      passwordInput = ''
      toastStore.success(
        status.wifi_enabled
          ? $LL.hotspot.tethering.toasts.softapReloaded({ ssid: ssidInput, band: selectedBand.toUpperCase(), mode: securityType.toUpperCase() })
          : $LL.hotspot.tethering.toasts.softapSaved({ ssid: ssidInput, band: selectedBand.toUpperCase(), mode: securityType.toUpperCase() })
      )
      await onRefresh()
      await fetchRouterSettings()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.hotspot.tethering.toasts.hotspotConfigFailed())
    } finally {
      isSavingWifi = false
    }
  }

  async function saveRouterSettings() {
    try {
      isSavingRouter = true
      routerCfg.band = selectedBand
      const res = await api.post<{ saved?: boolean; applied?: boolean; error?: string }>(
        '/api/hotspot/settings',
        routerCfg
      )
      if (res && res.saved && routerCfg.gateway_mode !== 'static') {
        toastStore.success($LL.hotspot.tethering.toasts.routerSavedAuto())
      } else if (res && res.saved && res.applied) {
        toastStore.success($LL.hotspot.tethering.toasts.routerSavedApplied())
      } else if (res && res.saved) {
        toastStore.warning($LL.hotspot.tethering.toasts.savedNotApplied({ error: res.error || $LL.hotspot.tethering.toasts.runtimeUnchanged() }))
      } else {
        toastStore.error(res?.error || $LL.hotspot.tethering.toasts.saveRouterFailed())
      }
      await onRefresh()
      await fetchRouterSettings()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.hotspot.tethering.toasts.saveRouterFailed())
    } finally {
      isSavingRouter = false
    }
  }
</script>

<div class="space-y-4">
  <!-- Multi-channel Tethering Grid -->
  <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
    <!-- 1. Wi-Fi Hotspot Channel -->
    <div class="p-3.5 bg-card-sub border-2 border-border rounded-lg flex flex-col justify-between gap-3">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-2">
          <div class="p-2 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <Wifi class="w-4 h-4" />
          </div>
          <div>
            <h4 class="font-bold text-foreground text-xs uppercase tracking-wider">{$LL.hotspot.tethering.wifiTitle()}</h4>
            <span class="text-[10px] text-muted">{$LL.hotspot.tethering.wifiSub()}</span>
          </div>
        </div>
        <Badge variant={status.wifi_enabled ? 'success' : 'default'}>
          {status.wifi_enabled ? $LL.common.active() : $LL.hotspot.tethering.off()}
        </Badge>
      </div>

      <div class="text-[11px] font-mono text-muted space-y-0.5">
        <div>SSID: <span class="text-foreground font-bold">{status.ssid || 'AndroidAP'}</span></div>
        <div>Band: <span class="text-accent font-bold uppercase">{selectedBand === 'any' ? $LL.hotspot.tethering.dualAuto() : `${selectedBand} GHz`}</span></div>
      </div>

      <div class="flex items-center justify-between pt-2 border-t border-border/40 text-[11px]">
        <span class="text-muted font-medium">{$LL.hotspot.tethering.autoStart()}</span>
        <button
          type="button"
          class="px-2 py-0.5 rounded text-[10px] font-bold border transition-colors cursor-pointer {routerCfg.auto_start_wifi ? 'bg-emerald-500/20 text-emerald-400 border-emerald-500/40' : 'bg-card text-muted border-border hover:border-muted'}"
          disabled={isSavingAutoStart}
          onclick={() => toggleAutoStart('wifi')}
        >
          {routerCfg.auto_start_wifi ? 'ENABLED' : 'DISABLED'}
        </button>
      </div>

      <Button
        variant={status.wifi_enabled ? 'danger' : 'primary'}
        size="sm"
        disabled={isTogglingWifi}
        onclick={() => toggleChannel('wifi')}
      >
        <Power class="w-3.5 h-3.5 mr-1" />
        <span>{status.wifi_enabled ? $LL.hotspot.tethering.stopHotspot() : $LL.hotspot.tethering.startHotspot()}</span>
      </Button>
    </div>

    <!-- 2. USB Tethering Channel -->
    <div class="p-3.5 bg-card-sub border-2 border-border rounded-lg flex flex-col justify-between gap-3">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-2">
          <div class="p-2 rounded bg-sky-500/10 text-sky-400 border border-sky-500/20">
            <Usb class="w-4 h-4" />
          </div>
          <div>
            <h4 class="font-bold text-foreground text-xs uppercase tracking-wider">{$LL.hotspot.tethering.usbTitle()}</h4>
            <span class="text-[10px] text-muted">{$LL.hotspot.tethering.usbSub()}</span>
          </div>
        </div>
        <Badge variant={status.usb_enabled ? 'success' : 'default'}>
          {status.usb_enabled ? $LL.common.active() : $LL.hotspot.tethering.off()}
        </Badge>
      </div>

      <div class="text-[11px] font-mono text-muted space-y-0.5">
        <div>Port: <span class="text-foreground font-bold">rndis0 / usb0</span></div>
        <div>{$LL.hotspot.tethering.linkLabel()} <span class="text-accent font-bold">Host OTG</span></div>
      </div>

      <div class="flex items-center justify-between pt-2 border-t border-border/40 text-[11px]">
        <span class="text-muted font-medium">{$LL.hotspot.tethering.autoStart()}</span>
        <button
          type="button"
          class="px-2 py-0.5 rounded text-[10px] font-bold border transition-colors cursor-pointer {routerCfg.auto_start_usb ? 'bg-sky-500/20 text-sky-400 border-sky-500/40' : 'bg-card text-muted border-border hover:border-muted'}"
          disabled={isSavingAutoStart}
          onclick={() => toggleAutoStart('usb')}
        >
          {routerCfg.auto_start_usb ? 'ENABLED' : 'DISABLED'}
        </button>
      </div>

      <Button
        variant={status.usb_enabled ? 'danger' : 'secondary'}
        size="sm"
        disabled={isTogglingUsb}
        onclick={() => toggleChannel('usb')}
      >
        <Power class="w-3.5 h-3.5 mr-1" />
        <span>{status.usb_enabled ? $LL.hotspot.tethering.stopUsb() : $LL.hotspot.tethering.startUsb()}</span>
      </Button>
    </div>

    <!-- 3. Ethernet Tethering Channel -->
    <div class="p-3.5 bg-card-sub border-2 border-border rounded-lg flex flex-col justify-between gap-3">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-2">
          <div class="p-2 rounded bg-amber-500/10 text-amber-400 border border-amber-500/20">
            <Network class="w-4 h-4" />
          </div>
          <div>
            <h4 class="font-bold text-foreground text-xs uppercase tracking-wider">{$LL.hotspot.tethering.ethTitle()}</h4>
            <span class="text-[10px] text-muted">{$LL.hotspot.tethering.ethSub()}</span>
          </div>
        </div>
        <Badge variant={status.ethernet_enabled ? 'success' : 'default'}>
          {status.ethernet_enabled ? $LL.common.active() : $LL.hotspot.tethering.off()}
        </Badge>
      </div>

      <div class="text-[11px] font-mono text-muted space-y-0.5">
        <div>{$LL.hotspot.tethering.interfaceLabel()} <span class="text-foreground font-bold">eth0 gateway</span></div>
        <div>{$LL.hotspot.tethering.typeLabel()} <span class="text-accent font-bold">Gigabit OTG</span></div>
      </div>

      <div class="flex items-center justify-between pt-2 border-t border-border/40 text-[11px]">
        <span class="text-muted font-medium">{$LL.hotspot.tethering.autoStart()}</span>
        <button
          type="button"
          class="px-2 py-0.5 rounded text-[10px] font-bold border transition-colors cursor-pointer {routerCfg.auto_start_ethernet ? 'bg-amber-500/20 text-amber-400 border-amber-500/40' : 'bg-card text-muted border-border hover:border-muted'}"
          disabled={isSavingAutoStart}
          onclick={() => toggleAutoStart('ethernet')}
        >
          {routerCfg.auto_start_ethernet ? 'ENABLED' : 'DISABLED'}
        </button>
      </div>

      <Button
        variant={status.ethernet_enabled ? 'danger' : 'secondary'}
        size="sm"
        disabled={isTogglingEthernet}
        onclick={() => toggleChannel('ethernet')}
      >
        <Power class="w-3.5 h-3.5 mr-1" />
        <span>{status.ethernet_enabled ? $LL.hotspot.tethering.stopEth() : $LL.hotspot.tethering.startEth()}</span>
      </Button>
    </div>
  </div>

  <!-- Gateway & SoftAP Credentials Section -->
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
    <!-- Wi-Fi SSID, Band & Security Form -->
    <Card
      title={$LL.hotspot.tethering.wifi.title()}
      subtitle={$LL.hotspot.tethering.wifi.subtitle()}
      tone="mint"
    >
      <div class="space-y-3 font-mono text-xs">
        <!-- Security Mode Switcher -->
        <div class="flex items-center gap-2 pb-2 border-b border-border">
          <span class="text-muted uppercase font-bold text-[11px]">{$LL.hotspot.tethering.wifi.securityMode()}</span>
          <button
            type="button"
            class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {securityType === 'wpa2' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
            onclick={() => (securityType = 'wpa2')}
          >
            {$LL.hotspot.tethering.wifi.wpa2Secured()}
          </button>
          <button
            type="button"
            class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {securityType === 'open' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
            onclick={() => { securityType = 'open'; passwordInput = ''; }}
          >
            {$LL.hotspot.tethering.wifi.openNoPassword()}
          </button>
        </div>

        <!-- Frequency Band Switcher -->
        <div class="flex items-center gap-2 pb-2 border-b border-border">
          <span class="text-muted uppercase font-bold text-[11px]">{$LL.hotspot.tethering.wifi.radioBand()}</span>
          <button
            type="button"
            class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {selectedBand === 'any' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
            onclick={() => (selectedBand = 'any')}
          >
            {$LL.hotspot.tethering.wifi.autoAny()}
          </button>
          <button
            type="button"
            class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {selectedBand === '5' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
            onclick={() => (selectedBand = '5')}
          >
            {$LL.hotspot.tethering.wifi.band5()}
          </button>
          <button
            type="button"
            class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {selectedBand === '2' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
            onclick={() => (selectedBand = '2')}
          >
            {$LL.hotspot.tethering.wifi.band24()}
          </button>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <Input
            label={$LL.hotspot.tethering.wifi.ssidLabel()}
            placeholder={$LL.hotspot.tethering.wifi.ssidPlaceholder()}
            bind:value={ssidInput}
          />
          {#if securityType === 'wpa2'}
            <Input
              type="password"
              label={$LL.hotspot.tethering.wifi.passphraseLabel()}
              placeholder={$LL.hotspot.tethering.wifi.passphrasePlaceholder()}
              bind:value={passwordInput}
            />
          {:else}
            <div class="space-y-1">
              <span class="text-[10px] text-muted uppercase font-bold">{$LL.hotspot.tethering.wifi.passphraseOpen()}</span>
              <div class="p-2.5 bg-card border border-border rounded text-muted text-xs italic">
                {$LL.hotspot.tethering.wifi.noPasswordRequired()}
              </div>
            </div>
          {/if}
        </div>

        <div class="flex justify-between items-center pt-2 border-t border-border">
          <span class="text-[11px] text-muted">
            {status.wifi_enabled
              ? $LL.hotspot.tethering.wifi.reloadNote()
              : $LL.hotspot.tethering.wifi.savedNote()}
          </span>
          <Button
            variant="secondary"
            size="sm"
            disabled={isSavingWifi || !ssidInput}
            onclick={saveWifiConfig}
          >
            <Check class="w-3.5 h-3.5 mr-1" />
            <span>{status.wifi_enabled ? $LL.hotspot.tethering.wifi.applyReload() : $LL.hotspot.tethering.wifi.saveCredentials()}</span>
          </Button>
        </div>
      </div>
    </Card>

    <!-- Modular Gateway & Router Suite (L3 Engine) -->
    <Card
      title={$LL.hotspot.tethering.router.title()}
      subtitle={$LL.hotspot.tethering.router.subtitle()}
      tone="lavender"
    >
      <div class="space-y-3 font-mono text-xs">
        <!-- Gateway Mode Selector -->
        <div class="flex flex-wrap items-center justify-between gap-2 pb-2 border-b border-border">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-muted uppercase font-bold text-[11px]">{$LL.hotspot.tethering.router.gatewayMode()}</span>
            <button
              type="button"
              class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {routerCfg.gateway_mode === 'auto' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
              onclick={() => (routerCfg.gateway_mode = 'auto')}
            >
              {$LL.hotspot.tethering.router.autoManaged()}
            </button>
            <button
              type="button"
              class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {routerCfg.gateway_mode === 'static' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
              onclick={() => (routerCfg.gateway_mode = 'static')}
            >
              {$LL.hotspot.tethering.router.customStatic()}
            </button>
          </div>

          <Badge variant={applyStateVariant(routerStatus.apply_state)}>
            {applyStateLabel(routerStatus.apply_state, routerStatus.desired_mode)}
          </Badge>
        </div>

        <!-- Live / Observed state -->
        <div class="space-y-2 p-3 bg-card-sub border border-border rounded-lg">
          <div class="flex items-center justify-between">
            <span class="text-muted">{$LL.hotspot.tethering.router.clientGateway()}</span>
            <span class="font-bold text-accent">{routerStatus.client_gateway || status.gateway_ip || '—'}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">{$LL.hotspot.tethering.router.clientSubnet()}</span>
            <span class="font-bold text-foreground">{routerStatus.client_subnet || status.subnet || '—'}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">{$LL.hotspot.tethering.router.tetherInterface()}</span>
            <span class="font-bold text-foreground">{routerStatus.observed_iface || (status.active_interfaces || [])[0] || '—'}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">{$LL.hotspot.tethering.router.upstreamTable()}</span>
            <span class="font-bold text-foreground">{routerStatus.upstream_table || '—'}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">{$LL.hotspot.tethering.router.dhcpDnsServer()}</span>
            <span class="font-bold text-emerald-400">{$LL.hotspot.tethering.router.networkStackOwned()}</span>
          </div>
        </div>

        {#if routerCfg.gateway_mode === 'static' && routerStatus.apply_state === 'applied' && !routerStatus.gateway_effective}
          <div class="p-2.5 bg-card border border-warning/40 rounded-lg text-[10px] text-warning space-y-1">
            <p class="font-bold uppercase">{$LL.hotspot.tethering.router.staticWarningTitle()}</p>
            <p>
              {$LL.hotspot.tethering.router.staticWarningBody({ gateway: routerStatus.observed_gateway || $LL.hotspot.tethering.router.staticAddress(), clientGateway: routerStatus.client_gateway || $LL.hotspot.tethering.router.ownAddress() })}
            </p>
          </div>
        {/if}

        {#if routerCfg.gateway_mode === 'static'}
          <!-- Custom Static Gateway Inputs -->
          <div class="space-y-2">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
              <Input
                label={$LL.hotspot.tethering.router.staticGatewayIp()}
                placeholder="192.168.1.1"
                bind:value={routerCfg.gateway_ip}
              />
              <Input
                label={$LL.hotspot.tethering.router.subnetMask()}
                placeholder="255.255.255.0"
                bind:value={routerCfg.subnet_mask}
              />
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
              <Input
                label={$LL.hotspot.tethering.router.upstreamInterface()}
                placeholder="auto"
                bind:value={routerCfg.upstream_interface}
              />
              <Input
                label={$LL.hotspot.tethering.router.leaseTime()}
                placeholder="12h"
                bind:value={routerCfg.lease_time}
              />
            </div>

            <div class="p-2.5 bg-card border border-border rounded-lg text-[10px] text-muted space-y-1">
              <p class="font-bold text-foreground uppercase">{$LL.hotspot.tethering.router.dhcpPoolTitle()}</p>
              <p>
                <span class="text-foreground">{$LL.hotspot.tethering.router.dhcpRange()}</span>
                {routerCfg.dhcp_start} — {routerCfg.dhcp_end} ·
                <span class="text-foreground">DNS:</span>
                {routerCfg.dns_primary}, {routerCfg.dns_secondary}
              </p>
              <p class="text-warning">
                {$LL.hotspot.tethering.router.dhcpPoolWarning()}
              </p>
            </div>
          </div>
        {:else}
          <p class="text-[10px] text-muted pt-1 border-t border-border/60">
            {$LL.hotspot.tethering.router.autoNote()}
          </p>
        {/if}

        {#if routerStatus.last_apply_error}
          <p class="text-[10px] text-red-400 font-bold">{$LL.hotspot.tethering.router.lastApplyError()} {routerStatus.last_apply_error}</p>
        {/if}

        <div class="flex justify-between items-center pt-2 border-t border-border">
          <div class="text-[10px] text-muted">
            {#if routerStatus.apply_state === 'applied'}
              <span class="text-emerald-400 font-bold">{$LL.hotspot.tethering.router.stateApplied()}</span>
            {:else if routerStatus.apply_state === 'auto'}
              <span>{$LL.hotspot.tethering.router.stateAuto()}</span>
            {:else if routerStatus.apply_state === 'inactive'}
              <span>{$LL.hotspot.tethering.router.stateInactive()}</span>
            {:else if routerStatus.apply_state === 'pending'}
              <span class="text-warning">{$LL.hotspot.tethering.router.statePending()}</span>
            {:else if routerStatus.apply_state === 'failed'}
              <span class="text-red-400">{$LL.hotspot.tethering.router.stateFailed()}</span>
            {:else}
              <span>{$LL.hotspot.tethering.router.stateNotApplied()}</span>
            {/if}
          </div>
          <Button
            variant="primary"
            size="sm"
            disabled={isSavingRouter}
            onclick={saveRouterSettings}
          >
            <Check class="w-3.5 h-3.5 mr-1" />
            <span>{$LL.hotspot.tethering.router.saveRouter()}</span>
          </Button>
        </div>
      </div>
    </Card>
  </div>
</div>
