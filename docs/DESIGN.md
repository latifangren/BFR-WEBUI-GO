# BFR-WEBUI-GO - UI/UX Design System Specification

This document provides a comprehensive, exact specification of the UI/UX architecture, visual design paradigms, color theme presets, layout structures, component anatomy, typography, and navigation governance used in **BFR-WEBUI-GO**.

> **Note**: This specification covers **UI/UX visual design and layout architecture only**, excluding backend functionality and business logic.

---

## 1. Design Architecture Overview

`BFR-WEBUI-GO` features a **100% offline-ready, dual-paradigm modular UI system**. The styling engine is built on top of precompiled **Tailwind CSS**, enhanced with custom CSS variable tokens, Alpine.js dynamic state binding, and custom CSS presentation layers.

### Key Visual Principles
- **Dual Design Paradigms**: Instant switching between **⚡ Neobrutalism** (retro industrial, bold offset shadows) and **✨ Modern Clean** (glassmorphic, rounded-xl corners, smooth drop shadows).
- **Preset Color Theme Engine**: 7 color palettes bound dynamically to `[data-theme="..."]` CSS variables on `<html>`.
- **Anti-FOUC (Flash of Unstyled Content) Engine**: Inline `<head>` script reading `localStorage` state before DOM rendering.
- **Adaptive Layout**: Desktop category dropdowns (`z-1050`) and mobile 5-column bottom navigation bar with a slide-up bottom sheet sub-menu (`z-999`).

```
web/static/css/
├── base.css            # Base tokens, theme variables, resets, badges, z-index rules
├── styles/
│   ├── neobrutal.css   # Neobrutal design paradigm overrides (2px border, 4px shadow)
│   └── modern.css      # Modern Clean design paradigm overrides (rounded-2xl, glassmorphism)
├── style.css           # Consolidator importing base.css + paradigm styles
├── tailwind.min.css    # Precompiled Tailwind CSS utility classes
└── xterm.css           # Terminal component styling
```

---

## 2. UI Design Paradigms (Styles)

The application supports two distinct presentation paradigms controlled by the `data-ui-style` attribute on `<html>`.

### ⚡ Neobrutalism Style (`[data-ui-style="neobrutal"]`)
Emphasizes high contrast, sharp angles, retro-industrial aesthetics, and physical offset shadows.

| Component | Design Attributes |
|---|---|
| **Background Grid** | Radial dot grid overlay (`background-image: radial-gradient(var(--neo-border) 1px, transparent 1px); background-size: 24px 24px; opacity: 0.03`) |
| **Cards (`.bg-card`)** | `background: var(--neo-card)`, `border: 2px solid var(--neo-border)`, `box-shadow: 4px 4px 0px 0px var(--neo-shadow)`, `border-radius: 4px` |
| **Modal Cards** | `box-shadow: 8px 8px 0px 0px var(--neo-shadow)` |
| **Sub-Containers (`.bg-black`)** | `background: var(--neo-card-sub)`, `border: 2px solid var(--neo-border)`, `border-radius: 4px` |
| **Buttons (`button`, `.btn`)** | `border: 2px solid var(--neo-border)`, `border-radius: 4px`, `box-shadow: 4px 4px 0px 0px var(--neo-shadow)`, `font-weight: 700`, `text-transform: uppercase`, `letter-spacing: 0.05em` |
| **Button States** | **Hover**: `transform: translate(-1px, -1px)`, `box-shadow: 5px 5px 0px 0px var(--neo-shadow)`<br>**Active**: `transform: translate(2px, 2px)`, `box-shadow: 2px 2px 0px 0px var(--neo-shadow)` |
| **Form Controls** | `border: 2px solid var(--neo-border)`, `border-radius: 4px`, `box-shadow: 2px 2px 0px 0px var(--neo-shadow)`<br>**Focus**: `border-color: var(--neo-accent)`, `box-shadow: 3px 3px 0px 0px var(--neo-accent)` |

### ✨ Modern Clean Style (`[data-ui-style="modern"]`)
Emphasizes smooth curves, glassmorphism backdrop blurs, soft multi-layered drop shadows, and subtle border lines.

| Component | Design Attributes |
|---|---|
| **Cards (`.bg-card`)** | `background: var(--neo-card)`, `border: 1px solid var(--neo-border)`, `box-shadow: 0 10px 30px -5px rgba(0, 0, 0, 0.12), 0 4px 10px -2px rgba(0, 0, 0, 0.05)`, `border-radius: 16px` (`rounded-2xl`), `backdrop-filter: blur(12px)` |
| **Modal Cards** | `box-shadow: 0 20px 50px -10px rgba(0, 0, 0, 0.4)` |
| **Sub-Containers (`.bg-black`)** | `background: var(--neo-card-sub)`, `border: 1px solid var(--neo-border)`, `border-radius: 12px` (`rounded-xl`) |
| **Buttons (`button`, `.btn`)** | `border: 1px solid var(--neo-border)`, `border-radius: 12px`, `box-shadow: 0 2px 6px rgba(0,0,0,0.06)`, `font-weight: 600` |
| **Button States** | **Hover**: `transform: translateY(-1px)`, `box-shadow: 0 4px 12px rgba(0,0,0,0.1)`<br>**Active**: `transform: translateY(0) scale(0.98)` |
| **Form Controls** | `border: 1px solid var(--neo-border)`, `border-radius: 12px`<br>**Focus**: `border-color: var(--neo-accent)`, `box-shadow: 0 0 0 3px rgba(124, 58, 237, 0.2)` |
| **Table Formatting** | Headers: `border-bottom: 1px solid var(--neo-border)`, `color: var(--neo-muted)`. Cells: `border-bottom: 1px solid rgba(255, 255, 255, 0.05)` |

---

## 3. Color Theme Presets

All color themes drive standard CSS custom property variables defined in `base.css`:

```css
--neo-bg        /* Page body background color */
--neo-card      /* Primary card surface background */
--neo-card-sub  /* Secondary container / input background */
--neo-text      /* Primary body text color */
--neo-muted     /* Secondary / muted text color */
--neo-border    /* Component border color */
--neo-accent    /* Active focus / highlight accent color */
--neo-shadow    /* Neobrutalist hard shadow color */
```

### Complete Color Palette Reference

| Theme Preset | `--neo-bg` | `--neo-card` | `--neo-card-sub` | `--neo-text` | `--neo-muted` | `--neo-border` | `--neo-accent` | `--neo-shadow` |
|---|---|---|---|---|---|---|---|---|
| **🌙 Dark** *(Default)* | `#090d16` | `#131927` | `#1e293b` | `#f8fafc` | `#94a3b8` | `#334155` | `#7c3aed` | `#1e293b` |
| **☀️ Light** | `#f4f6fa` | `#ffffff` | `#f8fafc` | `#0f172a` | `#64748b` | `#cbd5e1` | `#2563eb` | `#0f172a` |
| **🧛 Dracula** | `#1e1f29` | `#282a36` | `#44475a` | `#f8f8f2` | `#6272a4` | `#44475a` | `#bd93f9` | `#191a21` |
| **❄️ Nord** | `#242933` | `#2e3440` | `#3b4252` | `#eceff4` | `#d8dee9` | `#434c5e` | `#88c0d0` | `#1b1f27` |
| **🤖 Cyberpunk** | `#0d0f18` | `#181b28` | `#222638` | `#fff066` | `#a3a7c2` | `#facc15` | `#facc15` | `#facc15` |
| **🌲 Emerald** | `#042f2e` | `#064e3b` | `#0f766e` | `#ecfdf5` | `#6ee7b7` | `#115e59` | `#10b981` | `#022c22` |
| **🌅 Sunset** | `#120b18` | `#1f1329` | `#2d1b3a` | `#fed7aa` | `#fb923c` | `#4c1d95` | `#f97316` | `#0a050e` |

---

## 4. Typography & Font Hierarchy

- **Primary Sans Font**: System UI Stack / Inter (`font-sans`) — used for card titles, buttons, navigation links, and body copy.
- **Monospace Font**: JetBrains Mono / Consolas / Fira Code / monospace fallback (`font-mono`) — used for code snippets, IP addresses, status gauges, system metrics, file paths, terminals, and logs.
- **Font Weights**:
  - `font-normal` (400): Standard body text.
  - `font-medium` (500): Secondary button text, tab items.
  - `font-semibold` (600): Navigation headers, input labels.
  - `font-bold` (700): Card headings, action buttons, table th.
  - `font-extrabold` (800): Main section titles, brand headers.
  - `font-black` (900): Badges, PRO pill, highlight numbers.
- **Text Case Governance**: Uppercase + letter spacing (`uppercase tracking-wider` or `uppercase tracking-wide`) is strictly applied to badge labels, category titles, section headers, and primary action buttons.

---

## 5. Navigation & Layout Systems

### Desktop Navigation Structure (`md:flex`)
- **Header Bar (`header`)**: Sticky top bar (`sticky top-0 z-[1000]`), translucent background with backdrop blur (`bg-card/80 backdrop-blur`), max-width 7xl container (`max-w-7xl mx-auto`).
- **Brand Block**:
  - Logo Box: `w-9 h-9 rounded-xl bg-blue-600` containing letter `B`.
  - Title: `BFR WEBUI` + **PRO Badge** (`bg-[#f59e0b] text-[#000000] border-[1.5px] border-black text-[10px] font-black uppercase tracking-wider`).
- **Category Navigation Dropdowns (`nav`)**:
  Pill wrapper (`bg-black/90 p-1.5 rounded-xl border border-border/80 shadow-lg`) containing 5 primary category dropdown triggers:
  1. **📊 Status**: Overview, System Info, System Logs.
  2. **⚙️ System**: File Manager, Local NAS & Media, Web Terminal, System Tools & Modules.
  3. **🛠️ Services**: Telegram Bot, Smart Charger, SSH Daemon, Remote Screen (Scrcpy).
  4. **🌐 Network**: Network & Tweaks, QoS Control, Modem & Band Lock, Remote Access Tunnel, Speedtest Engine, Proxy Core.
  5. **⭐ Extras**: SMS Viewer, About & Docs.

### Mobile Navigation Structure (`md:hidden`)
- **Fixed 5-Column Bottom Bar**:
  - Fixed at screen bottom (`fixed bottom-0 left-0 right-0 z-50 bg-card/85 backdrop-blur-xl border-t border-border/60`).
  - Grid columns: `📊 Status`, `⚙️ System`, `🛠️ Services`, `🌐 Network`, `⭐ Extras`.
  - Includes a top-right minimize/hide pill button (`absolute -top-3.5 right-3 bg-card border border-border/80 text-[9px] font-bold`).
- **Slide-Up Bottom Sheet Modal**:
  - Triggered when tapping a mobile category icon (`fixed inset-0 z-[999] bg-black/70 backdrop-blur-md flex items-end`).
  - Container: Rounded-top card (`rounded-2xl mb-16 mx-3 bg-card/85 backdrop-blur-2xl border border-border/80`).
  - Includes top centered drag handle pill (`w-12 h-1 bg-gray-600/50 rounded-full`).

---

## 6. Component Anatomy & UI Patterns

### Cards (`.bg-card`)
The fundamental surface container for content blocks.
- **Card Header**: Icon box (`w-10 h-10 rounded-xl bg-accent/10 border-2 border-accent/30 flex items-center justify-center`), section title, and optional status badge.
- **Card Sub-box (`.bg-black`)**: Nested data box for secondary metrics, code outputs, or input forms.

### Badges & Pill Indicators
- **PRO Badge**: Gold accent (`bg-[#f59e0b] text-[#000000] border-[1.5px] border-black shadow-[1px_1px_0px_0px_rgba(0,0,0,1)]`).
- **Active / Enabled Badge**: `bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 font-bold text-[10px]`.
- **Warning / Pending Badge**: `bg-amber-500/10 text-amber-400 border border-amber-500/30 font-bold text-[10px]`.
- **Inactive / Disabled Badge**: `bg-gray-500/10 text-gray-400 border border-gray-500/30 font-bold text-[10px]`.
- **Error / Failed Badge**: `bg-red-500/10 text-red-400 border border-red-500/30 font-bold text-[10px]`.

### Toast Notification System (`toasts`)
- **Container**: `fixed top-4 right-4 z-50 flex flex-col gap-2.5 max-w-sm w-full pointer-events-none p-4`.
- **Item**: Card with 2px status border (`border-green-500` for success, `border-red-500` for error, `border-amber-500` for info), hard shadow (`shadow-[4px_4px_0px_0px_var(--neo-shadow)]`), status icon badge, uppercase title, message text, and dismissal button (`✕`).

### Appearance Modal Dialog (`#showAppearanceModal`)
- Triggered by the **🎨 Appearance** button in the Header (`z-[120]`).
- Features real-time selection between:
  1. **UI Paradigm & Style**: Neobrutalism vs Modern Clean with instant preview cards.
  2. **Color Theme Presets**: Interactive grid of all 7 color themes with dual color swatch previews.

### Settings & Backup Modal Dialog (`#showBackupModal`)
- Triggered by the **⚙️** button in the Header (`z-[200]`).
- Features a prominent 3-Tab sub-navigation bar:
  1. **📦 Backup**: Export/Import configuration JSON bundle.
  2. **🔐 Security**: Password update form with strength status pills.
  3. **☁️ Cloud**: WebDAV cloud auto-sync configuration.

---

## 7. Z-Index & Stacking Governance

To prevent z-index collision and clipping across dropdowns and fixed overlays:

| Element | Z-Index Value | Class / Target |
|---|---|---|
| **Utility Custom Z-Index** | `z-60` to `z-200` | `.z-60` ... `.z-200`, `.z-999` |
| **Mobile Bottom Bar** | `z-50` | `fixed bottom-0` |
| **Toast Notifications Container** | `z-50` | `fixed top-4 right-4` |
| **Header Bar** | `z-[1000]` *(forced)* | `header` |
| **Desktop Nav Dropdown Wrapper** | `z-[1050]` *(forced)* | `nav div.absolute` |
| **Nav Dropdown Inner Menu** | `z-[110]` | `.nav-dropdown` |
| **Appearance Customization Modal** | `z-[120]` | `x-show="showAppearanceModal"` |
| **Settings & Backup Modal** | `z-[200]` | `x-show="showBackupModal"` |
| **Mobile Bottom Sheet Sub-Menu** | `z-[999]` | `x-show="mobileCategory !== null"` |

---

## 8. Anti-FOUC & State Persistence

Theme state is persisted in browser `localStorage`:
- `colorTheme`: Saves active color preset (`dark`, `light`, `dracula`, `nord`, `cyberpunk`, `emerald`, `sunset`).
- `uiStyle`: Saves active visual paradigm (`neobrutal`, `modern`).

The `<head>` anti-FOUC script executes synchronously before body parsing:
```html
<script>
    const colorTheme = localStorage.getItem('colorTheme') || (localStorage.getItem('theme') === 'light' ? 'light' : 'dark');
    document.documentElement.setAttribute('data-theme', colorTheme);
    if (colorTheme === 'light') {
        document.documentElement.classList.remove('dark');
        document.documentElement.classList.add('light');
    } else {
        document.documentElement.classList.add('dark');
        document.documentElement.classList.remove('light');
    }
    const uiStyle = localStorage.getItem('uiStyle') || 'neobrutal';
    document.documentElement.setAttribute('data-ui-style', uiStyle);
</script>
```

---
*End of UI/UX Design Specification.*
