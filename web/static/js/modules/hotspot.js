const HotspotModule = {
    hotspotStatus: {},
    hotspotPass: '',
    hotspotClients: [],

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
    }
};
