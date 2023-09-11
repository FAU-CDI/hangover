
const progressElement = document.getElementById('progress') as HTMLElement
const API_PROGRESS = '/api/v1/progress';


interface Progress {
    Done: boolean
    Stage: string
    Current: number
    Total: number
}

let counter = 0;
let interval: null | number = null;
async function updateStatus() {
    counter++

    const lastCounter = counter;
    try {
        const progress: Progress = await fetch(API_PROGRESS).then(r => r.json())

        if (lastCounter !== counter) {
            console.warn('Received out-of-order response')
            return;
        }

        // we are done loading
        if (progress.Done) {
            progressElement.innerHTML = 'Finished, reloading page ...'
            if (interval !== null) {
                clearInterval(interval)
            }
            location.reload()
            return
        }

        if (progress.Total !== 0) {
            let html = `<code>${progress.Stage}</code> (<code>`;
            if (progress.Total !== progress.Current) {
                html += `${progress.Current}/${progress.Total}`;
            } else {
                html += progress.Current.toString();
            }
            html += '</code>)';
            progressElement.innerHTML = html;
        } else if(progress.Stage != "") {
            progressElement.innerHTML = `<code>${progress.Stage}</code>`
        }
    } catch(e: any) {
        console.error(e)
    }
}
updateStatus()

function startInterval() {
    if (interval !== null) {
        return
    }

    interval = setInterval(updateStatus, 500)
}
startInterval()