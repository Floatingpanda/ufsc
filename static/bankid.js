let qrInterval;
let pollInterval;

async function getEndUserIp() {
    const response = await fetch('https://api.ipify.org?format=json');
    const data = await response.json();
    return data.ip;
}

async function checkAuthOrder(orderRef, onComplete, onTimeout) {
    const response = await fetch('/bankid/collect', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ orderRef })
    });

    if (response.status === 202) {
        // do nothing, keep polling
        return
    } else {
        if (pollInterval) clearInterval(pollInterval);
    }

    if (response.status === 204) {
        let data = await response.json()
        onTimeout(data)
        return
    }

    if (response.status === 200) {
        let data = await response.json()
        onComplete(data)
        return
    }

    if (response.status === 401) {
        console.log('Authentication failed');
        return
    }
    console.log('Something went wrong', "code", response.status);
}

export function startBankIDApp(autoStartToken) {
    window.location.href = `bankid:///?autostarttoken=${autoStartToken}&redirect=null`;
}

async function getQRCode(orderRef, onNewCode, onComplete, onTimeout) {
    console.log("Getting QR Code")
    const response = await fetch('/bankid/qrcode', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ orderRef })
    });


    if (response.status === 202) {
        const blob = await response.blob();
        const url = URL.createObjectURL(blob);
        onNewCode(url);
    } else if (response.status === 204) {
        // canceled by client
        clearInterval(qrInterval);
        onTimeout();
    } else if (response.status === 401) {
        clearInterval(qrInterval);
        console.log('Authentication failed');
    } else if (response.status === 200) {
        // completed collect
        clearInterval(qrInterval);
        checkAuthOrder(orderRef, onComplete);
    } else {
        clearInterval(qrInterval);
        console.log('Error generating QR code:', response.status);
    }
}

export function startPollingQRCode(orderRef, onNewCode, onComplete, onTimeout) {
    getQRCode(orderRef, onNewCode, onComplete, onTimeout)
    qrInterval = setInterval(() => {
        getQRCode(orderRef, onNewCode, onComplete, onTimeout)
    }, 1000);
}

export function startPollingResult(orderRef, onComplete, onTimeout) {
    pollInterval = setInterval(() => { checkAuthOrder(orderRef, onComplete, onTimeout) }, 1000);
}

export async function startBankidAuth(openApp) {
    const endUserIp = await getEndUserIp();
    const response = await fetch(`/bankid/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ endUserIp })
    });
    const data = await response.json();
    if (openApp) {
        startBankIDApp(data.autoStartToken);
    }
    return data.orderRef
}

export async function cancelBankidAuth(orderRef) {
    const response = await fetch('/bankid/cancel', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ orderRef })
    });

    if (response.ok) {
        console.log('BankID authentication canceled');
    } else {
        console.log('Failed to cancel BankID authentication');
    }
}


