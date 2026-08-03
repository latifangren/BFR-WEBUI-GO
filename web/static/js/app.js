function dashboard() {
    return Object.assign({},
        CommonModule,
        SysinfoModule,
        NetworkModule,
        SpeedtestModule,
        BackupModule,
        FilemanagerModule,
        ProxyModule,
        TerminalModule,
        HotspotModule,
        VnstatModule,
        ChargerModule,
        SmsModule,
        ScrcpyModule,
        DonationModule,
        SshModule,
        TelegramModule,
        LogsModule,
        ModulesModule,
        LogcatModule,
        {
            // Authenticated root orchestration flow
            async init() {
                this.applyTheme();
                this.initShortcuts();
                await this.checkAuth();
                this.initHashNavigation();
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
                    this.fetchSSHStatus();
                    this.fetchTelegramStatus();
                    this.fetchCloudConfig();
                    this.startLogsPolling();
                }
            },

            initHashNavigation() {
                const initialHash = window.location.hash.substring(1);
                if (initialHash && this.validTabs && this.validTabs.includes(initialHash)) {
                    this.activeTab = initialHash;
                    this.triggerTabLoad(initialHash);
                } else if (this.activeTab) {
                    window.location.hash = this.activeTab;
                }

                this.$watch('activeTab', (newTab) => {
                    if (newTab && this.validTabs && this.validTabs.includes(newTab)) {
                        if (window.location.hash.substring(1) !== newTab) {
                            window.location.hash = newTab;
                        }
                    }
                });

                window.addEventListener('hashchange', () => {
                    const hashTab = window.location.hash.substring(1);
                    if (hashTab && this.validTabs && this.validTabs.includes(hashTab) && this.activeTab !== hashTab) {
                        this.activeTab = hashTab;
                    }
                });
            },

            triggerTabLoad(tab) {
                if (tab === 'logs' && typeof this.fetchLogs === 'function') {
                    this.fetchLogs();
                } else if (tab === 'files' && typeof this.fetchFileList === 'function') {
                    this.fetchFileList(this.currentPath || '/');
                } else if (tab === 'terminal' && typeof this.initTerminal === 'function') {
                    this.initTerminal();
                } else if (tab === 'sms' && typeof this.fetchSMS === 'function') {
                    this.fetchSMS();
                }
            },

            async checkAuth() {
                try {
                    const res = await fetch('/api/auth/status');
                    const data = await res.json();
                    this.authenticated = data.authenticated;
                    if (data.is_default_pass !== undefined) {
                        this.isDefaultPass = data.is_default_pass;
                    }
                } catch (e) {
                    this.authenticated = false;
                }
            },

            async login() {
                this.authError = '';
                this.loggingIn = true;
                try {
                    const res = await fetch('/api/auth/login', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ password: this.loginPassword })
                    });
                    const data = await res.json();
                    if (res.ok && data.success) {
                        this.authenticated = true;
                        if (data.is_default_pass !== undefined) {
                            this.isDefaultPass = data.is_default_pass;
                        }
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
                        this.fetchSSHStatus();
                        this.fetchTelegramStatus();
                        this.fetchCloudConfig();
                        this.startLogsPolling();
                    } else {
                        this.authError = data.error || 'Invalid password';
                    }
                } catch (e) {
                    this.authError = 'Connection failed';
                } finally {
                    this.loggingIn = false;
                }
            },

            async logout() {
                await fetch('/api/auth/logout', { method: 'POST' });
                this.authenticated = false;
                if (this.pollTimer) clearInterval(this.pollTimer);
                if (this.eventSource) this.eventSource.close();
                this.stopLogsPolling();
            },

            formatBytes(bytes) {
                if (!bytes) return '0 B';
                const k = 1024;
                const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
                const i = Math.floor(Math.log(bytes) / Math.log(k));
                return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
            }
        }
    );
}
