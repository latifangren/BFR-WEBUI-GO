const CommonModule = {
    authenticated: false,
    loginPassword: '',
    authError: '',
    activeTab: 'overview',
    isDark: localStorage.getItem('theme') !== 'light',
    toasts: [],
    confirmModal: { show: false, title: 'Confirmation', message: '', onConfirm: null },
    modal: { show: false, action: '', actionName: '' },
    showBatteryModal: false,
    showNetworkModal: false,

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
    }
};
