import { Graph, instance } from "@viz-js/viz";

type GraphData = {
    triples: Array<[string, string, string]>,
    data: Array<[string, string, any]>,
}

/** represents a namespace map obj[long] = short */
class NamespaceMap {
    private elements = new Map<string, string>();
    add(long: string, short: string) {
        if (short === "") return; // ignore empty prefix
        this.elements.set(long, short);
    }
    apply(uri: string): string {
        const prefix = this.prefix(uri);
        if (prefix === "") return uri;
        return this.elements.get(prefix) + ":" + uri.substring(prefix.length);
    }
    prefix(uri: string): string {
        let prefix = "";     // prefix used
        this.elements.forEach((short, long) => {
            // must actually be a prefix
            if (!uri.startsWith(long)) {
                return;
            }

            // if we already have a shorter prefix
            // then don't apply it at all!
            if (prefix != "" && (long <= prefix)) {
                return;
            }
            prefix = long;
        });
        return prefix;
    }
    toTable(): HTMLTableElement {
        const table = document.createElement('table');

        const header = document.createElement('tr');
        table.append(header);

        const th0 = document.createElement('th');
        header.append(th0);
        th0.append(document.createTextNode('NS'));

        const th1 = document.createElement('th');
        header.append(th1);
        th1.append(document.createTextNode('URI'));

        this.elements.forEach((short, long) => {
            const tr = document.createElement('tr');
            table.append(tr);

            const td0 = document.createElement('td');
            tr.append(td0);
            td0.append(document.createTextNode(short));

            const td1 = document.createElement('td');
            tr.append(td1);
            td1.append(document.createTextNode(long));
        })

        return table;
    }

    static generate(uris: Set<string>, separators: string = "/#", len = 30): NamespaceMap {
        const prefixes = new Set<string>();
        uris.forEach(uri => {
            const until = Math.max(...Array.from(separators).map(c => uri.lastIndexOf(c)));
            // no valid prefix
            if (until === -1) {
                return;
            }

            // compute the prefix
            const prefix = uri.substring(0, until + 1);

            // we already have a prefix
            if (prefixes.has(prefix)) {
                return;
            }

            let hadPrefix = false;
            prefixes.forEach(old => {
                // we have a prefix that is longer
                // so delete it
                if (old.startsWith(prefix)) {
                    prefixes.delete(old);
                }

                // we had a subset of this one already
                // so don't add it!
                if (prefix.startsWith(old)) {
                    hadPrefix = true;
                }
            })

            // don't add the prefix
            if (hadPrefix) {
                return;
            }
            prefixes.add(prefix);
        })

        const ns = new NamespaceMap();

        const seen = new Map<string, number>();
        prefixes.forEach(prefix => {
            const name = (prefix.indexOf('://') >= 0) ? prefix.substring(prefix.indexOf('://') + '://'.length) : prefix;
            const match = (name.match(/([a-zA-Z0-9]+)/g) ?? []).find(v => v !== "www") ?? "prefix";
            
            let theName = match.substring(0, len);
            if (seen.has(theName)) {
                const counter = seen.get(theName)!;
                seen.set(theName, counter + 1);
                theName = `${theName}_${counter}`;
            } else {
                seen.set(theName, 1);
            }

            ns.add(prefix, theName); // TODO: smarter prefixing
        })
        return ns;
    }

}

// renderData renders data as a string
function renderData(data: unknown, line = 30): string {
    let asString = data + '';
    
    const lines: Array<string> = [];
    while(asString.length > line) {
        lines.push(asString.substring(0, line));
        asString = asString.substring(line);
    }
    if (asString !== "") {
        lines.push(asString);
    }
    return lines.join('\n');
}

function renderGraph(element: HTMLElement, data: GraphData) {
    // build the graph 
    const graph: Graph = {directed: true, nodes: [], edges: []}

    // add the nodes that we know of
    const nodeSet = new Set<string>();
    const uriSet = new Set<string>();
    data.triples.forEach(triple => {
        nodeSet.add(triple[0]);
        nodeSet.add(triple[2]);
        uriSet.add(triple[1]);
    });
    data.data.forEach(triple => {
        nodeSet.add(triple[0]);
        uriSet.add(triple[1]);
    });

    nodeSet.forEach(e => uriSet.add(e));

    // generate a namespace map
    const mp = NamespaceMap.generate(uriSet);
    mp.add("http://www.w3.org/1999/02/22-rdf-syntax-ns#", "rdf"); // we always use this prefix

    // add the nodes to the underlying dataset
    nodeSet.forEach(node => {
        graph.nodes?.push({
            name: node,
            attributes: {
                label: mp.apply(node),
                shape: 'ellipse',
            },
        })
    })

    // add the known edges
    data.triples.forEach(triple => {
        graph.edges?.push({
            head: triple[2],
            tail: triple[0],
            attributes: {
                label: mp.apply(triple[1]),
            },
        })
    })

    let counter = 0;

    // add data edges
    data.data.forEach(triple => {
        counter++
        const id = '_:' + counter.toString()

        graph.nodes?.push({
            name: id,
            attributes: {
                label: renderData(triple[2]),
                shape: 'box',
                color: 'orange',
            }
        })

        graph.edges?.push({
            head: id,
            tail: triple[0],
            attributes: {
                label: mp.apply(triple[1]),
            }
        })
    })

    const statusSpan = document.createElement("span")
    statusSpan.append("Loading ...")
    element.appendChild(statusSpan)

    // and render it
    instance().then(viz => {
        const canon = viz.renderString(graph, { format: 'canon' })
        const svg =  viz.renderString(canon, { format: 'svg' })
        return  { canon, svg }
    }).then(
        ({canon, svg }) => {
            // create an svg element!
            let svgElem: SVGElement
            {
                const temp = document.createElement('span')
                temp.innerHTML = svg
                const svgQuery = temp.querySelector('svg')
                if (svgQuery == null) {
                    throw new Error('no svg rendered')
                }
                svgElem = svgQuery
            }
            
            // create URLs for the gv and svg formats
            const svgURL = URL.createObjectURL(new Blob([svg], { type: 'image/svg+xml' }))
            const gvURL = URL.createObjectURL(new Blob([canon], {type: 'text/vnd.graphviz'}))

            // create download link for svg
            const downloadSVG = document.createElement('a')
            downloadSVG.href = svgURL
            downloadSVG.download = 'graph.svg'
            downloadSVG.append('SVG')

            // create download link for gv
            const downloadGV = document.createElement('a')
            downloadGV.href = gvURL
            downloadGV.download = 'graph.gv'
            downloadGV.append('GV')

            // make a <p>
            const p = document.createElement('p')
            p.append(
                'Download As: ',
                downloadSVG,
                ' ',
                downloadGV,
            )
            
            element.removeChild(statusSpan)
            element.append(
                svgElem,
                p,
                mp.toTable()
            )
        }, error => {
            statusSpan.innerHTML = '';
            statusSpan.append("Failed to render graph")
            console.error(error)
        },
    )
}

document.querySelectorAll("script").forEach((script) => {
    // ensure that we have the data-render-graph attribute
    if (script.getAttribute('data-render-graph') !== 'true') {
        return
    }

    // get the graph data
    const graph = JSON.parse(script.textContent ?? '') as GraphData;

    // create a toggle (to show / hide the graph)
    const toggle: HTMLElement = document.createElement('div')
    script.parentNode!.insertBefore(toggle, script);

    // create an element to hold the graph
    const element = document.createElement('div');
    script.parentNode!.insertBefore(element, script);

    const ToggleText = 'Show Entity Triples Graph'
    const unToggleText = 'Hide Entity Triples Graph' 
    let hidden = false
    let initialized = false
    const onclick = (event) => {
        if (event) event.preventDefault()

        toggle.innerText = hidden ? unToggleText : ToggleText

        // update the content itself
        if (hidden) {
            (element as HTMLElement).style.display = "block";
            hidden = false
            if (!initialized) {
                renderGraph(element, graph)
                initialized = true
            }
        } else {
            (element as HTMLElement).style.display = "none"
            hidden = true
        }
    }

    toggle.addEventListener('click', onclick)
    onclick(null);
})