<script lang="ts">
  import {
    Info,
    Heart,
    Zap,
    ExternalLink,
    Code2,
    Sliders,
    BatteryCharging,
    MessageSquare,
    Tv,
    QrCode,
    Send,
    Share2,
    Download,
    Maximize2,
    X,
    Sparkles,
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Button from '../../ui/Button.svelte'
  import Input from '../../ui/Input.svelte'
  import { toastStore } from '../../../stores/toast.svelte'
  import { LL } from '../../../i18n/i18n-svelte'
  import { fireSakugaFx } from '../../../lib/sakuga-fx'
  import qrisImg from '../../../assets/qris.jpg'

  let donorName = $state('')
  let donationAmount = $state('')
  let donorMessage = $state('')
  let isZoomed = $state(false)

  function getDonationMessage(): string {
    return `Halo Bang Latifan, saya sudah kirim donasi untuk BFR WEBUI GO:\n- Nama: ${donorName.trim() || 'Anonymous'}\n- Nominal: Rp ${donationAmount.trim() || '-'}\n- Pesan: ${donorMessage.trim() || '-'}\nTerima kasih atas karyanya!`
  }

  async function confirmTelegram() {
    const text = getDonationMessage()
    try {
      if (typeof navigator !== 'undefined' && navigator.clipboard) {
        await navigator.clipboard.writeText(text)
      }
      toastStore.success($LL.about.toasts.copiedTelegram())
      window.open(`https://t.me/skrprojects`, '_blank')
    } catch {
      window.open(`https://t.me/skrprojects`, '_blank')
    }
  }

  async function confirmFacebook() {
    const text = getDonationMessage()
    try {
      if (typeof navigator !== 'undefined' && navigator.clipboard) {
        await navigator.clipboard.writeText(text)
      }
      toastStore.success($LL.about.toasts.copiedFacebook())
      window.open('https://www.facebook.com/latifan.latifan.latifan.latif', '_blank')
    } catch {
      window.open('https://www.facebook.com/latifan.latifan.latifan.latif', '_blank')
    }
  }

  function openZoom(e: MouseEvent) {
    isZoomed = true
    // Sakuga FX: a small sparkle at the tap point when the card opens.
    fireSakugaFx(e, 'sparkle')
  }

  function closeZoom() {
    isZoomed = false
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && isZoomed) {
      closeZoom()
    }
  }

  function downloadQR(e: MouseEvent) {
    const link = document.createElement('a')
    link.href = qrisImg
    link.download = 'BFR-WEBUI-QRIS.jpg'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    toastStore.success($LL.about.toasts.qrisDownloaded())
    fireSakugaFx(e, 'sparkle')
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="space-y-3 sm:space-y-6">
  <!-- 1. Header Hero -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-2 border-b border-border font-mono">
    <div class="flex items-center gap-2.5 flex-wrap">
      <div class="p-2 rounded bg-card-sub border border-border text-accent">
        <Info class="w-5 h-5" />
      </div>
      <div class="flex items-center gap-2 flex-wrap">
        <h2 class="text-base sm:text-lg font-bold uppercase tracking-wider text-foreground">
          {$LL.about.title()}
        </h2>
        <span class="px-2 py-0.5 rounded text-[10px] font-black uppercase tracking-wider bg-amber-400 text-black shadow-neobrutal-sm">
          PRO
        </span>
        <Badge variant="default" class="text-[10px]">
          v1.2.8 (Build 128)
        </Badge>
      </div>
    </div>

    <div class="flex items-center gap-2 flex-wrap">
      <a
        href="https://github.com/latifangren/BFR-WEBUI-GO"
        target="_blank"
        rel="noreferrer"
        class="neo-button inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono font-bold rounded bg-card-sub border border-border text-foreground hover:border-accent hover:text-accent cursor-pointer transition-colors"
      >
        <Code2 class="w-4 h-4 text-accent" />
        <span>{$LL.about.githubRelease()}</span>
        <ExternalLink class="w-3 h-3 text-muted" />
      </a>
      <a
        href="https://t.me/skrprojects"
        target="_blank"
        rel="noreferrer"
        class="neo-button inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono font-bold rounded bg-card-sub border border-border text-foreground hover:border-sky-400 hover:text-sky-400 cursor-pointer transition-colors"
      >
        <Send class="w-4 h-4 text-sky-400" />
        <span>{$LL.about.telegramChannel()}</span>
        <ExternalLink class="w-3 h-3 text-muted" />
      </a>
    </div>
  </div>

  <!-- Main Content Grid -->
  <div class="space-y-3 sm:space-y-6">
    <!-- Top Row: Card 1 & Card 2 in 2-Column Desktop Grid -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6 items-start">
      <!-- 2. Card 1: Project Overview & Specs (tone="ice") -->
      <Card
        title={$LL.about.overview.title()}
        subtitle={$LL.about.overview.subtitle()}
        tone="ice"
      >
        <div class="space-y-4 font-mono text-xs">
          <!-- Logo Box & Tagline -->
          <div class="flex items-center gap-3.5 p-3.5 bg-card-sub border border-border rounded-lg">
            <div class="w-12 h-12 rounded-lg bg-accent text-accent-text border-2 border-border shadow-neobrutal-sm flex items-center justify-center font-black text-2xl font-mono shrink-0 select-none">
              B
            </div>
            <div>
              <h3 class="text-sm sm:text-base font-black uppercase tracking-wider text-foreground">
                BFR WEBUI PRO
              </h3>
              <p class="text-[11px] text-muted mt-0.5">
                {$LL.about.overview.tagline()}
              </p>
            </div>
          </div>

          <!-- 4 Specifications Grid -->
          <div class="grid grid-cols-2 sm:grid-cols-2 gap-2 sm:gap-2.5">
            <div class="p-2.5 sm:p-3 bg-card-sub border border-border rounded space-y-0.5 sm:space-y-1">
              <span class="text-[10px] uppercase font-bold text-muted">{$LL.about.overview.appVersion()}</span>
              <p class="font-bold text-accent">GO v1.2.8 (128)</p>
            </div>
            <div class="p-2.5 sm:p-3 bg-card-sub border border-border rounded space-y-0.5 sm:space-y-1">
              <span class="text-[10px] uppercase font-bold text-muted">{$LL.about.overview.environment()}</span>
              <p class="font-bold text-foreground truncate">{$LL.about.overview.environmentValue()}</p>
            </div>

            <div class="p-2.5 sm:p-3 bg-card-sub border border-border rounded space-y-0.5 sm:space-y-1">
              <span class="text-[10px] uppercase font-bold text-muted">{$LL.about.overview.maintainer()}</span>
              <p class="font-bold text-foreground">latifangren</p>
            </div>

            <div class="p-2.5 sm:p-3 bg-card-sub border border-border rounded space-y-0.5 sm:space-y-1">
              <span class="text-[10px] uppercase font-bold text-muted">{$LL.about.overview.license()}</span>
              <p class="font-bold text-emerald-400">GNU GPL v3.0</p>
            </div>
          </div>

          <!-- Architecture Paragraph -->
          <p class="leading-relaxed text-muted p-3 bg-card-sub border border-border rounded text-[11px]">
            {$LL.about.overview.archBodyA()}<strong class="text-foreground">Svelte 5</strong>{$LL.about.overview.archBodyB()}<strong class="text-foreground">Tailwind CSS</strong>{$LL.about.overview.archBodyC()}
          </p>
        </div>
      </Card>

      <!-- 3. Card 2: Magisk Module Key Features (tone="mint") -->
      <Card
        title={$LL.about.features.title()}
        subtitle={$LL.about.features.subtitle()}
        tone="mint"
      >
        <div class="space-y-2.5 font-mono text-xs">
          <!-- Feature 1: BBR2 -->
          <div class="p-3 bg-card-sub border border-border rounded space-y-1 hover:border-accent/40 transition-colors">
            <div class="flex items-center gap-2 text-foreground font-bold">
              <Zap class="w-4 h-4 text-amber-400 shrink-0" />
              <span>{$LL.about.features.bbrTitle()}</span>
            </div>
            <p class="text-[11px] text-muted leading-relaxed pl-6">
              {$LL.about.features.bbrBody()}
            </p>
          </div>

          <!-- Feature 2: Optimizer Tweaks -->
          <div class="p-3 bg-card-sub border border-border rounded space-y-1 hover:border-accent/40 transition-colors">
            <div class="flex items-center gap-2 text-foreground font-bold">
              <Sliders class="w-4 h-4 text-blue-400 shrink-0" />
              <span>{$LL.about.features.optimizerTitle()}</span>
            </div>
            <p class="text-[11px] text-muted leading-relaxed pl-6">
              {$LL.about.features.optimizerBody()}
            </p>
          </div>

          <!-- Feature 3: Charge Limiter -->
          <div class="p-3 bg-card-sub border border-border rounded space-y-1 hover:border-accent/40 transition-colors">
            <div class="flex items-center gap-2 text-foreground font-bold">
              <BatteryCharging class="w-4 h-4 text-emerald-400 shrink-0" />
              <span>{$LL.about.features.chargeTitle()}</span>
            </div>
            <p class="text-[11px] text-muted leading-relaxed pl-6">
              {$LL.about.features.chargeBody()}
            </p>
          </div>

          <!-- Feature 4: Telephony SQLite -->
          <div class="p-3 bg-card-sub border border-border rounded space-y-1 hover:border-accent/40 transition-colors">
            <div class="flex items-center gap-2 text-foreground font-bold">
              <MessageSquare class="w-4 h-4 text-cyan-400 shrink-0" />
              <span>{$LL.about.features.telephonyTitle()}</span>
            </div>
            <p class="text-[11px] text-muted leading-relaxed pl-6">
              {$LL.about.features.telephonyBody()}
            </p>
          </div>

          <!-- Feature 5: Terminal & Screen Mirroring -->
          <div class="p-3 bg-card-sub border border-border rounded space-y-1 hover:border-accent/40 transition-colors">
            <div class="flex items-center gap-2 text-foreground font-bold">
              <Tv class="w-4 h-4 text-purple-400 shrink-0" />
              <span>{$LL.about.features.terminalTitle()}</span>
            </div>
            <p class="text-[11px] text-muted leading-relaxed pl-6">
              {$LL.about.features.terminalBody()}
            </p>
          </div>
        </div>
      </Card>
    </div>

    <!-- 4. Card 3: Support & Donation Hub (tone="peach") -->
    <Card
      title={$LL.about.donation.title()}
      subtitle={$LL.about.donation.subtitle()}
      tone="peach"
    >
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3 sm:gap-6 font-mono text-xs">
        <!-- Left Column: Scan QRIS Card -->
        <div class="flex flex-col items-center justify-between p-4 bg-card-sub border border-border rounded-lg text-center space-y-4">
          <div class="space-y-1">
            <div class="flex items-center justify-center gap-2 text-foreground font-bold text-sm">
              <QrCode class="w-4 h-4 text-accent" />
              <span>{$LL.about.donation.scanTitle()}</span>
            </div>
            <p class="text-[11px] text-muted">
              {$LL.about.donation.scanBody()}
            </p>
          </div>

          <!-- Idle Interactive Pokemon-Style Card -->
          <div class="my-auto py-2 flex flex-col items-center">
            <button
              type="button"
              onclick={openZoom}
              class="pokemon-idle-card group relative p-3 rounded-2xl bg-card border-2 border-amber-400/70 shadow-neobrutal-md hover:shadow-neobrutal-lg hover:border-amber-400 cursor-pointer transition-all duration-300 text-left focus:outline-none focus:ring-2 focus:ring-amber-400/50"
              title={$LL.about.donation.zoomTitle()}
            >
              <!-- Foil Shimmer Layer -->
              <div class="holo-shimmer absolute inset-0 rounded-2xl pointer-events-none opacity-60 group-hover:opacity-100 transition-opacity"></div>

              <!-- Card Header Pill -->
              <div class="flex items-center justify-between gap-2 mb-2 pb-1.5 border-b border-border/80 text-[10px] font-bold">
                <span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-amber-400/20 text-amber-500 border border-amber-400/40 font-mono">
                  <Sparkles class="w-3 h-3" />
                  <span>QRIS FOIL</span>
                </span>
                <span class="inline-flex items-center gap-1 text-muted group-hover:text-foreground transition-colors font-mono">
                  <Maximize2 class="w-3 h-3 text-accent" />
                  <span>{$LL.about.donation.tapToZoom()}</span>
                </span>
              </div>

              <!-- QR Code Image Inside Card -->
              <div class="relative bg-white p-2 rounded-xl border border-border/60 shadow-sm flex items-center justify-center">
                <img
                  src={qrisImg}
                  alt={$LL.about.donation.qrisAlt()}
                  loading="eager"
                  class="w-full max-w-[170px] sm:max-w-[200px] aspect-square object-contain mx-auto rounded"
                />
              </div>

              <!-- Card Footer Hint -->
              <div class="mt-2 text-center">
                <p class="text-[10px] font-mono text-muted group-hover:text-accent font-bold transition-colors">
                  🔍 {$LL.about.donation.zoomHint()}
                </p>
              </div>
            </button>

            <!-- Quick Download Button Under Idle Card -->
            <button
              type="button"
              onclick={downloadQR}
              class="mt-3 inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono font-bold rounded-lg bg-card border border-border text-foreground hover:border-emerald-400 hover:text-emerald-400 hover:bg-emerald-500/10 cursor-pointer transition-colors shadow-neobrutal-sm"
              title={$LL.about.donation.downloadQrisTitle()}
            >
              <Download class="w-3.5 h-3.5 text-emerald-400" />
              <span>{$LL.about.donation.downloadQris()}</span>
            </button>
          </div>

          <p class="text-[10px] text-muted leading-relaxed max-w-xs">
            {$LL.about.donation.supported()}
          </p>
        </div>
        <!-- Right Column: Donation Confirmation Form -->
        <div class="p-4 bg-card-sub border border-border rounded-lg space-y-4 flex flex-col justify-between">
          <div class="space-y-3">
            <div class="flex items-center gap-2 text-foreground font-bold text-sm">
              <Heart class="w-4 h-4 text-red-500 fill-red-500/20" />
              <span>{$LL.about.donation.confirmTitle()}</span>
            </div>
            <p class="text-[11px] text-muted leading-relaxed">
              {$LL.about.donation.confirmBody()}
            </p>

            <Input
              label={$LL.about.donation.donorName()}
              placeholder={$LL.about.donation.donorNamePlaceholder()}
              bind:value={donorName}
            />

            <Input
              label={$LL.about.donation.amount()}
              placeholder={$LL.about.donation.amountPlaceholder()}
              bind:value={donationAmount}
            />

            <div class="space-y-1">
              <label for="donor-message" class="block text-xs font-mono font-bold uppercase text-muted tracking-wider">
                {$LL.about.donation.messageLabel()}
              </label>
              <textarea
                id="donor-message"
                rows="3"
                bind:value={donorMessage}
                placeholder={$LL.about.donation.messagePlaceholder()}
                class="neo-input w-full bg-card border border-border rounded px-3 py-2 text-xs font-mono text-foreground placeholder:text-muted focus:outline-none focus:border-accent resize-none"
              ></textarea>
            </div>
          </div>

          <!-- Action Buttons -->
          <div class="space-y-2 pt-2 border-t border-border">
            <Button
              variant="primary"
              size="md"
              fullWidth={true}
              onclick={confirmTelegram}
              title={$LL.about.donation.confirmTelegramTitle()}
            >
              <Send class="w-4 h-4 mr-2" />
              <span>{$LL.about.donation.confirmTelegram()}</span>
            </Button>

            <Button
              variant="secondary"
              size="md"
              fullWidth={true}
              onclick={confirmFacebook}
              title={$LL.about.donation.confirmFacebookTitle()}
            >
              <Share2 class="w-4 h-4 mr-2 text-blue-400" />
              <span>{$LL.about.donation.confirmFacebook()}</span>
            </Button>
          </div>
        </div>
      </div>

      <!-- Official Community & Project Links Row -->
      <div class="mt-4 pt-4 border-t border-border">
        <div class="flex items-center justify-between gap-3 flex-wrap">
          <div class="text-[11px] font-mono text-muted">
            <span class="font-bold text-foreground">{$LL.about.donation.communityTitle()}</span>{$LL.about.donation.communityBody()}
          </div>
          <div class="flex items-center gap-2 flex-wrap">
            <a
              href="https://github.com/latifangren/BFR-WEBUI-GO"
              target="_blank"
              rel="noreferrer"
              class="neo-button inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono font-bold rounded bg-card border border-border text-foreground hover:border-accent hover:text-accent cursor-pointer transition-colors"
            >
              <Code2 class="w-3.5 h-3.5 text-accent" />
              <span>{$LL.about.donation.githubRepo()}</span>
              <ExternalLink class="w-3 h-3 text-muted" />
            </a>
            <a
              href="https://t.me/skrprojects"
              target="_blank"
              rel="noreferrer"
              class="neo-button inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono font-bold rounded bg-card border border-border text-foreground hover:border-sky-400 hover:text-sky-400 cursor-pointer transition-colors"
            >
              <Send class="w-3.5 h-3.5 text-sky-400" />
              <span>{$LL.about.donation.telegramHandle()}</span>
              <ExternalLink class="w-3 h-3 text-muted" />
            </a>
          </div>
        </div>
      </div>
    </Card>
  </div>

  <!-- Pokemon GO Style Card Zoom Modal -->
  {#if isZoomed}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div
      class="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-6 bg-black/85 backdrop-blur-md animate-fade-in"
      onclick={closeZoom}
      onkeydown={handleKeydown}
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <!-- Ambient Card Aura Burst -->
      <div class="aura-burst absolute -inset-10 bg-gradient-to-tr from-amber-400/25 via-accent/30 to-rose-500/25 rounded-full blur-3xl pointer-events-none"></div>

      <!-- Pokemon Zoom Card Container -->
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
      <div
        class="pokemon-card-reveal relative w-full max-w-sm sm:max-w-md bg-card border-2 border-amber-400 rounded-3xl p-4 sm:p-5 shadow-2xl shadow-amber-500/20 font-mono text-xs overflow-hidden"
        onclick={(e) => e.stopPropagation()}
        role="document"
      >
        <!-- Holographic Sheen Overlay on Zoomed Card -->
        <div class="holo-shimmer absolute inset-0 pointer-events-none rounded-3xl opacity-75"></div>

        <!-- Modal Card Header -->
        <div class="relative flex items-center justify-between pb-3 mb-3 border-b border-border/80">
          <div class="flex items-center gap-2">
            <div class="p-1.5 rounded-lg bg-amber-400/20 border border-amber-400/40 text-amber-500">
              <Sparkles class="w-4 h-4" />
            </div>
            <div>
              <h3 class="font-bold text-sm text-foreground uppercase tracking-wider flex items-center gap-1.5">
                <span>{$LL.about.modal.title()}</span>
                <span class="text-[10px] px-1.5 py-0.2 rounded bg-amber-400 text-black font-black">PRO</span>
              </h3>
              <p class="text-[10px] text-muted">{$LL.about.modal.subtitle()}</p>
            </div>
          </div>

          <button
            type="button"
            onclick={closeZoom}
            class="p-1.5 rounded-lg bg-card-sub border border-border text-muted hover:text-foreground hover:border-red-400 hover:bg-red-500/10 transition cursor-pointer"
            title={$LL.about.modal.closeTitle()}
          >
            <X class="w-4 h-4" />
          </button>
        </div>

        <!-- Large High-Contrast QR Code for Easy Scanning -->
        <div class="relative bg-white p-4 sm:p-5 rounded-2xl border-2 border-border/80 shadow-inner flex flex-col items-center justify-center">
          <img
            src={qrisImg}
            alt={$LL.about.donation.qrisAltLarge()}
            class="w-full max-w-[260px] sm:max-w-[300px] aspect-square object-contain mx-auto"
          />
          <div class="mt-2 text-center">
            <span class="text-[10px] font-bold text-neutral-800 tracking-wider uppercase font-mono">
              {$LL.about.modal.scanHint()}
            </span>
          </div>
        </div>

        <!-- Scan Instructions Note -->
        <p class="relative text-[11px] text-center text-muted mt-3 leading-relaxed">
          {$LL.about.modal.instructions()}
        </p>

        <!-- Modal Actions: Download & Close -->
        <div class="relative grid grid-cols-2 gap-2 mt-4 pt-3 border-t border-border">
          <button
            type="button"
            onclick={downloadQR}
            class="neo-button flex items-center justify-center gap-2 py-2 px-3 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-black font-black cursor-pointer transition shadow-neobrutal-sm"
          >
            <Download class="w-4 h-4" />
            <span>{$LL.about.modal.downloadQr()}</span>
          </button>

          <button
            type="button"
            onclick={closeZoom}
            class="neo-button flex items-center justify-center gap-2 py-2 px-3 rounded-xl bg-card-sub hover:bg-card border border-border text-foreground font-bold cursor-pointer transition shadow-neobrutal-sm"
          >
            <X class="w-4 h-4 text-muted" />
            <span>{$LL.common.close()}</span>
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  @keyframes cardIdleFloat {
    0%, 100% {
      transform: translateY(0px) rotate(0deg);
    }
    50% {
      transform: translateY(-6px) rotate(0.8deg);
    }
  }

  @keyframes cardHoloShine {
    0% {
      background-position: -200% -200%;
      opacity: 0.2;
    }
    50% {
      opacity: 0.65;
    }
    100% {
      background-position: 200% 200%;
      opacity: 0.2;
    }
  }

  @keyframes pokemonCardPop {
    0% {
      opacity: 0;
      transform: scale(0.32) rotate(-14deg) translateY(60px);
    }
    60% {
      opacity: 1;
      transform: scale(1.06) rotate(2deg) translateY(-8px);
    }
    85% {
      transform: scale(0.98) rotate(-0.5deg) translateY(2px);
    }
    100% {
      opacity: 1;
      transform: scale(1) rotate(0deg) translateY(0);
    }
  }

  @keyframes auraPulse {
    0%, 100% {
      opacity: 0.45;
      transform: scale(0.95);
    }
    50% {
      opacity: 0.8;
      transform: scale(1.08);
    }
  }

  .pokemon-idle-card {
    animation: cardIdleFloat 4s ease-in-out infinite;
    transform-style: preserve-3d;
  }

  .pokemon-idle-card:hover {
    animation-play-state: paused;
    transform: translateY(-8px) scale(1.03);
  }

  .holo-shimmer {
    background: linear-gradient(
      115deg,
      transparent 0%,
      rgba(255, 255, 255, 0.08) 25%,
      rgba(255, 215, 0, 0.35) 45%,
      rgba(56, 189, 248, 0.4) 50%,
      rgba(244, 63, 94, 0.35) 55%,
      transparent 75%
    );
    background-size: 250% 250%;
    animation: cardHoloShine 4s linear infinite;
  }

  .pokemon-card-reveal {
    animation: pokemonCardPop 0.48s cubic-bezier(0.2, 0.9, 0.3, 1.2) forwards;
  }

  .aura-burst {
    animation: auraPulse 3s ease-in-out infinite;
  }
</style>
