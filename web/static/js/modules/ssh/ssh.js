const SshModule = {
    sshData: {
        config: { enabled: false, port: 2222, bind: "127.0.0.1", key_auth_only: true },
        running: false,
        pid: 0,
        binary_path: ""
    },

    async fetchSSHStatus() {
        try {
            const res = await fetch('/api/ssh/status');
            if (res.ok) {
                const data = await res.json();
                this.sshData.config = data.config || data.Config || this.sshData.config;
                this.sshData.running = data.running || data.Running || false;
                this.sshData.pid = data.pid || data.Pid || 0;
                this.sshData.binary_path = data.binary_path || data.BinaryPath || '';
            }
        } catch (e) {}
    },

    async saveSSHConfig() {
        try {
            const res = await fetch('/api/ssh/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(this.sshData.config)
            });
            if (res.ok) {
                const data = await res.json();
                this.sshData.config = data.config || this.sshData.config;
                this.showToast('SSH Config', 'SSH configurations saved successfully!', 'success');
            } else {
                this.showToast('SSH Error', 'Failed to save SSH configurations.', 'error');
            }
        } catch (e) {
            this.showToast('SSH Error', 'Request failed.', 'error');
        }
    },

    async controlSSH(action) {
        try {
            const res = await fetch('/api/ssh/control', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action: action })
            });
            if (res.ok) {
                const data = await res.json();
                this.sshData.running = data.running;
                this.sshData.pid = data.pid;
                this.showToast('SSH Service', `SSH daemon ${action} successful!`, 'success');
                await this.fetchSSHStatus();
            } else {
                this.showToast('SSH Error', `Failed to execute ${action} on SSH daemon.`, 'error');
            }
        } catch (e) {
            this.showToast('SSH Error', 'Connection failed.', 'error');
        }
    }
};
