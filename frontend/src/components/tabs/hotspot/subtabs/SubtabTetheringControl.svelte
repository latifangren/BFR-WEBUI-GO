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
        return 'APPLIED'
      case 'auto':
        return 'ANDROID MANAGED'
      case 'pending':
        return 'SAVED — PENDING'
      case 'failed':
        return 'FAILED'
      case 'inactive':
        return mode === 'static' ? 'HOTSPOT OFF' : 'ANDROID MANAGED'
      default:
        return 'UNKNOWN'
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
      }
      if (res && res.status) {
        routerStatus = res.status
      }
    } catch {
      // Ignored
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
      toastStore.success(`${type.toUpperCase()} tethering ${nextState ? 'enabled' : 'disabled'}.`)
      await onRefresh()
      await fetchRouterSettings()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : `Failed to toggle ${type} tethering`)
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
      // Also update routerCfg.band
      routerCfg.band = selectedBand
      await api.post('/api/hotspot/settings', routerCfg)

      await api.post('/api/hotspot/control', {
        type: 'wifi',
        enable: true,
        ssid: ssidInput,
        security: securityType,
        band: selectedBand,
        password: securityType === 'open' ? 'none' : (passwordInput || undefined),
      })
      status.ssid = ssidInput
      status.security_type = securityType
      passwordInput = ''
      toastStore.success(`SoftAP updated (SSID: ${ssidInput}, Band: ${selectedBand.toUpperCase()}, Mode: ${securityType.toUpperCase()})`)
      await onRefresh()
      await fetchRouterSettings()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to update hotspot config')
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
        toastStore.success('Router settings saved (Android manages gateway, DHCP and DNS).')
      } else if (res && res.saved && res.applied) {
        toastStore.success('Router settings saved and applied to the kernel.')
      } else if (res && res.saved) {
        toastStore.warning(`Settings saved but not applied: ${res.error || 'runtime state unchanged'}`)
      } else {
        toastStore.error(res?.error || 'Failed to save router settings')
      }
      await onRefresh()
      await fetchRouterSettings()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to save router settings')
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
            <h4 class="font-bold text-foreground text-xs uppercase tracking-wider">Wi-Fi SoftAP</h4>
            <span class="text-[10px] text-muted">Wireless LAN Hotspot</span>
          </div>
        </div>
        <Badge variant={status.wifi_enabled ? 'success' : 'default'}>
          {status.wifi_enabled ? 'Active' : 'Off'}
        </Badge>
      </div>

      <div class="text-[11px] font-mono text-muted space-y-0.5">
        <div>SSID: <span class="text-foreground font-bold">{status.ssid || 'AndroidAP'}</span></div>
        <div>Band: <span class="text-accent font-bold uppercase">{selectedBand === 'any' ? 'Dual/Auto' : `${selectedBand} GHz`}</span></div>
      </div>

      <Button
        variant={status.wifi_enabled ? 'danger' : 'primary'}
        size="sm"
        disabled={isTogglingWifi}
        onclick={() => toggleChannel('wifi')}
      >
        <Power class="w-3.5 h-3.5 mr-1" />
        <span>{status.wifi_enabled ? 'Stop Hotspot' : 'Start Hotspot'}</span>
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
            <h4 class="font-bold text-foreground text-xs uppercase tracking-wider">USB RNDIS / NCM</h4>
            <span class="text-[10px] text-muted">Direct USB Cable Link</span>
          </div>
        </div>
        <Badge variant={status.usb_enabled ? 'success' : 'default'}>
          {status.usb_enabled ? 'Active' : 'Off'}
        </Badge>
      </div>

      <div class="text-[11px] font-mono text-muted space-y-0.5">
        <div>Port: <span class="text-foreground font-bold">rndis0 / usb0</span></div>
        <div>Link: <span class="text-accent font-bold">Host OTG</span></div>
      </div>

      <Button
        variant={status.usb_enabled ? 'danger' : 'secondary'}
        size="sm"
        disabled={isTogglingUsb}
        onclick={() => toggleChannel('usb')}
      >
        <Power class="w-3.5 h-3.5 mr-1" />
        <span>{status.usb_enabled ? 'Stop USB Tether' : 'Start USB Tether'}</span>
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
            <h4 class="font-bold text-foreground text-xs uppercase tracking-wider">Ethernet OTG</h4>
            <span class="text-[10px] text-muted">USB to LAN Dongle</span>
          </div>
        </div>
        <Badge variant={status.ethernet_enabled ? 'success' : 'default'}>
          {status.ethernet_enabled ? 'Active' : 'Off'}
        </Badge>
      </div>

      <div class="text-[11px] font-mono text-muted space-y-0.5">
        <div>Interface: <span class="text-foreground font-bold">eth0 gateway</span></div>
        <div>Type: <span class="text-accent font-bold">Gigabit OTG</span></div>
      </div>

      <Button
        variant={status.ethernet_enabled ? 'danger' : 'secondary'}
        size="sm"
        disabled={isTogglingEthernet}
        onclick={() => toggleChannel('ethernet')}
      >
        <Power class="w-3.5 h-3.5 mr-1" />
        <span>{status.ethernet_enabled ? 'Stop Ethernet' : 'Start Ethernet'}</span>
      </Button>
    </div>
  </div>

  <!-- Gateway & SoftAP Credentials Section -->
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
    <!-- Wi-Fi SSID, Band & Security Form -->
    <Card
      title="Wi-Fi Hotspot Credentials & Radio"
      subtitle="Broadcast configuration, frequency band and security mode"
      tone="mint"
    >
      <div class="space-y-3 font-mono text-xs">
        <!-- Security Mode Switcher -->
        <div class="flex items-center gap-2 pb-2 border-b border-border">
          <span class="text-muted uppercase font-bold text-[11px]">Security Mode:</span>
          <button
            type="button"
            class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {securityType === 'wpa2' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
            onclick={() => (securityType = 'wpa2')}
          >
            WPA2-PSK (Secured)
          </button>
          <button
            type="button"
            class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {securityType === 'open' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
            onclick={() => { securityType = 'open'; passwordInput = ''; }}
          >
            Open (No Password)
          </button>
        </div>

        <!-- Frequency Band Switcher -->
        <div class="flex items-center gap-2 pb-2 border-b border-border">
          <span class="text-muted uppercase font-bold text-[11px]">Radio Band:</span>
          <button
            type="button"
            class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {selectedBand === 'any' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
            onclick={() => (selectedBand = 'any')}
          >
            Auto / Any
          </button>
          <button
            type="button"
            class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {selectedBand === '5' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
            onclick={() => (selectedBand = '5')}
          >
            5 GHz (High Speed)
          </button>
          <button
            type="button"
            class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {selectedBand === '2' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
            onclick={() => (selectedBand = '2')}
          >
            2.4 GHz (Long Range)
          </button>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <Input
            label="SSID Name (1-32 chars)"
            placeholder="Network SSID"
            bind:value={ssidInput}
          />
          {#if securityType === 'wpa2'}
            <Input
              type="password"
              label="Passphrase (8-63 chars)"
              placeholder="Leave blank to keep current"
              bind:value={passwordInput}
            />
          {:else}
            <div class="space-y-1">
              <span class="text-[10px] text-muted uppercase font-bold">Passphrase</span>
              <div class="p-2.5 bg-card border border-border rounded text-muted text-xs italic">
                No password required (Open Network)
              </div>
            </div>
          {/if}
        </div>

        <div class="flex justify-between items-center pt-2 border-t border-border">
          <span class="text-[11px] text-muted">
            Radio will reload when credentials change.
          </span>
          <Button
            variant="secondary"
            size="sm"
            disabled={isSavingWifi || !ssidInput}
            onclick={saveWifiConfig}
          >
            <Check class="w-3.5 h-3.5 mr-1" />
            <span>Apply & Reload SoftAP</span>
          </Button>
        </div>
      </div>
    </Card>

    <!-- Modular Gateway & Router Suite (L3 Engine) -->
    <Card
      title="Modular Gateway & Router Suite"
      subtitle="Static gateway address, upstream steering and OS-owned DHCP"
      tone="lavender"
    >
      <div class="space-y-3 font-mono text-xs">
        <!-- Gateway Mode Selector -->
        <div class="flex flex-wrap items-center justify-between gap-2 pb-2 border-b border-border">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-muted uppercase font-bold text-[11px]">Gateway Mode:</span>
            <button
              type="button"
              class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {routerCfg.gateway_mode === 'auto' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
              onclick={() => (routerCfg.gateway_mode = 'auto')}
            >
              Auto (Android Managed)
            </button>
            <button
              type="button"
              class="px-2.5 py-1 text-[11px] uppercase font-bold rounded border transition-all cursor-pointer {routerCfg.gateway_mode === 'static' ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
              onclick={() => (routerCfg.gateway_mode = 'static')}
            >
              Custom Static Gateway
            </button>
          </div>

          <Badge variant={applyStateVariant(routerStatus.apply_state)}>
            {applyStateLabel(routerStatus.apply_state, routerStatus.desired_mode)}
          </Badge>
        </div>

        <!-- Live / Observed state -->
        <div class="space-y-2 p-3 bg-card-sub border border-border rounded-lg">
          <div class="flex items-center justify-between">
            <span class="text-muted">Client Gateway (DHCP):</span>
            <span class="font-bold text-accent">{routerStatus.client_gateway || status.gateway_ip || '—'}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">Client Subnet (DHCP):</span>
            <span class="font-bold text-foreground">{routerStatus.client_subnet || status.subnet || '—'}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">Tether Interface:</span>
            <span class="font-bold text-foreground">{routerStatus.observed_iface || (status.active_interfaces || [])[0] || '—'}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">Upstream Routing Table:</span>
            <span class="font-bold text-foreground">{routerStatus.upstream_table || '—'}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">DHCP &amp; DNS Server:</span>
            <span class="font-bold text-emerald-400">Android NetworkStack (OS owned)</span>
          </div>
        </div>

        {#if routerCfg.gateway_mode === 'static' && routerStatus.apply_state === 'applied' && !routerStatus.gateway_effective}
          <div class="p-2.5 bg-card border border-warning/40 rounded-lg text-[10px] text-warning space-y-1">
            <p class="font-bold uppercase">Static gateway installed but not handed to clients</p>
            <p>
              BFR installed {routerStatus.observed_gateway || 'the static address'} and its policy rules in the kernel,
              but Android's DHCP server still advertises {routerStatus.client_gateway || 'its own address'} as the client gateway.
              Clients keep using the Android address until Android's DHCP server is replaced.
            </p>
          </div>
        {/if}

        {#if routerCfg.gateway_mode === 'static'}
          <!-- Custom Static Gateway Inputs -->
          <div class="space-y-2">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
              <Input
                label="Static Gateway IP"
                placeholder="192.168.1.1"
                bind:value={routerCfg.gateway_ip}
              />
              <Input
                label="Subnet Mask"
                placeholder="255.255.255.0"
                bind:value={routerCfg.subnet_mask}
              />
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
              <Input
                label="Upstream Interface (auto)"
                placeholder="auto"
                bind:value={routerCfg.upstream_interface}
              />
              <Input
                label="Lease Time (OS managed)"
                placeholder="12h"
                bind:value={routerCfg.lease_time}
              />
            </div>

            <div class="p-2.5 bg-card border border-border rounded-lg text-[10px] text-muted space-y-1">
              <p class="font-bold text-foreground uppercase">Dhcp Pool &amp; Dns (Android owned)</p>
              <p>
                <span class="text-foreground">DHCP Range:</span>
                {routerCfg.dhcp_start} — {routerCfg.dhcp_end} ·
                <span class="text-foreground">DNS:</span>
                {routerCfg.dns_primary}, {routerCfg.dns_secondary}
              </p>
              <p class="text-warning">
                Android's NetworkStack still hands out leases and the DNS addresses to clients.
                These values are stored for a future BFR-managed DHCP mode and are not applied yet.
              </p>
            </div>
          </div>
        {:else}
          <p class="text-[10px] text-muted pt-1 border-t border-border/60">
            Android OS owns the AP address, DHCP, DNS and forwarding. BFR only reports the live values
            and can layer scoped features (QoS, MAC filtering) on top.
          </p>
        {/if}

        {#if routerStatus.last_apply_error}
          <p class="text-[10px] text-red-400 font-bold">Last apply error: {routerStatus.last_apply_error}</p>
        {/if}

        <div class="flex justify-between items-center pt-2 border-t border-border">
          <div class="text-[10px] text-muted">
            {#if routerStatus.apply_state === 'applied'}
              <span class="text-emerald-400 font-bold">● static gateway verified in kernel</span>
            {:else if routerStatus.apply_state === 'auto'}
              <span>● Android manages gateway, DHCP and DNS — no BFR state applied</span>
            {:else if routerStatus.apply_state === 'inactive'}
              <span>hotspot disabled — state idle</span>
            {:else if routerStatus.apply_state === 'pending'}
              <span class="text-warning">● saved, waiting for tether interface</span>
            {:else if routerStatus.apply_state === 'failed'}
              <span class="text-red-400">● apply failed</span>
            {:else}
              <span>not applied</span>
            {/if}
          </div>
          <Button
            variant="primary"
            size="sm"
            disabled={isSavingRouter}
            onclick={saveRouterSettings}
          >
            <Check class="w-3.5 h-3.5 mr-1" />
            <span>Save Router Settings</span>
          </Button>
        </div>
      </div>
    </Card>
  </div>
</div>
