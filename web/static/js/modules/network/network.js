const NetworkModule = {
    networkData: {},
    rpsConfigs: [],
    pingHost: '1.1.1.1',
    pingOutput: '',
    ttlValue: 64,
    showAllRpsInterfaces: false,

    async fetchNetworkData() {
        try {
            const res = await fetch('/api/network/tweaks');
            if (res.ok) {
                this.networkData = await res.json();
            }
        } catch (e) {}
    },

    async saveTweakConfig() {
        try {
            const res = await fetch('/api/network/tweaks?action=save_tweaks', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(this.networkData.tweaks_json)
            });
            const data = await res.json();
            if (data.success) {
                this.showToast('Tweaks Saved', 'System optimizations updated successfully!', 'success');
                this.fetchNetworkData();
            } else {
                this.showToast('Save Error', 'Failed to save tweaks: ' + data.error, 'error');
            }
        } catch (e) {
            this.showToast('Save Error', 'Save request failed', 'error');
        }
    },

    async restoreSysctlDefaults() {
        try {
            const res = await fetch('/api/network/tweaks/restore', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            const data = await res.json();
            if (res.ok && data.success) {
                this.showToast('Defaults Restored', 'Original sysctl defaults restored successfully!', 'success');
                this.fetchNetworkData();
            } else {
                this.showToast('Restore Error', data.error || 'Failed to restore sysctl defaults', 'error');
            }
        } catch (e) {
            this.showToast('Restore Error', 'Request failed', 'error');
        }
    },

    async fetchRPSConfigs() {
        try {
            const res = await fetch('/api/network/rps');
            if (res.ok) {
                const data = await res.json();
                this.rpsConfigs = data.configs || [];
            }
        } catch (e) {}
    },

    async setRPS(iface, bitmask) {
        try {
            const res = await fetch('/api/network/rps', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ interface: iface, bitmask: bitmask })
            });
            if (res.ok) {
                this.showToast('RPS Updated', 'Packet steering bitmask configured!', 'success');
            } else {
                this.showToast('RPS Error', 'Failed to update RPS configuration', 'error');
            }
            this.fetchRPSConfigs();
        } catch (e) {
            this.showToast('RPS Error', 'RPS request failed', 'error');
        }
    },

    async applyTTLConfig(enable) {
        try {
            const res = await fetch('/api/network/tweaks?action=ttl', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ enable: enable, ttl: parseInt(this.ttlValue || 64) })
            });
            if (res.ok) {
                this.showToast('TTL Updated', 'Target TTL configured successfully!', 'success');
            } else {
                this.showToast('TTL Error', 'Failed to configure TTL', 'error');
            }
            this.fetchNetworkData();
        } catch (e) {
            this.showToast('TTL Error', 'TTL request failed', 'error');
        }
    },

    async setDNS(primary, secondary) {
        try {
            const res = await fetch('/api/network/tweaks?action=dns', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ primary, secondary })
            });
            if (res.ok) {
                this.showToast('DNS Updated', 'System DNS changed successfully!', 'success');
            } else {
                this.showToast('DNS Error', 'Failed to update DNS resolver', 'error');
            }
        } catch (e) {
            this.showToast('DNS Error', 'DNS request failed', 'error');
        }
    },

    async runPing() {
        this.pingOutput = 'Pinging...';
        try {
            const res = await fetch('/api/network/ping', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ host: this.pingHost, count: 4 })
            });
            const data = await res.json();
            this.pingOutput = data.output;
        } catch (e) {
            this.pingOutput = 'Ping failed';
        }
    },

    isPhysicalInterface(name) {
        if (!name) return false;
        const lower = name.toLowerCase();
        return lower.startsWith('wlan') ||
               lower.startsWith('rmnet') ||
               lower.startsWith('r_rmnet') ||
               lower.startsWith('rndis') ||
               lower.startsWith('eth') ||
               lower.startsWith('p2p');
    },

    getRPSBitmaskLabel(bitmask) {
        if (!bitmask) return 'Disabled';
        const bm = bitmask.toLowerCase().trim();
        if (bm === '00') return 'Disabled';
        if (bm === 'cc') return 'Qualcomm / Vendor Default (Core 2,3,6,7)';
        if (bm === '02') return 'Core 1 Dedicated';
        if (bm === '0f') return 'Core 0-3 (Efficiency)';
        if (bm === 'f0') return 'Core 4-7 (Performance)';
        if (bm === 'ff') return 'All Cores (Max Throughput)';
        return `Custom Hex (0x${bitmask})`;
    }
};
