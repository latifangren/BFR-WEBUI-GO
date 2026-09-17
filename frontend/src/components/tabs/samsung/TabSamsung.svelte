<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Smartphone,
    Radio,
    Lock,
    RotateCcw,
    RefreshCw,
    AlertTriangle,
    CheckCircle,
    Sliders,
    ShieldCheck,
    Zap,
    Check,
    Search,
    Layers,
    Info,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'

  interface SamsungDeviceStatus {
    is_samsung: boolean
    manufacturer: string
    model: string
    device: string
    oneui_version: string
    dex_exists: boolean
    cli_exists: boolean
    cli_path: string
    supported_slots: number
  }

  interface BandItem {
    id: number
    name: string
    active: boolean
  }

  interface SamsungBandStatus {
    slot: number
    mode: string // "AUTOMATIC" | "LOCKED"
    active_bands: number[]
    bands: BandItem[]
    error?: string
  }

  // State
  let deviceStatus = $state<SamsungDeviceStatus | null>(null)
  let isLoadingStatus = $state(false)

  let selectedSlot = $state<number>(0)
  let bandData = $state<SamsungBandStatus | null>(null)
  let isLoadingBands = $state(false)
  let bandsError = $state<string | null>(null)

  let selectedBandIds = $state<number[]>([])
  let searchQuery = $state('')
  let isLocking = $state(false)
  let isResetting = $state(false)
  let showResetModal = $state(false)

  // Popular LTE bands in Indonesia (Telkomsel, Indosat, XL, Smartfren)
  const popularBandIds = [1, 3, 8, 40]

  // Derived computations
  const isSamsung = $derived(deviceStatus?.is_samsung ?? false)
  const isLockedMode = $derived(bandData?.mode === 'LOCKED')
  const allBands = $derived(bandData?.bands ?? [])
  const activeBandsList = $derived(
    allBands.filter((b) => b.active || (bandData?.active_bands ?? []).includes(b.id))
  )

  const filteredBands = $derived(
    allBands.filter((b) => {
      if (!searchQuery.trim()) return true
      const q = searchQuery.toLowerCase().trim()
      return (
        b.name.toLowerCase().includes(q) ||
        b.id.toString().includes(q) ||
        `b${b.id}`.toLowerCase().includes(q)
      )
    })
  )

  const isAllSelected = $derived(
    allBands.length > 0 && selectedBandIds.length === allBands.length
  )

  const availablePopularIds = $derived(
    allBands.filter((b) => popularBandIds.includes(b.id)).map((b) => b.id)
  )

  const isPopularSelected = $derived(
    availablePopularIds.length > 0 &&
      availablePopularIds.every((id) => selectedBandIds.includes(id)) &&
      selectedBandIds.length === availablePopularIds.length
  )

  onMount(async () => {
    await loadInitialData()
  })

  async function loadInitialData() {
    await fetchDeviceStatus()
    await fetchBands(selectedSlot)
  }

  async function fetchDeviceStatus() {
    try {
      isLoadingStatus = true
      const res = await api.get<SamsungDeviceStatus>('/api/samsung/status')
      deviceStatus = res
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Gagal memuat status perangkat Samsung'
      toastStore.error(msg, 'Status Error')
    } finally {
      isLoadingStatus = false
    }
  }

  async function fetchBands(slot: number) {
    try {
      isLoadingBands = true
      bandsError = null
      const res = await api.get<SamsungBandStatus>(`/api/samsung/bands?slot=${slot}`)
      bandData = res

      // Initialize selected band IDs based on current active bands or mode
      if (res.mode === 'LOCKED' && res.active_bands && res.active_bands.length > 0) {
        selectedBandIds = [...res.active_bands]
      } else {
        const activeIds = (res.bands || []).filter((b) => b.active).map((b) => b.id)
        if (activeIds.length > 0) {
          selectedBandIds = activeIds
        } else if (res.active_bands && res.active_bands.length > 0) {
          selectedBandIds = [...res.active_bands]
        } else {
          // If in automatic mode and no active list returned, default to popular available bands
          const pop = (res.bands || [])
            .filter((b) => popularBandIds.includes(b.id))
            .map((b) => b.id)
          selectedBandIds = pop.length > 0 ? pop : (res.bands || []).map((b) => b.id)
        }
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : `Gagal mengambil daftar band SIM ${slot + 1}`
      bandsError = msg
      toastStore.error(msg, 'Query Band Error')
    } finally {
      isLoadingBands = false
    }
  }

  async function handleSlotChange(slot: number) {
    if (slot === selectedSlot || isLoadingBands) return
    selectedSlot = slot
    await fetchBands(slot)
  }

  function toggleBand(id: number) {
    if (selectedBandIds.includes(id)) {
      selectedBandIds = selectedBandIds.filter((b) => b !== id)
    } else {
      selectedBandIds = [...selectedBandIds, id]
    }
  }

  function selectPopular() {
    if (availablePopularIds.length === 0) {
      toastStore.warning('Band populer (B1, B3, B8, B40) tidak tersedia di modem ini.')
      return
    }
    selectedBandIds = [...availablePopularIds]
    toastStore.info(`Memilih ${selectedBandIds.length} frekuensi populer (B1, B3, B8, B40)`)
  }

  function selectAll() {
    selectedBandIds = allBands.map((b) => b.id)
  }

  function clearAll() {
    selectedBandIds = []
  }

  async function applyLock() {
    if (selectedBandIds.length === 0) {
      toastStore.warning('Pilih minimal satu band sebelum mengunci frekuensi.')
      return
    }

    try {
      isLocking = true
      await api.post<{ status: string; message: string }>('/api/samsung/lock', {
        slot: selectedSlot,
        bands: selectedBandIds,
      })
      toastStore.success(
        `Band lock berhasil diterapkan ke SIM ${selectedSlot + 1}. Radio modem sedang diperbarui.`,
        'Lock Applied'
      )
      await fetchBands(selectedSlot)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Gagal mengirim perintah band lock'
      toastStore.error(msg, 'Lock Failed')
    } finally {
      isLocking = false
    }
  }

  async function confirmResetAuto() {
    try {
      isResetting = true
      await api.post<{ status: string; message: string }>('/api/samsung/auto', {
        slot: selectedSlot,
      })
      toastStore.success(
        `SIM ${selectedSlot + 1} berhasil dikembalikan ke mode otomatis (All Bands).`,
        'Reset Selesai'
      )
      showResetModal = false
      await fetchBands(selectedSlot)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Gagal mereset modem ke mode otomatis'
      toastStore.error(msg, 'Reset Failed')
    } finally {
      isResetting = false
    }
  }

  function formatBandDisplay(band: BandItem): { label: string; desc: string } {
    const raw = band.name || ''
    const match = raw.match(/^(B\d+|n\d+|LTE\s*B\d+|NR\s*n\d+)\s*(.*)$/i)
    if (match) {
      let label = match[1].replace(/LTE\s*/i, '').replace(/NR\s*/i, '')
      let desc = match[2].replace(/[()]/g, '').trim()
      return { label: label.toUpperCase(), desc: desc || `Band ${band.id}` }
    }
    return {
      label: `B${band.id}`,
      desc: raw.replace(/[()]/g, '').trim() || 'LTE Band',
    }
  }
</script>

<div class="space-y-6 pb-12 font-mono">
  <!-- TOP HERO & REFRESH -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b-2 border-border pb-4">
    <div>
      <div class="flex items-center gap-2">
        <Smartphone class="w-6 h-6 text-accent" />
        <h1 class="text-xl sm:text-2xl font-black uppercase tracking-wider text-foreground">
          Samsung RIL Manager
        </h1>
        <Badge variant="info" class="text-[10px]">SecRIL</Badge>
      </div>
      <p class="text-xs text-muted mt-1">
        Kontrol modulasi frekuensi modem langsung melalui Samsung RIL framework & binary band_manager.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoadingStatus || isLoadingBands}
        onclick={loadInitialData}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoadingStatus || isLoadingBands ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <!-- NON-SAMSUNG WARNING BANNER -->
  {#if deviceStatus && !isSamsung}
    <div
      class="p-4 rounded-lg border-2 border-amber-500 bg-amber-950/25 shadow-neobrutal-sm text-foreground space-y-2 select-none"
      role="alert"
    >
      <div class="flex items-center gap-2 text-amber-400 font-bold uppercase text-xs tracking-wider">
        <AlertTriangle class="w-5 h-5 shrink-0 text-amber-400" />
        <span>Peringatan Kompatibilitas OEM (Bukan Samsung)</span>
      </div>
      <p class="text-xs text-muted leading-relaxed">
        Perangkat Anda terdeteksi bermerek <span class="text-foreground font-bold">{deviceStatus.manufacturer || 'Non-Samsung'}</span> model <span class="text-foreground font-bold">{deviceStatus.model || 'Unknown'}</span>. Fitur ini dirancang khusus untuk modul baseband Samsung Exynos/Qualcomm yang mendukung hook SecRIL. Perintah penguncian band kemungkinan besar tidak akan berpengaruh pada modem perangkat ini.
      </p>
    </div>
  {/if}

  <!-- DEVICE HARDWARE & TOOL ENVIRONMENT -->
  <Card title="Hardware & Environment RIL" subtitle="Informasi OEM, versi One UI, dan ketersediaan tool baseband" tone="ice">
    {#snippet action()}
      {#if isSamsung}
        <Badge variant="success" class="flex items-center gap-1">
          <ShieldCheck class="w-3 h-3" />
          <span>SAMSUNG VERIFIED</span>
        </Badge>
      {:else}
        <Badge variant="warning" class="flex items-center gap-1">
          <AlertTriangle class="w-3 h-3" />
          <span>NON-SAMSUNG</span>
        </Badge>
      {/if}
    {/snippet}

    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
      <!-- Device & Model -->
      <div class="p-3 bg-card-sub border-2 border-border rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Perangkat / Model</span>
        <p class="font-bold text-foreground truncate">
          {deviceStatus?.manufacturer || 'Unknown'} {deviceStatus?.model || '-'}
        </p>
        <span class="text-[10px] text-muted block truncate font-mono">
          Codename: {deviceStatus?.device || '-'}
        </span>
      </div>

      <!-- One UI Version -->
      <div class="p-3 bg-card-sub border-2 border-border rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">One UI Version</span>
        <p class="font-bold text-foreground truncate">
          {deviceStatus?.oneui_version ? `One UI ${deviceStatus.oneui_version}` : 'Standard Android'}
        </p>
        <span class="text-[10px] text-muted block truncate">
          Framework Samsung RIL
        </span>
      </div>

      <!-- DEX Hook Status -->
      <div class="p-3 bg-card-sub border-2 border-border rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">SecRIL DEX Hook</span>
        <div class="pt-0.5">
          {#if deviceStatus?.dex_exists}
            <Badge variant="success" class="flex items-center gap-1 w-fit">
              <CheckCircle class="w-3 h-3" />
              <span>DEX READY</span>
            </Badge>
          {:else}
            <Badge variant="danger" class="flex items-center gap-1 w-fit">
              <AlertTriangle class="w-3 h-3" />
              <span>DEX MISSING</span>
            </Badge>
          {/if}
        </div>
        <span class="text-[10px] text-muted block truncate">
          RIL Class Injector
        </span>
      </div>

      <!-- CLI Tool Status -->
      <div class="p-3 bg-card-sub border-2 border-border rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Binary Tool CLI</span>
        <div class="pt-0.5">
          {#if deviceStatus?.cli_exists}
            <Badge variant="success" class="flex items-center gap-1 w-fit">
              <CheckCircle class="w-3 h-3" />
              <span>CLI OK</span>
            </Badge>
          {:else}
            <Badge variant="warning" class="flex items-center gap-1 w-fit">
              <AlertTriangle class="w-3 h-3" />
              <span>FALLBACK CLI</span>
            </Badge>
          {/if}
        </div>
        <span class="text-[10px] text-muted block truncate font-mono" title={deviceStatus?.cli_path || 'band_manager'}>
          {deviceStatus?.cli_path ? deviceStatus.cli_path.split('/').pop() : 'band_manager'}
        </span>
      </div>
    </div>
  </Card>

  <!-- SIM SELECTOR & CURRENT MODE STATUS -->
  <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
    <!-- SIM SELECTOR (1 col) -->
    <Card title="SIM Card Slot" subtitle="Pilih kartu SIM yang akan dikonfigurasi" class="md:col-span-1">
      <div class="flex flex-col gap-2.5">
        <button
          type="button"
          class="neo-button p-3 rounded border-2 text-left cursor-pointer flex items-center justify-between transition-all {selectedSlot === 0 ? 'border-accent bg-accent/15 text-foreground shadow-neobrutal-sm font-bold' : 'border-border bg-card-sub text-muted hover:border-foreground/30 hover:bg-card'}"
          disabled={isLoadingBands}
          onclick={() => handleSlotChange(0)}
        >
          <div class="flex items-center gap-2.5">
            <Radio class="w-4 h-4 {selectedSlot === 0 ? 'text-accent' : 'text-muted'}" />
            <div>
              <div class="text-sm font-bold">SIM 1</div>
              <div class="text-[10px] text-muted">Slot Indeks 0</div>
            </div>
          </div>
          {#if selectedSlot === 0}
            <Badge variant="info">ACTIVE</Badge>
          {/if}
        </button>

        <button
          type="button"
          class="neo-button p-3 rounded border-2 text-left cursor-pointer flex items-center justify-between transition-all {selectedSlot === 1 ? 'border-accent bg-accent/15 text-foreground shadow-neobrutal-sm font-bold' : 'border-border bg-card-sub text-muted hover:border-foreground/30 hover:bg-card'}"
          disabled={isLoadingBands}
          onclick={() => handleSlotChange(1)}
        >
          <div class="flex items-center gap-2.5">
            <Radio class="w-4 h-4 {selectedSlot === 1 ? 'text-accent' : 'text-muted'}" />
            <div>
              <div class="text-sm font-bold">SIM 2</div>
              <div class="text-[10px] text-muted">Slot Indeks 1</div>
            </div>
          </div>
          {#if selectedSlot === 1}
            <Badge variant="info">ACTIVE</Badge>
          {/if}
        </button>
      </div>
    </Card>

    <!-- STATUS MODE CARD (2 cols) -->
    <Card title="Status Baseband & Mode SIM" subtitle="Status transmisi frekuensi pada modem saat ini" tone="mint" class="md:col-span-2">
      {#snippet action()}
        {#if bandData}
          {#if isLockedMode}
            <Badge variant="warning" class="px-2 py-1 text-xs">
              <Lock class="w-3 h-3 mr-1 inline" />
              <span>LOCKED</span>
            </Badge>
          {:else}
            <Badge variant="success" class="px-2 py-1 text-xs">
              <Zap class="w-3 h-3 mr-1 inline" />
              <span>AUTOMATIC</span>
            </Badge>
          {/if}
        {/if}
      {/snippet}

      <div class="space-y-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 p-3 bg-card-sub border-2 border-border rounded">
          <div>
            <span class="text-[10px] text-muted uppercase font-bold block">Mode Operasi Saat Ini</span>
            <p class="text-sm font-bold text-foreground flex items-center gap-1.5 mt-0.5">
              {#if isLockedMode}
                <Lock class="w-4 h-4 text-amber-400 inline" />
                <span class="text-amber-400">Modem Dikunci ke Frekuensi Terpilih</span>
              {:else}
                <CheckCircle class="w-4 h-4 text-emerald-400 inline" />
                <span class="text-emerald-400">Pemilihan Otomatis (All Bands Dinamis)</span>
              {/if}
            </p>
          </div>
          <div class="text-left sm:text-right">
            <span class="text-[10px] text-muted uppercase font-bold block">Frekuensi Terdaftar</span>
            <span class="text-sm font-bold text-foreground">
              {activeBandsList.length} Band Aktif / {allBands.length} Total
            </span>
          </div>
        </div>

        <!-- ACTIVE BANDS CHIP LIST -->
        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-1.5">
            Daftar Frekuensi Sedang Aktif di Modem:
          </span>
          {#if isLoadingBands}
            <div class="flex items-center gap-2 text-xs text-muted">
              <RefreshCw class="w-3.5 h-3.5 animate-spin" />
              <span>Membaca telemetry baseband...</span>
            </div>
          {:else if activeBandsList.length > 0}
            <div class="flex flex-wrap gap-1.5">
              {#each activeBandsList as activeBand}
                {@const parsed = formatBandDisplay(activeBand)}
                <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded border border-emerald-500/40 bg-emerald-950/40 text-emerald-300 text-xs font-bold font-mono">
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                  <span>{parsed.label}</span>
                  <span class="text-[10px] text-emerald-400/80 font-normal">({parsed.desc})</span>
                </span>
              {/each}
            </div>
          {:else}
            <p class="text-xs text-muted italic">
              Tidak ada band spesifik yang dilaporkan atau modem dalam mode auto penuh.
            </p>
          {/if}
        </div>
      </div>
    </Card>
  </div>

  <!-- BAND SELECTION GRID & CONTROLS -->
  <Card title="Matriks Frekuensi Baseband" subtitle="Pilih satu atau beberapa band frekuensi lalu kunci untuk mengoptimalkan throughput dan kestabilan BTS" tone="none">
    {#snippet action()}
      <div class="flex items-center gap-2">
        <Badge variant="default" class="text-xs font-mono">
          {selectedBandIds.length} Band Dipilih
        </Badge>
      </div>
    {/snippet}

    <div class="space-y-4">
      <!-- FILTER & QUICK PRESET BAR -->
      <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-3 p-3 bg-card-sub border-2 border-border rounded">
        <!-- Search bar -->
        <div class="relative flex-1 min-w-[200px] max-w-md">
          <Search class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
          <input
            type="text"
            bind:value={searchQuery}
            placeholder="Cari band (contoh: 1, 3, B40, 1800)..."
            class="neo-input w-full bg-card border-2 border-border rounded pl-9 pr-3 py-1.5 text-xs font-mono text-foreground placeholder:text-muted focus:outline-none focus:border-accent"
          />
        </div>

        <!-- Quick selection buttons -->
        <div class="flex flex-wrap items-center gap-2">
          <!-- Popular Indonesia -->
          <Button
            variant={isPopularSelected ? 'primary' : 'outline'}
            size="sm"
            onclick={selectPopular}
            title="Pilih band umum operator Indonesia (B1, B3, B8, B40)"
          >
            <Zap class="w-3.5 h-3.5 mr-1 {isPopularSelected ? 'text-amber-300' : 'text-amber-500'}" />
            <span>Pilih Populer ID (B1, B3, B8, B40)</span>
          </Button>

          <!-- Select All -->
          <Button
            variant={isAllSelected ? 'secondary' : 'outline'}
            size="sm"
            onclick={selectAll}
          >
            <Check class="w-3.5 h-3.5 mr-1" />
            <span>Select All</span>
          </Button>

          <!-- Clear -->
          <Button
            variant="ghost"
            size="sm"
            onclick={clearAll}
            disabled={selectedBandIds.length === 0}
          >
            <span>Clear</span>
          </Button>
        </div>
      </div>

      <!-- ERROR STATE -->
      {#if bandsError}
        <div class="p-4 border-2 border-red-500 bg-red-950/20 rounded text-red-400 space-y-2">
          <div class="flex items-center gap-2 font-bold text-xs uppercase">
            <AlertTriangle class="w-4 h-4" />
            <span>Gagal Mendapatkan Daftar Band</span>
          </div>
          <p class="text-xs text-muted">{bandsError}</p>
          <Button variant="danger" size="sm" onclick={() => fetchBands(selectedSlot)}>
            <RefreshCw class="w-3 h-3 mr-1" />
            <span>Coba Lagi</span>
          </Button>
        </div>
      {/if}

      <!-- LOADING STATE -->
      {#if isLoadingBands}
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
          {#each Array(12) as _, idx}
            <div class="p-3 rounded-lg border-2 border-border bg-card-sub animate-pulse h-24 flex flex-col justify-between">
              <div class="flex items-center justify-between">
                <div class="w-10 h-5 bg-border rounded"></div>
                <div class="w-4 h-4 bg-border rounded"></div>
              </div>
              <div class="w-20 h-3 bg-border rounded"></div>
              <div class="w-12 h-3 bg-border rounded"></div>
            </div>
          {/each}
        </div>
      {:else if filteredBands.length === 0}
        <div class="p-8 text-center border-2 border-dashed border-border rounded bg-card-sub space-y-2">
          <Layers class="w-8 h-8 mx-auto text-muted" />
          <p class="text-sm font-bold text-foreground">Tidak Ada Band Ditemukan</p>
          <p class="text-xs text-muted max-w-sm mx-auto">
            {#if searchQuery.trim()}
              Tidak ada band yang cocok dengan filter pencarian "{searchQuery}".
            {:else}
              Modem tidak mengembalikan daftar band yang didukung. Pastikan perangkat di-root dengan izin Superuser (su) dan binary band_manager tersedia.
            {/if}
          </p>
        </div>
      {:else}
        <!-- BAND CARDS GRID -->
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
          {#each filteredBands as band (band.id)}
            {@const isChecked = selectedBandIds.includes(band.id)}
            {@const isLiveActive = band.active || (bandData?.active_bands ?? []).includes(band.id)}
            {@const parsed = formatBandDisplay(band)}
            <button
              type="button"
              class="relative p-3 rounded-lg border-2 text-left cursor-pointer select-none transition-all flex flex-col justify-between min-h-[96px] group {isChecked
                ? 'border-accent bg-accent/15 text-foreground shadow-neobrutal-sm -translate-x-0.5 -translate-y-0.5'
                : 'border-border bg-card-sub text-muted hover:border-foreground/30 hover:bg-card'}"
              onclick={() => toggleBand(band.id)}
            >
              <!-- CARD TOP -->
              <div class="flex items-center justify-between gap-1 w-full">
                <span class="font-mono font-black text-base tracking-tight {isChecked ? 'text-accent' : 'text-foreground'}">
                  {parsed.label}
                </span>

                <!-- CHECKBOX INDICATOR -->
                <span
                  class="w-4 h-4 rounded border-2 flex items-center justify-center shrink-0 transition-colors {isChecked
                    ? 'border-accent bg-accent text-accent-text'
                    : 'border-border bg-card text-transparent group-hover:border-foreground/40'}"
                >
                  {#if isChecked}
                    <Check class="w-3 h-3 stroke-[3]" />
                  {/if}
                </span>
              </div>

              <!-- CARD BODY -->
              <div class="my-1.5 truncate">
                <span class="text-[11px] block truncate text-muted font-medium" title={band.name}>
                  {parsed.desc}
                </span>
              </div>

              <!-- CARD FOOTER -->
              <div class="flex items-center justify-between text-[10px] w-full pt-1 border-t border-border/50">
                <span class="text-muted font-mono">ID {band.id}</span>
                {#if isLiveActive}
                  <Badge variant="success" class="text-[8px] px-1 py-0 font-bold">ACTIVE</Badge>
                {/if}
              </div>
            </button>
          {/each}
        </div>
      {/if}
    </div>
  </Card>

  <!-- ACTION BAR (LOCKED / RESET / APPLY) -->
  <div class="p-4 bg-card border-2 border-border shadow-neobrutal rounded-lg flex flex-col sm:flex-row items-center justify-between gap-4 sticky bottom-4 z-20 backdrop-blur-md">
    <div class="flex items-center gap-3 w-full sm:w-auto">
      <div class="p-2 rounded bg-accent/20 border border-accent text-accent">
        <Sliders class="w-5 h-5" />
      </div>
      <div>
        <div class="text-xs font-bold uppercase text-foreground">
          Konfigurasi SIM {selectedSlot + 1}
        </div>
        <div class="text-[11px] text-muted">
          {#if selectedBandIds.length === 0}
            <span class="text-red-400 font-bold">Pilih minimal 1 band sebelum mengunci</span>
          {:else}
            <span class="text-foreground font-bold">{selectedBandIds.length} band</span> siap dikunci
          {/if}
        </div>
      </div>
    </div>

    <div class="flex flex-wrap items-center justify-end gap-2.5 w-full sm:w-auto">
      <!-- Reset to Auto Button -->
      <Button
        variant="secondary"
        size="md"
        disabled={isLoadingBands || isResetting || isLocking}
        onclick={() => (showResetModal = true)}
        title="Kembalikan SIM ke mode otomatis"
      >
        <RotateCcw class="w-4 h-4 mr-1.5" />
        <span>Reset to Auto</span>
      </Button>

      <!-- Lock Selected Bands Button -->
      <Button
        variant="primary"
        size="md"
        disabled={selectedBandIds.length === 0 || isLoadingBands || isLocking}
        onclick={applyLock}
        title="Terapkan kunci ke frekuensi yang dipilih"
      >
        {#if isLocking}
          <RefreshCw class="w-4 h-4 mr-1.5 animate-spin" />
          <span>Locking...</span>
        {:else}
          <Lock class="w-4 h-4 mr-1.5" />
          <span>Lock Selected Bands</span>
        {/if}
      </Button>
    </div>
  </div>
</div>

<!-- RESET CONFIRMATION MODAL -->
<Modal
  open={showResetModal}
  title="Konfirmasi Reset ke Mode Auto"
  onclose={() => (showResetModal = false)}
>
  <div class="space-y-4 font-mono text-xs">
    <div class="p-3.5 rounded border-2 border-amber-500/40 bg-amber-950/30 flex items-start gap-3 text-amber-200">
      <AlertTriangle class="w-5 h-5 shrink-0 text-amber-400 mt-0.5" />
      <div class="space-y-1">
        <p class="font-bold text-sm text-foreground">
          Reset Kunci Frekuensi SIM {selectedSlot + 1}?
        </p>
        <p class="text-muted leading-relaxed">
          Tindakan ini akan menghapus seluruh pembatasan band frekuensi pada SIM {selectedSlot + 1} dan mengembalikan modem Samsung ke pemilihan dinamis (All Bands Permitted).
        </p>
      </div>
    </div>

    <p class="text-muted text-[11px] leading-relaxed">
      Koneksi data seluler Anda mungkin akan mengalami radio reset singkat (1-3 detik) saat modem kembali menegosiasikan sinyal terbaik dengan BTS terdekat.
    </p>
  </div>

  {#snippet footer()}
    <div class="flex items-center justify-end gap-2 w-full font-mono">
      <Button
        variant="outline"
        size="sm"
        disabled={isResetting}
        onclick={() => (showResetModal = false)}
      >
        Batal
      </Button>
      <Button
        variant="danger"
        size="sm"
        disabled={isResetting}
        onclick={confirmResetAuto}
      >
        {#if isResetting}
          <RefreshCw class="w-3.5 h-3.5 mr-1.5 animate-spin" />
          <span>Mereset Modem...</span>
        {:else}
          <RotateCcw class="w-3.5 h-3.5 mr-1.5" />
          <span>Ya, Reset ke Auto</span>
        {/if}
      </Button>
    </div>
  {/snippet}
</Modal>
