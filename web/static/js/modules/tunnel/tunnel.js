const TunnelModule = {
    tunnelRunning: true,
    tunnelEngine: 'cloudflare_quick', // 'cloudflare_quick', 'cloudflare_token', 'tailscale', 'zerotier'
    tunnelPublicUrl: 'https://bfr-android.trycloudflare.com',
    tunnelPing: '18 ms',
    tunnelProtocol: 'QUIC / WireGuard (TLS 1.3)',
    tunnelUptime: '02h 45m 12s',
    cfTunnelToken: '',
    tailscaleAuthKey: '',
    zeroTierNetworkId: '',

    fetchTunnelStatus() {
        if (typeof showToast === 'function') {
            showToast('Memperbarui status Remote Access Tunnel...', 'info');
        }
    },

    toggleTunnelEngine() {
        this.tunnelRunning = !this.tunnelRunning;
        if (typeof showToast === 'function') {
            const state = this.tunnelRunning ? 'Tunnel Remote Access diaktifkan' : 'Tunnel Remote Access dihentikan';
            showToast(state, this.tunnelRunning ? 'success' : 'warning');
        }
    },

    copyTunnelUrl() {
        if (navigator.clipboard) {
            navigator.clipboard.writeText(this.tunnelPublicUrl);
            if (typeof showToast === 'function') showToast('Public Tunnel URL berhasil disalin!', 'success');
        }
    },

    saveTunnelConfig() {
        if (typeof showToast === 'function') {
            showToast(`Pengaturan Tunnel (${this.tunnelEngine}) berhasil disimpan!`, 'success');
        }
    }
};
