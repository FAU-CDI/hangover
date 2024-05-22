import { Network } from "vis-network";
import { DataSet } from "vis-data";

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
    applyLabel(uri: string) {
        const prefix = this.prefix(uri);
        if (prefix === "") return uri;
        return this.elements.get(prefix) + ":\n" + uri.substring(prefix.length);
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

function renderGraph(element: HTMLElement, graph: GraphData): Network {
    // create a graph element and append it
    const graphElement = document.createElement("div");
    element.append(graphElement);



    const nodes = new DataSet<{ id?: string, label: string, shape?: string, color?: string; font?: unknown }>([]);
    const edges = new DataSet<{ from: string, to: string, id?: never, label: string, color?: string; font?: unknown, arrows?: "to" }>();

    // add the nodes that we know of
    const nodeSet = new Set<string>();
    const uriSet = new Set<string>();
    graph.triples.forEach(triple => {
        nodeSet.add(triple[0]);
        nodeSet.add(triple[2]);
        uriSet.add(triple[1]);
    });
    graph.data.forEach(triple => {
        nodeSet.add(triple[0]);
        uriSet.add(triple[1]);
    });

    nodeSet.forEach(e => uriSet.add(e));

    // generate a namespace map
    const mp = NamespaceMap.generate(uriSet);
    mp.add("http://www.w3.org/1999/02/22-rdf-syntax-ns#", "rdf"); // we always use this prefix
    element.append(mp.toTable());

    // add the nodes to the underlying dataset
    nodeSet.forEach(node => {
        nodes.add({ id: node, label: mp.applyLabel(node), shape: "ellipse" });
    })

    // add the known edges
    graph.triples.forEach(triple => {
        edges.add({
            from: triple[0], to: triple[2],
            arrows: "to",
            label: mp.applyLabel(triple[1]), font: { background: 'white' },
        });
    })


    // add data edges
    graph.data.forEach(triple => {
        const id = nodes.add({ label: renderData(triple[2]), shape: "box", color: "orange" })[0] as string;

        edges.add({
            from: triple[0], to: id,
            arrows: "to",
            label: mp.applyLabel(triple[1]), font: { background: 'white' }, color: 'orange',
        });
    })

    graphElement.style.width = '100%';
    graphElement.style.height = '500px';

    const options = {
        layout: {
            hierarchical: {
                enabled: true,
            }
        },
        physics: {
            hierarchicalRepulsion: {
                avoidOverlap: 100,
            },
        },
    }
    return new Network(graphElement, { nodes: nodes as unknown as any, edges: edges as unknown as any }, options);
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