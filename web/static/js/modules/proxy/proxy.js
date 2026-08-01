const ProxyModule = {
    proxyData: {},
    proxyLogs: [],

    async fetchProxyStatus() {
        try {
            const res = await fetch('/api/proxy/status');
            if (res.ok) {
                this.proxyData = await res.json();
            }
        } catch (e) {}
    },

    async toggleProxyWatchdog() {
        const next = !this.proxyData.watchdog;
        try {
            await fetch('/api/proxy/watchdog', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ enable: next })
            });
            this.fetchProxyStatus();
        } catch (e) {}
    },

    async openYamlConfigEditor() {
        try {
            const res = await fetch('/api/proxy/config');
            if (res.ok) {
                const data = await res.json();
                this.editingFilePath = data.path;
                this.editorContent = data.content;
                this.showEditorModal = true;
            }
        } catch (e) {}
    },

    async controlProxy(action) {
        try {
            const res = await fetch('/api/proxy/control', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action })
            });
            if (res.ok) {
                this.showToast('Proxy Core Action', 'Instruction: ' + action + ' sent to proxy service.', 'info');
            } else {
                this.showToast('Proxy Error', 'Failed to control proxy core', 'error');
            }
            setTimeout(() => this.fetchProxyStatus(), 1500);
        } catch (e) {
            this.showToast('Proxy Error', 'Proxy control request failed', 'error');
        }
    },

    async setProxyMode(mode) {
        try {
            const res = await fetch('/api/proxy/control', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ mode })
            });
            if (res.ok) {
                this.showToast('Proxy Mode Config', 'Routing mode set to: ' + mode, 'success');
            } else {
                this.showToast('Proxy Error', 'Failed to update routing mode', 'error');
            }
            this.fetchProxyStatus();
        } catch (e) {
            this.showToast('Proxy Error', 'Proxy mode request failed', 'error');
        }
    },

    setupProxyLogs() {
        if (this.eventSource) this.eventSource.close();
        this.eventSource = new EventSource('/api/proxy/logs');
        this.eventSource.onmessage = (e) => {
            this.proxyLogs.push(e.data);
            if (this.proxyLogs.length > 200) this.proxyLogs.shift();
        };
    }
};
