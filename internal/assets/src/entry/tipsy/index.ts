function createTipsyFrame(tipsy: string, filename: string, type: string, data: string): HTMLIFrameElement {
    // create an iframe with the source pointing to tipsy
    const iframe = document.createElement('iframe')
    iframe.setAttribute('src', tipsy)

    // wait for a message from the frame to tell us that tipsy is ready
    const handleTipsyReady = (/** @type {MessageEvent} */message) => {
        if (message.source !== iframe.contentWindow || message.data !== 'tipsy:ready') {
            return
        }
        message.preventDefault()
    
        window.removeEventListener('message', handleTipsyReady)
        message.source.postMessage({'filename': filename, 'type': type, 'data': data}, '*');
    }
    window.addEventListener('message', handleTipsyReady)

    // and return the iframe
    return iframe
}

function loadTipsy(tipsy: HTMLDivElement): HTMLIFrameElement | null {
    const { url, data, filename } = tipsy.dataset
    if (typeof data !== 'string' || typeof url !== 'string' || typeof filename !== 'string') {
        return null
    }
    return createTipsyFrame(url, filename, 'application/xml', data)
}

window.onload = () => {
    const tipsy = document.getElementById('tipsy');
    if (tipsy === null) {
        return
    }

    const frame = loadTipsy(tipsy as HTMLDivElement)
    if (frame === null) {
        tipsy.innerHTML = ''
        tipsy.append(document.createTextNode('Something went wrong trying to load tipsy. '))
        return
    }

    tipsy.remove()
    document.body.append(frame)
}
