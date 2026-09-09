<script lang="ts">
  import { onMount } from 'svelte'
  import { Lock, Construction } from '@lucide/svelte'
  import Header from './components/layout/Header.svelte'
  import Sidebar from './components/layout/Sidebar.svelte'
  import BottomNav from './components/layout/BottomNav.svelte'
  import TabOverview from './components/tabs/overview/TabOverview.svelte'
  import TabSysinfo from './components/tabs/sysinfo/TabSysinfo.svelte'
  import TabNetwork from './components/tabs/network/TabNetwork.svelte'
  import TabHotspot from './components/tabs/hotspot/TabHotspot.svelte'
  import TabPower from './components/tabs/power/TabPower.svelte'
  import TabCharger from './components/tabs/charger/TabCharger.svelte'
  import TabModem from './components/tabs/modem/TabModem.svelte'
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
  import Modal from './components/ui/Modal.svelte'
  import Input from './components/ui/Input.svelte'
  import Button from './components/ui/Button.svelte'
  import Card from './components/ui/Card.svelte'
  import { authStore } from './stores/auth.svelte'
  import { navigationStore, AVAILABLE_TABS } from './stores/navigation.svelte'
  import { sysinfoStore } from './stores/sysinfo.svelte'

  let password = $state('')

  onMount(() => {
    authStore.checkStatus()
    sysinfoStore.init()
  })

  $effect(() => {
    const isFocused = navigationStore.activeTab === 'overview' || navigationStore.activeTab === 'sysinfo'
    sysinfoStore.setTelemetryFocus(isFocused)
  })

  async function handleLogin(e: SubmitEvent) {
    e.preventDefault()
    if (!password) return
    const success = await authStore.login({ password })
    if (success) {
      password = ''
    }
  }

  const currentTab = $derived(
    AVAILABLE_TABS.find((t) => t.id === navigationStore.activeTab) || AVAILABLE_TABS[0]
  )
</script>

<div class="min-h-screen flex flex-col bg-background text-foreground antialiased selection:bg-accent selection:text-accent-text">
  <!-- Top Navigation Header -->
  <Header />

  <!-- App Body Layout -->
  <div class="flex-1 flex max-w-7xl w-full mx-auto">
    <!-- Navigation Sidebar -->
    <Sidebar />

    <!-- Main Content Area -->
    <main class="flex-1 min-w-0 px-4 pt-4 pb-20 md:py-6 md:px-6 space-y-6 overflow-x-hidden">
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
    </main>
  </div>

  <!-- Global Toast Notifications -->
  <ToastContainer />

  <!-- Mobile Bottom Navigation Bar (md:hidden) -->
  <BottomNav />

  <!-- Auth Modal (Blocking if unauthenticated) -->
  {#if !authStore.authenticated}
    <Modal open={true} title="Authentication Required">
      <form onsubmit={handleLogin} class="space-y-4">
        <div class="p-3 bg-card-sub border border-border rounded flex items-center gap-3 text-xs font-mono text-muted">
          <Lock class="w-4 h-4 text-accent shrink-0" />
          <span>Please enter your administrator password to manage system controls.</span>
        </div>

        <Input
          type="password"
          label="Administrator Password"
          placeholder="Enter password..."
          bind:value={password}
          disabled={authStore.isLoading}
          error={authStore.loginError || undefined}
        />

        <div class="flex justify-end pt-2">
          <Button
            type="submit"
            variant="primary"
            disabled={authStore.isLoading || !password}
            fullWidth={true}
          >
            {authStore.isLoading ? 'Authenticating...' : 'Sign In'}
          </Button>
        </div>
      </form>
    </Modal>
  {/if}
</div>

