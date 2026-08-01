const LogcatModule = {
    logcatSocket: null,
    logcatEntries: [],
    logcatFilter: 'ALL',
    logcatSearch: '',
    logcatAutoscroll: true,

    initLogcat() {
        if (this.logcatSocket) return;
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/logs/logcat/stream`;

        try {
            this.logcatSocket = new WebSocket(wsUrl);

            this.logcatSocket.onopen = () => {
                this.showToast('Logcat', 'Live Logcat stream connected', 'info');
            };

            this.logcatSocket.onmessage = (evt) => {
                const line = evt.data || '';
                if (!line) return;

                let level = 'INFO';
                if (line.includes(' D ') || line.includes('D/')) level = 'DEBUG';
                else if (line.includes(' I ') || line.includes('I/')) level = 'INFO';
                else if (line.includes(' W ') || line.includes('W/')) level = 'WARN';
                else if (line.includes(' E ') || line.includes('E/')) level = 'ERROR';

                this.logcatEntries.push({
                    raw: line,
                    level: level,
                    time: new Date().toLocaleTimeString()
                });

                if (this.logcatEntries.length > 500) {
                    this.logcatEntries.shift();
                }

                if (this.logcatAutoscroll) {
                    this.$nextTick(() => {
                        const container = document.getElementById('logcat-stream-container');
                        if (container) container.scrollTop = container.scrollHeight;
                    });
                }
            };

            this.logcatSocket.onclose = () => {
                this.logcatSocket = null;
            };

            this.logcatSocket.onerror = () => {
                this.logcatSocket = null;
            };
        } catch (e) {
            this.showToast('Logcat Error', 'Failed to connect to Logcat WebSocket', 'error');
        }
    },

    stopLogcat() {
        if (this.logcatSocket) {
            this.logcatSocket.close();
            this.logcatSocket = null;
        }
    },

    clearLogcat() {
        this.logcatEntries = [];
    },

    filteredLogcatEntries() {
        let list = this.logcatEntries;
        if (this.logcatFilter !== 'ALL') {
            list = list.filter(e => e.level === this.logcatFilter);
        }
        if (this.logcatSearch) {
            const q = this.logcatSearch.toLowerCase();
            list = list.filter(e => e.raw.toLowerCase().includes(q));
        }
        return list;
    }
};
