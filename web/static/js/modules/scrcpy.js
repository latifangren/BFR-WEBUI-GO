const ScrcpyModule = {
    isMirroring: false,
    screenRate: 0,
    scrcpyImgUrl: '',
    scrcpyWs: null,
    scrcpyTextInput: '',
    scrcpySwipeStart: null,
    scrcpyLastFrameTime: 0,

    startMirroring() {
        if (this.isMirroring) return;
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/scrcpy/ws`;

        this.scrcpyWs = new WebSocket(wsUrl);
        this.scrcpyWs.binaryType = 'blob';

        this.scrcpyWs.onopen = () => {
            this.isMirroring = true;
            this.scrcpyLastFrameTime = Date.now();
        };

        this.scrcpyWs.onmessage = (event) => {
            if (event.data instanceof Blob) {
                const now = Date.now();
                if (this.scrcpyLastFrameTime > 0) {
                    const delta = now - this.scrcpyLastFrameTime;
                    if (delta > 0) {
                        this.screenRate = (1000 / delta).toFixed(1);
                    }
                }
                this.scrcpyLastFrameTime = now;

                const url = URL.createObjectURL(event.data);
                if (this.scrcpyImgUrl) {
                    URL.revokeObjectURL(this.scrcpyImgUrl);
                }
                this.scrcpyImgUrl = url;
            }
        };

        this.scrcpyWs.onclose = () => {
            this.isMirroring = false;
            this.screenRate = 0;
        };

        this.scrcpyWs.onerror = () => {
            this.isMirroring = false;
            this.screenRate = 0;
        };
    },

    stopMirroring() {
        if (this.scrcpyWs) {
            this.scrcpyWs.close();
            this.scrcpyWs = null;
        }
        if (this.scrcpyImgUrl) {
            URL.revokeObjectURL(this.scrcpyImgUrl);
            this.scrcpyImgUrl = '';
        }
        this.isMirroring = false;
        this.screenRate = 0;
    },

    sendScrcpyEvent(evt) {
        if (this.scrcpyWs && this.scrcpyWs.readyState === WebSocket.OPEN) {
            this.scrcpyWs.send(JSON.stringify(evt));
        }
    },

    sendKey(action, keycode = 0) {
        this.sendScrcpyEvent({ action: action, keycode: keycode });
    },

    sendScrcpyText() {
        if (!this.scrcpyTextInput) return;
        this.sendScrcpyEvent({ action: 'text', text: this.scrcpyTextInput });
        this.scrcpyTextInput = '';
    },

    handleScreenClick(event) {
        const img = event.target;
        const rect = img.getBoundingClientRect();
        const clickX = event.clientX - rect.left;
        const clickY = event.clientY - rect.top;

        const natW = img.naturalWidth || 1080;
        const natH = img.naturalHeight || 2400;

        const targetX = Math.round((clickX / rect.width) * natW);
        const targetY = Math.round((clickY / rect.height) * natH);

        this.sendScrcpyEvent({ action: 'click', x: targetX, y: targetY });
    },

    handleScreenTouchStart(event) {
        if (!event.touches || event.touches.length === 0) return;
        const touch = event.touches[0];
        const img = event.target;
        const rect = img.getBoundingClientRect();
        this.scrcpySwipeStart = {
            x: touch.clientX - rect.left,
            y: touch.clientY - rect.top,
            time: Date.now(),
            rectWidth: rect.width,
            rectHeight: rect.height,
            natW: img.naturalWidth || 1080,
            natH: img.naturalHeight || 2400
        };
    },

    handleScreenTouchEnd(event) {
        if (!this.scrcpySwipeStart || !event.changedTouches || event.changedTouches.length === 0) return;
        const touch = event.changedTouches[0];
        const img = event.target;
        const rect = img.getBoundingClientRect();
        const endX = touch.clientX - rect.left;
        const endY = touch.clientY - rect.top;
        const duration = Math.max(100, Date.now() - this.scrcpySwipeStart.time);

        const natW = this.scrcpySwipeStart.natW;
        const natH = this.scrcpySwipeStart.natH;

        const x1 = Math.round((this.scrcpySwipeStart.x / this.scrcpySwipeStart.rectWidth) * natW);
        const y1 = Math.round((this.scrcpySwipeStart.y / this.scrcpySwipeStart.rectHeight) * natH);
        const x2 = Math.round((endX / rect.width) * natW);
        const y2 = Math.round((endY / rect.height) * natH);

        const dist = Math.hypot(x2 - x1, y2 - y1);
        if (dist > 30) {
            this.sendScrcpyEvent({ action: 'swipe', x: x1, y: y1, x2: x2, y2: y2, duration: duration });
        } else {
            this.sendScrcpyEvent({ action: 'click', x: x1, y: y1 });
        }
        this.scrcpySwipeStart = null;
    }
};
