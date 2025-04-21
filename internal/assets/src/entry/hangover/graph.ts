import { Graph, instance } from "@viz-js/viz";

type GraphData = {
    triples: Array<[string, string, string]>,
    data: Array<[string, string, string, string]>,
}

// renderData renders data as a string
function renderData(data: unknown, language: string, line: number = 30): string {
    let buf = JSON.stringify(data)
    if (language !== "") {
        buf += "@" + language
    }
    
    if (line <= 0) {
        return buf
    }

    const lines: Array<string> = [];
    while(buf.length > line) {
        lines.push(buf.substring(0, line));
        buf = buf.substring(line);
    }
    if (buf !== "") {
        lines.push(buf);
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
    
    // add the nodes to the underlying dataset
    nodeSet.forEach(node => {
        graph.nodes?.push({
            name: node,
            attributes: {
                label: node,
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
                label: triple[1],
            },
        })
    })

    let counter = 0;

    // add data edges
    data.data.forEach(triple => {
        counter++
        const id = '_:d' + counter.toString()

        graph.nodes?.push({
            name: id,
            attributes: {
                fontname: 'Courier',
                label: renderData(triple[2], triple[3]),
                shape: 'box',
                color: 'orange',
            }
        })

        graph.edges?.push({
            head: id,
            tail: triple[0],
            attributes: {
                label: triple[1],
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
            )
        }, error => {
            statusSpan.innerHTML = '';
            statusSpan.append("failed to render graph")
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

    renderGraph(element, graph);
})