document.querySelectorAll(".tripletable").forEach(element => {
    const uris: Array<HTMLElement> = Array.from(element.querySelectorAll('code.uri'))
    const rows: Array<HTMLTableRowElement> = Array.from(element.querySelectorAll('tbody tr'))


    // colors that are being used
    const colors = ['violet', 'blue', 'red', 'indigo', 'green', 'orange']
    let colorIndex = 0
    const highlighted = new Map<string, string>()

    // apply colors to elements
    const rerenderColors = () => {
        uris.forEach(uri => {
            const actual = uri.innerText
            const color = highlighted.get(actual)
            if (color) {
                uri.style.color = color
            } else {
                uri.style.color = ""
            }
        })
    }

    // function to reset the order
    const resetColors = () => {
        highlighted.clear()
        rerenderColors()
        colorIndex = 0
    }
    

    // toggle the highlight of an element
    const toggleHighlight = (uri: HTMLElement) => {
        const actual = uri.innerText
        console.log(uri.innerText)
        if (!highlighted.has(actual)) {
            highlighted.set(actual, colors[colorIndex])
            colorIndex = (colorIndex + 1) % colors.length
        } else {
            highlighted.delete(actual)
        }
        rerenderColors()
    }

    // hold the elements for each row
    const rowdata = new Array<Set<string>>(rows.length)
    rows.forEach((row, index) => {
        row.setAttribute('data-original-index', index.toString(10))
        rowdata[index] = new Set(
            Array.from(row.querySelectorAll('code.uri')).map(c => (c as HTMLElement).innerText)
        )
    })

    // function to rerender the rows
    const rerenderRows = () => {
        rows.forEach(row => {
            const parent = row.parentNode!
            parent.removeChild(row)
            parent.appendChild(row)
        })
    }

    // function to reset the order
    const resetRowOrder = () => {
        rows.sort((a, b) => {
            const aIndex = parseInt(a.getAttribute('data-original-index')!, 10)
            const bIndex = parseInt(b.getAttribute('data-original-index')!, 10)

            return aIndex - bIndex
        })
        rerenderRows()
    }

    // bring an uri to the front of the table
    const bringToFront = (uri: HTMLElement) => {
        const actual = uri.innerText
        rows.sort((a, b) => {
            const aIndex = parseInt(a.getAttribute('data-original-index')!, 10)
            const bIndex = parseInt(b.getAttribute('data-original-index')!, 10)

            const aPrio = rowdata[aIndex].has(actual)
            const bPrio = rowdata[bIndex].has(actual)

            if (aPrio === bPrio) {
                return aIndex - bIndex
            } else if (aPrio) {
                return -1
            } else {
                return 1
            }
        })
        rerenderRows()
        uri.scrollIntoView({
            behavior: 'smooth',
        })
    }


    uris.forEach(uri => {
        uri.addEventListener('click', (event) => {
            if (event.ctrlKey || event.metaKey) {
                event.preventDefault()
                toggleHighlight(uri)
                return
            }
            if (event.altKey) {
                event.preventDefault()
                bringToFront(uri)
                return
            }
        })
    })

    element.querySelectorAll('thead tr').forEach(tr =>
        tr.addEventListener('click', (event) => {
            (function(event: MouseEvent){
                if (event.ctrlKey || event.metaKey) {
                    event.preventDefault()
                    resetColors()
                    return
                }
                if (event.altKey) {
                    event.preventDefault()
                    resetRowOrder()
                    return
                }
            })(event as MouseEvent)
        })
    )
})