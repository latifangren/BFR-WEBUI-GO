const ModulesModule = {
    modulesList: [],
    modulesLoading: false,

    async fetchModules() {
        this.modulesLoading = true;
        try {
            const res = await fetch('/api/modules');
            if (res.ok) {
                const data = await res.json();
                this.modulesList = data.modules || [];
            }
        } catch (e) {
            this.showToast('Modules Error', 'Failed to fetch root modules list', 'error');
        } finally {
            this.modulesLoading = false;
        }
    },

    async toggleModule(id, enable) {
        try {
            const res = await fetch('/api/modules/toggle', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ id, enable })
            });
            const data = await res.json();
            if (data.success) {
                this.showToast('Module Updated', `Module "${id}" ${enable ? 'enabled' : 'disabled'}`, 'success');
                this.fetchModules();
            } else {
                this.showToast('Module Error', data.error || 'Failed to toggle module', 'error');
            }
        } catch (e) {
            this.showToast('Module Error', 'Module toggle request failed', 'error');
        }
    },

    async installModule() {
        const input = document.getElementById('module-zip-input');
        if (!input || !input.files.length) return;

        const file = input.files[0];
        if (!file.name.toLowerCase().endsWith('.zip')) {
            this.showToast('Install Error', 'Module package must be a .zip file', 'error');
            return;
        }

        const formData = new FormData();
        formData.append('module', file);

        this.showToast('Installing Module', `Flashing ${file.name}...`, 'info');
        try {
            const res = await fetch('/api/modules/install', {
                method: 'POST',
                body: formData
            });
            const data = await res.json();
            if (data.success) {
                this.showToast('Module Installed', 'Module flashed successfully! Reboot to apply.', 'success');
                input.value = '';
                this.fetchModules();
            } else {
                this.showToast('Install Error', data.error || 'Module flashing failed', 'error');
            }
        } catch (e) {
            this.showToast('Install Error', 'Module installation request failed', 'error');
        }
    }
};
