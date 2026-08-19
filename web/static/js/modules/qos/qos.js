const QoSModule = {
    qosEnabled: true,
    qosInterface: 'wlan0',
    qosAlgorithm: 'cake',
    qosDownLimit: 100,
    qosUpLimit: 20,
    qosTargetRtt: '15ms',
    qosActiveQueues: '3 Flow Queues',
    qosDroppedPackets: 0,
    showAddRuleModal: false,

    fetchQoSStatus() {
        if (typeof showToast === 'function') {
            showToast('Memperbarui status QoS Kernel...', 'info');
        }
    },

    toggleQoSEngine() {
        this.qosEnabled = !this.qosEnabled;
        if (typeof showToast === 'function') {
            const stateMsg = this.qosEnabled ? 'QoS CAKE Engine diaktifkan' : 'QoS Engine dinonaktifkan';
            showToast(stateMsg, this.qosEnabled ? 'success' : 'warning');
        }
    },

    saveQoSConfig() {
        if (typeof showToast === 'function') {
            showToast(`Batas SQM (${this.qosDownLimit}M/${this.qosUpLimit}M) berhasil diterapkan di ${this.qosInterface}!`, 'success');
        }
    },

    applyQoSPreset(presetName) {
        if (presetName === 'gaming') {
            this.qosDownLimit = 80;
            this.qosUpLimit = 15;
            this.qosTargetRtt = '10ms';
            if (typeof showToast === 'function') showToast('Preset Ultra-Low Latency Gaming diterapkan!', 'success');
        } else if (presetName === 'streaming') {
            this.qosDownLimit = 120;
            this.qosUpLimit = 25;
            this.qosTargetRtt = '20ms';
            if (typeof showToast === 'function') showToast('Preset Media Streaming diterapkan!', 'success');
        } else if (presetName === 'balanced') {
            this.qosDownLimit = 100;
            this.qosUpLimit = 20;
            this.qosTargetRtt = '15ms';
            if (typeof showToast === 'function') showToast('Preset Fair-Share Balance diterapkan!', 'success');
        }
    }
};
