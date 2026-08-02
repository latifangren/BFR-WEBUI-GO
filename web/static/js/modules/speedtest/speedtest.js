const SpeedtestModule = {
    speedtest: {
        running: false,
        phase: 'idle',
        progress: 0,
        ping: 0,
        jitter: 0,
        download: 0,
        upload: 0,
        client_ip: '',
        isp: '',
        location: '',
        server_colo: '',
        server_name: '',
        history: []
    },
    speedtestTimer: null,

    async fetchSpeedtestStatus() {
        try {
            const res = await fetch('/api/speedtest/status');
            const data = await res.json();
            if (data) {
                this.speedtest.running = !!data.running;
                this.speedtest.phase = data.phase || (data.running ? 'testing' : 'idle');
                this.speedtest.progress = data.progress_pct !== undefined ? data.progress_pct : (data.progress || 0);
                this.speedtest.ping = data.ping_ms !== undefined ? data.ping_ms : (data.ping || 0);
                this.speedtest.jitter = data.jitter_ms !== undefined ? data.jitter_ms : (data.jitter || 0);
                this.speedtest.download = data.download_mbps !== undefined ? data.download_mbps : (data.download || 0);
                this.speedtest.upload = data.upload_mbps !== undefined ? data.upload_mbps : (data.upload || 0);

                this.speedtest.client_ip = data.client_ip || '';
                this.speedtest.isp = data.isp || '';
                this.speedtest.location = data.location || '';
                this.speedtest.server_colo = data.server_colo || '';
                this.speedtest.server_name = data.server_name || '';

                if (data.history) {
                    this.speedtest.history = data.history;
                }
            }

            if (this.speedtest.running && !this.speedtestTimer) {
                this.speedtestTimer = setInterval(() => this.fetchSpeedtestStatus(), 500);
            } else if (!this.speedtest.running && this.speedtestTimer) {
                clearInterval(this.speedtestTimer);
                this.speedtestTimer = null;
                this.fetchSpeedtestHistory();
            }
        } catch (e) {
            console.error('Failed to fetch speedtest status', e);
        }
    },

    async fetchSpeedtestHistory() {
        try {
            const res = await fetch('/api/speedtest/history');
            const data = await res.json();
            if (data && Array.isArray(data.history)) {
                this.speedtest.history = data.history;
            } else if (Array.isArray(data)) {
                this.speedtest.history = data;
            }
        } catch (e) {
            console.error('Failed to fetch speedtest history', e);
        }
    },

    async startSpeedtest() {
        try {
            const res = await fetch('/api/speedtest/start', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            const data = await res.json();
            if (res.ok && data.success !== false) {
                this.showToast('Speedtest Started', 'Running multi-threaded benchmark...', 'info');
                this.speedtest.running = true;
                this.speedtest.phase = 'ping';
                this.speedtest.progress = 5;
                if (this.speedtestTimer) {
                    clearInterval(this.speedtestTimer);
                }
                this.speedtestTimer = setInterval(() => this.fetchSpeedtestStatus(), 500);
                this.fetchSpeedtestStatus();
            } else {
                this.showToast('Speedtest Error', data.error || 'Failed to start speedtest', 'error');
            }
        } catch (e) {
            this.showToast('Speedtest Error', 'Connection failed', 'error');
        }
    },

    async stopSpeedtest() {
        try {
            const res = await fetch('/api/speedtest/stop', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            const data = await res.json();
            if (res.ok && data.success !== false) {
                this.showToast('Speedtest Stopped', 'Benchmark test cancelled.', 'info');
                this.speedtest.running = false;
                this.speedtest.phase = 'stopped';
                if (this.speedtestTimer) {
                    clearInterval(this.speedtestTimer);
                    this.speedtestTimer = null;
                }
            } else {
                this.showToast('Stop Error', data.error || 'Failed to stop speedtest', 'error');
            }
        } catch (e) {
            this.showToast('Stop Error', 'Connection failed', 'error');
        }
    }
};