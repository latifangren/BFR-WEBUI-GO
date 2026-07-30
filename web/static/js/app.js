function dashboard() {
    return {
        authenticated: false,
        loginPassword: '',
        authError: '',
        activeTab: 'overview',
        isDark: localStorage.getItem('theme') !== 'light',
        stats: {},
        lastNetRx: 0,
        lastNetTx: 0,
        netSpeedRx: 0,
        netSpeedTx: 0,
        throughputHistory: [],
        fileManagerShortcuts: [],
        showAddShortcutModal: false,
        newShortcutName: '',
        newShortcutPath: '',
        showFileModal: false,
        newFileName: '',
        showRenameModal: false,
        renameOldPath: '',
        renameNewName: '',
        confirmModal: { show: false, title: 'Confirmation', message: '', onConfirm: null },
        toasts: [],
        donorName: '',
        donationAmount: '',
        donorMessage: '',
        ttlValue: 64,
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

        // Smart Charger State
        chargerConfig: {
            config: { enabled: false, start_percent: 70, stop_percent: 80 },
            detected_path: '',
            detected_type: '',
            charging_disabled: false,
            current_level: -1,
            logs: []
        },

        // SMS Viewer State
        smsList: [],
        smsTotal: 0,
        smsLimit: 20,
        smsOffset: 0,
        smsSearchQuery: '',
        isLoadingSms: false,

        // Remote Screen (Scrcpy) State
        scrcpyWs: null,
        isMirroring: false,
        screenRate: 0,
        scrcpyTextInput: '',
        scrcpyImgUrl: '',
        scrcpyLastFrameTime: 0,
        scrcpySwipeStart: null,

        // Vnstat Bandwidth State
        vnstatData: {
            daily: { total: { rx_bytes: 0, tx_bytes: 0, total: 0 }, interfaces: {} },
            monthly: { total: { rx_bytes: 0, tx_bytes: 0, total: 0 }, interfaces: {} }
        },
        showVnstatResetModal: false,

        modal: { show: false, action: '', actionName: '' },

        get currentPathSegments() {
            return this.currentPath.split('/').filter(Boolean);
        },

        get vnstatInterfaceList() {
            if (!this.vnstatData) return [];
            const dailyIfaces = (this.vnstatData.daily && this.vnstatData.daily.interfaces) || {};
            const monthlyIfaces = (this.vnstatData.monthly && this.vnstatData.monthly.interfaces) || {};

            const names = new Set([...Object.keys(dailyIfaces), ...Object.keys(monthlyIfaces)]);
            const list = [];

            names.forEach(name => {
                const d = dailyIfaces[name] || { rx_bytes: 0, tx_bytes: 0 };
                const m = monthlyIfaces[name] || { rx_bytes: 0, tx_bytes: 0 };
                let type = 'Other';
                const lower = name.toLowerCase();
                if (lower.startsWith('wlan') || lower.startsWith('ap') || lower.startsWith('softap') || lower.startsWith('swlan')) {
                    type = 'WLAN';
                } else if (lower.startsWith('rmnet') || lower.startsWith('ccmni') || lower.startsWith('pdp') || lower.startsWith('wwan') || lower.startsWith('v4-rmnet')) {
                    type = 'LTE / Mobile Data';
                }
                list.push({
                    name: name,
                    type: type,
                    dailyRx: d.rx_bytes,
                    dailyTx: d.tx_bytes,
                    dailyTotal: d.rx_bytes + d.tx_bytes,
                    monthlyRx: m.rx_bytes,
                    monthlyTx: m.tx_bytes,
                    monthlyTotal: m.rx_bytes + m.tx_bytes
                });
            });

            return list.sort((a, b) => b.monthlyTotal - a.monthlyTotal);
        },

        async init() {
            this.applyTheme();
            this.initShortcuts();
            await this.checkAuth();
            if (this.authenticated) {
                this.startPolling();
                this.fetchNetworkData();
                this.fetchProxyStatus();
                this.setupProxyLogs();
                this.fetchHotspotStatus();
                this.fetchHotspotClients();
                this.fetchRPSConfigs();
                this.fetchVnstatData();
                this.fetchChargerConfig();
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
                    this.fetchVnstatData();
                    this.fetchChargerConfig();
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
                    const data = await res.json();
                    if (this.lastNetRx > 0 && this.lastNetTx > 0) {
                        const rxDiff = data.net_rx - this.lastNetRx;
                        const txDiff = data.net_tx - this.lastNetTx;
                        if (rxDiff >= 0 && txDiff >= 0) {
                            this.netSpeedRx = rxDiff / 2;
                            this.netSpeedTx = txDiff / 2;
                            this.throughputHistory.push({ rx: this.netSpeedRx, tx: this.netSpeedTx });
                            if (this.throughputHistory.length > 20) {
                                this.throughputHistory.shift();
                            }
                        }
                    }
                    this.lastNetRx = data.net_rx;
                    this.lastNetTx = data.net_tx;
                    this.stats = data;
                }
                this.fetchVnstatData();
                this.fetchChargerConfig();
            } catch (e) {}
        },

        async fetchChargerConfig() {
            try {
                const res = await fetch('/api/charger/config');
                if (res.ok) {
                    this.chargerConfig = await res.json();
                }
            } catch (e) {}
        },

        async saveChargerConfig() {
            try {
                const res = await fetch('/api/charger/toggle', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(this.chargerConfig.config)
                });
                if (res.ok) {
                    this.chargerConfig = await res.json();
                    this.showToast('Charger Limiter', 'Battery charge settings updated!', 'success');
                } else {
                    this.showToast('Charger Error', 'Failed to update battery settings.', 'error');
                }
            } catch (e) {
                this.showToast('Charger Error', 'Battery settings request failed.', 'error');
            }
        },

        async fetchVnstatData() {
            try {
                const res = await fetch('/api/vnstat/stats');
                if (res.ok) {
                    this.vnstatData = await res.json();
                }
            } catch (e) {}
        },

        async resetVnstatData() {
            this.showVnstatResetModal = false;
            try {
                const res = await fetch('/api/vnstat/reset', { method: 'POST' });
                if (res.ok) {
                    this.vnstatData = await res.json();
                    this.showToast('Reset Success', 'Bandwidth logs database cleared.', 'success');
                } else {
                    this.showToast('Reset Error', 'Failed to reset traffic logs.', 'error');
                }
            } catch (e) {
                this.showToast('Reset Error', 'Reset request failed.', 'error');
            }
        },

        // SMS Viewer Methods
        async fetchSMS(newSearch = false) {
            if (newSearch) {
                this.smsOffset = 0;
            }
            this.isLoadingSms = true;
            try {
                const query = new URLSearchParams({
                    limit: this.smsLimit,
                    offset: this.smsOffset,
                    q: this.smsSearchQuery || ''
                });
                const res = await fetch('/api/sms/inbox?' + query.toString());
                if (res.ok) {
                    const data = await res.json();
                    this.smsList = data.messages || [];
                    this.smsTotal = data.total || 0;
                }
            } catch (e) {
            } finally {
                this.isLoadingSms = false;
            }
        },

        nextSMSPage() {
            if (this.smsOffset + this.smsLimit < this.smsTotal) {
                this.smsOffset += this.smsLimit;
                this.fetchSMS();
            }
        },

        prevSMSPage() {
            if (this.smsOffset >= this.smsLimit) {
                this.smsOffset -= this.smsLimit;
                this.fetchSMS();
            } else {
                this.smsOffset = 0;
                this.fetchSMS();
            }
        },

        formatSMSDate(timestamp) {
            if (!timestamp) return 'N/A';
            let t = timestamp;
            if (t < 10000000000) {
                t = t * 1000;
            }
            const date = new Date(t);
            return date.toLocaleString();
        },

        formatSMSBody(body) {
            if (!body) return '';
            // Escape HTML characters
            let safe = body.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
            // Highlight OTP/verification code phrases (4-8 digits or keywords)
            const regex = /\b(\d{4,8})\b/g;
            return safe.replace(regex, '<mark class="bg-[#f59e0b]/20 px-1 text-[#f59e0b] rounded font-bold border-b border-[#f59e0b]/30 font-mono">$1</mark>');
        },

        // Remote Screen (Scrcpy) Methods
        startMirroring() {
            if (this.isMirroring) return;
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/api/scrcpy/ws`;

            this.scrcpyWs = new WebSocket(wsUrl);
            this.scrcpyWs.binaryType = 'blob';

            this.scrcpyWs.onopen = () => {
                this.isMirroring = true;
                this.scrcpyLastFrameTime = Date.now();
            };

            this.scrcpyWs.onmessage = (event) => {
                if (event.data instanceof Blob) {
                    const now = Date.now();
                    if (this.scrcpyLastFrameTime > 0) {
                        const delta = now - this.scrcpyLastFrameTime;
                        if (delta > 0) {
                            this.screenRate = (1000 / delta).toFixed(1);
                        }
                    }
                    this.scrcpyLastFrameTime = now;

                    const url = URL.createObjectURL(event.data);
                    if (this.scrcpyImgUrl) {
                        URL.revokeObjectURL(this.scrcpyImgUrl);
                    }
                    this.scrcpyImgUrl = url;
                }
            };

            this.scrcpyWs.onclose = () => {
                this.isMirroring = false;
                this.screenRate = 0;
            };

            this.scrcpyWs.onerror = () => {
                this.isMirroring = false;
                this.screenRate = 0;
            };
        },

        stopMirroring() {
            if (this.scrcpyWs) {
                this.scrcpyWs.close();
                this.scrcpyWs = null;
            }
            if (this.scrcpyImgUrl) {
                URL.revokeObjectURL(this.scrcpyImgUrl);
                this.scrcpyImgUrl = '';
            }
            this.isMirroring = false;
            this.screenRate = 0;
        },

        sendScrcpyEvent(evt) {
            if (this.scrcpyWs && this.scrcpyWs.readyState === WebSocket.OPEN) {
                this.scrcpyWs.send(JSON.stringify(evt));
            }
        },

        sendKey(action, keycode = 0) {
            this.sendScrcpyEvent({ action: action, keycode: keycode });
        },

        sendScrcpyText() {
            if (!this.scrcpyTextInput) return;
            this.sendScrcpyEvent({ action: 'text', text: this.scrcpyTextInput });
            this.scrcpyTextInput = '';
        },

        handleScreenClick(event) {
            const img = event.target;
            const rect = img.getBoundingClientRect();
            const clickX = event.clientX - rect.left;
            const clickY = event.clientY - rect.top;

            const natW = img.naturalWidth || 1080;
            const natH = img.naturalHeight || 2400;

            const targetX = Math.round((clickX / rect.width) * natW);
            const targetY = Math.round((clickY / rect.height) * natH);

            this.sendScrcpyEvent({ action: 'click', x: targetX, y: targetY });
        },

        handleScreenTouchStart(event) {
            if (!event.touches || event.touches.length === 0) return;
            const touch = event.touches[0];
            const img = event.target;
            const rect = img.getBoundingClientRect();
            this.scrcpySwipeStart = {
                x: touch.clientX - rect.left,
                y: touch.clientY - rect.top,
                time: Date.now(),
                rectWidth: rect.width,
                rectHeight: rect.height,
                natW: img.naturalWidth || 1080,
                natH: img.naturalHeight || 2400
            };
        },

        handleScreenTouchEnd(event) {
            if (!this.scrcpySwipeStart || !event.changedTouches || event.changedTouches.length === 0) return;
            const touch = event.changedTouches[0];
            const img = event.target;
            const rect = img.getBoundingClientRect();
            const endX = touch.clientX - rect.left;
            const endY = touch.clientY - rect.top;
            const duration = Math.max(100, Date.now() - this.scrcpySwipeStart.time);

            const natW = this.scrcpySwipeStart.natW;
            const natH = this.scrcpySwipeStart.natH;

            const x1 = Math.round((this.scrcpySwipeStart.x / this.scrcpySwipeStart.rectWidth) * natW);
            const y1 = Math.round((this.scrcpySwipeStart.y / this.scrcpySwipeStart.rectHeight) * natH);
            const x2 = Math.round((endX / rect.width) * natW);
            const y2 = Math.round((endY / rect.height) * natH);

            const dist = Math.hypot(x2 - x1, y2 - y1);
            if (dist > 30) {
                this.sendScrcpyEvent({ action: 'swipe', x: x1, y: y1, x2: x2, y2: y2, duration: duration });
            } else {
                this.sendScrcpyEvent({ action: 'click', x: x1, y: y1 });
            }
            this.scrcpySwipeStart = null;
        },

        // File Manager Methods
        getParentPath(path) {
            if (!path || path === '/') return '/';
            let clean = path;
            if (clean.endsWith('/') && clean.length > 1) {
                clean = clean.slice(0, -1);
            }
            const idx = clean.lastIndexOf('/');
            if (idx <= 0) return '/';
            return clean.substring(0, idx);
        },

        async fetchFileList(path) {
            try {
                const res = await fetch('/api/files/list?path=' + encodeURIComponent(path || ''));
                if (res.ok) {
                    const data = await res.json();
                    this.currentPath = data.path;
                    let files = data.files || [];
                    if (this.currentPath !== '/') {
                        const parent = this.getParentPath(this.currentPath);
                        files.unshift({
                            name: '..',
                            path: parent,
                            is_dir: true,
                            permissions: 'd--r--r--',
                            size: 0,
                            is_parent: true
                        });
                    }
                    this.fileList = files;
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
                    this.showToast('Read Error', 'Cannot read file (may exceed 5MB or be binary)', 'error');
                }
            } catch (e) {
                this.showToast('Read Error', 'Failed to retrieve file content.', 'error');
            }
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
                    this.showToast('File Saved', 'File contents written successfully!', 'success');
                } else {
                    this.showToast('Save Error', data.error, 'error');
                }
            } catch (e) {
                this.showToast('Save Error', 'File writing request failed.', 'error');
            }
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
                    this.showToast('Upload Success', 'File uploaded successfully!', 'success');
                } else {
                    this.showToast('Upload Error', 'Upload failed: ' + data.error, 'error');
                }
            } catch (e) {
                this.showToast('Upload Error', 'File upload request failed.', 'error');
            }
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
                    this.showToast('Folder Created', 'New directory created successfully!', 'success');
                } else {
                    this.showToast('Directory Error', 'Create directory failed: ' + data.error, 'error');
                }
            } catch (e) {
                this.showToast('Directory Error', 'Create directory request failed.', 'error');
            }
        },

        deletePath(targetPath) {
            this.showConfirm(
                'Delete Confirmation',
                `Are you sure you want to delete: ${targetPath}?`,
                async () => {
                    try {
                        const res = await fetch('/api/files/delete', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ path: targetPath })
                        });
                        const data = await res.json();
                        if (data.success) {
                            this.fetchFileList(this.currentPath);
                            this.showToast('Delete Success', 'Item deleted successfully!', 'success');
                        } else {
                            this.showToast('Delete Error', 'Delete failed: ' + data.error, 'error');
                        }
                    } catch (e) {
                        this.showToast('Delete Error', 'Delete request failed.', 'error');
                    }
                }
            );
        },

        openCreateFileModal() {
            this.newFileName = '';
            this.showFileModal = true;
        },

        async createFile() {
            if (!this.newFileName) return;
            const fullPath = this.currentPath + (this.currentPath.endsWith('/') ? '' : '/') + this.newFileName.trim();
            try {
                const res = await fetch('/api/files/create', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: fullPath })
                });
                const data = await res.json();
                if (data.success) {
                    this.showFileModal = false;
                    this.fetchFileList(this.currentPath);
                    this.showToast('File Created', 'New empty file created successfully!', 'success');
                } else {
                    this.showToast('Create File Error', 'Create file failed: ' + data.error, 'error');
                }
            } catch (e) {
                this.showToast('Create File Error', 'Create file request failed.', 'error');
            }
        },

        openRenameModal(file) {
            this.renameOldPath = file.path;
            this.renameNewName = file.name;
            this.showRenameModal = true;
        },

        async renamePath() {
            if (!this.renameNewName) return;
            const idx = this.renameOldPath.lastIndexOf('/');
            const parent = idx >= 0 ? this.renameOldPath.substring(0, idx + 1) : '';
            const newPath = parent + this.renameNewName.trim();
            try {
                const res = await fetch('/api/files/rename', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ old_path: this.renameOldPath, new_path: newPath })
                });
                const data = await res.json();
                if (data.success) {
                    this.showRenameModal = false;
                    this.fetchFileList(this.currentPath);
                    this.showToast('Rename Success', 'Item renamed successfully!', 'success');
                } else {
                    this.showToast('Rename Error', 'Rename failed: ' + data.error, 'error');
                }
            } catch (e) {
                this.showToast('Rename Error', 'Rename request failed.', 'error');
            }
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
                    this.showToast('Tweaks Saved', 'System optimizations updated successfully!', 'success');
                    this.fetchNetworkData();
                } else {
                    this.showToast('Save Error', 'Failed to save tweaks: ' + data.error, 'error');
                }
            } catch (e) {
                this.showToast('Save Error', 'Save request failed', 'error');
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
            this.showConfirm(
                'Confirm Action',
                `Are you sure you want to perform: ${name}?`,
                async () => {
                    try {
                        await fetch('/api/power', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ action: action })
                        });
                    } catch (e) {}
                }
            );
        },

        showConfirm(title, message, callback) {
            this.confirmModal = {
                show: true,
                title: title,
                message: message,
                onConfirm: () => {
                    this.confirmModal.show = false;
                    if (callback) callback();
                }
            };
        },

        showToast(title, message, type = 'info', duration = 3000) {
            const id = Date.now() + Math.random();
            this.toasts.push({ id, title, message, type });
            setTimeout(() => {
                this.removeToast(id);
            }, duration);
        },

        removeToast(id) {
            this.toasts = this.toasts.filter(t => t.id !== id);
        },

        sendDonationConfirmation(platform) {
            const name = this.donorName.trim() || 'Hamba Allah';
            const amount = this.donationAmount || '0';
            const msg = this.donorMessage.trim() || '-';
            const text = `Halo, saya ingin mengkonfirmasi donasi BFR WebUI Go.\n\nNama: ${name}\nJumlah: Rp ${parseInt(amount).toLocaleString('id-ID')}\nPesan/Doa: ${msg}`;
            navigator.clipboard.writeText(text).then(() => {
                this.showToast('Copied confirmation text', 'Confirmation message copied to clipboard!', 'success');
            }).catch(() => {});
            if (platform === 'telegram') {
                const url = `https://t.me/Latifan_id?text=${encodeURIComponent(text)}`;
                window.open(url, '_blank');
            } else if (platform === 'facebook') {
                const url = `https://www.facebook.com/latifan.latifan.latifan.latif`;
                window.open(url, '_blank');
            }
        },

        toggleTheme() {
            this.isDark = !this.isDark;
            localStorage.setItem('theme', this.isDark ? 'dark' : 'light');
            this.applyTheme();
        },

        applyTheme() {
            if (this.isDark) {
                document.documentElement.classList.add('dark');
                document.documentElement.classList.remove('light');
            } else {
                document.documentElement.classList.add('light');
                document.documentElement.classList.remove('dark');
            }
        },

        initShortcuts() {
            let saved = localStorage.getItem('fileManagerShortcuts');
            if (saved) {
                try {
                    this.fileManagerShortcuts = JSON.parse(saved);
                } catch(e) {
                    this.fileManagerShortcuts = this.getDefaultShortcuts();
                }
            } else {
                this.fileManagerShortcuts = this.getDefaultShortcuts();
                localStorage.setItem('fileManagerShortcuts', JSON.stringify(this.fileManagerShortcuts));
            }
        },

        getDefaultShortcuts() {
            return [
                { name: "/sdcard", path: "/sdcard" },
                { name: "/data/adb", path: "/data/adb" },
                { name: "/modules", path: "/data/adb/modules" }
            ];
        },

        addShortcut(name, path) {
            if (!name || !path) return;
            this.fileManagerShortcuts.push({ name, path });
            localStorage.setItem('fileManagerShortcuts', JSON.stringify(this.fileManagerShortcuts));
        },

        removeShortcut(index) {
            this.fileManagerShortcuts.splice(index, 1);
            localStorage.setItem('fileManagerShortcuts', JSON.stringify(this.fileManagerShortcuts));
        },

        openAddShortcutModal() {
            this.newShortcutName = '';
            this.newShortcutPath = '';
            this.showAddShortcutModal = true;
        },

        saveShortcut() {
            if (this.newShortcutName && this.newShortcutPath) {
                this.addShortcut(this.newShortcutName.trim(), this.newShortcutPath.trim());
                this.showAddShortcutModal = false;
            }
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
