const ChargerModule = {
    chargerConfig: {
        config: { enabled: false, start_percent: 70, stop_percent: 80 }
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
    }
};
