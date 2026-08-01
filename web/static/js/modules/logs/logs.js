const LogsModule = {
    logsData: {
        entries: [],
        filter: '',
        _pollTimer: null
    },

    async fetchLogs() {
        try {
            const params = new URLSearchParams();
            if (this.logsData.filter) params.set('category', this.logsData.filter);
            params.set('limit', '200');
            const res = await fetch('/api/logs?' + params.toString());
            if (res.ok) {
                const data = await res.json();
                const prev = this.logsData.entries.length;
                this.logsData.entries = data.entries || [];
                if (this.logsData.entries.length !== prev) {
                    this.$nextTick(() => {
                        const el = document.getElementById('logs-container');
                        if (el) el.scrollTop = el.scrollHeight;
                    });
                }
            }
        } catch (e) {}
    },

    async clearLogs() {
        try {
            const res = await fetch('/api/logs/clear', { method: 'POST' });
            if (res.ok) {
                this.logsData.entries = [];
                this.showToast('Logs', 'Log entries cleared.', 'success');
            }
        } catch (e) {
            this.showToast('Logs', 'Failed to clear logs.', 'error');
        }
    },

    startLogsPolling() {
        if (this.logsData._pollTimer) clearInterval(this.logsData._pollTimer);
        this.logsData._pollTimer = setInterval(() => {
            if (this.activeTab === 'logs') {
                this.fetchLogs();
            }
        }, 5000);
    },

    stopLogsPolling() {
        if (this.logsData._pollTimer) {
            clearInterval(this.logsData._pollTimer);
            this.logsData._pollTimer = null;
        }
    }
};