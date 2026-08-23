const CommonModule = {
    authenticated: false,
    loginPassword: '',
    showPassword: false,
    loggingIn: false,
    authError: '',
    isDefaultPass: true,
    changePassData: { current_password: '', new_password: '', confirm_password: '' },
    changePassMsg: '',
    changePassError: '',
    activeTab: 'overview',
    settingsTab: 'backup',
    validTabs: ['overview', 'sysinfo', 'logs', 'files', 'terminal', 'tools', 'scrcpy', 'network', 'qos', 'modem', 'tunnel', 'nas', 'speedtest', 'proxy', 'sms', 'about'],
    mobileCategory: null,
    hideMobileNav: false,
    isDark: localStorage.getItem('theme') !== 'light',
    colorTheme: localStorage.getItem('colorTheme') || (localStorage.getItem('theme') === 'light' ? 'light' : 'dark'),
    uiStyle: localStorage.getItem('uiStyle') || 'neobrutal',
    toasts: [],
    confirmModal: { show: false, title: 'Confirmation', message: '', onConfirm: null },
    modal: { show: false, action: '', actionName: '' },
    showBatteryModal: false,
    showNetworkModal: false,
    showBackupModal: false,
    showAppearanceModal: false,

    async exportBackup() {
        try {
            window.location.href = '/api/backup/export';
            this.showToast('Backup Export', 'Downloading configuration backup...', 'info');
        } catch (e) {
            this.showToast('Backup Error', 'Failed to export backup', 'error');
        }
    },

    async importBackup() {
        const input = document.getElementById('backup-import-input');
        if (!input || !input.files.length) return;

        const file = input.files[0];
        try {
            const content = await file.text();
            const res = await fetch('/api/backup/import', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: content
            });
            const data = await res.json();
            if (data.success) {
                this.showToast('Backup Imported', 'Configurations restored successfully!', 'success');
                this.showBackupModal = false;
                input.value = '';
                setTimeout(() => location.reload(), 1000);
            } else {
                this.showToast('Import Error', data.error || 'Failed to import backup', 'error');
            }
        } catch (e) {
            this.showToast('Import Error', 'Failed reading backup file', 'error');
        }
    },

    async changePassword() {
        this.changePassMsg = '';
        this.changePassError = '';
        const current_password = this.changePassData.current_password || '';
        const new_password = this.changePassData.new_password || '';
        const confirm_password = this.changePassData.confirm_password || '';

        if (!current_password) {
            this.changePassError = 'Current password is required.';
            return;
        }
        if (!new_password || new_password.length < 4) {
            this.changePassError = 'New password must be at least 4 characters long.';
            return;
        }
        if (new_password !== confirm_password) {
            this.changePassError = 'New password confirmation does not match.';
            return;
        }
        try {
            const res = await fetch('/api/auth/change-password', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ current_password, new_password })
            });
            const data = await res.json();
            if (res.ok && data.success) {
                this.isDefaultPass = false;
                this.changePassMsg = 'Password successfully updated!';
                this.showToast('Password Updated', 'Access password has been updated successfully.', 'success');
                this.changePassData = { current_password: '', new_password: '', confirm_password: '' };
            } else {
                this.changePassError = data.error || 'Failed to update password.';
            }
        } catch (e) {
            this.changePassError = 'Connection error. Please try again.';
        }
    },

    setColorTheme(theme) {
        this.colorTheme = theme;
        localStorage.setItem('colorTheme', theme);
        if (theme === 'light') {
            this.isDark = false;
            localStorage.setItem('theme', 'light');
        } else {
            this.isDark = true;
            localStorage.setItem('theme', 'dark');
        }
        this.applyTheme();
    },

    toggleTheme() {
        this.isDark = !this.isDark;
        const newTheme = this.isDark ? 'dark' : 'light';
        this.setColorTheme(newTheme);
    },

    applyTheme() {
        const theme = this.colorTheme || 'dark';
        document.documentElement.setAttribute('data-theme', theme);
        if (theme === 'light') {
            document.documentElement.classList.add('light');
            document.documentElement.classList.remove('dark');
            this.isDark = false;
        } else {
            document.documentElement.classList.add('dark');
            document.documentElement.classList.remove('light');
            this.isDark = true;
        }
    },

    toggleUIStyle() {
        this.uiStyle = this.uiStyle === 'neobrutal' ? 'modern' : 'neobrutal';
        localStorage.setItem('uiStyle', this.uiStyle);
        this.applyUIStyle();
    },

    setUIStyle(style) {
        this.uiStyle = style;
        localStorage.setItem('uiStyle', style);
        this.applyUIStyle();
    },

    applyUIStyle() {
        const style = this.uiStyle || 'neobrutal';
        document.documentElement.setAttribute('data-ui-style', style);
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
