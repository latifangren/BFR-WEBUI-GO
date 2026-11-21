<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Smartphone,
    RefreshCw,
    Shield,
    Check,
    AlertTriangle,
    Copy,
    Wrench,
    Radio,
    Terminal,
    HardDrive,
    RotateCw,
    Cpu,
    ExternalLink,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import { navigationStore } from '../../../stores/navigation.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'
  import { themeStore } from '../../../stores/theme.svelte'
  import SakugaCheckbox from '../../ui/SakugaCheckbox.svelte'
  import { LL } from '../../../i18n/i18n-svelte'

  interface IMEIInfo {
    imei1: string
    imei2: string
    success: boolean
    error?: string
  }

  interface IMEIWriteResult {
    success: boolean
    slot: number
    imei: string
    method_used: string
    step_logs: string[]
    error?: string
  }

  // ─── State ───────────────────────────────────────────────────────────────────

  let imeiInfo = $state<IMEIInfo | null>(null)
  let isLoadingInfo = $state(false)

  // Write Form
  let targetSlot = $state<0 | 1>(0)
  let inputIMEI = $state('')
  let selectedMethod = $state<'auto' | 'normal' | 'efs'>('auto')
  let useOfflineMode = $state(true)
  let autoReboot = $state(false)

  // Execution state
  let isWriting = $state(false)
  let executionLogs = $state<string[]>([])
  let lastResult = $state<IMEIWriteResult | null>(null)
  let confirmModalOpen = $state(false)

  // SSR state
  let isRestartingModem = $state(false)

  // ─── Luhn Helpers in Frontend ────────────────────────────────────────────────

  function calcLuhnDigit(core14: string): string {
    if (core14.length !== 14) return '0'
    let sum = 0
    for (let i = 0; i < 14; i++) {
      let d = parseInt(core14[i], 10)
      if (i % 2 === 1) {
        d *= 2
        if (d > 9) d -= 9
      }
      sum += d
    }
    const check = (10 - (sum % 10)) % 10
    return check.toString()
  }

  const cleanedIMEI = $derived(inputIMEI.replace(/\D/g, ''))

  const luhnStatus = $derived.by(() => {
    if (!cleanedIMEI) return { state: 'empty', message: '' }
    if (cleanedIMEI.length < 14) {
      return {
        state: 'incomplete',
        message: $LL.qualcomm.imei.digitsEntered({ count: cleanedIMEI.length }),
      }
    }
    if (cleanedIMEI.length === 14) {
      const needed = calcLuhnDigit(cleanedIMEI)
      return {
        state: 'fixable',
        message: $LL.qualcomm.imei.checksum14({ digit: needed }),
        fixedDigit: needed,
      }
    }
    if (cleanedIMEI.length === 15) {
      const expected = calcLuhnDigit(cleanedIMEI.slice(0, 14))
      if (cleanedIMEI[14] === expected) {
        return {
          state: 'valid',
          message: $LL.qualcomm.imei.valid(),
        }
      }
      return {
        state: 'invalid',
        message: $LL.qualcomm.imei.invalidLuhn({ expected, got: cleanedIMEI[14] }),
        fixedDigit: expected,
      }
    }
    return {
      state: 'too_long',
      message: $LL.qualcomm.imei.tooMany({ count: cleanedIMEI.length }),
    }
  })

  function autoFixLuhn() {
    if (cleanedIMEI.length === 14) {
      const digit = calcLuhnDigit(cleanedIMEI)
      inputIMEI = cleanedIMEI + digit
      toastStore.success($LL.qualcomm.imei.appended({ digit }))
    } else if (cleanedIMEI.length === 15) {
      const digit = calcLuhnDigit(cleanedIMEI.slice(0, 14))
      inputIMEI = cleanedIMEI.slice(0, 14) + digit
      toastStore.success($LL.qualcomm.imei.corrected({ digit }))
    }
  }


  function copyToClipboard(text: string, label: string) {
    if (!text || text === 'Not Set') return
    navigator.clipboard.writeText(text)
    toastStore.success($LL.qualcomm.imei.copied({ label }))
  }

  function loadIntoForm(imei: string, slot: 0 | 1) {
    if (!imei || imei === 'Not Set') return
    targetSlot = slot
    inputIMEI = imei
    toastStore.info($LL.qualcomm.imei.loaded({ imei, slot: slot + 1 }))
  }

  // ─── Lifecycle & API ─────────────────────────────────────────────────────────

  onMount(() => {
    fetchCurrentIMEI()
  })

  async function fetchCurrentIMEI() {
    try {
      isLoadingInfo = true
      const res = await api.get<{ ok: boolean; imei: IMEIInfo; error?: string }>('/api/efs/imei')
      if (res && res.ok && res.imei) {
        imeiInfo = res.imei
      } else {
        toastStore.error(res?.error || $LL.qualcomm.imei.readFailed())
      }
    } catch (e: any) {
      toastStore.error(e?.message || $LL.qualcomm.imei.diagError())
    } finally {
      isLoadingInfo = false
    }
  }

  function requestWrite() {
    if (luhnStatus.state !== 'valid') {
      toastStore.error($LL.qualcomm.imei.enterValid())
      return
    }
    confirmModalOpen = true
  }

  async function executeWrite() {
    confirmModalOpen = false
    isWriting = true
    executionLogs = [
      `[${new Date().toLocaleTimeString()}] ${$LL.qualcomm.imei.logStart()}`,
      $LL.qualcomm.imei.logTarget({ slot: targetSlot + 1, nv: targetSlot === 0 ? 'Primary NV 550' : 'Secondary NV 2497/5014' }),
      $LL.qualcomm.imei.logNewImei({ imei: cleanedIMEI }),
      $LL.qualcomm.imei.logMethod({ method: selectedMethod.toUpperCase() }),
      $LL.qualcomm.imei.logOffline({ mode: useOfflineMode ? 'ENABLED' : 'DISABLED' }),
    ]

    try {
      const res = await api.post<{ ok: boolean; result: IMEIWriteResult; error?: string }>(
        '/api/efs/imei/write',
        {
          slot: targetSlot,
          imei: cleanedIMEI,
          method: selectedMethod,
          use_offline_mode: useOfflineMode,
          reboot_modem: autoReboot,
        },
      )

      if (res?.result) {
        lastResult = res.result
        if (res.result.step_logs) {
          executionLogs = [
            ...executionLogs,
            ...res.result.step_logs.map((log) => `[DIAG] ${log}`),
          ]
        }
      }

      if (res?.ok && res.result?.success) {
        toastStore.success($LL.qualcomm.imei.updated({ method: res.result.method_used.toUpperCase() }))
        executionLogs = [
          ...executionLogs,
          `[SUCCESS] ${$LL.qualcomm.imei.logSuccess()}`,
          autoReboot ? `[INFO] ${$LL.qualcomm.imei.logRestarting()}` : `[NOTICE] ${$LL.qualcomm.imei.logRebootNotice()}`,
        ]
        await fetchCurrentIMEI()
      } else {
        const errMsg = res?.error || res?.result?.error || $LL.qualcomm.imei.writeFailed()
        toastStore.error(errMsg)
        executionLogs = [
          ...executionLogs,
          `[FAILED] ${errMsg}`,
        ]
      }
    } catch (e: any) {
      const msg = e?.message || $LL.qualcomm.imei.networkError()
      toastStore.error(msg)
      executionLogs = [...executionLogs, `[EXCEPTION] ${msg}`]
    } finally {
      isWriting = false
    }
  }

  async function handleModemSSR() {
    try {
      isRestartingModem = true
      const res = await api.post<{ ok: boolean; message?: string }>('/api/efs/ssr', {})
      if (res && res.ok) {
        toastStore.success($LL.qualcomm.imei.ssrIssued())
        executionLogs = [
          ...executionLogs,
          `[${new Date().toLocaleTimeString()}] ${$LL.qualcomm.imei.logSsr()}`,
        ]
        setTimeout(() => fetchCurrentIMEI(), 4000)
      } else {
        toastStore.error($LL.qualcomm.imei.ssrFailed())
      }
    } catch (e: any) {
      toastStore.error(e?.message || $LL.qualcomm.imei.ssrRequestFailed())
    } finally {
      isRestartingModem = false
    }
  }

  function jumpToPartitions() {
    navigationStore.activeTab = 'partitions'
  }
</script>

<div class="space-y-4 font-mono text-xs">
  <!-- SAFETY BACKUP ALERT BANNER -->
  <div class="p-3.5 bg-amber-950/30 border-2 border-amber-600/70 rounded-lg flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-neobrutal-sm">
    <div class="flex items-start gap-2.5">
      <AlertTriangle class="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
      <div>
        <div class="flex items-center gap-2">
          <span class="font-black uppercase tracking-wider text-amber-300">{$LL.qualcomm.imei.safetyTitle()}</span>
          <Badge variant="warning">{$LL.qualcomm.imei.crucial()}</Badge>
        </div>
        <p class="text-xs text-muted mt-0.5">
          {$LL.qualcomm.imei.safetyA()}<code class="text-white font-bold">fsg, modemst1, modemst2, efs</code>{$LL.qualcomm.imei.safetyB()}
        </p>
      </div>
    </div>
    <Button
      variant="secondary"
      size="sm"
      class="shrink-0 flex items-center gap-1.5 border-amber-600/50 hover:border-amber-400"
      onclick={jumpToPartitions}
    >
      <HardDrive class="w-3.5 h-3.5 text-amber-400" />
      <span>{$LL.qualcomm.imei.backupPartitions()}</span>
      <ExternalLink class="w-3 h-3 text-muted" />
    </Button>
  </div>

  <!-- CURRENT IMEI CARDS (READ STATUS) -->
  <div class="p-4 bg-card border-2 border-border rounded-lg shadow-neobrutal-sm space-y-3">
    <div class="flex items-center justify-between border-b border-border pb-2.5">
      <div class="flex items-center gap-2">
        <Smartphone class="w-4 h-4 text-accent" />
        <span class="font-black uppercase tracking-wider text-foreground">{$LL.qualcomm.imei.currentTitle()}</span>
      </div>
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoadingInfo || isWriting}
        onclick={fetchCurrentIMEI}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoadingInfo ? 'animate-spin' : ''}" />
        <span>{$LL.qualcomm.imei.readImei()}</span>
      </Button>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <!-- SIM 1 / NV 550 -->
      <div class="p-3 bg-card-sub border border-border rounded-lg space-y-2">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1.5">
            <Badge variant="info">SIM 1</Badge>
            <span class="text-[11px] text-muted">{$LL.qualcomm.imei.primaryNv()}</span>
          </div>
          {#if imeiInfo?.imei1 && imeiInfo.imei1 !== 'Not Set'}
            <Badge variant="success">{$LL.common.active()}</Badge>
          {:else}
            <Badge variant="default">{$LL.qualcomm.imei.notPopulated()}</Badge>
          {/if}
        </div>

        <div class="flex items-center justify-between gap-2 p-2 bg-black/40 border border-border/80 rounded font-mono">
          <span class="text-sm font-bold text-emerald-400 tracking-widest truncate">
            {imeiInfo?.imei1 || (isLoadingInfo ? $LL.qualcomm.imei.reading() : $LL.qualcomm.imei.notSet())}
          </span>
          {#if imeiInfo?.imei1 && imeiInfo.imei1 !== 'Not Set'}
            <div class="flex items-center gap-1 shrink-0">
              <button
                type="button"
                class="p-1 hover:bg-card-sub rounded transition-colors text-muted hover:text-white"
                title={$LL.qualcomm.imei.copyImei({ slot: 1 })}
                onclick={() => copyToClipboard(imeiInfo!.imei1, 'IMEI 1')}
              >
                <Copy class="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                class="p-1 hover:bg-card-sub rounded transition-colors text-muted hover:text-accent"
                title={$LL.qualcomm.imei.loadIntoForm()}
                onclick={() => loadIntoForm(imeiInfo!.imei1, 0)}
              >
                <Wrench class="w-3.5 h-3.5" />
              </button>
            </div>
          {/if}
        </div>
      </div>

      <!-- SIM 2 / NV 2497 / 5014 -->
      <div class="p-3 bg-card-sub border border-border rounded-lg space-y-2">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1.5">
            <Badge variant="pop-purple">SIM 2</Badge>
            <span class="text-[11px] text-muted">{$LL.qualcomm.imei.secondaryNv()}</span>
          </div>
          {#if imeiInfo?.imei2 && imeiInfo.imei2 !== 'Not Set'}
            <Badge variant="success">{$LL.common.active()}</Badge>
          {:else}
            <Badge variant="default">{$LL.qualcomm.imei.notPopulated()}</Badge>
          {/if}
        </div>

        <div class="flex items-center justify-between gap-2 p-2 bg-black/40 border border-border/80 rounded font-mono">
          <span class="text-sm font-bold text-purple-400 tracking-widest truncate">
            {imeiInfo?.imei2 || (isLoadingInfo ? $LL.qualcomm.imei.reading() : $LL.qualcomm.imei.notSet())}
          </span>
          {#if imeiInfo?.imei2 && imeiInfo.imei2 !== 'Not Set'}
            <div class="flex items-center gap-1 shrink-0">
              <button
                type="button"
                class="p-1 hover:bg-card-sub rounded transition-colors text-muted hover:text-white"
                title={$LL.qualcomm.imei.copyImei({ slot: 2 })}
                onclick={() => copyToClipboard(imeiInfo!.imei2, 'IMEI 2')}
              >
                <Copy class="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                class="p-1 hover:bg-card-sub rounded transition-colors text-muted hover:text-accent"
                title={$LL.qualcomm.imei.loadIntoForm()}
                onclick={() => loadIntoForm(imeiInfo!.imei2, 1)}
              >
                <Wrench class="w-3.5 h-3.5" />
              </button>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>

  <!-- REWRITE CONFIGURATION & FORM -->
  <div class="p-4 bg-card border-2 border-border rounded-lg shadow-neobrutal-sm space-y-4">
    <div class="flex items-center justify-between border-b border-border pb-2.5">
      <div class="flex items-center gap-2">
        <Wrench class="w-4 h-4 text-accent" />
        <span class="font-black uppercase tracking-wider text-foreground">{$LL.qualcomm.imei.writeTitle()}</span>
      </div>
    </div>

    <!-- TARGET SLOT & METHOD SELECTION -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <!-- SLOT SELECTION -->
      <div class="space-y-1.5">
        <label for="target-slot-select" class="block text-[11px] font-bold uppercase text-muted tracking-wider">
          {$LL.qualcomm.imei.targetSlot()}
        </label>
        <div id="target-slot-select" class="grid grid-cols-2 gap-2">
          <button
            type="button"
            class="p-2.5 rounded border-2 text-center transition-all cursor-pointer {targetSlot === 0
              ? 'border-accent bg-accent/20 text-white font-black shadow-neobrutal-sm'
              : 'border-border bg-card-sub text-muted hover:text-foreground'}"
            onclick={() => (targetSlot = 0)}
          >
            <div class="text-xs">SIM 1</div>
            <div class="text-[10px] text-muted font-normal">{$LL.qualcomm.imei.primaryNv()}</div>
          </button>
          <button
            type="button"
            class="p-2.5 rounded border-2 text-center transition-all cursor-pointer {targetSlot === 1
              ? 'border-accent bg-accent/20 text-white font-black shadow-neobrutal-sm'
              : 'border-border bg-card-sub text-muted hover:text-foreground'}"
            onclick={() => (targetSlot = 1)}
          >
            <div class="text-xs">SIM 2</div>
            <div class="text-[10px] text-muted font-normal">{$LL.qualcomm.imei.secondaryNvShort()}</div>
          </button>
        </div>
      </div>

      <!-- METHOD SELECTION -->
      <div class="space-y-1.5">
        <label for="write-method-select" class="block text-[11px] font-bold uppercase text-muted tracking-wider">
          {$LL.qualcomm.imei.engine()}
        </label>
        <div id="write-method-select" class="grid grid-cols-3 gap-1.5">
          <button
            type="button"
            class="p-2 rounded border-2 text-center transition-all cursor-pointer {selectedMethod === 'auto'
              ? 'border-emerald-500 bg-emerald-950/40 text-emerald-300 font-bold'
              : 'border-border bg-card-sub text-muted hover:text-foreground'}"
            onclick={() => (selectedMethod = 'auto')}
          >
            <div class="text-[11px] font-bold">{$LL.qualcomm.imei.methodAuto()}</div>
            <div class="text-[9px] text-muted">{$LL.qualcomm.imei.methodAutoDesc()}</div>
          </button>

          <button
            type="button"
            class="p-2 rounded border-2 text-center transition-all cursor-pointer {selectedMethod === 'normal'
              ? 'border-cyan-500 bg-cyan-950/40 text-cyan-300 font-bold'
              : 'border-border bg-card-sub text-muted hover:text-foreground'}"
            onclick={() => (selectedMethod = 'normal')}
          >
            <div class="text-[11px] font-bold">{$LL.qualcomm.imei.methodNormal()}</div>
            <div class="text-[9px] text-muted">{$LL.qualcomm.imei.methodNormalDesc()}</div>
          </button>

          <button
            type="button"
            class="p-2 rounded border-2 text-center transition-all cursor-pointer {selectedMethod === 'efs'
              ? 'border-purple-500 bg-purple-950/40 text-purple-300 font-bold'
              : 'border-border bg-card-sub text-muted hover:text-foreground'}"
            onclick={() => (selectedMethod = 'efs')}
          >
            <div class="text-[11px] font-bold">{$LL.qualcomm.imei.methodEfs()}</div>
            <div class="text-[9px] text-muted">{$LL.qualcomm.imei.methodEfsDesc()}</div>
          </button>
        </div>
      </div>
    </div>

    <!-- IMEI INPUT & LUHN VALIDATOR -->
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <label for="imei-input" class="text-[11px] font-bold uppercase text-muted tracking-wider">
          {$LL.qualcomm.imei.newImeiLabel()}
        </label>
        {#if luhnStatus.state === 'valid'}
          <Badge variant="success" class="flex items-center gap-1">
            <Check class="w-3 h-3" />
            <span>{luhnStatus.message}</span>
          </Badge>
        {:else if luhnStatus.state === 'fixable' || luhnStatus.state === 'invalid'}
          <div class="flex items-center gap-1.5">
            <Badge variant="warning">{luhnStatus.message}</Badge>
            <Button variant="secondary" size="sm" class="py-0.5 px-2 text-[10px]" onclick={autoFixLuhn}>
              {$LL.qualcomm.imei.autoFix()}
            </Button>
          </div>
        {:else if luhnStatus.state === 'incomplete'}
          <Badge variant="default">{luhnStatus.message}</Badge>
        {:else if luhnStatus.state === 'too_long'}
          <Badge variant="danger">{luhnStatus.message}</Badge>
        {/if}
      </div>

      <div class="relative">
        <input
          id="imei-input"
          type="text"
          bind:value={inputIMEI}
          placeholder={$LL.qualcomm.imei.imeiPlaceholder()}
          maxlength={17}
          disabled={isWriting}
          class="neo-input w-full bg-card-sub border-2 border-border rounded px-3 py-2.5 text-base font-mono tracking-widest text-foreground placeholder:text-muted focus:outline-none focus:border-accent disabled:opacity-50 transition-colors"
        />
      </div>
    </div>

    <!-- SAFETY TOGGLES & ACTIONS -->
    <div class="pt-2 border-t border-border grid grid-cols-1 sm:grid-cols-2 gap-3">
      <label class="flex items-start gap-2.5 p-2 bg-card-sub border border-border rounded cursor-pointer hover:border-border/80">
        {#if themeStore.useSakugaControls()}
          <SakugaCheckbox bind:checked={useOfflineMode} disabled={isWriting} />
        {:else}
        <input
          type="checkbox"
          bind:checked={useOfflineMode}
          class="mt-1 neo-checkbox"
          disabled={isWriting}
        />
        {/if}
        <div class="text-[11px]">
          <span class="font-bold text-foreground block">{$LL.qualcomm.imei.offlineTitle()}</span>
          <span class="text-muted text-[10px]">
            {$LL.qualcomm.imei.offlineDesc()}
          </span>
        </div>
      </label>

      <label class="flex items-start gap-2.5 p-2 bg-card-sub border border-border rounded cursor-pointer hover:border-border/80">
        {#if themeStore.useSakugaControls()}
          <SakugaCheckbox bind:checked={autoReboot} disabled={isWriting} />
        {:else}
        <input
          type="checkbox"
          bind:checked={autoReboot}
          class="mt-1 neo-checkbox"
          disabled={isWriting}
        />
        {/if}
        <div class="text-[11px]">
          <span class="font-bold text-foreground block">{$LL.qualcomm.imei.autoRebootTitle()}</span>
          <span class="text-muted text-[10px]">
            {$LL.qualcomm.imei.autoRebootDesc()}
          </span>
        </div>
      </label>
    </div>

    <!-- ACTION BUTTONS -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-2">
      <div class="text-[11px] text-muted">
        {$LL.qualcomm.imei.methodLabel()} <span class="font-bold text-foreground uppercase">{selectedMethod}</span> |
        {$LL.qualcomm.imei.slotLabel()} <span class="font-bold text-foreground">SIM {targetSlot + 1}</span>
      </div>

      <div class="flex items-center gap-2">
        <Button
          variant="secondary"
          size="sm"
          disabled={isRestartingModem || isWriting}
          onclick={handleModemSSR}
        >
          <RotateCw class="w-3.5 h-3.5 mr-1.5 {isRestartingModem ? 'animate-spin' : ''}" />
          <span>{$LL.qualcomm.imei.restartModem()}</span>
        </Button>

        <Button
          variant="primary"
          size="sm"
          class="px-5 font-bold"
          disabled={isWriting || luhnStatus.state !== 'valid'}
          onclick={requestWrite}
        >
          <Wrench class="w-3.5 h-3.5 mr-1.5 {isWriting ? 'animate-spin' : ''}" />
          <span>{$LL.qualcomm.imei.applyWrite()}</span>
        </Button>
      </div>
    </div>
  </div>

  <!-- INTERACTIVE DIAG EXECUTION LOGS -->
  <div class="p-4 bg-card border-2 border-border rounded-lg shadow-neobrutal-sm space-y-2">
    <div class="flex items-center justify-between border-b border-border pb-2">
      <div class="flex items-center gap-2">
        <Terminal class="w-4 h-4 text-accent" />
        <span class="font-black uppercase tracking-wider text-foreground">{$LL.qualcomm.imei.logsTitle()}</span>
        {#if isWriting}
          <Badge variant="warning" class="animate-pulse">{$LL.qualcomm.imei.writing()}</Badge>
        {:else if lastResult?.success}
          <Badge variant="success">{$LL.toast.success()}</Badge>
        {:else if lastResult && !lastResult.success}
          <Badge variant="danger">{$LL.toast.error()}</Badge>
        {/if}
      </div>

      {#if executionLogs.length > 0}
        <div class="flex items-center gap-1.5">
          <Button
            variant="secondary"
            size="sm"
            class="text-[10px] py-0.5 px-2"
            onclick={() => copyToClipboard(executionLogs.join('\n'), 'Execution Logs')}
          >
            <Copy class="w-3 h-3 mr-1" />
            <span>{$LL.common.copy()}</span>
          </Button>
          <Button
            variant="secondary"
            size="sm"
            class="text-[10px] py-0.5 px-2"
            onclick={() => (executionLogs = [])}
          >
            <span>{$LL.common.clear()}</span>
          </Button>
        </div>
      {/if}
    </div>

    <div class="p-3 bg-black/70 border border-border/80 rounded-lg min-h-[140px] max-h-[260px] overflow-y-auto font-mono text-[11px] space-y-1">
      {#if executionLogs.length === 0}
        <div class="text-muted/60 italic py-6 text-center">
          {$LL.qualcomm.imei.noLogs()}
        </div>
      {:else}
        {#each executionLogs as log}
          <div class="leading-relaxed {log.includes('FAIL') || log.includes('rejected') || log.includes('FAILED')
            ? 'text-red-400 font-bold'
            : log.includes('SUCCESS') || log.includes('OK:')
              ? 'text-emerald-400 font-bold'
              : log.includes('WARN:')
                ? 'text-amber-300'
                : log.includes('INFO')
                  ? 'text-cyan-300'
                  : 'text-muted'}">
            {log}
          </div>
        {/each}
      {/if}
    </div>
  </div>
</div>

<!-- CONFIRMATION MODAL -->
<Modal bind:open={confirmModalOpen} title={$LL.qualcomm.imei.confirmTitle()}>
  <div class="space-y-4 font-mono text-xs">
    <div class="p-3 bg-amber-950/40 border border-amber-800 rounded text-amber-300 space-y-1.5">
      <div class="flex items-center gap-1.5 font-bold">
        <AlertTriangle class="w-4 h-4 text-amber-400 shrink-0" />
        <span>{$LL.qualcomm.imei.confirmWarning()}</span>
      </div>
      <p class="text-[11px] text-muted leading-relaxed">
        {$LL.qualcomm.imei.confirmBody()}
      </p>
    </div>

    <div class="p-3 bg-card-sub border border-border rounded space-y-2">
      <div class="flex justify-between">
        <span class="text-muted">{$LL.qualcomm.imei.confirmSlot()}</span>
        <span class="font-bold text-white">{$LL.qualcomm.imei.confirmSlotValue({ slot: targetSlot + 1, nv: targetSlot === 0 ? 'NV 550' : 'NV 2497/5014' })}</span>
      </div>
      <div class="flex justify-between">
        <span class="text-muted">{$LL.qualcomm.imei.confirmNewImei()}</span>
        <span class="font-bold text-emerald-400 font-mono text-sm tracking-wider">{cleanedIMEI}</span>
      </div>
      <div class="flex justify-between">
        <span class="text-muted">{$LL.qualcomm.imei.methodLabel()}</span>
        <span class="font-bold text-cyan-400 uppercase">{$LL.qualcomm.imei.confirmMethodValue({ method: selectedMethod })}</span>
      </div>
      <div class="flex justify-between">
        <span class="text-muted">{$LL.qualcomm.imei.confirmOffline()}</span>
        <span class="font-bold {useOfflineMode ? 'text-emerald-400' : 'text-muted'}">{useOfflineMode ? $LL.common.yes() : $LL.common.no()}</span>
      </div>
    </div>

    <div class="flex justify-end gap-2 pt-2 border-t border-border">
      <Button variant="secondary" size="sm" onclick={() => (confirmModalOpen = false)}>
        {$LL.common.cancel()}
      </Button>
      <Button variant="primary" size="sm" class="font-bold" onclick={executeWrite}>
        {$LL.qualcomm.imei.proceedWrite()}
      </Button>
    </div>
  </div>
</Modal>
