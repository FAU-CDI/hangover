document.querySelectorAll(".showable").forEach(element => {
    // get or create a toggle
    const toggle: HTMLElement = element.querySelector('.toggle') ?? document.createElement('div')
    if (toggle.parentNode) {
        toggle.parentNode.removeChild(toggle)
    }
    element.parentElement?.insertBefore(toggle, element)
    const display = (element as HTMLElement).style.display || 'block'

    const ToggleText = element.getAttribute('data-placeholder') ?? 'Show'
    const unToggleText = toggle.innerText || 'Hide'


    const onclick = (event) => {
        if(event) event.preventDefault()

        toggle.innerText = hidden ? unToggleText : ToggleText

        // update the content itself
        if (hidden) {
            (element as HTMLElement).style.display = display
            hidden = false
        } else {
            (element as HTMLElement).style.display = "none"
            hidden = true
        }
    }
    let hidden = false
    
    toggle.addEventListener('click', onclick)
    onclick(null)
})