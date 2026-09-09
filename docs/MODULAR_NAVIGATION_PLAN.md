# Modular Navigation Architecture & Extensible Theming Engine

This document specifies the technical design, component architecture, and implementation plan for introducing a **Modular Navigation System** while preserving the refined card layout, visual responsiveness, and vibrant AMOLED/Light theme tokens of `BFR-WEBUI-GO`.

---

## 1. Context & Architectural Directive

During review and hands-on PC demonstration, the following user feedback was established:
1. **Preserve Current Aesthetics:** The modern card styling, bento grid layout, and color palettes developed in Svelte 5 are significantly superior, fluid, and less rigid than the legacy WebUI. These must be **100% preserved**.
2. **Navigation Modularization:** The user requested support for multiple navigation layout styles:
   - **Gaya 1 (Classic WebUI `topbar`):** Desktop top header category pills with hover flyout submenus + Mobile fixed bottom bar with upward popover cards.
   - **Gaya 2 (Modern Dashboard `sidebar`):** Desktop collapsible accordion sidebar + Mobile left slide-over drawer (without duplicate bottom bar).
3. **Future Extensibility:** The system must be truly modular and extensible (data-driven registries) so additional navigation layouts, UI paradigms, and color palettes can be introduced without modifying core business logic.

---

## 2. The Tri-Engine Customization Architecture

The visual and navigational experience is decoupled into three orthogonal, composable engines:

```text
┌────────────────────────────────────────────────────────────────────────────────┐
│                         BFR-WEBUI TRI-ENGINE MATRIX                            │
│                                                                                │
│  1. Navigation Layout Engine    2. UI Style Paradigm      3. Color Palette     │
│     [data-nav-layout]              [data-ui-style]           [data-theme]      │
│  ├── "topbar"   (Classic Web)   ├── "neobrutal" (Bold)   ├── "dark"            │
│  ├── "sidebar"  (Modern Dash)   ├── "modern"    (Clean)  ├── "light"           │
│  └── (Extensible: "dock", etc)  └── (Extensible)         ├── "amoled"          │
│                                                          ├── "dracula"         │
│                                                          ├── "nord"            │
│                                                          ├── "cyberpunk"       │
│                                                          ├── "emerald"         │
│                                                          └── "sunset"          │
└────────────────────────────────────────────────────────────────────────────────┘
```

Each engine operates independently: any Navigation Layout can be paired with any UI Style Paradigm and any Color Palette.

---

## 3. Navigation Layout Specification

### 3.1 Mode A: Top Bar Navigation (`topbar`) — Classic WebUI Feel
Designed for users who want maximum screen space for data and monitoring telemetry.

#### Desktop View (≥ 768px)
- **Zero Horizontal Sidebar:** Eliminates the left 240px sidebar. The content area expands to **100% full width**, allowing the Bento Grid in Overview and Sysinfo to breathe.
- **Header Category Dropdowns:**
  Five grouped category pills centered or aligned in the top header:
  1. `📊 Status` → Submenu: Overview, Sysinfo, Logs, About
  2. `⚙️ System` → Submenu: Power, Charger, Modules, Terminal, SSH, Files
  3. `🛠️ Services` → Submenu: Scrcpy, NAS, Telegram, Tools
  4. `🌐 Network` → Submenu: Network, Hotspot, Proxy, Modem, SMS, QoS, Vnstat, Tunnel
  5. `⚡ Speedtest` → Direct action or extras
- **Hover & Click Behavior:**
  - Hovering over a category pill displays a floating dropdown menu with a smooth 150ms transition.
  - Clicking locks the menu open; clicking outside dismisses it.
  - Active tab displays a high-contrast accent highlight and indicator dot.

#### Mobile View (< 768px)
- **Single Navigation Mechanism:** Fixed 5-column bottom navigation bar (`fixed bottom-0 inset-x-0 z-50 md:hidden`).
- **Upward Popover Cards:**
  - Tapping a category (e.g., `⚙️ System`) opens a compact, floating popover card positioned directly above the clicked button.
  - The popover displays clean sub-menu buttons with icons. Tapping an item navigates immediately and closes the popover.
- **Clean Header:** Hamburger menu button is **automatically hidden** in `topbar` mode on mobile to avoid duplicate controls.

---

### 3.2 Mode B: Collapsible Sidebar Navigation (`sidebar`) — Modern Dashboard Feel
Designed for power users managing dense server operations who prefer direct sidebar access.

#### Desktop View (≥ 768px)
- **Left Vertical Sidebar (240px):** Fixed or floating rail on the left.
- **Collapsible Accordion Groups:**
  - Categories (`Core Dashboard`, `Network & Connectivity`, `Hardware & System`, `Tools & Utilities`) can be expanded or collapsed via chevron toggles.
  - Category collapse states persist in `localStorage`.
  - **Active State Indicator:** If a category is collapsed but contains the currently active tab, an accent glow or dot indicates that the active tab resides within that group.
- **No Infinite Scrolling:** Users can collapse unused categories (e.g., collapse Tools and Services) to view only the tabs they actively monitor.

#### Mobile View (< 768px)
- **Single Navigation Mechanism:** The hamburger icon in the top header opens a smooth slide-over **Left Drawer**.
- **No Duplicate Bottom Bar:** In `sidebar` mode on mobile, the bottom navigation bar is **completely unmounted/disabled**.
- The slide-over drawer contains the full categorized list and an instant search filter.

---

## 4. Future-Proof Registry Design (Data-Driven)

To prevent code refactoring when adding new options in the future, all options are defined as extensible TypeScript registries:

### 4.1 Navigation Layout Registry (`types/navigation.ts`)
```ts
export type NavLayoutId = 'topbar' | 'sidebar'

export interface NavLayoutOption {
  id: NavLayoutId
  name: string
  description: string
  icon: string
  previewType: 'horizontal' | 'vertical'
}

export const NAV_LAYOUT_REGISTRY: NavLayoutOption[] = [
  {
    id: 'topbar',
    name: 'Classic Top Bar',
    description: 'Header dropdowns on desktop with 100% full-width content, upward popovers on mobile',
    icon: 'PanelTop',
    previewType: 'horizontal',
  },
  {
    id: 'sidebar',
    name: 'Modern Sidebar',
    description: 'Collapsible accordion sidebar on desktop, left slide drawer on mobile',
    icon: 'PanelLeft',
    previewType: 'vertical',
  },
]
```

### 4.2 Category & Tab Structure (`stores/navigation.svelte.ts`)
```ts
export interface NavSubItem {
  id: string
  label: string
  icon: string
}

export interface NavCategory {
  id: string
  label: string
  icon: string
  tabs: NavSubItem[]
}

export const NAV_CATEGORIES: NavCategory[] = [
  {
    id: 'status',
    label: 'Status',
    icon: 'Activity',
    tabs: [
      { id: 'overview', label: 'Overview', icon: 'LayoutDashboard' },
      { id: 'sysinfo', label: 'Sysinfo', icon: 'Cpu' },
      { id: 'logs', label: 'Logs', icon: 'FileText' },
      { id: 'about', label: 'About', icon: 'Info' },
    ],
  },
  {
    id: 'system',
    label: 'System',
    icon: 'Settings',
    tabs: [
      { id: 'power', label: 'Power', icon: 'Power' },
      { id: 'charger', label: 'Charger', icon: 'BatteryCharging' },
      { id: 'modules', label: 'Modules', icon: 'Box' },
      { id: 'terminal', label: 'Terminal', icon: 'Terminal' },
      { id: 'ssh', label: 'SSH', icon: 'KeyRound' },
      { id: 'files', label: 'Files', icon: 'FolderOpen' },
    ],
  },
  {
    id: 'network',
    label: 'Network',
    icon: 'Globe',
    tabs: [
      { id: 'network', label: 'Network', icon: 'Network' },
      { id: 'hotspot', label: 'Hotspot', icon: 'Wifi' },
      { id: 'proxy', label: 'Proxy', icon: 'Shield' },
      { id: 'modem', label: 'Modem', icon: 'Radio' },
      { id: 'sms', label: 'SMS', icon: 'MessageSquare' },
      { id: 'qos', label: 'QoS', icon: 'Gauge' },
      { id: 'vnstat', label: 'Vnstat', icon: 'Activity' },
      { id: 'tunnel', label: 'Tunnel', icon: 'Globe' },
    ],
  },
  {
    id: 'services',
    label: 'Services',
    icon: 'Wrench',
    tabs: [
      { id: 'scrcpy', label: 'Scrcpy', icon: 'Smartphone' },
      { id: 'nas', label: 'NAS', icon: 'HardDrive' },
      { id: 'speedtest', label: 'Speedtest', icon: 'Zap' },
      { id: 'tools', label: 'Tools', icon: 'Wrench' },
      { id: 'telegram', label: 'Telegram', icon: 'Send' },
    ],
  },
]
```

---

## 5. UI Implementation Architecture

```text
frontend/src/
├── stores/
│   └── navigation.svelte.ts     # Holds layout mode ('topbar' | 'sidebar') + activeTab
├── components/
│   └── layout/
│       ├── Header.svelte         # Top bar with Logo, Quick Metrics, Appearance Button
│       ├── TopNav.svelte         # Desktop Top Category Pills with hover flyouts (for 'topbar' mode)
│       ├── BottomPopoverNav.svelte # Mobile 5-column bar with upward popovers (for 'topbar' mode)
│       ├── Sidebar.svelte        # Desktop Collapsible Accordion Sidebar (for 'sidebar' mode)
│       ├── MobileDrawer.svelte   # Mobile Left Slide-over Drawer (for 'sidebar' mode)
│       └── AppearanceModal.svelte # 3D Studio (Nav Layout, UI Style, Color Theme)
```

### 5.1 Conditional Shell Rendering in `App.svelte`
```svelte
{#if navigationStore.layout === 'sidebar'}
  <!-- Mode 1: Sidebar Layout -->
  <div class="flex h-screen overflow-hidden">
    <Sidebar />
    <div class="flex-1 flex flex-col overflow-y-auto">
      <Header showHamburger={true} />
      <main class="flex-1 p-4 md:p-6">
        <!-- Active Tab Content -->
      </main>
    </div>
  </div>
{:else}
  <!-- Mode 2: Full-Width TopBar Layout -->
  <div class="min-h-screen flex flex-col">
    <Header showTopNav={true} showHamburger={false} />
    <main class="flex-1 max-w-7xl w-full mx-auto p-4 md:p-6 pb-20 md:pb-6">
      <!-- Active Tab Content (Full Width) -->
    </main>
    <BottomPopoverNav />
  </div>
{/if}
```

### 5.2 Dedicated Hero Login Stage (`LoginCard.svelte`)

Based on visual comparison with the legacy WebUI (`loginpagelama.png`), the authentication interface must be upgraded from a generic popup modal to a **Dedicated Center Stage Login Experience**:

1. **Top Header with Appearance Studio:**
   - The top header remains visible even before authentication, featuring the `BFR WEBUI [PRO]` logo and the **`🎨 Appearance`** studio button.
   - Users can customize their Theme Palette (`amoled`, `dracula`, `cyberpunk`, `nord`, etc.) and UI Style Paradigm (`neobrutal` vs `modern`) **prior to logging in**.
2. **Hero Identity & Branding Card:**
   - Centered glowing blue squircle avatar with 'B' monogram.
   - Heading **BFR WebUI [PRO]** with amber PRO pill badge and subtitle `Android System WebUI`.
   - Card styling automatically reflects the active UI style (2px borders & hard offset shadow in `neobrutal`, 16px soft blur & diffused glow in `modern`).
3. **Smart Quick Access Auto-Fill:**
   - Prominent 1-click pill button: `Quick Access: [Default: bfr (Tap to auto-fill)]`.
   - Tapping auto-fills the default password `bfr` without requiring virtual keyboard typing on touchscreens.
4. **Password Visibility Toggle:**
   - Integrated eye icon inside the password input to toggle between masked (`password`) and plain text (`text`).
5. **Connect & Community Footer:**
   - Section divider labeled `CONNECT & COMMUNITY`.
   - 3 Social/Support pills:
     - ✈️ **Telegram**
     - 📘 **Facebook**
     - 🐙 **GitHub**

---

## 6. Phased Execution Roadmap

### Phase 1: Navigation Store & Layout Engine
- [x] Add `layout: 'topbar' | 'sidebar'` to `navigationStore` with `localStorage.getItem('navLayout')` persistence.
- [x] Implement `setNavLayout(layout: NavLayoutId)`.
- [x] Export `NAV_LAYOUT_REGISTRY` and `NAV_CATEGORIES`.

### Phase 2: Refactor `Sidebar.svelte` (Collapsible Accordion)
- [x] Update `Sidebar.svelte` to implement accordion collapse toggles for each category.
- [x] Save accordion expanded states to `localStorage`.
- [x] Add category active dot indicators when an accordion group is collapsed.
- [x] Ensure mobile drawer is cleanly isolated without duplicate bottom bars.

### Phase 3: Implement `TopNav.svelte` & `BottomPopoverNav.svelte`
- [x] Create `TopNav.svelte`: Desktop header category pills with hover flyout submenus and smooth enter/leave transitions.
- [x] Create `BottomPopoverNav.svelte`: Mobile 5-column bottom bar with upward popovers for touch devices.
- [x] Ensure submenus have high-contrast borders and elevation shadows matching the active UI style (`neobrutal` vs `modern`).

### Phase 4: Dedicated Hero Login Stage (`LoginCard.svelte`)
- [x] Implement `frontend/src/components/layout/LoginCard.svelte` replacing the blocking generic modal.
- [x] Add top header minimal view with `🎨 Appearance` studio button accessible prior to login.
- [x] Implement `Quick Access: Default: bfr (Tap to auto-fill)` button.
- [x] Add password visibility toggle (eye icon) and `CONNECT & COMMUNITY` links (Telegram, Facebook, GitHub).
- [x] Wire login transition cleanly to unmount `LoginCard` and reveal the active dashboard shell.

### Phase 5: Dynamic Shell Orchestration (`App.svelte` & `Header.svelte`)
- [x] Wire `App.svelte` to switch between full-width topbar shell and sidebar rail shell based on `navigationStore.layout`.
- [x] Ensure mobile viewport never renders both hamburger and bottom bar simultaneously.
- [x] Preserve full card layouts, bento grids, and typography untouched.

### Phase 6: 3D Appearance Studio (`AppearanceModal.svelte`) & Verification
- [x] Add **Navigation Style Selector** as Section 1 in `AppearanceModal.svelte`:
  - Cards for *Classic Top Bar* and *Modern Sidebar* with visual preview badges.
- [x] Section 2: **UI Style Paradigm** (*Neobrutalism* vs *Modern Clean*).
- [x] Section 3: **Color Palette Presets** (8 palettes with swatches).
- [x] Verification: `pnpm run check`, Vite build, and Android cross-compile.

---

## 7. Verification Criteria

1. **Card & Color Preservation:** All 23 tabs retain their current polished card structures, bento grids, and color theme tokens.
2. **Topbar Mode Desktop:** Content takes 100% full width without left sidebar; hovering top categories opens responsive flyout dropdowns.
3. **Topbar Mode Mobile:** Bottom bar displays 5 categories; tapping a category smoothly displays an upward popover card with sub-items. Header has no hamburger icon.
4. **Sidebar Mode Desktop:** Left sidebar groups tabs into collapsible accordions; no vertical clutter.
5. **Sidebar Mode Mobile:** Only hamburger slide drawer is active; no bottom bar rendered.
6. **Hero Login Stage:** Dedicated centered card with B monogram, amber PRO badge, Quick Access auto-fill button, password eye toggle, community links, and pre-login Appearance Studio access.
7. **Persistence:** `navLayout`, `uiStyle`, and `colorTheme` persist across browser reloads.
