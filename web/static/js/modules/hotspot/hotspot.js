const HotspotModule = {
    hotspotStatus: {},
    hotspotPass: '',
    hotspotClients: [],
    macFilterMode: 'allow', // 'allow' (Whitelist) or 'deny' (Blacklist)
    macAddressInput: '',
    macFilterList: [
        { mac: 'AA:BB:CC:DD:EE:11', name: 'Authorized Laptop', mode: 'allow' },
        { mac: '11:22:33:44:55:66', name: 'Blocked Leech Device', mode: 'deny' }
    ],

    async fetchHotspotStatus() {
        try {
            const res = await fetch('/api/hotspot/status');
            if (res.ok) {
                this.hotspotStatus = await res.json();
            }
        } catch (e) {}
    },

    async toggleHotspot() {
        const next = !this.hotspotStatus.enabled;
        try {
            const res = await fetch('/api/hotspot/control', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ enable: next, ssid: this.hotspotStatus.ssid, password: this.hotspotPass })
            });
            if (res.ok) {
                this.showToast('Hotspot Access Point', next ? 'Starting softAP hotspot...' : 'Stopping hotspot...', 'info');
            } else {
                this.showToast('Hotspot Error', 'Failed to change hotspot status', 'error');
            }
            setTimeout(() => {
                this.fetchHotspotStatus();
                this.fetchHotspotClients();
            }, 2000);
        } catch (e) {
            this.showToast('Hotspot Error', 'Hotspot request failed', 'error');
        }
    },

    async fetchHotspotClients() {
        try {
            const res = await fetch('/api/hotspot/clients');
            if (res.ok) {
                this.hotspotClients = await res.json() || [];
            }
        } catch (e) {}
    },

    addMacFilterRule() {
        if (!this.macAddressInput.trim()) {
            if (typeof showToast === 'function') showToast('Hotspot MAC Filter', 'Ketik alamat MAC terlebih dahulu', 'error');
            return;
        }
        const macClean = this.macAddressInput.trim().toUpperCase();
        this.macFilterList.push({ mac: macClean, name: 'Custom Device', mode: this.macFilterMode });
        this.macAddressInput = '';
        if (typeof showToast === 'function') {
            showToast('Hotspot MAC Filter', `Aturan MAC ${macClean} (${this.macFilterMode}) ditambahkan`, 'success');
        }
    },

    removeMacFilterRule(index) {
        const removed = this.macFilterList.splice(index, 1);
        if (typeof showToast === 'function' && removed.length > 0) {
            showToast('Hotspot MAC Filter', `Aturan MAC ${removed[0].mac} dihapus`, 'info');
        }
    },

    blockClientMAC(mac, name) {
        this.macFilterList.push({ mac: mac, name: name || 'Blocked Client', mode: 'deny' });
        if (typeof showToast === 'function') {
            showToast('Hotspot MAC Filter', `Perangkat ${mac} ditambahkan ke Blacklist`, 'warning');
        }
    }
};
