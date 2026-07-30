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
        } catch (e) {}
    },

    formatUptime(seconds) {
        if (!seconds) return '0s';
        const d = Math.floor(seconds / (3600*24));
        const h = Math.floor(seconds % (3600*24) / 3600);
        const m = Math.floor(seconds % 3600 / 60);
        const s = Math.floor(seconds % 60);
        return (d > 0 ? d + "d " : "") + (h > 0 ? h + "h " : "") + (m > 0 ? m + "m " : "") + (s > 0 ? s + "s" : "");
    }
};
