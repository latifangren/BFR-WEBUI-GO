const SmsModule = {
    smsList: [],
    smsTotal: 0,
    smsLimit: 10,
    smsOffset: 0,
    smsSearchQuery: '',
    isLoadingSms: false,

    async fetchSMS(newSearch = false) {
        if (newSearch) {
            this.smsOffset = 0;
        }
        this.isLoadingSms = true;
        try {
            const query = new URLSearchParams({
                limit: this.smsLimit,
                offset: this.smsOffset,
                q: this.smsSearchQuery || ''
            });
            const res = await fetch('/api/sms/inbox?' + query.toString());
            if (res.ok) {
                const data = await res.json();
                this.smsList = data.messages || [];
                this.smsTotal = data.total || 0;
            }
        } catch (e) {
        } finally {
            this.isLoadingSms = false;
        }
    },

    nextSMSPage() {
        if (this.smsOffset + this.smsLimit < this.smsTotal) {
            this.smsOffset += this.smsLimit;
            this.fetchSMS();
        }
    },

    prevSMSPage() {
        if (this.smsOffset >= this.smsLimit) {
            this.smsOffset -= this.smsLimit;
            this.fetchSMS();
        } else {
            this.smsOffset = 0;
            this.fetchSMS();
        }
    },

    formatSMSDate(timestamp) {
        if (!timestamp) return 'N/A';
        let t = timestamp;
        if (t < 10000000000) {
            t = t * 1000;
        }
        const date = new Date(t);
        return date.toLocaleString();
    },

    formatSMSBody(body) {
        if (!body) return '';
        let safe = body.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
        const regex = /\b(\d{4,8})\b/g;
        return safe.replace(regex, '<mark class="bg-[#f59e0b]/20 px-1 text-[#f59e0b] rounded font-bold border-b border-[#f59e0b]/30 font-mono">$1</mark>');
    }
};
