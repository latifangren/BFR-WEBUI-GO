const ModemModule = {
    modemState: 'MODEM READY',
    signalRSRP: '-85',
    signalRSRQ: '-10',
    signalSINR: '22',
    carrierName: 'Telkomsel SELULAR',
    cellID: '2489102',
    pciVal: '142',
    selectedBands: ['B1', 'B3'],
    preferredMode: 'lte_only',
    atInput: '',
    atLogs: [
        '> AT+CSQ',
        '+CSQ: 26,99',
        '> AT+COPS?',
        '+COPS: 0,0,"Telkomsel",7'
    ],
    targetEarfcn: '',
    targetPCI: '',

    fetchModemSignal() {
        if (typeof showToast === 'function') {
            showToast('Memperbarui sinyal & status modem...', 'info');
        }
    },

    applyBandLock() {
        if (typeof showToast === 'function') {
            const bandsStr = this.selectedBands.join(', ');
            showToast(`Band Lock (${bandsStr || 'Auto'}) diterapkan ke modem!`, 'success');
        }
    },

    resetBandLock() {
        this.selectedBands = ['B1', 'B3', 'B5', 'B8', 'B40'];
        if (typeof showToast === 'function') {
            showToast('Reset Band Lock ke Auto All Bands.', 'info');
        }
    },

    applyNetworkMode() {
        if (typeof showToast === 'function') {
            showToast(`Preferred Network Mode diset ke ${this.preferredMode}`, 'success');
        }
    },

    sendATCommand(cmd) {
        this.atLogs.push('> ' + cmd);
        this.atLogs.push('OK');
        if (typeof showToast === 'function') {
            showToast(`Jalankan: ${cmd}`, 'info');
        }
    },

    executeATInput() {
        if (!this.atInput.trim()) return;
        this.sendATCommand(this.atInput.trim());
        this.atInput = '';
    },

    applyCellLock() {
        if (!this.targetEarfcn || !this.targetPCI) {
            if (typeof showToast === 'function') showToast('Isi EARFCN dan PCI terlebih dahulu!', 'error');
            return;
        }
        if (typeof showToast === 'function') {
            showToast(`Locking Cell Tower: EARFCN ${this.targetEarfcn}, PCI ${this.targetPCI}...`, 'success');
        }
    },

    unlockCell() {
        this.targetEarfcn = '';
        this.targetPCI = '';
        if (typeof showToast === 'function') {
            showToast('Cell Tower Unlocked (Auto Hopping)', 'info');
        }
    }
};
