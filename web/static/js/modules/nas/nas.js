const NASModule = {
    nasEnabled: true,
    nasSharedPath: '/storage/emulated/0/Download',
    nasPort: 8088,
    nasPermission: 'read_write',
    nasAuthMode: false,
    nasUser: 'admin',
    nasPassword: '',
    nasFreeSpace: '48.5 GB',

    fetchNASStatus() {
        if (typeof showToast === 'function') {
            showToast('Memperbarui status NAS Lite server...', 'info');
        }
    },

    toggleNASServer() {
        this.nasEnabled = !this.nasEnabled;
        if (typeof showToast === 'function') {
            const msg = this.nasEnabled ? 'Server NAS Lite diaktifkan di Port ' + this.nasPort : 'Server NAS Lite dihentikan';
            showToast(msg, this.nasEnabled ? 'success' : 'warning');
        }
    },

    saveNASConfig() {
        if (typeof showToast === 'function') {
            showToast(`Pengaturan NAS Lite (${this.nasSharedPath}) disimpan & diterapkan!`, 'success');
        }
    }
};
