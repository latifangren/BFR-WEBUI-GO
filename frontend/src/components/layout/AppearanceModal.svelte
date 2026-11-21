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
    Languages,
  } from '@lucide/svelte'
  import { themeStore } from '../../stores/theme.svelte'
  import {
    navigationStore,
  } from '../../stores/navigation.svelte'
  import { toastStore } from '../../stores/toast.svelte'
  import { locale, LL, setLocale } from '../../i18n/i18n-svelte'
  import { loadLocaleAsync } from '../../i18n/i18n-util.async'
  import { request } from '../../api/client'
  import type { ThemeMode, UIStyle, WallpaperItem } from '../../types/common'
  import Button from '../ui/Button.svelte'
  import SakugaButton from '../ui/SakugaButton.svelte'
  import SakugaSlider from '../ui/SakugaSlider.svelte'

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

  // Theme names/taglines come from the locale so they follow the language.
  const themes = $derived.by((): ThemeOption[] => [
    {
      id: 'dark',
      name: $LL.appearance.themes.dark.name(),
      tagline: $LL.appearance.themes.dark.tagline(),
      bg: '#090d16',
      card: '#131927',
      border: '#2a3449',
      accent: '#7c3aed',
      text: '#f8fafc',
    },
    {
      id: 'light',
      name: $LL.appearance.themes.light.name(),
      tagline: $LL.appearance.themes.light.tagline(),
      bg: '#f4f6fa',
      card: '#ffffff',
      border: '#0f172a',
      accent: '#2563eb',
      text: '#0f172a',
    },
    {
      id: 'neobrutalism-light',
      name: $LL.appearance.themes.neobrutalismLight.name(),
      tagline: $LL.appearance.themes.neobrutalismLight.tagline(),
      bg: '#fff6e8',
      card: '#fffdf8',
      border: '#17151a',
      accent: '#ff5ca8',
      text: '#17151a',
    },
    {
      id: 'neobrutalism',
      name: $LL.appearance.themes.neobrutalism.name(),
      tagline: $LL.appearance.themes.neobrutalism.tagline(),
      bg: '#18151e',
      card: '#24202b',
      border: '#0d0b10',
      accent: '#ffe14a',
      text: '#fffaf4',
    },
  ])

  // Footer style name, translated like the style buttons.
  const styleName = $derived.by(() => {
    if (themeStore.currentStyle === 'modern') return $LL.appearance.styleModern()
    if (themeStore.currentStyle === 'glass') return $LL.appearance.styleGlass()
    return $LL.appearance.styleNeobrutal()
  })

  async function selectLanguage(lang: 'en' | 'id') {
    try {
      await loadLocaleAsync(lang)
    } catch {
      // A failed chunk keeps the current locale; never block the switcher.
    }
    setLocale(lang)
    try {
      localStorage.setItem('locale', lang)
    } catch {
      // Private mode etc. — the choice just won't persist.
    }
  }

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

  // Accent picker presets: the pop colors already used across the themes.
  const ACCENT_PRESETS = $derived.by(() => [
    { name: $LL.appearance.accentPresets.pink(), hex: '#ff5ca8' },
    { name: $LL.appearance.accentPresets.yellow(), hex: '#ffe14a' },
    { name: $LL.appearance.accentPresets.cyan(), hex: '#55dfff' },
    { name: $LL.appearance.accentPresets.lime(), hex: '#b8f34a' },
    { name: $LL.appearance.accentPresets.purple(), hex: '#9b7cff' },
    { name: $LL.appearance.accentPresets.orange(), hex: '#ff9955' },
  ])

  // The accent actually shown: the custom override when set, otherwise the
  // active theme's own accent.
  const effectiveAccent = $derived(
    themeStore.customAccent ||
      themes.find((t) => t.id === themeStore.currentTheme)?.accent ||
      '#ffe14a',
  )

  let wallpapers = $state<WallpaperItem[]>([])
  let isLoadingWallpapers = $state(false)
  let isUploading = $state(false)
  let showAdvancedTuning = $state(false)
  let showLoginWallpaper = $state(false)

  const PRESETS = $derived.by(() => [
    { id: 'preset:none', name: $LL.appearance.presets.none(), preview: 'linear-gradient(135deg, rgba(255,255,255,0.05), rgba(255,255,255,0.02))' },
    { id: 'preset:cyber-neon', name: $LL.appearance.presets.cyberNeon(), preview: 'linear-gradient(135deg, #0d0f18, #1a102f, #0d273a)' },
    { id: 'preset:aurora-mesh', name: $LL.appearance.presets.auroraMesh(), preview: 'linear-gradient(135deg, #120b29, #172554, #3b0764, #042f2e)' },
    { id: 'preset:sunset-glow', name: $LL.appearance.presets.sunsetGlow(), preview: 'linear-gradient(135deg, #1b0a2a, #3d1445, #5e1c43)' },
    { id: 'preset:dark-velvet', name: $LL.appearance.presets.darkVelvet(), preview: 'linear-gradient(160deg, #18151e, #2b1f38, #110e16)' },
  ])

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
        toastStore.success($LL.appearance.wallpaperUploaded({ name: res.data.name }))
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : $LL.appearance.uploadFailed()
      toastStore.error(msg)
    } finally {
      isUploading = false
      input.value = ''
    }
  }

  async function handleDeleteWallpaper(item: WallpaperItem, e: MouseEvent) {
    e.stopPropagation()
    if (!confirm($LL.appearance.deleteConfirm({ name: item.name }))) return
    try {
      await request(`/api/theme/wallpapers/delete?id=${encodeURIComponent(item.id)}`, { method: 'DELETE' })
      wallpapers = wallpapers.filter((w) => w.id !== item.id)
      if (themeStore.activeWallpaper === item.url) {
        themeStore.setActiveWallpaper('')
      }
      toastStore.success($LL.appearance.wallpaperDeleted({ name: item.name }))
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : $LL.appearance.deleteFailed()
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
              {$LL.appearance.title()}
            </h2>
            <p class="text-[10px] text-muted">{$LL.appearance.subtitle()}</p>
          </div>
        </div>

        <button
          type="button"
          class="p-1 rounded border border-border text-muted hover:text-foreground hover:border-accent cursor-pointer transition-colors"
          onclick={handleClose}
          aria-label={$LL.common.close()}
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Section 1: Color Themes (4 Palettes) -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
          <Sparkles class="w-3.5 h-3.5 text-accent" />
          {$LL.appearance.colorThemes({ count: themes.length })}
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
                  <span class="w-3.5 h-3.5 rounded-full border border-black/20 shrink-0" style="background-color: {isSelected && themeStore.customAccent ? themeStore.customAccent : t.accent};"></span>
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

      <!-- Section 1a: Language -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
          <Languages class="w-3.5 h-3.5 text-accent" />
          {$LL.appearance.language()}
        </span>
        <p class="text-[10px] text-muted">{$LL.appearance.languageDesc()}</p>
        <div class="grid grid-cols-2 gap-2 bg-card-sub p-1 border border-border rounded-lg">
          <button
            type="button"
            class="flex items-center justify-center gap-2 py-2 px-3 rounded text-xs font-bold transition-all cursor-pointer {$locale === 'en' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
            onclick={() => selectLanguage('en')}
          >
            <span>{$LL.appearance.languageEnglish()}</span>
            {#if $locale === 'en'}
              <Check class="w-3.5 h-3.5" />
            {/if}
          </button>
          <button
            type="button"
            class="flex items-center justify-center gap-2 py-2 px-3 rounded text-xs font-bold transition-all cursor-pointer {$locale === 'id' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
            onclick={() => selectLanguage('id')}
          >
            <span>{$LL.appearance.languageIndonesian()}</span>
            {#if $locale === 'id'}
              <Check class="w-3.5 h-3.5" />
            {/if}
          </button>
        </div>
      </div>

      <!-- Section 1b: Accent Color (all themes) -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
          <Palette class="w-3.5 h-3.5 text-accent" />
          {$LL.appearance.accentColor()}
        </span>
        <div class="p-2.5 bg-card-sub border border-border rounded-lg space-y-2.5">
          <div class="flex items-center gap-3">
            <!-- Native color picker (swatch preview) -->
            <label
              class="relative shrink-0 w-10 h-10 rounded-lg border-2 border-border overflow-hidden cursor-pointer"
              style="background-color: {effectiveAccent};"
              title={$LL.appearance.pickAccent()}
            >
              <input
                type="color"
                value={effectiveAccent}
                oninput={(e) => themeStore.setCustomAccent((e.target as HTMLInputElement).value)}
                class="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
                aria-label={$LL.appearance.pickAccent()}
              />
            </label>
            <div class="min-w-0 flex-1">
              <div class="text-xs font-bold text-foreground">
                {themeStore.customAccent ? $LL.appearance.customAccent() : $LL.appearance.themeDefaultAccent()}
              </div>
              <p class="text-[10px] text-muted">
                {themeStore.customAccent
                  ? $LL.appearance.customAccentNote()
                  : $LL.appearance.defaultAccentNote()}
              </p>
            </div>
            <span class="font-mono text-xs font-bold text-accent uppercase">{effectiveAccent}</span>
            {#if themeStore.customAccent}
              <button
                type="button"
                class="inline-flex items-center gap-1 text-[10px] font-bold py-1 px-2 rounded bg-card hover:bg-card-sub text-muted hover:text-foreground border border-border transition-colors cursor-pointer"
                title={$LL.appearance.resetAccentTitle()}
                onclick={() => themeStore.setCustomAccent(null)}
              >
                <RotateCcw class="w-2.5 h-2.5" />
                <span>{$LL.common.reset()}</span>
              </button>
            {/if}
          </div>
          <!-- Preset swatches -->
          <div class="flex items-center gap-1.5">
            {#each ACCENT_PRESETS as preset}
              <button
                type="button"
                class="w-7 h-7 rounded-full border-2 cursor-pointer transition-transform hover:scale-110 {themeStore.customAccent === preset.hex
                  ? 'border-foreground scale-110'
                  : 'border-black/20'}"
                style="background-color: {preset.hex};"
                title={preset.name}
                aria-label={$LL.appearance.accentPresetAria({ name: preset.name })}
                onclick={() => themeStore.setCustomAccent(preset.hex)}
              ></button>
            {/each}
          </div>
        </div>
      </div>

      <!-- Section 2: Navigation Layout (Top Bar vs Sidebar) -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
          <LayoutGrid class="w-3.5 h-3.5 text-accent" />
          {$LL.appearance.navLayout()}
        </span>
        <div class="grid grid-cols-2 gap-2 bg-card-sub p-1 border border-border rounded-lg">
          <button
            type="button"
            class="flex items-center justify-center gap-2 py-2 px-3 rounded text-xs font-bold transition-all cursor-pointer {navigationStore.layout === 'topbar' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
            onclick={() => navigationStore.setLayout('topbar')}
          >
            <LayoutGrid class="w-3.5 h-3.5" />
            <span>{$LL.appearance.layoutTopbar()}</span>
          </button>
          <button
            type="button"
            class="flex items-center justify-center gap-2 py-2 px-3 rounded text-xs font-bold transition-all cursor-pointer {navigationStore.layout === 'sidebar' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
            onclick={() => navigationStore.setLayout('sidebar')}
          >
            <PanelLeft class="w-3.5 h-3.5" />
            <span>{$LL.appearance.layoutSidebar()}</span>
          </button>
        </div>
      </div>

      {#if !themeStore.isNeobrutalTheme()}
      <!-- Section 3: UI Paradigm Style (Modern vs Glass) -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
          <Layers class="w-3.5 h-3.5 text-accent" />
          {$LL.appearance.uiStyle()}
        </span>
        <div class="grid grid-cols-2 gap-2 bg-card-sub p-1 border border-border rounded-lg">
          <button
            type="button"
            class="flex flex-col items-center justify-center py-2 px-3 rounded text-xs font-bold transition-all cursor-pointer {themeStore.currentStyle === 'modern' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
            onclick={() => selectStyle('modern')}
          >
            <span>{$LL.appearance.styleModern()}</span>
            <span class="text-[10px] font-normal opacity-80">{$LL.appearance.styleModernDesc()}</span>
          </button>
          <button
            type="button"
            class="flex flex-col items-center justify-center py-2 px-3 rounded text-xs font-bold transition-all cursor-pointer {themeStore.currentStyle === 'glass' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
            onclick={() => selectStyle('glass')}
          >
            <span>{$LL.appearance.styleGlass()}</span>
            <span class="text-[10px] font-normal opacity-80">{$LL.appearance.styleGlassDesc()}</span>
          </button>
        </div>
      </div>
      {:else}
      <div class="p-3 bg-card-sub border border-border rounded-lg text-[11px] text-muted">
        {$LL.appearance.neoLockedNote()}
      </div>
      {/if}

      <!-- Sakuga FX (all themes) -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
          <Sparkles class="w-3.5 h-3.5 text-accent" />
          {$LL.appearance.sakugaFx()}
        </span>
        <div class="flex items-center justify-between gap-3 p-2.5 bg-card-sub border border-border rounded-lg">
          <div class="min-w-0">
            <div class="text-xs font-bold text-foreground">{$LL.appearance.sakugaFxTitle()}</div>
            <p class="text-[10px] text-muted">{$LL.appearance.sakugaFxDesc()}</p>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={themeStore.sakugaFx}
            aria-label={$LL.appearance.sakugaFx()}
            class="shrink-0 w-11 h-6 rounded-full border-2 border-border transition-colors cursor-pointer relative {themeStore.sakugaFx ? 'bg-accent' : 'bg-card'}"
            onclick={() => themeStore.setSakugaFx(!themeStore.sakugaFx)}
          >
            <span class="absolute top-0.5 w-4 h-4 rounded-full bg-white border border-black/20 transition-all {themeStore.sakugaFx ? 'left-[22px]' : 'left-0.5'}"></span>
          </button>
        </div>
      </div>

      <!-- Advanced Tuning (collapsed by default) -->
      {#if themeStore.currentStyle === 'neobrutal' || themeStore.currentStyle === 'glass'}
      <button
        type="button"
        class="w-full flex items-center justify-between p-2.5 bg-card-sub border border-border rounded-lg text-xs font-bold text-foreground hover:border-accent transition-colors cursor-pointer"
        onclick={() => showAdvancedTuning = !showAdvancedTuning}
      >
        <span class="flex items-center gap-1.5">
          <Sliders class="w-3.5 h-3.5 text-accent" />
          {$LL.appearance.advancedTuning()}
        </span>
        <span class="text-muted">{showAdvancedTuning ? $LL.common.close() : $LL.common.open()}</span>
      </button>
      {/if}

      <!-- Section 2.4: Neobrutalism Extended Tuning -->
      {#if showAdvancedTuning && themeStore.currentStyle === 'neobrutal'}
        <div class="p-3.5 bg-card-sub border border-border rounded-xl space-y-3.5">
          <div class="flex items-center justify-between border-b border-border/50 pb-2">
            <span class="text-xs font-bold uppercase text-foreground flex items-center gap-1.5">
              <Sliders class="w-3.5 h-3.5 text-accent" />
              {$LL.appearance.neoTuning()}
            </span>
            <button
              type="button"
              class="inline-flex items-center gap-1 text-[10px] font-bold py-1 px-2 rounded bg-card hover:bg-card-sub text-muted hover:text-foreground border border-border transition-colors cursor-pointer"
              title={$LL.appearance.resetNeoTitle()}
              onclick={() => {
                themeStore.resetNeobrutalSettings()
                toastStore.success($LL.appearance.neoResetToast())
              }}
            >
              <RotateCcw class="w-2.5 h-2.5" />
              <span>{$LL.appearance.resetDefault()}</span>
            </button>
          </div>

          <!-- Sliders Grid (4 sliders) -->
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <!-- Shadow Depth -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">{$LL.appearance.shadow()}</span>
                <span class="text-accent font-mono font-bold">{themeStore.neoShadowSize}px</span>
              </div>
              {#if themeStore.useSakugaControls()}
                <SakugaSlider value={themeStore.neoShadowSize} min={0} max={8} step={1} oninput={(e) => themeStore.setNeoShadowSize(Number((e.target as HTMLInputElement).value))} />
              {:else}
              <input
                type="range"
                min="0"
                max="8"
                step="1"
                value={themeStore.neoShadowSize}
                oninput={(e) => themeStore.setNeoShadowSize(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
              {/if}
            </div>

            <!-- Border Thickness -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">{$LL.appearance.border()}</span>
                <span class="text-accent font-mono font-bold">{themeStore.neoBorderWidth}px</span>
              </div>
              {#if themeStore.useSakugaControls()}
                <SakugaSlider value={themeStore.neoBorderWidth} min={1} max={4} step={1} oninput={(e) => themeStore.setNeoBorderWidth(Number((e.target as HTMLInputElement).value))} />
              {:else}
              <input
                type="range"
                min="1"
                max="4"
                step="1"
                value={themeStore.neoBorderWidth}
                oninput={(e) => themeStore.setNeoBorderWidth(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
              {/if}
            </div>

            <!-- Corner Radius -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">{$LL.appearance.radius()}</span>
                <span class="text-accent font-mono font-bold">{themeStore.neoRadius}px</span>
              </div>
              {#if themeStore.useSakugaControls()}
                <SakugaSlider value={themeStore.neoRadius} min={0} max={14} step={1} oninput={(e) => themeStore.setNeoRadius(Number((e.target as HTMLInputElement).value))} />
              {:else}
              <input
                type="range"
                min="0"
                max="14"
                step="1"
                value={themeStore.neoRadius}
                oninput={(e) => themeStore.setNeoRadius(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
              {/if}
            </div>

            <!-- Press Depth -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">{$LL.appearance.clickPop()}</span>
                <span class="text-accent font-mono font-bold">{themeStore.neoPressDepth}px</span>
              </div>
              {#if themeStore.useSakugaControls()}
                <SakugaSlider value={themeStore.neoPressDepth} min={1} max={4} step={1} oninput={(e) => themeStore.setNeoPressDepth(Number((e.target as HTMLInputElement).value))} />
              {:else}
              <input
                type="range"
                min="1"
                max="4"
                step="1"
                value={themeStore.neoPressDepth}
                oninput={(e) => themeStore.setNeoPressDepth(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
              {/if}
            </div>
          </div>

          <!-- Pattern Texture Options -->
          <div class="space-y-2 pt-1 border-t border-border/50">
            <div class="flex items-center justify-between">
              <span class="text-[11px] font-bold uppercase text-muted">{$LL.appearance.patternTexture()}</span>
              <div class="flex items-center gap-2">
                <span class="text-[10px] text-muted">{$LL.appearance.opacity()}</span>
                <span class="text-[10px] font-mono text-accent font-bold">{Math.round(themeStore.neoPatternOpacity * 100)}%</span>
                {#if themeStore.useSakugaControls()}
                  <SakugaSlider value={themeStore.neoPatternOpacity} min={0} max={0.12} step={0.01} oninput={(e) => themeStore.setNeoPatternOpacity(Number((e.target as HTMLInputElement).value))} />
                {:else}
                <input
                  type="range"
                  min="0"
                  max="0.12"
                  step="0.01"
                  value={themeStore.neoPatternOpacity}
                  oninput={(e) => themeStore.setNeoPatternOpacity(Number((e.target as HTMLInputElement).value))}
                  class="w-20 accent-accent cursor-pointer"
                />
                {/if}
              </div>
            </div>

            <div class="grid grid-cols-3 gap-2">
              <button
                type="button"
                class="py-1.5 px-2 rounded-lg border text-xs font-bold transition-all cursor-pointer {themeStore.neoPattern === 'dots' ? 'bg-accent text-accent-text border-accent shadow-neobrutal-sm' : 'bg-card text-muted hover:text-foreground border-border'}"
                onclick={() => themeStore.setNeoPattern('dots')}
              >
                {$LL.appearance.patternDots()}
              </button>
              <button
                type="button"
                class="py-1.5 px-2 rounded-lg border text-xs font-bold transition-all cursor-pointer {themeStore.neoPattern === 'grid' ? 'bg-accent text-accent-text border-accent shadow-neobrutal-sm' : 'bg-card text-muted hover:text-foreground border-border'}"
                onclick={() => themeStore.setNeoPattern('grid')}
              >
                {$LL.appearance.patternGrid()}
              </button>
              <button
                type="button"
                class="py-1.5 px-2 rounded-lg border text-xs font-bold transition-all cursor-pointer {themeStore.neoPattern === 'none' ? 'bg-accent text-accent-text border-accent shadow-neobrutal-sm' : 'bg-card text-muted hover:text-foreground border-border'}"
                onclick={() => themeStore.setNeoPattern('none')}
              >
                {$LL.appearance.patternPlain()}
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
              {$LL.appearance.glassTuning()}
            </span>
            <span class="text-[10px] text-muted">{$LL.appearance.perDevice()}</span>
          </div>

          <!-- Sliders Grid -->
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <!-- Blur Slider -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">{$LL.appearance.blurAmount()}</span>
                <span class="text-accent font-mono font-bold">{themeStore.glassBlur}px</span>
              </div>
              {#if themeStore.useSakugaControls()}
                <SakugaSlider value={themeStore.glassBlur} min={4} max={32} step={2} oninput={(e) => themeStore.setGlassBlur(Number((e.target as HTMLInputElement).value))} />
              {:else}
              <input
                type="range"
                min="4"
                max="32"
                step="2"
                value={themeStore.glassBlur}
                oninput={(e) => themeStore.setGlassBlur(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
              {/if}
            </div>

            <!-- Opacity Slider -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">{$LL.appearance.cardOpacity()}</span>
                <span class="text-accent font-mono font-bold">{Math.round(themeStore.glassOpacity * 100)}%</span>
              </div>
              {#if themeStore.useSakugaControls()}
                <SakugaSlider value={themeStore.glassOpacity} min={0.2} max={0.95} step={0.05} oninput={(e) => themeStore.setGlassOpacity(Number((e.target as HTMLInputElement).value))} />
              {:else}
              <input
                type="range"
                min="0.2"
                max="0.95"
                step="0.05"
                value={themeStore.glassOpacity}
                oninput={(e) => themeStore.setGlassOpacity(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
              {/if}
            </div>

            <!-- Dim Slider -->
            <div class="space-y-1">
              <div class="flex justify-between text-[10px]">
                <span class="text-muted font-bold">{$LL.appearance.wallpaperDim()}</span>
                <span class="text-accent font-mono font-bold">{Math.round(themeStore.glassDim * 100)}%</span>
              </div>
              {#if themeStore.useSakugaControls()}
                <SakugaSlider value={themeStore.glassDim} min={0.2} max={1} step={0.05} oninput={(e) => themeStore.setGlassDim(Number((e.target as HTMLInputElement).value))} />
              {:else}
              <input
                type="range"
                min="0.2"
                max="1.0"
                step="0.05"
                value={themeStore.glassDim}
                oninput={(e) => themeStore.setGlassDim(Number((e.target as HTMLInputElement).value))}
                class="w-full accent-accent cursor-pointer"
              />
              {/if}
            </div>
          </div>

          <!-- Wallpaper Gallery & Presets -->
          <div class="space-y-2 pt-1 border-t border-border/50">
            <div class="flex items-center justify-between">
              <span class="text-[11px] font-bold uppercase text-muted flex items-center gap-1.5">
                <Image class="w-3.5 h-3.5 text-accent" />
                {$LL.appearance.activeWallpaper()}
              </span>

              <!-- Upload Button -->
              <label class="cursor-pointer inline-flex items-center gap-1.5 text-[10px] font-bold py-1 px-2.5 rounded bg-accent/20 hover:bg-accent/30 text-accent border border-accent/40 transition-colors">
                {#if isUploading}
                  <Loader2 class="w-3 h-3 animate-spin" />
                  <span>{$LL.common.uploading()}</span>
                {:else}
                  <Upload class="w-3 h-3" />
                  <span>{$LL.appearance.uploadNew()}</span>
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
                    title={$LL.appearance.deleteWallpaperTitle()}
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
          {$LL.appearance.loginWallpaper()}
        </span>
        <span class="text-muted">{showLoginWallpaper ? $LL.common.close() : $LL.common.open()}</span>
      </button>
      {#if showLoginWallpaper}
        <!-- Section 5: Login Page Wallpaper (Static & Independent) -->
        <div class="space-y-3 pt-3 border-t border-border">
          <div class="flex items-center justify-between">
            <div>
              <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
                <Lock class="w-3.5 h-3.5 text-accent" />
                {$LL.appearance.loginWallpaperTitle()}
              </span>
              <p class="text-[10px] text-muted">
                {$LL.appearance.loginWallpaperDesc()}
              </p>
            </div>
            {#if themeStore.loginWallpaper && themeStore.loginWallpaper !== 'preset:none'}
              <button
                type="button"
                onclick={() => themeStore.setLoginWallpaper('preset:none')}
                class="text-[10px] font-bold text-red-400 hover:text-red-300 underline cursor-pointer"
              >
                {$LL.appearance.resetToPlain()}
              </button>
            {/if}
          </div>

          <!-- Preset / Uploaded Wallpaper Gallery Grid -->
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-[10px] text-muted font-bold uppercase">{$LL.appearance.chooseLoginBg()}</span>
              <label class="cursor-pointer inline-flex items-center gap-1.5 text-[10px] font-bold py-1 px-2.5 rounded bg-accent/20 hover:bg-accent/30 text-accent border border-accent/40 transition-colors">
                {#if isUploading}
                  <Loader2 class="w-3 h-3 animate-spin" />
                  <span>{$LL.common.uploading()}</span>
                {:else}
                  <Upload class="w-3 h-3" />
                  <span>{$LL.appearance.uploadWallpaper()}</span>
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
                    title={$LL.appearance.deleteWallpaperTitle()}
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
                  <span class="text-muted font-bold">{$LL.appearance.dimming()}</span>
                  <span class="font-bold text-accent">{Math.round(themeStore.loginWallpaperDim * 100)}%</span>
                </div>
                {#if themeStore.useSakugaControls()}
                  <SakugaSlider value={themeStore.loginWallpaperDim} min={0} max={0.8} step={0.05} oninput={(e) => themeStore.setLoginWallpaperDim(Number((e.target as HTMLInputElement).value))} />
                {:else}
                <input
                  type="range"
                  min="0"
                  max="0.8"
                  step="0.05"
                  value={themeStore.loginWallpaperDim}
                  oninput={(e) => themeStore.setLoginWallpaperDim(Number((e.target as HTMLInputElement).value))}
                  class="w-full accent-accent cursor-pointer"
                />
                {/if}
              </div>

              <!-- Blur -->
              <div class="space-y-1">
                <div class="flex items-center justify-between text-[11px] font-mono">
                  <span class="text-muted font-bold">{$LL.appearance.backgroundBlur()}</span>
                  <span class="font-bold text-accent">{themeStore.loginWallpaperBlur}px</span>
                </div>
                {#if themeStore.useSakugaControls()}
                  <SakugaSlider value={themeStore.loginWallpaperBlur} min={0} max={16} step={1} oninput={(e) => themeStore.setLoginWallpaperBlur(Number((e.target as HTMLInputElement).value))} />
                {:else}
                <input
                  type="range"
                  min="0"
                  max="16"
                  step="1"
                  value={themeStore.loginWallpaperBlur}
                  oninput={(e) => themeStore.setLoginWallpaperBlur(Number((e.target as HTMLInputElement).value))}
                  class="w-full accent-accent cursor-pointer"
                />
                {/if}
              </div>
            </div>
          {/if}
        </div>
      {/if}

      <!-- Footer: Summary and Done Button -->
      <div class="flex items-center justify-between pt-3 border-t border-border">
        <span class="text-[11px] text-muted">
          {$LL.appearance.activeLabel()} <strong class="text-foreground">{themes.find((t) => t.id === themeStore.currentTheme)?.name}</strong> ({styleName})
        </span>
        {#if themeStore.useSakugaControls()}
          <SakugaButton onclick={handleClose}>{$LL.common.doneApply()}</SakugaButton>
        {:else}
          <Button variant="primary" size="sm" onclick={handleClose}>
            <Check class="w-3.5 h-3.5 mr-1" />
            <span>{$LL.common.doneApply()}</span>
          </Button>
        {/if}
      </div>
    </div>
  </div>
{/if}
