<script lang="ts">
  import {
    Info,
    Heart,
    Shield,
    Smartphone,
    Terminal,
    Zap,
    ExternalLink,
    Code2,
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Button from '../../ui/Button.svelte'
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Info class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        About BFR-WEBUI-GO
      </h2>
      <Badge variant="info">v1.2.2</Badge>
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

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Project Overview -->
    <Card
      title="BFR-WEBUI-GO Ecosystem"
      subtitle="Android Root System Control Panel & WebUI"
      tone="butter"
      class="lg:col-span-2"
    >
      <div class="space-y-4 font-mono text-xs leading-relaxed text-muted">
        <p>
          <strong class="text-foreground">BFR-WEBUI-GO</strong> is an ultra-lightweight, 100% offline-ready Android system control panel and management dashboard designed to run natively as a Magisk, KernelSU, or APatch root module.
        </p>

        <p>
          Re-architected with <strong class="text-foreground">Svelte 5</strong>, <strong class="text-foreground">Vite</strong>, and <strong class="text-foreground">Tailwind CSS</strong>, embedded directly into a single self-contained Go binary (~10-12MB) with near-zero idle CPU and tiny RAM consumption on Android devices.
        </p>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
          <div class="p-3 bg-card-sub border border-border rounded space-y-1">
            <span class="text-[10px] uppercase font-bold text-muted">Module ID</span>
            <p class="font-bold text-foreground">bfr_webui_go</p>
          </div>
          <div class="p-3 bg-card-sub border border-border rounded space-y-1">
            <span class="text-[10px] uppercase font-bold text-muted">Release Version</span>
            <p class="font-bold text-accent">v1.2.2 (Build 122)</p>
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
      </div>
    </Card>

    <!-- Architecture & Features -->
    <Card title="Modern Architecture" subtitle="Zero-Virtual-DOM stack" tone="butter">
      <div class="space-y-3 font-mono text-xs">
        <div class="flex items-center gap-2 p-2 bg-card-sub border border-border rounded">
          <Zap class="w-4 h-4 text-amber-400 shrink-0" />
          <div>
            <span class="font-bold text-foreground">Svelte 5 Runes</span>
            <p class="text-[10px] text-muted">Fine-grained reactive DOM mutation</p>
          </div>
        </div>

        <div class="flex items-center gap-2 p-2 bg-card-sub border border-border rounded">
          <Smartphone class="w-4 h-4 text-accent shrink-0" />
          <div>
            <span class="font-bold text-foreground">Android Root Native</span>
            <p class="text-[10px] text-muted">Direct /proc, sysfs, and su -c pipes</p>
          </div>
        </div>

        <div class="flex items-center gap-2 p-2 bg-card-sub border border-border rounded">
          <Terminal class="w-4 h-4 text-emerald-400 shrink-0" />
          <div>
            <span class="font-bold text-foreground">Low-Latency WebSockets</span>
            <p class="text-[10px] text-muted">PTY terminal, Scrcpy, and Logcat</p>
          </div>
        </div>
      </div>
    </Card>
  </div>
</div>
