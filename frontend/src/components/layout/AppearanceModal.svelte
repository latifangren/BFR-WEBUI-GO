<script lang="ts">
  import {
    Palette,
    Sparkles,
    Check,
    X,
    Layers,
    LayoutGrid,
    PanelLeft,
  } from '@lucide/svelte'
  import { themeStore } from '../../stores/theme.svelte'
  import {
    navigationStore,
  } from '../../stores/navigation.svelte'
  import type { ThemeMode, UIStyle } from '../../types/common'
  import Button from '../ui/Button.svelte'

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
    {
      id: 'retro',
      name: 'Retro Deck',
      tagline: 'Pastel ice canvas & tactile black borders',
      bg: '#daf0fc',
      card: '#ffffff',
      border: '#000000',
      accent: '#c8f5d0',
      text: '#111827',
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

      <!-- Section 1: Navigation Layout (Top Bar vs Sidebar) -->
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

      <!-- Section 2: UI Paradigm Style (Neobrutal vs Modern) -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
          <Layers class="w-3.5 h-3.5 text-accent" />
          UI Paradigm Style
        </span>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 bg-card-sub p-1 border border-border rounded-lg">
          <button
            type="button"
            class="flex flex-col items-center justify-center py-2 px-3 rounded text-xs font-bold transition-all cursor-pointer {themeStore.currentStyle === 'neobrutal' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
            onclick={() => selectStyle('neobrutal')}
          >
            <span>Neobrutalist</span>
            <span class="text-[10px] font-normal opacity-80">2px borders, hard shadows</span>
          </button>
          <button
            type="button"
            class="flex flex-col items-center justify-center py-2 px-3 rounded text-xs font-bold transition-all cursor-pointer {themeStore.currentStyle === 'modern' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
            onclick={() => selectStyle('modern')}
          >
            <span>Modern Clean</span>
            <span class="text-[10px] font-normal opacity-80">1px borders, smooth curves</span>
          </button>
        </div>
      </div>

      <!-- Section 3: Color Themes (9 Palettes) -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-bold uppercase text-muted tracking-wider flex items-center gap-1.5">
          <Sparkles class="w-3.5 h-3.5 text-accent" />
          Color Themes (9 Palettes)
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

      <!-- Footer: Summary and Done Button -->
      <div class="flex items-center justify-between pt-3 border-t border-border">
        <span class="text-[11px] text-muted">
          Active: <strong class="text-foreground">{themes.find(t => t.id === themeStore.currentTheme)?.name}</strong> ({themeStore.currentStyle})
        </span>
        <Button variant="primary" size="sm" onclick={handleClose}>
          <Check class="w-3.5 h-3.5 mr-1" />
          <span>Done & Apply</span>
        </Button>
      </div>
    </div>
  </div>
{/if}
