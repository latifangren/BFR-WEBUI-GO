const VnstatModule = {
    vnstatData: {
        daily: { total: { rx_bytes: 0, tx_bytes: 0, total: 0 }, interfaces: {} },
        monthly: { total: { rx_bytes: 0, tx_bytes: 0, total: 0 }, interfaces: {} }
    },
    showVnstatResetModal: false,

    async fetchVnstatData() {
        try {
            const res = await fetch('/api/vnstat/stats');
            if (res.ok) {
                this.vnstatData = await res.json();
            }
        } catch (e) {}
    },

    async resetVnstatData() {
        this.showVnstatResetModal = false;
        try {
            const res = await fetch('/api/vnstat/reset', { method: 'POST' });
            if (res.ok) {
                this.vnstatData = await res.json();
                this.showToast('Reset Success', 'Bandwidth logs database cleared.', 'success');
            } else {
                this.showToast('Reset Error', 'Failed to reset traffic logs.', 'error');
            }
        } catch (e) {
            this.showToast('Reset Error', 'Reset request failed.', 'error');
        }
    },

    get vnstatInterfaceList() {
        return this.getVnstatInterfaces();
    },

    getVnstatInterfaces() {
        const dailyIfaces = this.vnstatData?.daily?.interfaces || {};
        const monthlyIfaces = this.vnstatData?.monthly?.interfaces || {};

        const names = new Set([...Object.keys(dailyIfaces), ...Object.keys(monthlyIfaces)]);
        const list = [];

        names.forEach(name => {
            const d = dailyIfaces[name] || { rx_bytes: 0, tx_bytes: 0 };
            const m = monthlyIfaces[name] || { rx_bytes: 0, tx_bytes: 0 };
            let type = 'Unknown';
            if (name.startsWith('wlan') || name.startsWith('ap')) {
                type = 'WLAN / Wi-Fi Hotspot';
            } else if (name.startsWith('rmnet') || name.startsWith('ccmni')) {
                type = 'LTE / Mobile Data';
            }
            list.push({
                name: name,
                type: type,
                dailyRx: d.rx_bytes,
                dailyTx: d.tx_bytes,
                dailyTotal: d.rx_bytes + d.tx_bytes,
                monthlyRx: m.rx_bytes,
                monthlyTx: m.tx_bytes,
                monthlyTotal: m.rx_bytes + m.tx_bytes
            });
        });

        return list.sort((a, b) => b.monthlyTotal - a.monthlyTotal);
    }
};
