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
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Button from '../../ui/Button.svelte'
  import Input from '../../ui/Input.svelte'
  import { toastStore } from '../../../stores/toast.svelte'
  import qrisImg from '../../../assets/qris.jpg'

  let donorName = $state('')
  let donationAmount = $state('')
  let donorMessage = $state('')

  function getDonationMessage(): string {
    return `Halo Bang Latifan, saya sudah kirim donasi untuk BFR WEBUI GO:\n- Nama: ${donorName.trim() || 'Anonymous'}\n- Nominal: Rp ${donationAmount.trim() || '-'}\n- Pesan: ${donorMessage.trim() || '-'}\nTerima kasih atas karyanya!`
  }

  async function confirmTelegram() {
    const text = getDonationMessage()
    try {
      if (typeof navigator !== 'undefined' && navigator.clipboard) {
        await navigator.clipboard.writeText(text)
      }
      toastStore.success('Detail donasi disalin ke clipboard! Membuka Telegram...')
      window.open(`https://t.me/Latifan_id?text=${encodeURIComponent(text)}`, '_blank')
    } catch {
      window.open(`https://t.me/Latifan_id?text=${encodeURIComponent(text)}`, '_blank')
    }
  }

  async function confirmFacebook() {
    const text = getDonationMessage()
    try {
      if (typeof navigator !== 'undefined' && navigator.clipboard) {
        await navigator.clipboard.writeText(text)
      }
      toastStore.success('Detail donasi disalin ke clipboard! Membuka Facebook...')
      window.open('https://www.facebook.com/latifan.latifan.latifan.latif', '_blank')
    } catch {
      window.open('https://www.facebook.com/latifan.latifan.latifan.latif', '_blank')
    }
  }
</script>

<div class="space-y-6">
  <!-- 1. Header Hero -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-2 border-b border-border font-mono">
    <div class="flex items-center gap-2.5 flex-wrap">
      <div class="p-2 rounded bg-card-sub border border-border text-accent">
        <Info class="w-5 h-5" />
      </div>
      <div class="flex items-center gap-2 flex-wrap">
        <h2 class="text-base sm:text-lg font-bold uppercase tracking-wider text-foreground">
          About BFR-WEBUI-GO
        </h2>
        <span class="px-2 py-0.5 rounded text-[10px] font-black uppercase tracking-wider bg-amber-400 text-black shadow-neobrutal-sm">
          PRO
        </span>
        <Badge variant="default" class="text-[10px]">
          v1.2.3 (Build 123)
        </Badge>
      </div>
    </div>

    <div class="flex items-center gap-2">
      <a
        href="https://github.com/latifangren/BFR-WEBUI-GO"
        target="_blank"
        rel="noreferrer"
        class="neo-button inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono font-bold rounded bg-card-sub border border-border text-foreground hover:border-accent cursor-pointer transition-colors"
      >
        <Code2 class="w-4 h-4 text-accent" />
        <span>GitHub Repository</span>
        <ExternalLink class="w-3 h-3 text-muted" />
      </a>
    </div>
  </div>

  <!-- Main Content Grid -->
  <div class="space-y-6">
    <!-- Top Row: Card 1 & Card 2 in 2-Column Desktop Grid -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">
      <!-- 2. Card 1: Project Overview & Specs (tone="ice") -->
      <Card
        title="Project Overview & Specs"
        subtitle="Android Root System Control Panel & WebUI"
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
                High-Performance Android System WebUI & Control Panel
              </p>
            </div>
          </div>

          <!-- 4 Specifications Grid -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
            <div class="p-3 bg-card-sub border border-border rounded space-y-1">
              <span class="text-[10px] uppercase font-bold text-muted">App Version</span>
              <p class="font-bold text-accent">GO v1.2.3 (Build 123)</p>
            </div>

            <div class="p-3 bg-card-sub border border-border rounded space-y-1">
              <span class="text-[10px] uppercase font-bold text-muted">Environment</span>
              <p class="font-bold text-foreground">Magisk / KernelSU / APatch</p>
            </div>

            <div class="p-3 bg-card-sub border border-border rounded space-y-1">
              <span class="text-[10px] uppercase font-bold text-muted">Author / Maintainer</span>
              <p class="font-bold text-foreground">latifangren</p>
            </div>

            <div class="p-3 bg-card-sub border border-border rounded space-y-1">
              <span class="text-[10px] uppercase font-bold text-muted">License</span>
              <p class="font-bold text-emerald-400">MIT Open Source</p>
            </div>
          </div>

          <!-- Architecture Paragraph -->
          <p class="leading-relaxed text-muted p-3 bg-card-sub border border-border rounded text-[11px]">
            Re-architected with native Go backend, <strong class="text-foreground">Svelte 5</strong>, and <strong class="text-foreground">Tailwind CSS</strong>, embedded directly into a single self-contained Go binary (~10-12MB) with near-zero idle CPU and tiny RAM consumption on Android devices.
          </p>
        </div>
      </Card>

      <!-- 3. Card 2: Magisk Module Key Features (tone="mint") -->
      <Card
        title="Magisk Module Key Features"
        subtitle="Kernel-level tuning & automated background engine"
        tone="mint"
      >
        <div class="space-y-2.5 font-mono text-xs">
          <!-- Feature 1: BBR2 -->
          <div class="p-3 bg-card-sub border border-border rounded space-y-1 hover:border-accent/40 transition-colors">
            <div class="flex items-center gap-2 text-foreground font-bold">
              <Zap class="w-4 h-4 text-amber-400 shrink-0" />
              <span>BBR2 TCP Congestion Control</span>
            </div>
            <p class="text-[11px] text-muted leading-relaxed pl-6">
              Active congestion prevention algorithms tailored for low latency and high bandwidth wireless networks.
            </p>
          </div>

          <!-- Feature 2: Optimizer Tweaks -->
          <div class="p-3 bg-card-sub border border-border rounded space-y-1 hover:border-accent/40 transition-colors">
            <div class="flex items-center gap-2 text-foreground font-bold">
              <Sliders class="w-4 h-4 text-blue-400 shrink-0" />
              <span>System Optimizer Tweaks</span>
            </div>
            <p class="text-[11px] text-muted leading-relaxed pl-6">
              Kernel sysctl variables configurations, TCP FastOpen, Queue limits allocations, and dynamic SDK-aware TTL configurations.
            </p>
          </div>

          <!-- Feature 3: Charge Limiter -->
          <div class="p-3 bg-card-sub border border-border rounded space-y-1 hover:border-accent/40 transition-colors">
            <div class="flex items-center gap-2 text-foreground font-bold">
              <BatteryCharging class="w-4 h-4 text-emerald-400 shrink-0" />
              <span>Dynamic Charge Limiter</span>
            </div>
            <p class="text-[11px] text-muted leading-relaxed pl-6">
              Automated battery lifespan charging limits scanning based on device sysfs properties to prevent battery degradation.
            </p>
          </div>

          <!-- Feature 4: Telephony SQLite -->
          <div class="p-3 bg-card-sub border border-border rounded space-y-1 hover:border-accent/40 transition-colors">
            <div class="flex items-center gap-2 text-foreground font-bold">
              <MessageSquare class="w-4 h-4 text-cyan-400 shrink-0" />
              <span>Telephony SQLite OTP Scanner</span>
            </div>
            <p class="text-[11px] text-muted leading-relaxed pl-6">
              Direct system inbox database query controller retrieving recent incoming verifications, OTP tokens & bank notification codes.
            </p>
          </div>

          <!-- Feature 5: Terminal & Screen Mirroring -->
          <div class="p-3 bg-card-sub border border-border rounded space-y-1 hover:border-accent/40 transition-colors">
            <div class="flex items-center gap-2 text-foreground font-bold">
              <Tv class="w-4 h-4 text-purple-400 shrink-0" />
              <span>Websocket Terminal & Screen Mirroring</span>
            </div>
            <p class="text-[11px] text-muted leading-relaxed pl-6">
              Interactive low-latency root shells, canvas screen rendering and inputs/swipes injection overlay controls.
            </p>
          </div>
        </div>
      </Card>
    </div>

    <!-- 4. Card 3: Support & Donation Hub (tone="peach") -->
    <Card
      title="Support & Donation Hub"
      subtitle="Help support active maintenance, new features & continuous updates"
      tone="peach"
    >
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6 font-mono text-xs">
        <!-- Left Column: Scan QRIS -->
        <div class="flex flex-col items-center justify-between p-4 bg-card-sub border border-border rounded-lg text-center space-y-4">
          <div class="space-y-1">
            <div class="flex items-center justify-center gap-2 text-foreground font-bold text-sm">
              <QrCode class="w-4 h-4 text-accent" />
              <span>Scan QRIS to Donate</span>
            </div>
            <p class="text-[11px] text-muted">
              Fast, automated payment code valid for all Indonesian bank accounts & e-wallets.
            </p>
          </div>

          <!-- QR Code Image -->
          <div class="my-auto py-2">
            <img
              src={qrisImg}
              alt="QRIS Donation"
              loading="eager"
              class="w-full max-w-[220px] mx-auto rounded-xl border-2 border-border shadow-neobrutal-sm bg-white p-2"
            />
          </div>

          <p class="text-[10px] text-muted leading-relaxed max-w-xs">
            Supported by all Indonesian E-Wallets (GoPay, OVO, DANA, ShopeePay) & Mobile Banking (BCA, Mandiri, BRI, BNI, etc.).
          </p>
        </div>

        <!-- Right Column: Donation Confirmation Form -->
        <div class="p-4 bg-card-sub border border-border rounded-lg space-y-4 flex flex-col justify-between">
          <div class="space-y-3">
            <div class="flex items-center gap-2 text-foreground font-bold text-sm">
              <Heart class="w-4 h-4 text-red-500 fill-red-500/20" />
              <span>Donation Confirmation</span>
            </div>
            <p class="text-[11px] text-muted leading-relaxed">
              Confirm your contribution to developer via direct chat. Details will be automatically formatted and copied to your clipboard.
            </p>

            <Input
              label="Donor Name"
              placeholder="Your name or 'Anonymous'"
              bind:value={donorName}
            />

            <Input
              label="Donation Amount (Rp)"
              placeholder="e.g. 50000"
              bind:value={donationAmount}
            />

            <div class="space-y-1">
              <label for="donor-message" class="block text-xs font-mono font-bold uppercase text-muted tracking-wider">
                Message / Note
              </label>
              <textarea
                id="donor-message"
                rows="3"
                bind:value={donorMessage}
                placeholder="Write your message or note for developer..."
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
              title="Send confirmation via Telegram chat"
            >
              <Send class="w-4 h-4 mr-2" />
              <span>Confirm via Telegram</span>
            </Button>

            <Button
              variant="secondary"
              size="md"
              fullWidth={true}
              onclick={confirmFacebook}
              title="Send confirmation via Facebook message"
            >
              <Share2 class="w-4 h-4 mr-2 text-blue-400" />
              <span>Confirm via Facebook</span>
            </Button>
          </div>
        </div>
      </div>
    </Card>
  </div>
</div>
