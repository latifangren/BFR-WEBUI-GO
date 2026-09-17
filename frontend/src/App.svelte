<script lang="ts">
  import { onMount } from 'svelte'
  import Header from './components/layout/Header.svelte'
  import Sidebar from './components/layout/Sidebar.svelte'
  import BottomPopoverNav from './components/layout/BottomPopoverNav.svelte'
  import LoginCard from './components/layout/LoginCard.svelte'
  import TabOverview from './components/tabs/overview/TabOverview.svelte'
  import TabSysinfo from './components/tabs/sysinfo/TabSysinfo.svelte'
  import TabNetwork from './components/tabs/network/TabNetwork.svelte'
  import TabHotspot from './components/tabs/hotspot/TabHotspot.svelte'
  import TabPower from './components/tabs/power/TabPower.svelte'
  import TabCharger from './components/tabs/charger/TabCharger.svelte'
  import TabModem from './components/tabs/modem/TabModem.svelte'
  import TabSamsung from './components/tabs/samsung/TabSamsung.svelte'
  import TabTerminal from './components/tabs/terminal/TabTerminal.svelte'
  import TabScrcpy from './components/tabs/scrcpy/TabScrcpy.svelte'
  import TabFileManager from './components/tabs/files/TabFileManager.svelte'
  import TabLogs from './components/tabs/logs/TabLogs.svelte'
  import TabProxy from './components/tabs/proxy/TabProxy.svelte'
  import TabQoS from './components/tabs/qos/TabQoS.svelte'
  import TabVnstat from './components/tabs/vnstat/TabVnstat.svelte'
  import TabNAS from './components/tabs/nas/TabNAS.svelte'
  import TabTunnel from './components/tabs/tunnel/TabTunnel.svelte'
  import TabSpeedtest from './components/tabs/speedtest/TabSpeedtest.svelte'
  import TabTools from './components/tabs/tools/TabTools.svelte'
  import TabTelegram from './components/tabs/telegram/TabTelegram.svelte'
  import TabAbout from './components/tabs/about/TabAbout.svelte'
  import TabSSH from './components/tabs/ssh/TabSSH.svelte'
  import TabModules from './components/tabs/modules/TabModules.svelte'
  import TabSMS from './components/tabs/sms/TabSMS.svelte'
  import ToastContainer from './components/ui/ToastContainer.svelte'
  import { authStore } from './stores/auth.svelte'
  import { navigationStore, AVAILABLE_TABS } from './stores/navigation.svelte'
  import { sysinfoStore } from './stores/sysinfo.svelte'

  onMount(() => {
    authStore.checkStatus()
    sysinfoStore.init()
  })

  $effect(() => {
    const isFocused = navigationStore.activeTab === 'overview' || navigationStore.activeTab === 'sysinfo'
    sysinfoStore.setTelemetryFocus(isFocused)
  })

  const currentTab = $derived(
    AVAILABLE_TABS.find((t) => t.id === navigationStore.activeTab) || AVAILABLE_TABS[0]
  )
</script>

{#snippet tabContent()}
  {#if navigationStore.activeTab === 'overview'}
    <TabOverview />
  {:else if navigationStore.activeTab === 'sysinfo'}
    <TabSysinfo />
  {:else if navigationStore.activeTab === 'network'}
    <TabNetwork />
  {:else if navigationStore.activeTab === 'hotspot'}
    <TabHotspot />
  {:else if navigationStore.activeTab === 'power'}
    <TabPower />
  {:else if navigationStore.activeTab === 'charger'}
    <TabCharger />
  {:else if navigationStore.activeTab === 'modem'}
    <TabModem />
  {:else if navigationStore.activeTab === 'samsung'}
    <TabSamsung />
  {:else if navigationStore.activeTab === 'terminal'}
    <TabTerminal />
  {:else if navigationStore.activeTab === 'scrcpy'}
    <TabScrcpy />
  {:else if navigationStore.activeTab === 'files'}
    <TabFileManager />
  {:else if navigationStore.activeTab === 'logs'}
    <TabLogs />
  {:else if navigationStore.activeTab === 'proxy'}
    <TabProxy />
  {:else if navigationStore.activeTab === 'qos'}
    <TabQoS />
  {:else if navigationStore.activeTab === 'vnstat'}
    <TabVnstat />
  {:else if navigationStore.activeTab === 'nas'}
    <TabNAS />
  {:else if navigationStore.activeTab === 'tunnel'}
    <TabTunnel />
  {:else if navigationStore.activeTab === 'speedtest'}
    <TabSpeedtest />
  {:else if navigationStore.activeTab === 'tools'}
    <TabTools />
  {:else if navigationStore.activeTab === 'telegram'}
    <TabTelegram />
  {:else if navigationStore.activeTab === 'ssh'}
    <TabSSH />
  {:else if navigationStore.activeTab === 'modules'}
    <TabModules />
  {:else if navigationStore.activeTab === 'sms'}
    <TabSMS />
  {:else if navigationStore.activeTab === 'about'}
    <TabAbout />
  {:else}
    <!-- Fallback tab -->
    <TabOverview />
  {/if}
{/snippet}

{#if !authStore.authenticated}
  <!-- Full Screen Authentic Login Page (Matching Legacy WebUI) -->
  <LoginCard />
{:else if navigationStore.layout === 'sidebar'}
  <!-- Sidebar Rail Mode -->
  <div class="flex h-screen overflow-hidden bg-background text-foreground font-sans">
    <Sidebar />
    <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
      <Header />
      <main class="flex-1 overflow-y-auto p-4 md:p-6 custom-scrollbar">
        {@render tabContent()}
      </main>
    </div>
  </div>
{:else}
  <!-- Full-Width TopBar Mode -->
  <div class="min-h-screen flex flex-col bg-background text-foreground font-sans">
    <Header />
    <main class="flex-1 w-full max-w-7xl mx-auto p-4 md:p-6 pb-20 md:pb-6">
      {@render tabContent()}
    </main>
    <BottomPopoverNav />
  </div>
{/if}

<!-- Global Toast Notifications -->
<ToastContainer />
