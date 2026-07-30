const TerminalModule = {
    term: null,
    termWS: null,

    initTerminal() {
        this.$nextTick(() => {
            if (this.term) return;
            const container = document.getElementById('terminal-container');
            if (!container) return;

            this.term = new Terminal({
                theme: {
                    background: '#000000',
                    foreground: '#e5e7eb',
                    cursor: '#3b82f6'
                },
                fontFamily: 'monospace',
                fontSize: 13
            });
            const fitAddon = new FitAddon.FitAddon();
            this.term.loadAddon(fitAddon);
            this.term.open(container);
            fitAddon.fit();

            this.reconnectTerminal();
        });
    },

    reconnectTerminal() {
        if (this.termWS) this.termWS.close();
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        this.termWS = new WebSocket(`${protocol}//${window.location.host}/api/terminal/ws`);

        this.termWS.onmessage = (e) => {
            if (this.term) this.term.write(e.data);
        };

        this.term.onData((data) => {
            if (this.termWS && this.termWS.readyState === WebSocket.OPEN) {
                this.termWS.send(data);
            }
        });
    }
};
