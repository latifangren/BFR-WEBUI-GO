const DonationModule = {
    donorName: '',
    donationAmount: '',
    donorMessage: '',

    sendDonationConfirmation(platform) {
        const name = this.donorName.trim() || 'Anonymous';
        const amount = this.donationAmount ? 'Rp ' + parseInt(this.donationAmount).toLocaleString('id-ID') : 'Rp 0';
        const msg = this.donorMessage.trim() || '-';
        const text = `Donation Confirmation\nName: ${name}\nAmount: ${amount}\nMessage: ${msg}`;
        
        navigator.clipboard.writeText(text).then(() => {
            if (platform === 'telegram') {
                this.showToast('Copied confirmation text', 'Confirmation text copied to clipboard!', 'success');
            } else if (platform === 'facebook') {
                this.showToast('Copied confirmation text', 'Confirmation message copied! Opening Facebook chat...', 'success');
            }
        }).catch(() => {});

        if (platform === 'telegram') {
            const url = `https://t.me/Latifan_id?text=${encodeURIComponent(text)}`;
            window.open(url, '_blank');
        } else if (platform === 'facebook') {
            const url = `https://www.facebook.com/latifan.latifan.latifan.latif`;
            window.open(url, '_blank');
        }
    }
};
