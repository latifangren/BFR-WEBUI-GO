<script lang="ts">
  import {
    Users,
    RefreshCw,
    Smartphone,
    Laptop,
    Wifi,
    Usb,
    Network,
    Gauge,
    ShieldAlert,
  } from '@lucide/svelte'
  import type { ConnectedClient } from '../../../../types/hotspot'
  import Card from '../../../ui/Card.svelte'
  import Button from '../../../ui/Button.svelte'
  import Badge from '../../../ui/Badge.svelte'

  interface Props {
    clients: ConnectedClient[]
    isLoading: boolean
    onRefresh: () => Promise<void>
    onLimitClient: (client: ConnectedClient) => void
    onBlockClient: (mac: string) => void
  }

  let { clients = [], isLoading, onRefresh, onLimitClient, onBlockClient }: Props = $props()

  function getIfaceIcon(iface?: string) {
    if (!iface) return Wifi
    const lower = iface.toLowerCase()
    if (lower.startsWith('rndis') || lower.startsWith('usb')) return Usb
    if (lower.startsWith('eth')) return Network
    return Wifi
  }
</script>

<Card
  title="Connected Client Devices"
  subtitle="Live ARP and DHCP neighbour bindings discovered on tethering interfaces"
  tone="mint"
>
  {#snippet action()}
    <Button
      variant="ghost"
      size="sm"
      disabled={isLoading}
      onclick={onRefresh}
    >
      <RefreshCw class="w-3.5 h-3.5 mr-1 {isLoading ? 'animate-spin' : ''}" />
      <span>Scan Clients</span>
    </Button>
  {/snippet}

  {#if clients.length === 0}
    <div class="p-8 text-center font-mono text-xs text-muted">
      <Users class="w-8 h-8 mx-auto mb-2 opacity-40 text-muted" />
      <p class="font-bold text-foreground">No Active Tethering Clients</p>
      <p class="text-[11px] text-muted mt-1">
        Enable Wi-Fi Hotspot, USB Tethering, or Ethernet OTG and connect client devices.
      </p>
    </div>
  {:else}
    <!-- Mobile Cards (sm:hidden) -->
    <div class="sm:hidden space-y-2">
      {#each clients as client}
        {@const IconComp = getIfaceIcon(client.interface)}
        <div class="p-3 bg-card-sub border-2 border-border rounded-lg space-y-2 font-mono text-xs">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 min-w-0">
              <div class="p-1.5 rounded bg-card border border-border text-accent">
                <IconComp class="w-4 h-4" />
              </div>
              <div class="min-w-0">
                <div class="font-bold text-foreground truncate">{client.device || 'Client Device'}</div>
                <div class="text-[10px] text-muted">{client.interface || 'wlan1'}</div>
              </div>
            </div>
            <Badge variant="success" class="text-[9px]">ONLINE</Badge>
          </div>

          <div class="text-[11px] text-muted flex items-center justify-between pt-1 border-t border-border/60">
            <span class="text-accent font-bold">{client.ip}</span>
            <span class="truncate ml-2">{client.mac}</span>
          </div>

          <div class="flex items-center gap-2 pt-1 border-t border-border">
            <Button
              variant="outline"
              size="sm"
              class="w-full text-[10px]"
              onclick={() => onLimitClient(client)}
            >
              <Gauge class="w-3 h-3 mr-1 text-lavender" />
              <span>Limit Speed</span>
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="w-full text-[10px] text-red-400 hover:text-red-300"
              onclick={() => onBlockClient(client.mac)}
            >
              <ShieldAlert class="w-3 h-3 mr-1" />
              <span>Block</span>
            </Button>
          </div>
        </div>
      {/each}
    </div>

    <!-- Desktop Table View (hidden sm:block) -->
    <div class="hidden sm:block overflow-x-auto">
      <table class="w-full text-left font-mono text-xs">
        <thead>
          <tr class="border-b border-border text-muted uppercase text-[10px]">
            <th class="py-2.5 px-3">Device / Hostname</th>
            <th class="py-2.5 px-3">IP Address</th>
            <th class="py-2.5 px-3">MAC Address</th>
            <th class="py-2.5 px-3">Interface</th>
            <th class="py-2.5 px-3 text-right">Quick Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          {#each clients as client}
            {@const IconComp = getIfaceIcon(client.interface)}
            <tr class="hover:bg-card-sub transition-colors">
              <td class="py-2.5 px-3">
                <div class="flex items-center gap-2">
                  <IconComp class="w-3.5 h-3.5 text-accent" />
                  <span class="font-bold text-foreground">{client.device || 'Client Device'}</span>
                </div>
              </td>
              <td class="py-2.5 px-3 text-accent font-bold">{client.ip}</td>
              <td class="py-2.5 px-3 text-muted">{client.mac}</td>
              <td class="py-2.5 px-3">
                <Badge variant="default" class="text-[9px] uppercase">{client.interface || 'wlan1'}</Badge>
              </td>
              <td class="py-2.5 px-3 text-right">
                <div class="flex items-center justify-end gap-1.5">
                  <Button
                    variant="outline"
                    size="sm"
                    class="text-[10px] py-1 px-2"
                    onclick={() => onLimitClient(client)}
                  >
                    <Gauge class="w-3 h-3 mr-1 text-lavender" />
                    <span>QoS Limit</span>
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    class="text-[10px] py-1 px-2 text-red-400 hover:text-red-300"
                    onclick={() => onBlockClient(client.mac)}
                  >
                    <ShieldAlert class="w-3 h-3 mr-1" />
                    <span>Block MAC</span>
                  </Button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</Card>
