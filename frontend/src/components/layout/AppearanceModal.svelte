<script lang="ts">
  import {
    Palette,
    Sparkles,
    Check,
    X,
    Layers,
    LayoutGrid,
    PanelLeft,
    Upload,
    Trash2,
    Sliders,
    Image,
    Loader2,
    RotateCcw,
    Lock,
  } from '@lucide/svelte'
  import { themeStore } from '../../stores/theme.svelte'
  import {
    navigationStore,
  } from '../../stores/navigation.svelte'
  import { toastStore } from '../../stores/toast.svelte'
  import { request } from '../../api/client'
  import type { ThemeMode, UIStyle, WallpaperItem } from '../../types/common'
  import Button from '../ui/Button.svelte'
  import SakugaButton from '../ui/SakugaButton.svelte'

  interface Props {
    open?: boolean
    onclose?: () => void
  }

  let { open = false, onclose }: Props = $props()

  const isOpen = $derived(open || themeStore.showAppearanceModal)

  function handleClose() {
    onclose?.()
    themeStore.closeAppearanceModal()
  }

  interface ThemeOption {
    id: ThemeMode
    name: string
    tagline: string
    bg: string
    card: string
    border: string
    accent: string
    text: string
  }

  const themes: ThemeOption[] = [
    {
      id: 'dark',
      name: 'Dark Navy',
      tagline: 'Deep slate telemetry palette',
      bg: '#090d16',
      card: '#131927',
      border: '#2a3449',
      accent: '#7c3aed',
      text: '#f8fafc',
    },
    {
      id: 'light',
      name: 'Clean Light',
      tagline: 'High-contrast stark daylight mode',
      bg: '#f4f6fa',
      card: '#ffffff',
      border: '#0f172a',
      accent: '#2563eb',
      text: '#0f172a',
    },
    {
      id: 'neobrutalism-light',
      name: 'Neobrutalism Light',
      tagline: 'Warm anti-glare paper & pop accents',
      bg: '#fff6e8',
      card: '#fffdf8',
      border: '#17151a',
      accent: '#ff5ca8',
      text: '#17151a',
    },
    {
      id: 'neobrutalism',
      name: 'Neobrutalism',
      tagline: 'Retro-pop neobrutalism on purple charcoal',
      bg: '#18151e',
      card: '#24202b',
      border: '#0d0b10',
      accent: '#ffe14a',
      text: '#fffaf4',
    },
  ]

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && isOpen) {
      handleClose()
    }
  }

  function selectTheme(theme: ThemeMode) {
    themeStore.setTheme(theme)
  }

  function selectStyle(style: UIStyle) {
    themeStore.setStyle(style)
  }

  let wallpapers = $state<WallpaperItem[]>([])
  let isLoadingWallpapers = $state(false)
  let isUploading = $state(false)
  let showAdvancedTuning = $state(false)
  let showLoginWallpaper = $state(false)

  const PRESETS = [
    { id: 'preset:none', name: 'None (Pure Tint)', preview: 'linear-gradient(135deg, rgba(255,255,255,0.05), rgba(255,255,255,0.02))' },
    { id: 'preset:cyber-neon', name: 'Cyber Neon', preview: 'linear-gradient(135deg, #0d0f18, #1a102f, #0d273a)' },
    { id: 'preset:aurora-mesh', name: 'Aurora Mesh', preview: 'linear-gradient(135deg, #120b29, #172554, #3b0764, #042f2e)' },
    { id: 'preset:sunset-glow', name: 'Sunset Glow', preview: 'linear-gradient(135deg, #1b0a2a, #3d1445, #5e1c43)' },
    { id: 'preset:dark-velvet', name: 'Dark Velvet', preview: 'linear-gradient(160deg, #18151e, #2b1f38, #110e16)' },
  ]

  async function loadWallpapers() {
    isLoadingWallpapers = true
    try {
      const res = await request<{ success: boolean; data: WallpaperItem[] }>('/api/theme/wallpapers')
      if (res && res.success && Array.isArray(res.data)) {
        wallpapers = res.data
      }
    } catch (err) {
      console.error('Failed to load wallpapers', err)
    } finally {
      isLoadingWallpapers = false
    }
  }

  $effect(() => {
    if (isOpen) {
      loadWallpapers()
    }
  })

  async function handleUploadWallpaper(e: Event) {
    const input = e.target as HTMLInputElement
    if (!input.files || input.files.length === 0) return
    const file = input.files[0]
    isUploading = true
    try {
      const fd = new FormData()
      fd.append('file', file)
      const res = await request<{ success: boolean; data: WallpaperItem }>('/api/theme/wallpapers/upload', {
        method: 'POST',
        body: fd,
      })
      if (res && res.success && res.data) {
        wallpapers = [res.data, ...wallpapers]
        themeStore.setActiveWallpaper(res.data.url)
        toastStore.success(`Wallpaper "${res.data.name}" berhasil diunggah!`)
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Gagal mengunggah wallpaper'
      toastStore.error(msg)
    } finally {
      isUploading = false
      input.value = ''
    }
  }

  async function handleDeleteWallpaper(item: WallpaperItem, e: MouseEvent) {
    e.stopPropagation()
    if (!confirm(`Hapus wallpaper "${item.name}" dari galeri?`)) return
    try {
      await request(`/api/theme/wallpapers/delete?id=${encodeURIComponent(item.id)}`, { method: 'DELETE' })
      wallpapers = wallpapers.filter((w) => w.id !== item.id)
      if (themeStore.activeWallpaper === item.url) {
        themeStore.setActiveWallpaper('')
      }
      toastStore.success(`Wallpaper "${item.name}" dihapus`)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Gagal menghapus wallpaper'
      toastStore.error(msg)
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if isOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 overflow-y-auto">
    <!-- Backdrop -->
    <div
      class="fixed inset-0 bg-black/80 backdrop-blur-sm transition-opacity"
      onclick={handleClose}
    ></div>

    <!-- Modal Dialog Window -->
    <div
      class="neo-card relative w-full max-w-xl bg-card border-2 border-border p-4 sm:p-5 z-10 space-y-4 max-h-[92vh] overflow-y-auto my-auto shadow-2xl text-foreground font-mono"
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-border pb-3">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded bg-accent/15 text-accent border border-accent/30 flex items-center justify-center">
            <Palette class="w-4 h-4" />
          </div>
          <div>
            <h2 class="text-sm font-bold uppercase tracking-wider text-foreground">
              Appearance Studio
            </h2>
            <p class="text-[10px] text-muted">Theme palette & UI paradigm configuration</p>
          </div>
        </div>

        <button
          type="button"
          class="p-1 rounded border border-border text-muted hover:text-foreground hover:border-accent cursor-pointer transition-colors"
          onclick={handleClose}
          aria-label="Close"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Section 1: Color Themes (4 Palettes) -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
          <Sparkles class="w-3.5 h-3.5 text-accent" />
          Color Themes ({themes.length} Palettes)
        </span>
        <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
          {#each themes as t}
            {@const isSelected = themeStore.currentTheme === t.id}
            <button
              type="button"
              class="p-2.5 rounded-lg border-2 text-left transition-all cursor-pointer relative flex flex-col justify-between gap-1.5 {isSelected ? 'border-accent bg-accent/10 shadow-neobrutal-sm' : 'border-border bg-card-sub hover:border-accent/60'}"
              onclick={() => selectTheme(t.id)}
            >
              <div class="flex items-center justify-between">
                <!-- 3 Color Swatches -->
                <div class="flex items-center gap-1">
                  <span class="w-3.5 h-3.5 rounded-full border border-black/20 shrink-0" style="background-color: {t.bg};"></span>
                  <span class="w-3.5 h-3.5 rounded-full border border-black/20 shrink-0" style="background-color: {t.card};"></span>
                  <span class="w-3.5 h-3.5 rounded-full border border-black/20 shrink-0" style="background-color: {t.accent};"></span>
                </div>
                {#if isSelected}
                  <span class="w-4 h-4 rounded-full bg-accent text-accent-text flex items-center justify-center text-[10px] font-black">
                    <Check class="w-2.5 h-2.5" />
                  </span>
                {/if}
              </div>
              <div class="font-bold text-xs text-foreground truncate">
                {t.name}
              </div>
            </button>
          {/each}
        </div>
      </div>

      <!-- Section 2: Navigation Layout (Top Bar vs Sidebar) -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
          <LayoutGrid class="w-3.5 h-3.5 text-accent" />
          Navigation Layout
        </span>
        <div class="grid grid-cols-2 gap-2 bg-card-sub p-1 border border-border rounded-lg">
          <button
            type="button"
            class="flex items-center justify-center gap-2 py-2 px-3 rounded text-xs font-bold transition-all cursor-pointer {navigationStore.layout === 'topbar' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
            onclick={() => navigationStore.setLayout('topbar')}
          >
            <LayoutGrid class="w-3.5 h-3.5" />
            <span>Classic Top Bar</span>
          </button>
          <button
            type="button"
            class="flex items-center justify-center gap-2 py-2 px-3 rounded text-xs font-bold transition-all cursor-pointer {navigationStore.layout === 'sidebar' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
            onclick={() => navigationStore.setLayout('sidebar')}
          >
            <PanelLeft class="w-3.5 h-3.5" />
            <span>Modern Sidebar</span>
          </button>
        </div>
      </div>

      {#if !themeStore.isNeobrutalTheme()}
      <!-- Section 3: UI Paradigm Style (Modern vs Glass) -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
          <Layers class="w-3.5 h-3.5 text-accent" />
          UI Paradigm Style
        </span>
        <div class="grid grid-cols-2 gap-2 bg-card-sub p-1 border border-border rounded-lg">
          <button
            type="button"
            class="flex flex-col items-center justify-center py-2 px-3 rounded text-xs font-bold transition-all cursor-pointer {themeStore.currentStyle === 'modern' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
            onclick={() => selectStyle('modern')}
          >
            <span>Modern Clean</span>
            <span class="text-[10px] font-normal opacity-80">1px borders, smooth curves</span>
          </button>
          <button
            type="button"
            class="flex flex-col items-center justify-center py-2 px-3 rounded text-xs font-bold transition-all cursor-pointer {themeStore.currentStyle === 'glass' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
            onclick={() => selectStyle('glass')}
          >
            <span>Glassmorphism</span>
            <span class="text-[10px] font-normal opacity-80">Frosted acrylic & blur</span>
          </button>
        </div>
      </div>
      {:else}
      <div class="p-3 bg-card-sub border border-border rounded-lg text-[11px] text-muted">
        Bentuk Neobrutalism sudah bagian dari tema ini — style dikunci ke Neobrutalist.
      </div>
      {/if}

      {#if themeStore.isNeobrutalTheme()}
      <!-- Sakuga FX (Neobrutalism only) -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
          <Sparkles class="w-3.5 h-3.5 text-accent" />
          Sakuga FX
        </span>
        <div class="flex items-center justify-between gap-3 p-2.5 bg-card-sub border border-border rounded-lg">
          <div class="min-w-0">
            <div class="text-xs font-bold text-foreground">Efek anime Sakuga</div>
            <p class="text-[10px] text-muted">Tombol utama meledak ala Sakuga. Hanya aktif di tema Neobrutalism.</p>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={themeStore.sakugaFx}
            aria-label="Sakuga FX"
            class="shrink-0 w-11 h-6 rounded-full border-2 border-border transition-colors cursor-pointer relative {themeStore.sakugaFx ? 'bg-accent' : 'bg-card'}"
            onclick={() => themeStore.setSakugaFx(!themeStore.sakugaFx)}
          >
            <span class="absolute top-0.5 w-4 h-4 rounded-full bg-white border border-black/20 transition-all {themeStore.sakugaFx ? 'left-[22px]' : 'left-0.5'}"></span>
          </button>
        </div>
      </div>
      {/if}

      <!-- Advanced Tuning (collapsed by default) -->
      {#if themeStore.currentStyle === 'neobrutal' || themeStore.currentStyle === 'glass'}
      <button
        type="button"
        class="w-full flex items-center justify-between p-2.5 bg-card-sub border border-border rounded-lg text-xs font-bold text-foreground hover:border-accent transition-colors cursor-pointer"
        onclick={() => showAdvancedTuning = !showAdvancedTuning}
      >
        <span class="flex items-center gap-1.5">
          <Sliders class="w-3.5 h-3.5 text-accent" />
          Advanced Tuning
        </span>
        <span class="text-muted">{showAdvancedTuning ? 'Tutup' : 'Buka'}</span>
      </button>
      {/if}

      <!-- Section 2.4: Neobrutalism Extended Tuning -->
      {#if showAdvancedTuning && themeStore.currentStyle === 'neobrutal'}
        <div class="p-3.5 bg-card-sub border border-border rounded-xl space-y-3.5">
          <div class="flex items-center justify-between border-b border-border/50 pb-2">
            <span class="text-xs font-bold uppercase text-foreground flex items-center gap-1.5">
              <Sliders class="w-3.5 h-3.5 text-accent" />
              Neobrutalism Extended Tuning
            </span>
            <button
              type="button"
              class="inline-flex items-center gap-1 text-[10px] font-bold py-1 px-2 rounded bg-card hover:bg-card-sub text-muted hover:text-foreground border border-border transition-colors cursor-pointer"
              title="Reset ke pengaturan awal Neobrutalism"
              onclick={() => {
                themeStore.resetNeobrutalSettings()
                toastStore.success('Pengaturan Neobrutalism dikembalikan ke default')
              }}
            >
              <RotateCcw class="w-2.5 h-2.5" />
              <span>Reset Default</span>
            </button>
          </div>

          <!-- Sliders Grid (4 sliders) -->
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <!-- Shadow Depth -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">Shadow</span>
                <span class="text-accent font-mono font-bold">{themeStore.neoShadowSize}px</span>
              </div>
              <input
                type="range"
                min="0"
                max="8"
                step="1"
                value={themeStore.neoShadowSize}
                oninput={(e) => themeStore.setNeoShadowSize(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
            </div>

            <!-- Border Thickness -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">Border</span>
                <span class="text-accent font-mono font-bold">{themeStore.neoBorderWidth}px</span>
              </div>
              <input
                type="range"
                min="1"
                max="4"
                step="1"
                value={themeStore.neoBorderWidth}
                oninput={(e) => themeStore.setNeoBorderWidth(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
            </div>

            <!-- Corner Radius -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">Radius</span>
                <span class="text-accent font-mono font-bold">{themeStore.neoRadius}px</span>
              </div>
              <input
                type="range"
                min="0"
                max="14"
                step="1"
                value={themeStore.neoRadius}
                oninput={(e) => themeStore.setNeoRadius(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
            </div>

            <!-- Press Depth -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">Click Pop</span>
                <span class="text-accent font-mono font-bold">{themeStore.neoPressDepth}px</span>
              </div>
              <input
                type="range"
                min="1"
                max="4"
                step="1"
                value={themeStore.neoPressDepth}
                oninput={(e) => themeStore.setNeoPressDepth(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
            </div>
          </div>

          <!-- Pattern Texture Options -->
          <div class="space-y-2 pt-1 border-t border-border/50">
            <div class="flex items-center justify-between">
              <span class="text-[11px] font-bold uppercase text-muted">Canvas Pattern Texture</span>
              <div class="flex items-center gap-2">
                <span class="text-[10px] text-muted">Opacity:</span>
                <span class="text-[10px] font-mono text-accent font-bold">{Math.round(themeStore.neoPatternOpacity * 100)}%</span>
                <input
                  type="range"
                  min="0"
                  max="0.12"
                  step="0.01"
                  value={themeStore.neoPatternOpacity}
                  oninput={(e) => themeStore.setNeoPatternOpacity(Number((e.target as HTMLInputElement).value))}
                  class="w-20 accent-accent cursor-pointer"
                />
              </div>
            </div>

            <div class="grid grid-cols-3 gap-2">
              <button
                type="button"
                class="py-1.5 px-2 rounded-lg border text-xs font-bold transition-all cursor-pointer {themeStore.neoPattern === 'dots' ? 'bg-accent text-accent-text border-accent shadow-neobrutal-sm' : 'bg-card text-muted hover:text-foreground border-border'}"
                onclick={() => themeStore.setNeoPattern('dots')}
              >
                Dots (Titik)
              </button>
              <button
                type="button"
                class="py-1.5 px-2 rounded-lg border text-xs font-bold transition-all cursor-pointer {themeStore.neoPattern === 'grid' ? 'bg-accent text-accent-text border-accent shadow-neobrutal-sm' : 'bg-card text-muted hover:text-foreground border-border'}"
                onclick={() => themeStore.setNeoPattern('grid')}
              >
                Grid (Petak)
              </button>
              <button
                type="button"
                class="py-1.5 px-2 rounded-lg border text-xs font-bold transition-all cursor-pointer {themeStore.neoPattern === 'none' ? 'bg-accent text-accent-text border-accent shadow-neobrutal-sm' : 'bg-card text-muted hover:text-foreground border-border'}"
                onclick={() => themeStore.setNeoPattern('none')}
              >
                Polos (Clean)
              </button>
            </div>
          </div>
        </div>
      {/if}

      <!-- Section 2.5: Glassmorphism Customizer (Only shown when Glass style active) -->
      {#if showAdvancedTuning && themeStore.currentStyle === 'glass'}
        <div class="p-3.5 bg-card-sub border border-border rounded-xl space-y-3.5">
          <div class="flex items-center justify-between border-b border-border/50 pb-2">
            <span class="text-xs font-bold uppercase text-foreground flex items-center gap-1.5">
              <Sliders class="w-3.5 h-3.5 text-accent" />
              Glass & Wallpaper Tuning
            </span>
            <span class="text-[10px] text-muted">Per-Device Customization</span>
          </div>

          <!-- Sliders Grid -->
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <!-- Blur Slider -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">Blur Amount</span>
                <span class="text-accent font-mono font-bold">{themeStore.glassBlur}px</span>
              </div>
              <input
                type="range"
                min="4"
                max="32"
                step="2"
                value={themeStore.glassBlur}
                oninput={(e) => themeStore.setGlassBlur(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
            </div>

            <!-- Opacity Slider -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">Card Opacity</span>
                <span class="text-accent font-mono font-bold">{Math.round(themeStore.glassOpacity * 100)}%</span>
              </div>
              <input
                type="range"
                min="0.2"
                max="0.95"
                step="0.05"
                value={themeStore.glassOpacity}
                oninput={(e) => themeStore.setGlassOpacity(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
            </div>

            <!-- Dim Slider -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">Wallpaper Dim</span>
                <span class="text-accent font-mono font-bold">{Math.round(themeStore.glassDim * 100)}%</span>
              </div>
              <input
                type="range"
                min="0.2"
                max="1.0"
                step="0.05"
                value={themeStore.glassDim}
                oninput={(e) => themeStore.setGlassDim(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
            </div>
          </div>

          <!-- Wallpaper Gallery & Presets -->
          <div class="space-y-2 pt-1 border-t border-border/50">
            <div class="flex items-center justify-between">
              <span class="text-[11px] font-bold uppercase text-muted flex items-center gap-1.5">
                <Image class="w-3.5 h-3.5 text-accent" />
                Active Wallpaper
              </span>

              <!-- Upload Button -->
              <label class="cursor-pointer inline-flex items-center gap-1.5 text-[10px] font-bold py-1 px-2.5 rounded bg-accent/20 hover:bg-accent/30 text-accent border border-accent/40 transition-colors">
                {#if isUploading}
                  <Loader2 class="w-3 h-3 animate-spin" />
                  <span>Uploading...</span>
                {:else}
                  <Upload class="w-3 h-3" />
                  <span>+ Upload Baru</span>
                {/if}
                <input
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif,image/avif,image/jfif,image/bmp"
                  class="hidden"
                  onchange={handleUploadWallpaper}
                  disabled={isUploading}
                />
              </label>
            </div>

            <!-- Preset / Uploaded Grid -->
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 max-h-48 overflow-y-auto pr-1">
              <!-- Presets -->
              {#each PRESETS as preset}
                {@const isSelected = themeStore.activeWallpaper === preset.id || (!themeStore.activeWallpaper && preset.id === 'preset:none')}
                <button
                  type="button"
                  class="relative flex flex-col items-center justify-center h-16 rounded-lg border-2 transition-all overflow-hidden p-1 text-center cursor-pointer {isSelected ? 'border-accent shadow-neobrutal-sm' : 'border-border/60 hover:border-border'}"
                  style="background: {preset.preview}; background-size: cover;"
                  onclick={() => themeStore.setActiveWallpaper(preset.id)}
                >
                  <span class="text-[10px] font-bold bg-black/60 px-1.5 py-0.5 rounded text-white">
                    {preset.name}
                  </span>
                  {#if isSelected}
                    <div class="absolute top-1 right-1 w-4 h-4 bg-accent text-accent-text rounded-full flex items-center justify-center">
                      <Check class="w-2.5 h-2.5" />
                    </div>
                  {/if}
                </button>
              {/each}

              <!-- Uploaded Wallpapers from HP / Server -->
              {#each wallpapers as item}
                {@const isSelected = themeStore.activeWallpaper === item.url}
                <div
                  class="group relative flex flex-col items-center justify-end h-16 rounded-lg border-2 transition-all overflow-hidden p-1 text-center cursor-pointer {isSelected ? 'border-accent shadow-neobrutal-sm' : 'border-border/60 hover:border-border'}"
                  style="background-image: url('{item.url}'); background-size: cover; background-position: center;"
                  onclick={() => themeStore.setActiveWallpaper(item.url)}
                  role="button"
                  tabindex="0"
                  onkeydown={(e) => e.key === 'Enter' && themeStore.setActiveWallpaper(item.url)}
                >
                  <span class="text-[9px] font-bold bg-black/70 px-1.5 py-0.5 rounded text-white truncate max-w-full">
                    {item.name}
                  </span>

                  {#if isSelected}
                    <div class="absolute top-1 right-1 w-4 h-4 bg-accent text-accent-text rounded-full flex items-center justify-center">
                      <Check class="w-2.5 h-2.5" />
                    </div>
                  {/if}

                  <!-- Delete Button -->
                  <button
                    type="button"
                    class="absolute top-1 left-1 w-5 h-5 bg-red-600/90 hover:bg-red-600 text-white rounded flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer shadow"
                    title="Hapus wallpaper dari galeri"
                    onclick={(e) => handleDeleteWallpaper(item, e)}
                  >
                    <Trash2 class="w-3 h-3" />
                  </button>
                </div>
              {/each}
            </div>
          </div>
        </div>
      {/if}

      <!-- Login Screen Wallpaper (collapsed by default) -->
      <button
        type="button"
        class="w-full flex items-center justify-between p-2.5 bg-card-sub border border-border rounded-lg text-xs font-bold text-foreground hover:border-accent transition-colors cursor-pointer"
        onclick={() => showLoginWallpaper = !showLoginWallpaper}
      >
        <span class="flex items-center gap-1.5">
          <Lock class="w-3.5 h-3.5 text-accent" />
          Login Screen Wallpaper
        </span>
        <span class="text-muted">{showLoginWallpaper ? 'Tutup' : 'Buka'}</span>
      </button>
      {#if showLoginWallpaper}
        <!-- Section 5: Login Page Wallpaper (Static & Independent) -->
        <div class="space-y-3 pt-3 border-t border-border">
          <div class="flex items-center justify-between">
            <div>
              <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
                <Lock class="w-3.5 h-3.5 text-accent" />
                Login Screen Wallpaper (Static & Independent)
              </span>
              <p class="text-[10px] text-muted">
                Khusus latar belakang halaman masuk. Tidak mempengaruhi warna tema, kartu, atau dashboard.
              </p>
            </div>
            {#if themeStore.loginWallpaper && themeStore.loginWallpaper !== 'preset:none'}
              <button
                type="button"
                onclick={() => themeStore.setLoginWallpaper('preset:none')}
                class="text-[10px] font-bold text-red-400 hover:text-red-300 underline cursor-pointer"
              >
                Reset ke Polos
              </button>
            {/if}
          </div>

          <!-- Preset / Uploaded Wallpaper Gallery Grid -->
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-[10px] text-muted font-bold uppercase">Pilih Gambar Latar Login:</span>
              <label class="cursor-pointer inline-flex items-center gap-1.5 text-[10px] font-bold py-1 px-2.5 rounded bg-accent/20 hover:bg-accent/30 text-accent border border-accent/40 transition-colors">
                {#if isUploading}
                  <Loader2 class="w-3 h-3 animate-spin" />
                  <span>Uploading...</span>
                {:else}
                  <Upload class="w-3 h-3" />
                  <span>+ Upload Wallpaper</span>
                {/if}
                <input
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif,image/avif,image/jfif,image/bmp"
                  class="hidden"
                  onchange={handleUploadWallpaper}
                  disabled={isUploading}
                />
              </label>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 max-h-44 overflow-y-auto pr-1">
              <!-- Presets -->
              {#each PRESETS as preset}
                {@const isSelected = themeStore.loginWallpaper === preset.id || (!themeStore.loginWallpaper && preset.id === 'preset:none')}
                <button
                  type="button"
                  class="relative flex flex-col items-center justify-center h-16 rounded-lg border-2 transition-all overflow-hidden p-1 text-center cursor-pointer {isSelected ? 'border-accent shadow-[2px_2px_0px_var(--neo-shadow)]' : 'border-border/60 hover:border-border'}"
                  style="background: {preset.preview}; background-size: cover;"
                  onclick={() => themeStore.setLoginWallpaper(preset.id)}
                >
                  <span class="text-[10px] font-bold bg-black/60 px-1.5 py-0.5 rounded text-white">
                    {preset.name}
                  </span>
                  {#if isSelected}
                    <span class="absolute top-1 right-1 w-3.5 h-3.5 rounded-full bg-accent text-accent-text flex items-center justify-center text-[9px] font-black">
                      <Check class="w-2.5 h-2.5" />
                    </span>
                  {/if}
                </button>
              {/each}

              <!-- User Uploaded Wallpapers -->
              {#each wallpapers as item}
                {@const isSelected = themeStore.loginWallpaper === item.url}
                <div
                  class="group relative flex flex-col items-center justify-end h-16 rounded-lg border-2 transition-all overflow-hidden p-1 text-center cursor-pointer {isSelected ? 'border-accent shadow-[2px_2px_0px_var(--neo-shadow)]' : 'border-border/60 hover:border-border'}"
                  style="background-image: url('{item.url}'); background-size: cover; background-position: center;"
                  role="button"
                  tabindex="0"
                  onclick={() => themeStore.setLoginWallpaper(item.url)}
                  onkeydown={(e) => e.key === 'Enter' && themeStore.setLoginWallpaper(item.url)}
                >
                  <span class="text-[9px] font-mono bg-black/75 px-1 py-0.5 rounded text-white truncate max-w-[90%]">
                    {item.name}
                  </span>
                  {#if isSelected}
                    <span class="absolute top-1 right-1 w-3.5 h-3.5 rounded-full bg-accent text-accent-text flex items-center justify-center text-[9px] font-black">
                      <Check class="w-2.5 h-2.5" />
                    </span>
                  {/if}
                  <button
                    type="button"
                    class="absolute top-1 left-1 w-5 h-5 rounded bg-red-600/90 text-white flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-700 cursor-pointer"
                    title="Hapus wallpaper dari galeri"
                    onclick={(e) => handleDeleteWallpaper(item, e)}
                  >
                    <Trash2 class="w-3 h-3" />
                  </button>
                </div>
              {/each}
            </div>
          </div>

          <!-- Controls: Dim & Blur -->
          {#if themeStore.loginWallpaper && themeStore.loginWallpaper !== 'preset:none'}
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 p-3 bg-card-sub border-2 border-border rounded-lg">
              <!-- Dimming -->
              <div class="space-y-1">
                <div class="flex items-center justify-between text-[11px] font-mono">
                  <span class="text-muted font-bold">Dimming (Keterbacaan):</span>
                  <span class="font-bold text-accent">{Math.round(themeStore.loginWallpaperDim * 100)}%</span>
                </div>
                <input
                  type="range"
                  min="0"
                  max="0.8"
                  step="0.05"
                  value={themeStore.loginWallpaperDim}
                  oninput={(e) => themeStore.setLoginWallpaperDim(Number((e.target as HTMLInputElement).value))}
                  class="w-full accent-accent cursor-pointer"
                />
              </div>

              <!-- Blur -->
              <div class="space-y-1">
                <div class="flex items-center justify-between text-[11px] font-mono">
                  <span class="text-muted font-bold">Latar Blur:</span>
                  <span class="font-bold text-accent">{themeStore.loginWallpaperBlur}px</span>
                </div>
                <input
                  type="range"
                  min="0"
                  max="16"
                  step="1"
                  value={themeStore.loginWallpaperBlur}
                  oninput={(e) => themeStore.setLoginWallpaperBlur(Number((e.target as HTMLInputElement).value))}
                  class="w-full accent-accent cursor-pointer"
                />
              </div>
            </div>
          {/if}
        </div>
      {/if}

      <!-- Footer: Summary and Done Button -->
      <div class="flex items-center justify-between pt-3 border-t border-border">
        <span class="text-[11px] text-muted">
          Active: <strong class="text-foreground">{themes.find(t => t.id === themeStore.currentTheme)?.name}</strong> ({themeStore.currentStyle})
        </span>
        {#if themeStore.isSakugaFxActive()}
          <SakugaButton onclick={handleClose}>Done & Apply</SakugaButton>
        {:else}
          <Button variant="primary" size="sm" onclick={handleClose}>
            <Check class="w-3.5 h-3.5 mr-1" />
            <span>Done & Apply</span>
          </Button>
        {/if}
      </div>
    </div>
  </div>
{/if}
