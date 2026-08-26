const ModemModule = {
    modemState: 'MODEM READY',
    signalRSRP: '-99',
    signalRSRQ: '-12',
    signalSINR: '15',
    signalRSSI: '-80',
    qualityRSRP: 'Good',
    qualityRSRQ: 'Good',
    qualitySINR: 'Good',
    rsrpPct: 75,
    rsrqPct: 75,
    sinrPct: 75,
    carrierName: 'Searching...',
    cellID: '-',
    pciVal: '-',
    tacVal: '-',
    earfcnVal: '-',
    bandwidthVal: '20 MHz',
    selectedBands: ['B1', 'B3'],
    preferredMode: 'hybrid',
    engine: 'universal',
    atInput: '',
    atLogs: [
        '> AT+CSQ',
        '+CSQ: 26,99',
        '> AT+COPS?',
        '+COPS: 0,0,"Cellular",7'
    ],
    targetEarfcn: '',
    targetPCI: '',
    pollingInterval: null,

    initModem() {
        this.fetchModemSignal();
        this.loadBandConfig();
        this.startModemPolling();
    },

    startModemPolling() {
        if (this.pollingInterval) clearInterval(this.pollingInterval);
        this.pollingInterval = setInterval(() => {
            if (this.activeTab === 'modem') {
                this.fetchModemSignal();
            }
        }, 4000);
    },

    async fetchModemSignal() {
        try {
            const res = await fetch('/api/modem/signal');
            if (!res.ok) return;
            const data = await res.json();
            if (data) {
                this.signalRSRP = data.rsrp !== -999 ? String(data.rsrp) : 'N/A';
                this.signalRSRQ = data.rsrq !== -999 ? String(data.rsrq) : 'N/A';
                this.signalSINR = data.sinr !== -999 ? String(data.sinr) : 'N/A';
                this.signalRSSI = data.rssi !== -999 ? String(data.rssi) : 'N/A';
                this.carrierName = data.operator || 'Unknown Network';
                this.cellID = data.cell_id || '-';
                this.pciVal = data.pci || '-';
                this.tacVal = data.tac || '-';
                this.earfcnVal = data.earfcn ? String(data.earfcn) : '-';
                this.bandwidthVal = data.bandwidth || 'Auto';
                this.qualityRSRP = data.quality_rsrp || 'Good';
                this.qualityRSRQ = data.quality_rsrq || 'Good';
                this.qualitySINR = data.quality_sinr || 'Good';
                this.rsrpPct = data.rsrp_pct || 50;
                this.rsrqPct = data.rsrq_pct || 50;
                this.sinrPct = data.sinr_pct || 50;
                this.modemState = data.operator && data.operator !== 'Unknown' ? `${data.operator} (${data.network_type || 'LTE'})` : 'MODEM CONNECTED';
            }
        } catch (e) {
            console.error('Failed fetching modem signal', e);
        }
    },

    async loadBandConfig() {
        try {
            const res = await fetch('/api/modem/bands');
            if (!res.ok) return;
            const data = await res.json();
            if (data) {
                this.engine = data.engine || 'universal';
                this.preferredMode = data.preferred_rat || 'hybrid';
                const lteList = Array.isArray(data.lte_bands) ? data.lte_bands.map(b => 'B' + b) : [];
                const nrList = Array.isArray(data.nr_bands) ? data.nr_bands.map(b => 'n' + b) : [];
                if (lteList.length > 0 || nrList.length > 0) {
                    this.selectedBands = [...lteList, ...nrList];
                }
            }
        } catch (e) {}
    },

    async applyBandLock() {
        const lteNums = this.selectedBands.filter(b => b.startsWith('B')).map(b => parseInt(b.replace('B', ''), 10)).filter(n => !isNaN(n));
        const nrNums = this.selectedBands.filter(b => b.startsWith('n')).map(b => parseInt(b.replace('n', ''), 10)).filter(n => !isNaN(n));

        const payload = {
            engine: this.engine || 'universal',
            preferred_rat: this.preferredMode || 'hybrid',
            lte_bands: lteNums,
            nr_bands: nrNums,
            hex_bitmask: ''
        };

        try {
            const res = await fetch('/api/modem/bands', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            const data = await res.json();
            if (data && data.success) {
                if (typeof this.showToast === 'function') {
                    this.showToast('Band Lock Applied', `Bands [${this.selectedBands.join(', ') || 'Auto'}] applied successfully.`, 'success');
                }
                setTimeout(() => this.fetchModemSignal(), 1000);
            } else {
                if (typeof this.showToast === 'function') {
                    this.showToast('Band Lock Failed', data.error || 'Failed to apply band lock.', 'error');
                }
            }
        } catch (e) {
            if (typeof this.showToast === 'function') this.showToast('Network Error', 'Connection failed while applying band lock.', 'error');
        }
    },

    async resetBandLock() {
        try {
            const res = await fetch('/api/modem/reset', { method: 'POST' });
            const data = await res.json();
            if (data && data.success) {
                this.selectedBands = ['B1', 'B3', 'B5', 'B7', 'B8', 'B20', 'B28', 'B38', 'B40', 'B41', 'n1', 'n3', 'n5', 'n8', 'n28', 'n40', 'n41', 'n77', 'n78'];
                this.preferredMode = 'hybrid';
                if (typeof this.showToast === 'function') {
                    this.showToast('Modem Reset', 'Band lock reset to auto all bands.', 'info');
                }
                setTimeout(() => this.fetchModemSignal(), 1000);
            }
        } catch (e) {}
    },

    async sendATCommand(cmd) {
        if (!cmd) return;
        this.atLogs.push('> ' + cmd);
        try {
            const res = await fetch('/api/modem/at', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ command: cmd })
            });
            const data = await res.json();
            if (data && data.response) {
                this.atLogs.push(data.response);
            } else {
                this.atLogs.push('ERROR: Empty response');
            }
        } catch (e) {
            this.atLogs.push('ERROR: Connection failed');
        }
    },

    executeATInput() {
        if (!this.atInput.trim()) return;
        this.sendATCommand(this.atInput.trim());
        this.atInput = '';
    },

    applyCellLock() {
        if (!this.targetEarfcn || !this.targetPCI) {
            if (typeof this.showToast === 'function') this.showToast('Input Required', 'Please enter EARFCN and PCI.', 'error');
            return;
        }
        this.sendATCommand(`AT+QNWLOCK="common/4g",1,${this.targetEarfcn},${this.targetPCI}`);
    },

    unlockCell() {
        this.targetEarfcn = '';
        this.targetPCI = '';
        this.sendATCommand('AT+QNWLOCK="common/4g",0');
    }
};
