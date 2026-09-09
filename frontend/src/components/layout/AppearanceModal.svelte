<script lang="ts">
  import {
    Palette,
    Sparkles,
    Check,
    X,
    Layers,
    Cpu,
    BatteryCharging,
    Sliders,
    LayoutGrid,
    PanelLeft,
  } from '@lucide/svelte'
  import { themeStore } from '../../stores/theme.svelte'
  import {
    navigationStore,
    NAV_LAYOUT_REGISTRY,
    type NavLayoutId,
  } from '../../stores/navigation.svelte'
  import type { ThemeMode, UIStyle } from '../../types/common'

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
      id: 'amoled',
      name: 'Pure AMOLED',
      tagline: 'True #000000 with charcoal elevation',
      bg: '#000000',
      card: '#0a0a0a',
      border: '#262626',
      accent: '#10b981',
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
      id: 'dracula',
      name: 'Dracula',
      tagline: 'Classic vampire purple & pink accents',
      bg: '#1e1f29',
      card: '#282a36',
      border: '#44475a',
      accent: '#bd93f9',
      text: '#f8f8f2',
    },
    {
      id: 'nord',
      name: 'Nordic Frost',
      tagline: 'Arctic blue calm muted tones',
      bg: '#242933',
      card: '#2e3440',
      border: '#434c5e',
      accent: '#88c0d0',
      text: '#eceff4',
    },
    {
      id: 'cyberpunk',
      name: 'Cyberpunk 2077',
      tagline: 'High-voltage neon yellow on dark grid',
      bg: '#0d0f18',
      card: '#181b28',
      border: '#facc15',
      accent: '#facc15',
      text: '#fff066',
    },
    {
      id: 'emerald',
      name: 'Matrix Emerald',
      tagline: 'Deep sovereign terminal greens',
      bg: '#042f2e',
      card: '#064e3b',
      border: '#115e59',
      accent: '#10b981',
      text: '#ecfdf5',
    },
    {
      id: 'sunset',
      name: 'Retro Sunset',
      tagline: 'Synthwave violet & neon orange',
      bg: '#120b18',
      card: '#1f1329',
      border: '#4c1d95',
      accent: '#f97316',
      text: '#fed7aa',
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
      class="neo-card relative w-full max-w-2xl bg-card border-2 border-border p-5 sm:p-6 z-10 space-y-6 max-h-[92vh] overflow-y-auto my-auto shadow-2xl text-foreground font-mono"
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-border pb-4">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded bg-accent/15 text-accent border border-accent/30 flex items-center justify-center">
            <Palette class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-sm sm:text-base font-bold uppercase tracking-wider text-foreground">
              Appearance Studio
            </h2>
            <p class="text-xs text-muted">Theme palette & UI paradigm configuration</p>
          </div>
        </div>

        <button
          type="button"
          class="p-1.5 rounded border border-border text-muted hover:text-foreground hover:bg-card-sub cursor-pointer transition-colors"
          onclick={handleClose}
          aria-label="Close modal"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Section 1: Navigation Style Selector -->
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <span class="text-xs font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
            <LayoutGrid class="w-3.5 h-3.5 text-accent" />
            1. Navigation Layout
          </span>
          <span class="text-[11px] text-muted">Choose desktop header flyout or left sidebar rail</span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <!-- Option 1: Classic Top Bar -->
          <button
            type="button"
            class="text-left p-4 rounded border-2 transition-all cursor-pointer relative flex flex-col justify-between {navigationStore.layout === 'topbar' ? 'border-accent bg-card-sub shadow-[4px_4px_0px_0px_var(--neo-accent)]' : 'border-border bg-card hover:border-accent/50'}"
            onclick={() => navigationStore.setLayout('topbar')}
          >
            <div>
              <div class="flex items-center justify-between mb-1.5">
                <div class="flex items-center gap-2">
                  <LayoutGrid class="w-4 h-4 text-accent" />
                  <span class="text-xs font-bold uppercase text-foreground">Classic Top Bar</span>
                </div>
                {#if navigationStore.layout === 'topbar'}
                  <span class="w-5 h-5 rounded-full bg-accent text-accent-text flex items-center justify-center text-xs font-black">
                    <Check class="w-3 h-3" />
                  </span>
                {/if}
              </div>
              <p class="text-[11px] text-muted leading-relaxed">
                Desktop header flyout dropdowns with 100% full-width cards; mobile bottom popovers.
              </p>
            </div>

            <div class="mt-3 pt-2 border-t border-border flex items-center justify-between">
              <span class="text-[9px] uppercase font-bold text-muted">Layout Mode</span>
              <span class="text-[10px] font-bold {navigationStore.layout === 'topbar' ? 'text-accent' : 'text-muted'}">FULL WIDTH</span>
            </div>
          </button>

          <!-- Option 2: Modern Sidebar -->
          <button
            type="button"
            class="text-left p-4 rounded border-2 transition-all cursor-pointer relative flex flex-col justify-between {navigationStore.layout === 'sidebar' ? 'border-accent bg-card-sub shadow-[4px_4px_0px_0px_var(--neo-accent)]' : 'border-border bg-card hover:border-accent/50'}"
            onclick={() => navigationStore.setLayout('sidebar')}
          >
            <div>
              <div class="flex items-center justify-between mb-1.5">
                <div class="flex items-center gap-2">
                  <PanelLeft class="w-4 h-4 text-accent" />
                  <span class="text-xs font-bold uppercase text-foreground">Modern Sidebar</span>
                </div>
                {#if navigationStore.layout === 'sidebar'}
                  <span class="w-5 h-5 rounded-full bg-accent text-accent-text flex items-center justify-center text-xs font-black">
                    <Check class="w-3 h-3" />
                  </span>
                {/if}
              </div>
              <p class="text-[11px] text-muted leading-relaxed">
                Desktop collapsible accordion sidebar rail; mobile clean slide-over drawer.
              </p>
            </div>

            <div class="mt-3 pt-2 border-t border-border flex items-center justify-between">
              <span class="text-[9px] uppercase font-bold text-muted">Layout Mode</span>
              <span class="text-[10px] font-bold {navigationStore.layout === 'sidebar' ? 'text-accent' : 'text-muted'}">RAIL ACCORDION</span>
            </div>
          </button>
        </div>
      </div>

      <!-- Section 2: UI Paradigm (Neobrutal vs Modern) -->
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <span class="text-xs font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
            <Layers class="w-3.5 h-3.5 text-accent" />
            2. UI Paradigm Style
          </span>
          <span class="text-[11px] text-muted">Select visual grammar</span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <!-- Option 1: Neo-Brutalist -->
          <button
            type="button"
            class="text-left p-4 rounded border-2 transition-all cursor-pointer relative flex flex-col justify-between {themeStore.currentStyle === 'neobrutal' ? 'border-accent bg-card-sub shadow-[4px_4px_0px_0px_var(--neo-accent)]' : 'border-border bg-card hover:border-accent/50'}"
            onclick={() => selectStyle('neobrutal')}
          >
            <div>
              <div class="flex items-center justify-between mb-1.5">
                <span class="text-xs font-bold uppercase text-foreground">Neo-Brutalist</span>
                {#if themeStore.currentStyle === 'neobrutal'}
                  <span class="w-5 h-5 rounded-full bg-accent text-accent-text flex items-center justify-center text-xs font-black">
                    <Check class="w-3 h-3" />
                  </span>
                {/if}
              </div>
              <p class="text-[11px] text-muted leading-relaxed">
                Crisp 2px borders, hard mechanical drop-shadows, 4px corners, and subtle background dot-grid matrix.
              </p>
            </div>
            <div class="mt-3 pt-2 border-t border-border/60 flex items-center gap-1.5 text-[10px] font-bold text-accent uppercase">
              <span>● Solid Elevation</span>
              <span>• 4px Corners</span>
            </div>
          </button>

          <!-- Option 2: Modern Clean -->
          <button
            type="button"
            class="text-left p-4 rounded-xl border transition-all cursor-pointer relative flex flex-col justify-between {themeStore.currentStyle === 'modern' ? 'border-accent bg-card-sub shadow-lg ring-1 ring-accent' : 'border-border bg-card hover:border-accent/50'}"
            onclick={() => selectStyle('modern')}
          >
            <div>
              <div class="flex items-center justify-between mb-1.5">
                <span class="text-xs font-bold uppercase text-foreground">Modern Clean</span>
                {#if themeStore.currentStyle === 'modern'}
                  <span class="w-5 h-5 rounded-full bg-accent text-accent-text flex items-center justify-center text-xs font-black">
                    <Check class="w-3 h-3" />
                  </span>
                {/if}
              </div>
              <p class="text-[11px] text-muted leading-relaxed">
                Refined 1px borders, smooth ambient drop-shadows, generous 16px corner curves, and backdrop blur.
              </p>
            </div>
            <div class="mt-3 pt-2 border-t border-border/60 flex items-center gap-1.5 text-[10px] font-bold text-accent uppercase">
              <span>● Soft Ambient</span>
              <span>• 16px Curves</span>
            </div>
          </button>
        </div>
      </div>

      <!-- Section 3: Color Themes (8 Palettes) -->
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <span class="text-xs font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
            <Sliders class="w-3.5 h-3.5 text-accent" />
            3. Color Palettes (8 Themes)
          </span>
          <span class="text-[11px] text-muted">Real-time dynamic CSS tokens</span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
          {#each themes as t}
            {@const isSelected = themeStore.currentTheme === t.id}
            <button
              type="button"
              class="text-left p-3 rounded border transition-all cursor-pointer flex items-center justify-between gap-3 {isSelected ? 'border-accent bg-card-sub shadow-sm' : 'border-border bg-card hover:border-accent/40'}"
              onclick={() => selectTheme(t.id)}
            >
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="text-xs font-bold text-foreground truncate">{t.name}</span>
                  {#if isSelected}
                    <span class="px-1.5 py-0.2 rounded text-[9px] font-black uppercase tracking-wider bg-accent text-accent-text">
                      Active
                    </span>
                  {/if}
                </div>
                <p class="text-[10px] text-muted truncate mt-0.5">{t.tagline}</p>
              </div>

              <!-- Color Swatch Dots -->
              <div class="flex items-center gap-1 shrink-0 p-1 rounded bg-black/30 border border-white/5">
                <span class="w-3.5 h-3.5 rounded-full border border-black/30 shadow-inner" style="background-color: {t.bg};" title="Background"></span>
                <span class="w-3.5 h-3.5 rounded-full border border-black/30 shadow-inner" style="background-color: {t.card};" title="Card"></span>
                <span class="w-3.5 h-3.5 rounded-full border border-black/30 shadow-inner" style="background-color: {t.border};" title="Border"></span>
                <span class="w-3.5 h-3.5 rounded-full border border-black/30 shadow-inner" style="background-color: {t.accent};" title="Accent"></span>
              </div>
            </button>
          {/each}
        </div>
      </div>

      <!-- Section 3: Live Component Sandbox Preview -->
      <div class="space-y-2 pt-1 border-t border-border">
        <div class="flex items-center justify-between text-[11px] text-muted font-bold uppercase tracking-wider">
          <span class="flex items-center gap-1.5">
            <Sparkles class="w-3.5 h-3.5 text-accent" />
            Live Preview Sandbox
          </span>
          <span class="text-[10px] text-accent font-mono">
            {themeStore.currentStyle.toUpperCase()} / {themeStore.currentTheme.toUpperCase()}
          </span>
        </div>

        <div class="p-3.5 rounded bg-card-sub border border-border space-y-3">
          <!-- Mini Mock Card -->
          <div class="neo-card bg-card p-3 space-y-2.5">
            <div class="flex items-center justify-between border-b border-border pb-2">
              <div class="flex items-center gap-2">
                <Cpu class="w-4 h-4 text-accent" />
                <span class="text-xs font-bold text-foreground uppercase tracking-wider">Hardware Telemetry</span>
              </div>
              <div class="flex items-center gap-1.5 text-[10px] px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-bold">
                <BatteryCharging class="w-3 h-3" />
                <span>ONLINE</span>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-2 text-xs">
              <div class="p-2 rounded bg-card-sub border border-border">
                <span class="text-[9px] text-muted uppercase font-bold block">Kernel CPU</span>
                <span class="text-xs font-black text-foreground">2.84 GHz (8 Cores)</span>
              </div>
              <div class="p-2 rounded bg-card-sub border border-border">
                <span class="text-[9px] text-muted uppercase font-bold block">Memory RAM</span>
                <span class="text-xs font-black text-foreground">4.2 / 8.0 GB</span>
              </div>
            </div>

            <!-- Action Buttons Sample -->
            <div class="flex items-center gap-2 pt-1">
              <button
                type="button"
                class="neo-button px-3 py-1.5 text-xs bg-accent text-accent-text cursor-pointer"
              >
                Execute Action
              </button>
              <button
                type="button"
                class="neo-button px-3 py-1.5 text-xs bg-card text-foreground cursor-pointer"
              >
                Reset Settings
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Footer Actions -->
      <div class="border-t border-border pt-4 flex items-center justify-end gap-2">
        <button
          type="button"
          class="neo-button px-4 py-2 text-xs bg-accent text-accent-text cursor-pointer font-bold uppercase tracking-wider"
          onclick={handleClose}
        >
          Done & Apply
        </button>
      </div>
    </div>
  </div>
{/if}
