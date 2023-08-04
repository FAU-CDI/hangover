// global typescript for everything
const js_source = process.env.LEGAL_JS_SOURCE;
if(js_source) {
    const footer = document.createElement("footer");
    
    const text = process.env.LEGAL_JS_TEXT;
    if(text) {
        footer.append(document.createTextNode(text + " "));
    }

    const script = document.createElement("script");
    script.setAttribute("src", js_source);
    footer.append(script);

    if(text) {
        footer.append(document.createTextNode("."));
    }

    document.body.append(footer);
}