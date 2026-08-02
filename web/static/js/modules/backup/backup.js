const BackupModule = {
    cloudConfig: {
        enabled: false,
        url: '',
        username: '',
        password: '',
        interval_hours: 24,
        last_sync: 'Never'
    },
    cloudSyncing: false,

    async fetchCloudConfig() {
        try {
            const res = await fetch('/api/backup/cloud/config');
            const data = await res.json();
            if (data) {
                this.cloudConfig = Object.assign({}, this.cloudConfig, data);
            }
        } catch (e) {
            console.error('Failed to fetch WebDAV cloud config', e);
        }
    },

    async saveCloudConfig() {
        try {
            const res = await fetch('/api/backup/cloud/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(this.cloudConfig)
            });
            const data = await res.json();
            if (res.ok && data.success !== false) {
                this.showToast('Cloud Config Saved', 'WebDAV sync settings updated successfully.', 'success');
            } else {
                this.showToast('Save Error', data.error || 'Failed to save cloud config', 'error');
            }
        } catch (e) {
            this.showToast('Save Error', 'Connection failed', 'error');
        }
    },

    async syncCloudNow() {
        this.cloudSyncing = true;
        try {
            const res = await fetch('/api/backup/cloud/sync', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            const data = await res.json();
            if (res.ok && data.success !== false) {
                this.showToast('Cloud Sync Success', 'Backup synced to WebDAV server.', 'success');
                if (data.last_sync) {
                    this.cloudConfig.last_sync = data.last_sync;
                } else {
                    this.cloudConfig.last_sync = new Date().toLocaleString();
                }
            } else {
                this.showToast('Sync Error', data.error || 'Failed to sync with WebDAV server', 'error');
            }
        } catch (e) {
            this.showToast('Sync Error', 'Connection failed', 'error');
        } finally {
            this.cloudSyncing = false;
        }
    }
};
