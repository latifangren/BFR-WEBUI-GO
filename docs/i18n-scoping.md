# BFR-WEBUI-GO — i18n Scoping Doc (typesafe-i18n)

Status: **DRAFT for review** — the user first chose **Option B (svelte-i18n)** on 2026-11-21,
then asked whether the plan is the best, safest, lowest-risk, and future-proof one.
**It is not — see §0.** The recommendation below is revised on measured evidence.
Branch: `feat/i18n` (from `dev` @ `7fa5940`, post-merge PR #6). Nothing in this doc is
implemented yet; no code has been changed. This file lives outside the repo until the
user approves committing it to `docs/` on `feat/i18n`.

## 0. Reconsideration (2026-11-21) — why the recommendation changed

The original choice (svelte-i18n) was picked for "standard + features + lazy loading".
Re-examined against the user's own criteria (best, safe, minimal risk, future-proof),
it fails on the criterion that matters most for a 3,676-string extraction. Evidence,
all measured on 2026-11-21 (not assumed):

| Finding | Evidence | Consequence |
|---|---|---|
| **svelte-i18n is stale** | Last release **4.0.1, 2024-10-21** — 2+ years, no release since | Works today with Svelte 5, but if Svelte 6 / a Vite major breaks it, there is no maintainer to fix it. This is the single biggest future-proof hole. |
| **Bundle fear was overblown** | Probe build (same code + minimal svelte-i18n wiring): main chunk 409.48 kB → 426.49 kB raw, **gzip 98.81 kB → 104.41 kB (+5.6 kB)**; dist 1,664,069 → 1,726,564 B (+3.8%) | Size does **not** decide between libraries — both are small. The "tens of KB" estimate in §5 was conservative; the measured delta is +5.6 kB gzip. |
| **ICU plurals are barely used** | 384 interpolated nodes audited: dominant pattern is simple value interpolation (`{tab.label}`, `{freeRamText}`, ternaries). **Zero ICU plural constructs exist** in the codebase; counts render as plain numbers (`${n} items`, even `1 items`). | intl-messageformat — B's heavy dependency — buys almost nothing here. Simple `{var}` interpolation is all this UI needs, and any formatter does that. (Indonesian barely inflects for plural anyway.) |
| **The real long-term risk is key drift** | 3,676 candidates across 93 files. svelte-i18n keys are **untyped strings**: `$_('typo.key')` compiles fine and renders the raw key at runtime. | With this many strings, typos and en/id drift are a matter of time, and B detects neither. This — not bundle size — is the risk to engineer against. |
| **typesafe-i18n is active + tiny + typed** | 5.27.1, last release **2026-02-11**; **zero runtime dependencies** (peer: TypeScript only); ESM runtime + Svelte adapter **21,061 B raw** (minified/gzip far less); keys are **type-checked** (`LL.key()` — a typo is a TS error) and locale parity is enforced at type level (an `id` file missing a key does not compile). | Directly solves the dominant risk, is actively maintained, and has no dependency that can go stale underneath it. Framework-agnostic core (resilient to Svelte majors). |
| **Paraglide is best tech, highest churn** | @inlang/paraglide-js 3.2.2, last release **2026-11-10** (very active); compiler-based, type-safe, tiny runtime. | Requires a Vite plugin + inlang message format + generated files, and inlang has a history of breaking changes across majors. For a single maintainer wanting minimal risk, setup/churn cost outweighs the edge over typesafe-i18n. |
| **Custom (A) re-ranked** | Plurals barely needed ⇒ A's "no formatter" weakness mostly disappears. | A is zero-dep and can never go stale — the "most control" option — but it shares B's untyped-keys weakness and the user owns ~150 lines of plumbing forever. |

### Revised recommendation: **typesafe-i18n (Option C)**

- **Library:** typesafe-i18n 5.27.1, `bun add -d typesafe-i18n` (its runtime ships with the
  package; the generator is dev-time). Locale source is TypeScript
  (`src/i18n/en/index.ts`, `src/i18n/id/index.ts`), generator outputs types +
  the `LL` Svelte store + formatters.
- **Cost:** a codegen step — one command (`npx typesafe-i18n`), wired into
  `bun run check` so it can never be forgotten. API differs from B: `LL.key()` /
  `LL.key({ value })` instead of `$_('key')`. Locale files are TS modules, not JSON;
  lazy loading works via dynamic import of locale modules.
- **Why it wins on the user's criteria:** active maintenance (2026) + zero runtime
  deps + type-checked keys and locale parity + tiny runtime + no reliance on a
  Svelte-version-specific binding. The one thing it does not do is "zero setup" —
  that is B's only remaining advantage.
- **Option B stays acceptable** as the fallback: lowest friction, standard API,
  works now, measured cost only +5.6 kB gzip. If the user prefers "no codegen" over
  type safety, B with the §0.1 hardenings is still a sound choice — just not the
  most future-proof one.

### 0.1 Hardenings that apply whichever library wins

- **Never block mount on locale load.** The plan's `await waitLocale()` (B) — or
  `loadLocaleAsync` (C) — must be wrapped so a failed locale chunk still mounts the
  app in English (try/catch + fallback). A failed chunk must never produce a blank
  screen.
- **Bundle the default locale statically.** The English strings are already in the
  bundle today as hardcoded text — moving them into a locale file adds ~0 net size.
  Ship `en` as a static import and lazy-load only `id`. This removes the async
  dependency from the default path entirely. *Probe correction (2026-11-21): the
  generated API's natural pattern is lazy-both (`loadLocaleAsync` for `en` and
  `id`); static-`en` needs manual seeding of the exported `loadedLocales` record
  and is an **optional refinement**, not the default path — a local chunk fetch is
  ~ms, so measure first-paint timing before adding the complexity.*
- **Close the key-safety hole.** For B: a script that extracts every `$_('…')` key
  from source and verifies it exists in `en.json` (extends the §3.1 key check).
  For C: the compiler does this for free — no script needed.
- **Measure, don't assume.** §5 now carries measured numbers for B; the same probe
  build must be run for C in Phase 1 before the choice is locked in code.

## 1. Decision (revised — see §0)

| | |
|---|---|
| Library | **typesafe-i18n 5.27.1** (revised recommendation; svelte-i18n 4.0.1 remains the fallback) |
| Source locale | **English** — the e6f266c sweep made English the clean source; all keys are authored in English first |
| First target locale | **Indonesian (`id`)** |
| Fallback | Missing `id` keys are a **compile error** (type-level locale parity), so there is no runtime fallback gap to manage; the generated code still falls back to the base locale at runtime if a key is ever absent |
| Persistence | `localStorage`, key `locale` (mirrors the theme store's localStorage pattern) |

Compatibility: typesafe-i18n 5.27.1 ships a `svelte/` adapter in the current release
(2026, post-Svelte-5 era); Svelte stores work unchanged in Svelte 5 runes mode.
Definitive verification is a probe build in Phase 1 (§0.1). Install (Phase 1):
`bun add -d typesafe-i18n` (dev-time generator; the runtime ships inside the package
and has zero dependencies — peer is TypeScript only).

### Why C (recorded rationale)

The user asked for best + safest + lowest-risk + future-proof. On measured evidence
(§0) the deciding factor is not bundle size (all options are small) and not features
(the UI barely uses plurals) — it is **key safety across 3,676 strings** and
**maintenance health**. typesafe-i18n is the only option that is actively maintained,
has zero runtime dependencies, and makes a wrong or missing key a compile error.

## 2. Inventory (measured, heuristic)

Method: script scan of `frontend/src` (119 files: 91 `.svelte` + 28 `.ts`) counting
visible template text nodes and quoted string literals containing a space, with
`class="..."` / `style="..."` attributes stripped and Tailwind-class-like literals
excluded. This is a **heuristic, not an exact key count**: the true key count will be
lower after de-duplication (repeated labels such as "Save", "Cancel", "Loading…"
become one `common.*` key each), and some counted literals are not user-visible.

| Metric | Count |
|---|---|
| Files scanned | 119 (91 Svelte + 28 TS) |
| Files with ≥ 1 candidate | **93** (corrects the earlier "~60+" estimate) |
| Visible text nodes | 1,918 |
| Refined string literals | 1,758 |
| **Total candidates** | **3,676** |
| Interpolated text nodes (need `{values}`) | 384 |
| Files with 0 candidates | 20 (all `types/*.ts`, `registry.ts`, `vendor_tabs.ts`, `config/features.ts`, `main.ts`, `stores/toast.svelte.ts` — the toast *store* itself holds no strings; its call sites do) |

Distribution is extremely skewed: **tabs hold 3,416 of 3,676 candidates (93%)**;
layout 170, shared `ui` 39, stores 19, modals 17, `App.svelte` 9, `api`/`ws` 3 each.

Top files by candidates (nodes + literals):

| File | Candidates | Interpolated |
|---|---|---|
| `components/tabs/shannon/TabShannon.svelte` | 223 | 26 |
| `components/tabs/nas/TabNAS.svelte` | 157 | 11 |
| `components/tabs/shannon/ShannonCAEditor.svelte` | 148 | 18 |
| `components/tabs/overview/TabOverview.svelte` | 142 | 18 |
| `components/tabs/modem/TabModem.svelte` | 126 | 14 |
| `components/tabs/scrcpy/TabScrcpy.svelte` | 124 | 4 |
| `components/tabs/qualcomm/QualcommIMEI.svelte` | 108 | 8 |
| `components/tabs/samsung/TabSamsung.svelte` | 104 | 5 |
| `components/tabs/qualcomm/QualcommBands.svelte` | 103 | 12 |
| `components/tabs/charger/TabCharger.svelte` | 101 | 4 |
| `components/tabs/hotspot/subtabs/SubtabTetheringControl.svelte` | 93 | 11 |
| `components/tabs/qualcomm/QualcommCapability.svelte` | 91 | 3 |
| `components/tabs/qualcomm/QualcommEFS.svelte` | 90 | 5 |
| `components/layout/AppearanceModal.svelte` | 89 | 3 |

## 3. Architecture

### 3.1 New files (typesafe-i18n layout — verified by probe, 2026-11-21)

- `frontend/src/i18n/en/index.ts` — base locale as a TypeScript object (the source
  of truth; supports comments and structure). `frontend/src/i18n/id/index.ts` —
  same shape; **the generator enforces at type level that `id` implements every
  `en` key, so a missing Indonesian translation is a compile error, not a runtime
  fallback gap.**
- Generated files (`frontend/src/i18n/i18n-types.ts`, `i18n-util.ts`,
  `i18n-util.async.ts`, `i18n-util.sync.ts`, `formatters.ts`) — produced by
  `npx typesafe-i18n`, committed to the repo so the build never depends on
  running the generator, and regenerated by `bun run check` (see §4) so they
  cannot go stale unnoticed. **Generator default mode is watch — it never exits.
  Always run with `--no-watch`** (in `bun run check` and manually), otherwise the
  command hangs after generating.
- `frontend/src/i18n/i18n-svelte.ts` — **hand-written, NOT generated** (the
  generator never creates it): three lines wiring the generated dictionaries
  into the Svelte store —
  ```ts
  import { initI18nSvelte } from 'typesafe-i18n/svelte'
  import { loadedFormatters, loadedLocales } from './i18n-util'
  export const { locale, LL, setLocale } = initI18nSvelte(loadedLocales, loadedFormatters)
  ```
- Loading: `loadLocaleAsync(locale)` from the generated `i18n-util.async.ts`
  (dynamic `import()` per locale → separate lazy chunks; probe confirmed
  `en-*.js` + `id-*.js` chunks are emitted). **Do not use the generated sync
  loader (`i18n-util.sync.ts`) for lazy loading — it statically imports ALL
  locales, which defeats code-splitting.** `en` as a static import is possible
  only via manual seeding of the exported `loadedLocales` record (§0.1).
- No `scripts/check-i18n-keys.mjs` is needed for C — the compiler enforces parity.
  (It would be needed for the B fallback: a script extracting every `$_('…')` key
  from source and verifying it exists in `en.json`.)

### 3.2 Boot sequence (avoids flash of untranslated content)

`frontend/src/main.ts` is currently 3 lines (`mount(App)`). Change it to (verified
in the probe — this exact code type-checked and built):

```ts
import { mount } from 'svelte'
import './styles/app.css'
import App from './App.svelte'
import { loadLocaleAsync } from './i18n/i18n-util.async' // generated
import { setLocale } from './i18n/i18n-svelte'           // hand-written, §3.1

// 'en' loads first (lazy chunk); a failed chunk still mounts the app in English.
await loadLocaleAsync('en').catch(() => {})
setLocale('en')
const stored = localStorage.getItem('locale')
if (stored === 'id') {
  await loadLocaleAsync('id').catch(() => {})
  setLocale('id')
}
mount(App, { target: document.getElementById('app')! })
```

Because init runs before mount, the login screen is translated too — no separate
login-screen handling is needed. A failed `id` chunk still mounts the app in
English (§0.1).

### 3.3 Usage pattern

- Templates: `import LL from './i18n/i18n-svelte'` (generated store), then
  `{$LL.tabs.modem.title()}`. `LL` is a Svelte store, so it works in Svelte 5
  runes mode; keys autocomplete in the editor and a typo is a TS error.
- TypeScript (toast messages, dynamic strings): `import { get } from 'svelte/store'`
  then `get(LL).toast.saved({ name })`.
- Interpolation: locale strings use `{name}` placeholders, called as
  `LL.greeting({ name: 'world' })`. Plurals use the library's built-in formatter
  (exact syntax to confirm against the generated `formatters.ts` in Phase 1) —
  but the audit (§0) shows the UI barely needs plurals: 384 interpolated nodes
  are overwhelmingly simple value interpolation, and counts render as plain
  numbers today.
- The toast store's default titles (`'Success'` / `'Error'` / `'Warning'` / `'Info'`,
  `stores/toast.svelte.ts:29-46`) become `toast.success` / `toast.error` / … keys.
  Toast call sites (~dozens across tabs) pass translated strings, not literals.

### 3.4 Key naming

Nested objects mirroring the UI structure; `common` for shared labels (de-dup):

```ts
// src/i18n/en/index.ts (sketch)
export default {
  common: { save: 'Save', cancel: 'Cancel', close: 'Close', loading: 'Loading…', retry: 'Retry', error: 'Error' },
  nav: { overview: 'Overview', modem: 'Modem', hotspot: 'Hotspot' },
  appearance: { colorThemes: 'Color Themes', accentColor: 'Accent Color', language: 'Language' },
  tabs: { modem: { title: 'Modem', signal: 'Signal' }, shannon: { title: 'Shannon' } },
  toast: { success: 'Success', error: 'Error', saved: 'Saved' },
  login: { title: 'Sign in', password: 'Password', signIn: 'Sign In' },
}
```
Accessed as `LL.common.save()`, `LL.tabs.modem.title()`, …

### 3.5 Language switcher placement (design decision for the user)

Proposed: a **"Language" section in `AppearanceModal`**, directly under Color
Themes (the modal already opens with themes first, so language sits next to the
other identity settings). A two-option segmented control (English / Bahasa Indonesia)
bound to the `locale` store; switching persists to localStorage and re-renders
immediately (the `LL` store is reactive — no reload needed). This is a control inside
an existing modal, not a new tab, so it does not trigger the repo's new-tab rule.
Alternative (not recommended): a switcher in the Header — more visible, but the
header is space-constrained on phones and the modal is where all display
preferences already live.

### 3.6 What stays untranslated (deliberate)

- Donation WhatsApp message template (`TabAbout.svelte:33`) — message content to
  an Indonesian developer, not UI chrome. (Same exception as the e6f266c sweep.)
- Technical identifiers: band names, modem codes, model numbers, `5G`, `QRTR`,
  file paths, log output, terminal text, SSIDs, IP/MAC addresses.
- Go backend strings: out of scope for this phase. The backend surface is small and
  mostly English already; API error strings are returned as-is for now.
- `TabQoS.svelte` (87 candidates): **excluded from extraction** — the file is
  unreachable (no `qos` tab in navigation; `#qos` redirects to Hotspot) and the
  `fix/audit-findings` branch deletes it. Do not spend keys on it; if the audit
  merge lands, it is gone.

### 3.7 No-modem seam

Locale modules are shared and contain keys for vendor tabs (Qualcomm/Samsung/Shannon)
— harmless in no-modem builds, where those components are never bundled and the
keys are simply unused. No change to `features.ts`, `vendor_tabs.ts`, or the
`//go:build` seams is needed. The locale chunks are part of the frontend bundle
embedded via `frontend/embed.go` in both flavors.

## 4. Phases

### Phase 1 — Setup (small, verifiable alone)

1. `bun add -d typesafe-i18n`; create `src/i18n/en/index.ts` + `src/i18n/id/index.ts`
   (seeded with `common`, `nav`, `appearance` only); run the generator with
   **`npx typesafe-i18n --no-watch`** (default mode watches and never exits —
   probe finding); wire the generator into `bun run check` (package.json
   `check` script runs `typesafe-i18n --no-watch` first, so generated types can
   never go stale). Probe build for C is **done** (numbers in §5: dist +15,402 B).
2. `main.ts`: lazy `en` + `id` load with try/catch fallback (§3.2, verified code).
3. Language section in `AppearanceModal` (English / Bahasa Indonesia).
4. Translate the shell only (App, Header, LoginCard, AppearanceModal, toast titles)
   so the switcher is demonstrably working end-to-end.
5. Verify: svelte-check 0 errors, `bun run build`, bundle-size delta (§5),
   Go `GOOS=android GOARCH=arm64` build default + `-tags nomodem` + `go vet`,
   E2E headless: switch language → login screen + overview render in Indonesian,
   reload → choice persists, missing key is a compile error (not a runtime fallback).
   **Done 2026-11-21 ~21:30 WIB, all green:** `bun run check` 0 errors (2
   pre-existing `svelte:component` warnings); `bun run build` dist 1,695,923 B
   (+31,854 B, §5); Go android/arm64 29,358,829 B + nomodem 28,174,488 B + vet OK;
   E2E with the real Go backend on :18080 (default password `bfr`)
   **24/24 passed, 0 page errors** — script
   `~/workspace/your_files/bfr-sakuga-e2e/i18n-phase1-test.js` (login EN → switch
   ID pre-auth → login → topbar + sidebar ID labels → reload persists → back to
   EN → reload persists). Two test-harness fixes, not product bugs: the modal
   wait used a comma-union `text=` selector (Playwright treats the comma as
   literal text), and the flyout click toggled shut what mouseenter had opened
   (hover instead).

### Phase 2 — Extraction, batch 1 (shell + shared)

`common` de-dup across all files, `LoginCard`, `ToastContainer`, shared `ui`
components (Button/Input/Slider labels, Modal, Confirm dialogs). Small files, high
reuse — this is where the key count collapses.

### Phase 3 — Extraction, batch 2 (tabs, biggest first)

Order by candidate count: Shannon (+CAEditor), NAS, Overview, Modem, Scrcpy,
Qualcomm family (IMEI, Bands, Capability, EFS, NV, Searcher), Samsung, Charger,
Hotspot (+subtabs), Power, Sysinfo, then the rest. One tab (or one tab family) per
commit so regressions are bisectable; `svelte-check` + build after each batch.

**DONE — all 27 tab families, 2026-11-21 → 2026-11-22, branch `feat/i18n`.**
Marathon mode (user order 2026-11-22 ~04:06 WIB): translate all remaining tabs
sequentially, commit + push directly per family, no per-commit approval round.
Each family was implemented by a subagent and verified independently by the
coordinator (diff, `bun run check`, build, Go android/arm64 + nomodem + vet,
E2E vs the real backend) before commit + push. The leftovers sweep after SSH
found 4 more live tabs not in the original list (vnstat, tunnel, tools,
telegram) — all translated. `TabQoS.svelte` and `BottomNav.svelte` are dead
code (unimported; also deleted by the parked `fix/audit-findings` branch) and
were deliberately NOT translated. Layout labels in Sidebar / BottomPopoverNav /
TopNav resolve via `tNav()` from the `nav` namespace (complete in both
locales), so the hardcoded strings there are fallback-only.

| Family | Commit | Keys | Dist (B) | E2E |
|---|---|---|---|---|
| Shannon | `29333c1` | 219 + 5 common | 1,723,215 | 8/8, 0 page errors |
| NAS | `a23b796` | 146 | 1,737,880 | 8/8, 0 page errors |
| Overview | `d4e352f` | 102 | 1,749,510 | 8/8, 0 page errors |
| Modem | `cfafa90` | 75 | 1,757,662 | 8/8, 0 page errors |
| Scrcpy | `d28b2fa` | 90 | 1,766,910 | 8/8, 0 page errors |
| Qualcomm | `838e86d` | 652 + 2 common | 1,841,658 | 8/8, 0 page errors |
| Samsung | `d6f54e5` | 90 | 1,852,542 | 8/8, 0 page errors |
| Charger | `a785501` | 105 | 1,863,851 | 8/8, 0 page errors |
| Hotspot | `f51ee7e` | 155 | 1,883,274 | 8/8, 0 page errors |
| Power | `b8427d0` | 65 | 1,890,531 | 8/8, 0 page errors |
| Sysinfo | `688b898` | 52 | 1,895,967 | 23/23, 0 page errors |
| About | `80f011d` | 63 | 1,904,770 | 12/12, 0 page errors |
| Files | `059d32c` | 158 | 1,921,571 | 13/13, 0 page errors |
| Kernel | `5d16286` | 45 | 1,927,152 | 15/15, 0 page errors |
| Logs | `c70dead` | 16 | 1,929,029 | 15/15, 0 page errors |
| Modules | `3aae60e` | 21 | 1,931,404 | 13/13, 0 page errors |
| Network | `5a8686f` | 40 | 1,935,755 | 13/13, 0 page errors |
| Partitions | `bdcc2f0` | 147 | 1,954,008 | 13/13, 0 page errors |
| Proxy | `e93f441` | 117 | 1,967,594 | 13/13, 0 page errors |
| SMS | `e867619` | 12 | 1,968,915 | 10/10, 0 page errors |
| Speedtest | `31eff56` | 25 | 1,971,701 | 13/13, 0 page errors |
| Terminal | `5c8571e` | 25 | 1,974,219 | 12/12, 0 page errors |
| SSH | `fe6d41c` | 29 | 1,977,517 | 13/13, 0 page errors |
| Vnstat | `5ec7f47` | 67 | 1,984,501 | 13/13, 0 page errors |
| Tunnel | `59197ba` | 36 | 1,988,511 | 13/13, 0 page errors |
| Tools | `530a189` | 65 | 1,995,625 | 13/13, 0 page errors |
| Telegram | `4cf6883` | 36 | 1,999,730 | 13/13, 0 page errors |

E2E scripts: `~/workspace/your_files/bfr-sakuga-e2e/i18n-phase3-<family>-test.js`
(each: login EN → assert EN strings → switch to ID via Appearance modal →
assert ID strings → reload persists → back to EN; 0 page errors). Coverage
limits (VM has no modem/device): modem-dependent tabs assert their
not-detected/offline states; SMS asserts the empty inbox; no power/daemon/
tunnel/bot action was ever executed. Notable fixes found by E2E during the
marathon (pre-existing, not i18n-caused): Charger `available_switches?.find`
null guard (`a785501`); the `common.deselectAll` promotion (Samsung →
`common`, used by FileTable + TabSamsung, `059d32c`).

### Phase 4 — Polish

- `id` completion: every `en` key has an Indonesian translation — enforced at
  compile time by the generator (a missing key does not compile), so no separate
  audit script is needed.
- E2E per-tab spot check (screenshots in both locales).
- Docs: README/CHANGELOG entry for the release.

**DONE 2026-11-22:**
- Final tree verified: `bun run check` 0 errors (2 pre-existing `svelte:component`
  warnings), `bun run build` OK, Go android/arm64 + nomodem + vet OK.
  **35 top-level namespaces, 2,856 leaf keys (en). Dist 2,000,811 B**
  (+336,742 B / +20.2% vs the 1,664,069 B no-i18n baseline; +304,888 B vs
  Phase 1's 1,695,923 B) — after the post-audit follow-up below (12 leftover
  strings keyed, 10 unused keys removed; 2,852 → 2,856 leaves, +1,081 B).
- Full E2E sweep: **all 29 scripts re-run once against the final tree —
  354 checks, all passing, 0 page errors** (Phase 1 24/24, Phase 2 20/20,
  Shannon/NAS/Overview/Modem/Scrcpy/Qualcomm/Samsung/Charger/Hotspot/Power
  8/8 each, Sysinfo 23/23, About 12/12, Files 13/13, Kernel 15/15, Logs
  15/15, Modules 13/13, Network 13/13, Partitions 13/13, Proxy 13/13, SMS
  10/10, Speedtest 13/13, Terminal 12/12, SSH 13/13, Vnstat 13/13, Tunnel
  13/13, Tools 13/13, Telegram 13/13).
- Docs: CHANGELOG [1.2.9] entry added (this section's §4 update is the
  scoping-doc record). Known out-of-scope items preserved as-is: backend
  error strings stay English; `ChangePasswordModal` info line still says
  the default password is `admin` (real default `bfr`); login error strings
  in `auth.svelte.ts` stay English.

## 5. Bundle-size budget (measured for B and C)

Probe builds, same code + minimal wiring, 2026-11-21 (scratch copies in `/tmp`,
not the repo). Dist totals are stable across runs; per-chunk numbers shift
slightly with Rolldown's chunk rebalancing, so totals are the comparison basis.

| | dist total | total JS raw | main chunk gzip |
|---|---|---|---|
| Baseline (no i18n) | 1,664,069 B | 1,390,382 B | 97,946 B |
| + svelte-i18n (B) | 1,726,564 B | — | 104,410 B |
| + typesafe-i18n (C) | **1,679,471 B** | **1,404,317 B** | 90,090 B * |
| **Delta C** | **+15,402 B (+0.9%)** | **+13,935 B** | * chunk rebalanced; totals are the signal |

\* C's main chunk is smaller than baseline because Rolldown moved shared code into
other chunks (tab chunks each grew ~200–500 B); the dist total still grew by
15,402 B, so there is no real saving — just a boundary shift. C's total impact is
~4× smaller than B's (+62,495 B, +3.8%). Two lazy locale chunks (`en-*.js`,
`id-*.js`, ~113 B each for the 3-key samples) are emitted as designed.

Note: `bun add svelte-i18n` reported one blocked postinstall (bun's
trusted-dependencies gate, almost certainly the `esbuild` binary install pulled in
by svelte-i18n's CLI deps); the build itself was unaffected. Those CLI deps
(`esbuild`, `sade`, `cli-color`, `tiny-glob`) sit in `node_modules` only — the
runtime never uses them. `bun add -d typesafe-i18n` reported no such block.

The WebUI is embedded in a ~10–12 MB Magisk-module binary, so a +15.4 kB delta is
negligible. No further size stop is needed for C; the Phase 1 probe step is done.

**Phase 1 measured (2026-11-21, full shell translation, real repo):** dist
1,695,923 B (**+31,854 B, +1.9%** vs baseline) — larger than the probe's +15,402 B
because Phase 1 adds the full dictionaries (~140 keys × 2 locales) plus LL
usages across 9 files; main chunk 382.30 kB (gzip 89.97 kB, chunk rebalanced —
totals are the signal). Still negligible against the ~29 MB Go binary.

## 6. Risks

| Risk | Mitigation |
|---|---|
| Bundle grows more than expected | Measure in Phase 1 for C (§5); hard stop if delta is large |
| Flash of English / blank screen on first paint | Static `en` + try/catch around the async `id` load; mount always happens (§3.2, §0.1) |
| `en` / `id` drift (missing keys) | Compile-time: the generator enforces locale parity (C). For the B fallback: runtime `fallbackLocale: 'en'` + a key-usage check script (§0.1) |
| Generated types go stale (C) | Generator wired into `bun run check`; generated files committed |
| Interpolated strings with embedded HTML | No library renders HTML in translations — split into separate keys or use component slots; never inject translated HTML via `{@html}` |
| Svelte 5 runes + `LL` store | `LL` is a standard Svelte store; `{$LL.key()}` in templates works in runes mode (verify in Phase 1 probe build) |
| Translator churn on 3,676 candidates | Phase the work (shell first, then tabs by size); `common` de-dup collapses repeats |
| Scope creep into Go backend strings | Explicitly out of scope (§3.6); separate decision if ever wanted |
| Library goes stale (the §0 lesson) | C is actively maintained (2026) with zero runtime deps; if it ever stalls, the locale source is plain TS objects and the generated runtime is small enough to vendor or replace — the keys themselves are the asset, not the library |

## 7. Open decisions for the user

1. ~~Switch the recommendation to typesafe-i18n (C), or stay on svelte-i18n (B)?~~
   **DECIDED 2026-11-21 ~20:40 WIB: typesafe-i18n (C).**
2. ~~Commit this doc to `docs/` on `feat/i18n`?~~ **DONE 2026-11-21 ~21:41 WIB:**
   commit `a244b7a`, pushed to origin `feat/i18n` 2026-11-21 ~22:05 WIB (with the Phase 2 batch 1 push), verified via ls-remote.
3. ~~Start Phase 1 now (install + setup + shell translation)?~~ **DONE
   2026-11-21 ~21:30 WIB** — all verification green (see §4 step 5); committed
   + pushed as `33cfcac` 2026-11-21 ~21:38 WIB (see decision 4).
4. ~~Commit + push Phase 1 to `feat/i18n`?~~ **DONE 2026-11-21 ~21:38 WIB:**
   commit `33cfcac` (20 files, +2187/−160), pushed to origin `feat/i18n`,
   verified via ls-remote.
4. ~~Initial locale: default `'en'` always, or auto-detect?~~ — folded into (3)
   as the default; auto-detect (`navigator.language`, via the generated
   `detectLocale` + `typesafe-i18n/detectors`) remains available later.
5. ~~Confirm the switcher placement~~ — folded into (3) as the default.

## Appendix — verification evidence (2026-11-21)

- svelte-i18n 4.0.1: peer `svelte ^3 || ^4 || ^5`; **last release 2024-10-21** (npm
  `time` field); unpacked 57,567 B; deps incl. `intl-messageformat ^10.5.3`
  (117,447 B unpacked) + CLI tooling (`esbuild`, `sade`, `cli-color`, `tiny-glob`).
- typesafe-i18n 5.27.1: **last release 2026-02-11**; **zero runtime dependencies**
  (peer: TypeScript only); shipped ESM runtime + Svelte adapter `.mjs` total
  **21,061 B raw**; package unpacked 2,860,480 B is mostly dev-time codegen tooling,
  not shipped code.
- @inlang/paraglide-js 3.2.2: **last release 2026-11-10**; compiler-based, type-safe.
- Bundle probe for C (scratch copy in `/tmp/i18n-probe-c`, real `bun run build`):
  baseline dist 1,664,069 B / total JS 1,390,382 B / main chunk 409,480 B
  (gzip 97,946 B); + typesafe-i18n minimal wiring → dist 1,679,471 B /
  total JS 1,404,317 B / main chunk 379,364 B (gzip 90,090 B — chunk rebalanced,
  totals are the signal); locale chunks `en-*.js` + `id-*.js` emitted lazily.
- Generator probe: default `npx typesafe-i18n` **watches and never exits** — must
  run with `--no-watch` (exits 0). Generated files: `i18n-types.ts`,
  `i18n-util.ts`, `i18n-util.async.ts`, `i18n-util.sync.ts`, `formatters.ts`
  (empty stub). `i18n-svelte.ts` is **not generated** — hand-written via
  `initI18nSvelte(loadedLocales, loadedFormatters)` from `typesafe-i18n/svelte`,
  returning `{ locale, LL, setLocale }`. `loadLocaleAsync` from generated
  `i18n-util.async.ts`; sync loader statically imports all locales (no lazy
  loading). `{$LL.common.loading()}` in a template type-checked cleanly:
  **svelte-check 0 errors** (2 pre-existing warnings) with Svelte 5.56.10 +
  TypeScript 6.0.2 — Svelte 5 support confirmed empirically.
- Plural audit: 384 interpolated template nodes; dominant pattern is simple value
  interpolation; **zero ICU plural constructs** in the codebase; counts render as
  plain numbers (`${n} items`).
- Inventory method + counts: §2. Scan script was a throwaway heuristic in `/tmp`
  (not committed); re-run the same method after Phase 3 to measure the remainder.
- Key files read: `frontend/src/main.ts` (3 lines, mount only),
  `frontend/src/stores/theme.svelte.ts` (localStorage pattern to mirror),
  `frontend/src/components/layout/AppearanceModal.svelte` (sections: Color Themes,
  Accent Color, Sakuga FX, Advanced Tuning),
  `frontend/src/stores/toast.svelte.ts` (default titles at lines 29–46),
  `frontend/vite.config.ts`.
