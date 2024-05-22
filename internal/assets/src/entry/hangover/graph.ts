import { Network } from "vis-network";
import { DataSet } from "vis-data";

type GraphData = {
    triples: Array<[string, string, string]>,
    data: Array<[string, string, any]>,
}

function renderGraph(element: HTMLElement, graph: GraphData): Network {
    // TODO: Magically make these
    const namespaces: Record<string, string> = {
        'http://erlangen-crm.org/170309/': 'ecrm',
    }


    const nodes = new DataSet<{id?: string, label: string, shape?: string, color?: string; font?: unknown}>([]);
    const edges = new DataSet<{from: string, to: string, id?: never, label: string, font?: unknown, arrows?: "to"}>();

    // add the nodes that we know of
    const nodeSet = new Set<string>();    
    graph.triples.forEach(triple => {
        nodeSet.add(triple[0]);
        nodeSet.add(triple[2]); 
    });
    graph.data.forEach(triple => {
        nodeSet.add(triple[0]);
    });

    // add the nodes to the underlying dataset
    nodeSet.forEach(node => {
        nodes.add({id: node, label: node, shape: "ellipse"});
    })

    // add the known edges
    graph.triples.forEach(triple => {
        edges.add({
            from: triple[0], to: triple[2], 
            arrows: "to",
            label: triple[1], font: {background: 'white'},
        });
    })


    // add data edges
    graph.data.forEach(triple => {
        console.log("got data triple", triple);
        const id = nodes.add({label: triple[2], shape: "box", color: "orange"})[0] as string;

        edges.add({
            from: triple[0], to: id,
            arrows: "to",
            label: triple[1], font: {background: 'white'},
        });

    })

    
    // create and insert a graph div into the page
    const graphElement = document.createElement("div");
    element.append(graphElement);
    graphElement.style.width = '100%';
    graphElement.style.height = '500px';

    const network = new Network(graphElement, { nodes: nodes as unknown as any, edges: edges as unknown as any}, {});




    return network

}

document.querySelectorAll("script").forEach((script) => {
    // ensure that we have the data-render-graph attribute
    if(script.getAttribute('data-render-graph') !== 'true') {
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

    const ToggleText = 'Show Graph'
    const unToggleText = toggle.innerText || 'Hide'
    let hidden = false
    let initialized = false
    const onclick = (event) => {
        if(event) event.preventDefault()

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