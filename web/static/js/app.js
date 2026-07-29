function dashboard() {
    return {
        authenticated: false,
        loginPassword: '',
        authError: '',
        activeTab: 'overview',
        stats: {},
        networkData: {},
        proxyData: {},
        proxyLogs: [],
        pingHost: '1.1.1.1',
        pingOutput: '',
        pollTimer: null,
        term: null,
        termWS: null,
        eventSource: null,

        // File Manager State
        currentPath: '/sdcard',
        fileList: [],
        editingFilePath: '',
        editorContent: '',
        showEditorModal: false,
        showUploadModal: false,
        showDirModal: false,
        newDirName: '',

        // Hotspot & Extra State
        hotspotStatus: { enabled: false, ssid: 'AndroidAP' },
        hotspotPass: '',
        hotspotClients: [],
        rpsConfigs: [],

        modal: { show: false, action: '', actionName: '' },

        get currentPathSegments() {
            return this.currentPath.split('/').filter(Boolean);
        },

        async init() {
            await this.checkAuth();
            if (this.authenticated) {
                this.startPolling();
                this.fetchNetworkData();
                this.fetchProxyStatus();
                this.setupProxyLogs();
                this.fetchHotspotStatus();
                this.fetchHotspotClients();
                this.fetchRPSConfigs();
            }
        },

        async checkAuth() {
            try {
                const res = await fetch('/api/auth/status');
                const data = await res.json();
                this.authenticated = data.authenticated;
            } catch (e) {
                this.authenticated = false;
            }
        },

        async login() {
            this.authError = '';
            try {
                const res = await fetch('/api/auth/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ password: this.loginPassword })
                });
                const data = await res.json();
                if (res.ok && data.success) {
                    this.authenticated = true;
                    this.loginPassword = '';
                    this.startPolling();
                    this.fetchNetworkData();
                    this.fetchProxyStatus();
                    this.setupProxyLogs();
                    this.fetchHotspotStatus();
                    this.fetchHotspotClients();
                    this.fetchRPSConfigs();
                } else {
                    this.authError = data.error || 'Invalid password';
                }
            } catch (e) {
                this.authError = 'Connection failed';
            }
        },

        async logout() {
            await fetch('/api/auth/logout', { method: 'POST' });
            this.authenticated = false;
            if (this.pollTimer) clearInterval(this.pollTimer);
            if (this.eventSource) this.eventSource.close();
        },

        startPolling() {
            this.fetchStats();
            if (this.pollTimer) clearInterval(this.pollTimer);
            this.pollTimer = setInterval(() => this.fetchStats(), 2000);
        },

        async fetchStats() {
            try {
                const res = await fetch('/api/stats');
                if (res.status === 401) {
                    this.authenticated = false;
                    if (this.pollTimer) clearInterval(this.pollTimer);
                    return;
                }
                if (res.ok) {
                    this.stats = await res.json();
                }
            } catch (e) {}
        },

        // File Manager Methods
        async fetchFileList(path) {
            try {
                const res = await fetch('/api/files/list?path=' + encodeURIComponent(path || ''));
                if (res.ok) {
                    const data = await res.json();
                    this.currentPath = data.path;
                    this.fileList = data.files || [];
                }
            } catch (e) {}
        },

        navigateBreadcrumb(index) {
            const segments = this.currentPathSegments.slice(0, index + 1);
            const targetPath = '/' + segments.join('/');
            this.fetchFileList(targetPath);
        },

        async openEditor(filePath) {
            try {
                const res = await fetch('/api/files/read?path=' + encodeURIComponent(filePath));
                if (res.ok) {
                    const data = await res.json();
                    this.editingFilePath = data.path;
                    this.editorContent = data.content;
                    this.showEditorModal = true;
                } else {
                    alert('Cannot read file (may exceed 5MB or be binary)');
                }
            } catch (e) {}
        },

        async saveFileContent() {
            try {
                const res = await fetch('/api/files/save', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: this.editingFilePath, content: this.editorContent })
                });
                const data = await res.json();
                if (data.success) {
                    this.showEditorModal = false;
                    this.fetchFileList(this.currentPath);
                } else {
                    alert('Save failed: ' + data.error);
                }
            } catch (e) {}
        },

        async uploadFile() {
            const input = document.getElementById('upload-input');
            if (!input.files.length) return;

            const formData = new FormData();
            formData.append('path', this.currentPath);
            formData.append('file', input.files[0]);

            try {
                const res = await fetch('/api/files/upload', {
                    method: 'POST',
                    body: formData
                });
                const data = await res.json();
                if (data.success) {
                    this.showUploadModal = false;
                    this.fetchFileList(this.currentPath);
                } else {
                    alert('Upload failed: ' + data.error);
                }
            } catch (e) {}
        },

        openCreateDirModal() {
            this.newDirName = '';
            this.showDirModal = true;
        },

        async createDir() {
            if (!this.newDirName) return;
            const fullPath = this.currentPath + '/' + this.newDirName;
            try {
                const res = await fetch('/api/files/mkdir', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: fullPath })
                });
                const data = await res.json();
                if (data.success) {
                    this.showDirModal = false;
                    this.fetchFileList(this.currentPath);
                } else {
                    alert('Create directory failed: ' + data.error);
                }
            } catch (e) {}
        },

        async deletePath(targetPath) {
            if (!confirm('Are you sure you want to delete: ' + targetPath + '?')) return;
            try {
                const res = await fetch('/api/files/delete', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: targetPath })
                });
                const data = await res.json();
                if (data.success) {
                    this.fetchFileList(this.currentPath);
                } else {
                    alert('Delete failed: ' + data.error);
                }
            } catch (e) {}
        },

        // Network & Power Methods
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
                    alert('Tweaks configuration saved and applied.');
                    this.fetchNetworkData();
                } else {
                    alert('Failed to save tweaks: ' + data.error);
                }
            } catch (e) {
                alert('Save request failed');
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
                await fetch('/api/network/rps', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ interface: iface, bitmask: bitmask })
                });
                this.fetchRPSConfigs();
            } catch (e) {}
        },

        async toggleTTL() {
            const next = !this.networkData.ttl_spoof;
            try {
                await fetch('/api/network/tweaks?action=ttl', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ enable: next, ttl: 0 })
                });
                this.fetchNetworkData();
            } catch (e) {}
        },

        async setDNS(primary, secondary) {
            try {
                await fetch('/api/network/tweaks?action=dns', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ primary, secondary })
                });
                alert('DNS applied successfully');
            } catch (e) {}
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
                await fetch('/api/proxy/control', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ action })
                });
                setTimeout(() => this.fetchProxyStatus(), 1500);
            } catch (e) {}
        },

        async setProxyMode(mode) {
            try {
                await fetch('/api/proxy/control', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ mode })
                });
                this.fetchProxyStatus();
            } catch (e) {}
        },

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
                await fetch('/api/hotspot/control', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ enable: next, ssid: this.hotspotStatus.ssid, password: this.hotspotPass })
                });
                setTimeout(() => {
                    this.fetchHotspotStatus();
                    this.fetchHotspotClients();
                }, 2000);
            } catch (e) {}
        },

        async fetchHotspotClients() {
            try {
                const res = await fetch('/api/hotspot/clients');
                if (res.ok) {
                    this.hotspotClients = await res.json() || [];
                }
            } catch (e) {}
        },

        setupProxyLogs() {
            if (this.eventSource) this.eventSource.close();
            this.eventSource = new EventSource('/api/proxy/logs');
            this.eventSource.onmessage = (e) => {
                this.proxyLogs.push(e.data);
                if (this.proxyLogs.length > 200) this.proxyLogs.shift();
            };
        },

        initTerminal() {
            this.$nextTick(() => {
                if (this.term) return;
                const container = document.getElementById('terminal-container');
                if (!container) return;

                this.term = new Terminal({
                    theme: {
                        background: '#000000',
                        foreground: '#e5e7eb',
                        cursor: '#3b82f6'
                    },
                    fontFamily: 'monospace',
                    fontSize: 13
                });
                const fitAddon = new FitAddon.FitAddon();
                this.term.loadAddon(fitAddon);
                this.term.open(container);
                fitAddon.fit();

                this.reconnectTerminal();
            });
        },

        reconnectTerminal() {
            if (this.termWS) this.termWS.close();
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            this.termWS = new WebSocket(`${protocol}//${window.location.host}/api/terminal/ws`);

            this.termWS.onmessage = (e) => {
                if (this.term) this.term.write(e.data);
            };

            this.term.onData((data) => {
                if (this.termWS && this.termWS.readyState === WebSocket.OPEN) {
                    this.termWS.send(data);
                }
            });
        },

        confirmPower(action, name) {
            this.modal.action = action;
            this.modal.actionName = name;
            this.modal.show = true;
        },

        async executePowerAction() {
            const action = this.modal.action;
            this.modal.show = false;
            try {
                await fetch('/api/power', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ action })
                });
            } catch (e) {}
        },

        formatBytes(bytes) {
            if (!bytes) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
        },

        formatUptime(seconds) {
            if (!seconds) return '0s';
            const d = Math.floor(seconds / (3600*24));
            const h = Math.floor(seconds % (3600*24) / 365);
            const m = Math.floor(seconds % 3600 / 60);
            const s = Math.floor(seconds % 60);
            return (d > 0 ? d + "d " : "") + (h > 0 ? h + "h " : "") + (m > 0 ? m + "m " : "") + (s > 0 ? s + "s" : "");
        }
    }
}
