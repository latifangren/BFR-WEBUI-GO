function dashboard() {
    return Object.assign({},
        CommonModule,
        SysinfoModule,
        NetworkModule,
        FilemanagerModule,
        ProxyModule,
        TerminalModule,
        HotspotModule,
        VnstatModule,
        ChargerModule,
        SmsModule,
        ScrcpyModule,
        DonationModule,
        {
            // Authenticated root orchestration flow
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
