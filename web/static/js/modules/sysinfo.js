const SysinfoModule = {
    stats: {},
    lastNetRx: 0,
    lastNetTx: 0,
    netSpeedRx: 0,
    netSpeedTx: 0,
    throughputHistory: [],
    pollTimer: null,
    eventSource: null,

    startPolling() {
        this.fetchStats();
        if (this.pollTimer) clearInterval(this.pollTimer);
        this.pollTimer = setInterval(() => this.fetchStats(), 2000);
    },

    async fetchStats() {
        try {
            const res = await fetch('/api/stats');
            if (res.status === 401) {
                this.authenticated = false;
                if (this.pollTimer) clearInterval(this.pollTimer);
                return;
            }
            if (res.ok) {
                const data = await res.json();
                if (this.lastNetRx > 0 && this.lastNetTx > 0) {
                    const rxDiff = data.net_rx - this.lastNetRx;
                    const txDiff = data.net_tx - this.lastNetTx;
                    if (rxDiff >= 0 && txDiff >= 0) {
                        this.netSpeedRx = rxDiff / 2;
                        this.netSpeedTx = txDiff / 2;
                        this.throughputHistory.push({ rx: this.netSpeedRx, tx: this.netSpeedTx });
                        if (this.throughputHistory.length > 20) {
                            this.throughputHistory.shift();
                        }
                    }
                }
                this.lastNetRx = data.net_rx;
                this.lastNetTx = data.net_tx;
                this.stats = data;
            }
            this.fetchVnstatData();
            this.fetchChargerConfig();
            this.fetchSSHStatus();
        } catch (e) {}
    },

    formatUptime(seconds) {
        if (!seconds) return '0s';
        const d = Math.floor(seconds / (3600*24));
        const h = Math.floor(seconds % (3600*24) / 3600);
        const m = Math.floor(seconds % 3600 / 60);
        const s = Math.floor(seconds % 60);
        return (d > 0 ? d + "d " : "") + (h > 0 ? h + "h " : "") + (m > 0 ? m + "m " : "") + (s > 0 ? s + "s" : "");
    },

    formatBatteryCurrent(ua) {
        if (ua === null || ua === undefined) return '—';
        const mA = ua / 1000;
        return (mA > 0 ? '+' : '') + mA.toFixed(1) + ' mA';
    },

    formatBatteryVoltage(uv) {
        if (uv === null || uv === undefined) return '—';
        if (Math.abs(uv) > 10000) {
            return (uv / 1000000).toFixed(2) + ' V';
        }
        return uv + ' mV';
    },

    // ── Network Detail Helpers ──────────────────────────────
    primaryNetworkIP() {
        const nd = this.stats?.network_detail;
        if (nd && nd.ip_addresses && nd.ip_addresses.length > 0) {
            return nd.ip_addresses[0];
        }
        // Fallback: first interface IP from networkData
        const ifaces = this.networkData?.interfaces;
        if (ifaces && Array.isArray(ifaces)) {
            for (const iface of ifaces) {
                if (iface.addresses && iface.addresses.length > 0) {
                    return iface.addresses[0];
                }
            }
        }
        return '—';
    },

    formatNetworkList(list) {
        if (!list || !Array.isArray(list) || list.length === 0) return '—';
        return list.join(', ');
    },

    signalQualityBars(score) {
        if (score === null || score === undefined || score === '') return '—';
        const s = parseFloat(score);
        if (isNaN(s)) return '—';
        const filled = Math.max(0, Math.min(5, Math.round(s)));
        const empty = 5 - filled;
        return '▮'.repeat(filled) + '▯'.repeat(empty) + ' (' + s.toFixed(1) + ')';
    },

    networkCardSubtext() {
        const nd = this.stats?.network_detail;
        if (!nd) return 'Network Detail';
        const parts = [];
        if (nd.wifi_ssid) parts.push(nd.wifi_ssid);
        else parts.push('Network');
        if (nd.dns && nd.dns.length > 0) parts.push(nd.dns.length + ' DNS');
        if (nd.roaming && nd.roaming.toLowerCase() !== 'no' && nd.roaming !== '') parts.push('Roaming');
        return parts.join(' · ') || 'Network Detail';
    }
};
